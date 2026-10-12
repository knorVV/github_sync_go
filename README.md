# GitHub Sync Service (Go)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.27-00ADD8.svg)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-required-blue.svg)](https://www.postgresql.org)
[![Status: Experimental](https://img.shields.io/badge/status-experimental-red.svg)](#статус-проекта)

Сервис для синхронизации трендовых репозиториев GitHub.
Go-версия [Elixir-реализации](https://github.com/knorVV/github_sync).

> ⚠️ **Экспериментальный проект.** Не продакшн-код.

## Что делает

- Периодически запрашивает GitHub Search API
- Отбирает репозитории, созданные за последние 7 дней
- Сортирует по звёздам, сохраняет в PostgreSQL
- Отдаёт данные через HTTP API
- Предоставляет CLI для работы с сервисом

## Стек

- **Go 1.27.0**
- **PostgreSQL** — хранение репозиториев
- **pgx** — драйвер PostgreSQL с пулом соединений
- **chi** — HTTP-роутер
- **net/http** — встроенный HTTP-клиент для GitHub API

## Архитектура

- `cmd/server` — HTTP-сервер + планировщик
- `cmd/cli` — CLI-клиент для работы с API
- `internal/github` — клиент GitHub Search API
- `internal/storage` — работа с PostgreSQL (pgxpool + batch upsert)
- `internal/scheduler` — планировщик синхронизации на `time.Timer`
- `internal/api` — HTTP-хендлеры
- `internal/config` — конфигурация из переменных окружения

## Конфигурация

Все настройки — через переменные окружения с дефолтами:

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `PORT` | `4000` | Порт HTTP-сервера |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/github_sync` | Подключение к PostgreSQL |
| `SYNC_INTERVAL_MINUTES` | `5` | Интервал синхронизации (минуты) |
| `GITHUB_TOKEN` | (пусто) | Опциональный токен GitHub |

## GitHub API

Официального `/trending` endpoint у GitHub нет, поэтому используется
Search API с фильтром по дате создания:

```
GET https://api.github.com/search/repositories
  ?q=created:>{дата 7 дней назад}
  &sort=stars
  &order=desc
  &per_page=30
```

**Аутентификация (опционально):** токен передаётся через `GITHUB_TOKEN`.
Без токена работает, но с меньшим rate limit (60 запросов/час вместо 5000).

**Обработка ошибок:**

| Статус | Что означает | Что делает клиент |
|---|---|---|
| 200 | OK | Парсит `items`, нормализует |
| 403 | Rate limit | Парсит `X-RateLimit-Remaining` и `X-RateLimit-Reset` |
| 4xx/5xx | Ошибка GitHub | Общая ошибка |
| — | Таймаут HTTP | 15 секунд на запрос |

**Ограничение:** GitHub Search API отдаёт не больше 100 результатов
за запрос, поэтому `per_page = min(limit, 100)`.

## Планировщик без cron

Используется `time.Timer` + `time.AfterFunc` — без cron и без внешних
зависимостей.

- При старте сразу запускается первая синхронизация, затем планируется
  следующая через `interval`
- После каждой синхронизации `scheduleNext()` пересоздаёт таймер
- `ForceSync()` останавливает текущий таймер (`timer.Stop()`) и запускает
  синхронизацию немедленно, после чего ставит новый таймер
- Доступ к таймеру защищён `sync.Mutex` — потому что `ForceSync` может
  вызваться из HTTP-хендлера в горутине, а `scheduleNext` — из горутины таймера
- `Stop()` отменяет контекст (`context.WithCancel`) и останавливает таймер —
  для graceful shutdown
- Каждая синхронизация ограничена таймаутом 30 секунд

**Прямой аналог Elixir-версии:** там то же самое делается через `GenServer` +
`Process.send_after`, но **без мьютексов** — GenServer сериализует сообщения
автоматически. В Go эту работу приходится делать вручную.

## API

| Метод | Путь | Описание | Ответ |
|---|---|---|---|
| GET | `/api/repos` | Все репозитории, сортировка по звёздам | `{"count": N, "data": [...]}` |
| GET | `/api/repos/{id}` | По GitHub ID или имени | Объект репозитория или `{"error": "not_found"}` |
| POST | `/api/sync` | Принудительная синхронизация (сбрасывает таймер) | `202 Accepted`, `{"status": "sync_started"}` |

**Особенности:**

- `GET /api/repos/{id}` — если `id` парсится как число, ищет по GitHub ID,
  иначе — по имени (например, `elixir-lang`).
- `POST /api/sync` — запускает синхронизацию асинхронно (`go ForceSync()`)
  и сразу возвращает `202 Accepted`, не дожидаясь завершения.
- CORS включён (`Access-Control-Allow-Origin: *`) — для удобства тестирования
  с фронтенда или из Postman.

**Примеры:**

```bash
# Список репозиториев
curl http://localhost:4000/api/repos

# Конкретный репозиторий
curl http://localhost:4000/api/repos/elixir-lang

# Форсировать синхронизацию
curl -X POST http://localhost:4000/api/sync
```

## CLI

```bash
go run ./cmd/cli list          # список (топ-20 по звёздам)
go run ./cmd/cli get <name>    # по имени, full_name или GitHub ID
go run ./cmd/cli sync          # форсировать синхронизацию
```

CLI обращается к серверу через HTTP (`http://localhost:4000`) с таймаутом
10 секунд. Ответ на `sync` — `202 Accepted`.

## Запуск

### 1. Поднять PostgreSQL

```bash
docker compose up -d
```

### 2. Применить миграцию

```bash
psql -h localhost -U postgres -d github_sync -f migrations/001_create_repos.sql
```

### 3. Собрать

```bash
go build ./...
go build -o bin/server ./cmd/server
go build -o bin/cli ./cmd/cli
```

### 4. Запустить сервер

```bash
go run ./cmd/server    # или ./bin/server
```

Сервер поднимется на **http://localhost:4000** (порт можно переопределить
через `PORT`).

### 5. В другом терминале — CLI

```bash
go run ./cmd/cli list
go run ./cmd/cli get <name>
go run ./cmd/cli sync
```

## Graceful shutdown

Сервер корректно завершается по `SIGINT` / `SIGTERM`:

1. Ловит сигнал через `signal.Notify`.
2. Даёт 10 секунд на завершение текущих запросов (`srv.Shutdown`).
3. Останавливает планировщик (`sched.Stop()`).
4. Закрывает пул соединений с PostgreSQL.

Таймауты сервера: `ReadTimeout: 10s`, `WriteTimeout: 30s`.

## Сравнение с Elixir-версией

Один и тот же сервис реализован на двух языках. **API полностью совместим** —
эндпоинты, форматы запросов и ответов идентичны.

| | github_sync_go (этот) | [github_sync](https://github.com/knorVV/github_sync) |
|---|---|---|
| Язык | Go 1.27 | Elixir |
| HTTP-роутер | chi | Plug.Router |
| HTTP-клиент | net/http | Req |
| БД | pgx (pgxpool + batch upsert) | Ecto |
| CLI | cmd/cli | escript |
| Конфигурация | ENV (`PORT`, `DATABASE_URL`, ...) | `config/*.exs` |
| Интервал синхронизации | `SYNC_INTERVAL_MINUTES` (минуты) | миллисекунды |
| GitHub API | Search API, `created:>7д`, `sort=stars` | то же |
| User-Agent | `github-sync-service` | то же |
| Формат ответа `/api/repos` | `{"count": N, "data": [...]}` | то же |
| `/api/sync` | `go ForceSync()` → `202` | `GenServer.cast` → `202` |

### Планировщик: ключевое различие

| Аспект | Go | Elixir |
|---|---|---|
| Запуск таймера | `time.AfterFunc` | `Process.send_after` |
| Отмена | `timer.Stop()` | `Process.cancel_timer` |
| Защита состояния | `sync.Mutex` вручную | GenServer сериализует автоматически |
| Graceful shutdown | `context.WithCancel` + `srv.Shutdown` | Остановка GenServer |
| Таймаут синхронизации | `context.WithTimeout(30s)` | — |

В Elixir **не нужны мьютексы**: GenServer обрабатывает сообщения
последовательно, и состояние защищено самим процессом. В Go эту работу
приходится делать вручную через `sync.Mutex`.

Проект — упражнение в сравнении подходов: как одни и те же задачи
решаются в разных экосистемах, но дают **одинаковый внешний API**.

## Статус проекта

Экспериментальный проект. Создан для сравнения подходов Go и Elixir
к одной и той же задаче.

Не предназначен для продакшн-использования.

## Лицензия

Распространяется под лицензией MIT. Полный текст — в файле [LICENSE](LICENSE).

Copyright (c) 2026 knorVV