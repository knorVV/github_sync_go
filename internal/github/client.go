package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Repository struct {
	GithubID        int64      `json:"github_id"`
	Name            string     `json:"name"`
	FullName        string     `json:"full_name"`
	Description     string     `json:"description"`
	HTMLURL         string     `json:"html_url"`
	Stars           int        `json:"stars"`
	Language        string     `json:"language"`
	OwnerLogin      string     `json:"owner_login"`
	GithubCreatedAt *time.Time `json:"github_created_at"`
	GithubUpdatedAt *time.Time `json:"github_updated_at"`
}

type Client struct {
	httpClient *http.Client
	token      string
}

func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		token:      token,
	}
}

type searchResponse struct {
	Items []struct {
		ID              int64     `json:"id"`
		Name            string    `json:"name"`
		FullName        string    `json:"full_name"`
		Description     string    `json:"description"`
		HTMLURL         string    `json:"html_url"`
		StargazersCount int       `json:"stargazers_count"`
		Language        string    `json:"language"`
		CreatedAt       time.Time `json:"created_at"`
		UpdatedAt       time.Time `json:"updated_at"`
		Owner           struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"items"`
}

// FetchTrending ищет трендовые репозитории через GitHub Search API.
// Официального /trending endpoint нет, поэтому используем search
// с сортировкой по звёздам и фильтром по дате создания.
func (c *Client) FetchTrending(ctx context.Context, limit int) ([]Repository, error) {
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	q := fmt.Sprintf("created:>%s", weekAgo)

	params := url.Values{}
	params.Set("q", q)
	params.Set("sort", "stars")
	params.Set("order", "desc")
	params.Set("per_page", fmt.Sprintf("%d", min(limit, 100)))

	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://api.github.com/search/repositories?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "github-sync-service")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("rate limited: remaining=%s reset=%s",
			resp.Header.Get("X-RateLimit-Remaining"),
			resp.Header.Get("X-RateLimit-Reset"))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github error: status=%d", resp.StatusCode)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}

	repos := make([]Repository, 0, len(sr.Items))
	for _, item := range sr.Items {
		created := item.CreatedAt
		updated := item.UpdatedAt
		repos = append(repos, Repository{
			GithubID:        item.ID,
			Name:            item.Name,
			FullName:        item.FullName,
			Description:     item.Description,
			HTMLURL:         item.HTMLURL,
			Stars:           item.StargazersCount,
			Language:        item.Language,
			OwnerLogin:      item.Owner.Login,
			GithubCreatedAt: &created,
			GithubUpdatedAt: &updated,
		})
	}
	return repos, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
