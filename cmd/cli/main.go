package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const baseURL = "http://localhost:4000"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		list()
	case "get":
		if len(os.Args) < 3 {
			fmt.Println("Usage: cli get <name-or-id>")
			os.Exit(1)
		}
		get(os.Args[2])
	case "sync":
		sync()
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`GitHub Sync CLI

Usage:
  cli list          List all repositories
  cli get <name>    Get repo by name or ID
  cli sync          Force sync with GitHub (resets timer)`)
}

func list() {
	var resp struct {
		Count int `json:"count"`
		Data  []struct {
			FullName string `json:"full_name"`
			Stars    int    `json:"stars"`
			Language string `json:"language"`
		} `json:"data"`
	}

	if err := getJSON("/api/repos", &resp); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Total repositories: %d\n\n", resp.Count)
	for i, r := range resp.Data {
		if i >= 20 {
			break
		}
		lang := r.Language
		if lang == "" {
			lang = "n/a"
		}
		fmt.Printf("  %s — %d ⭐ (%s)\n", r.FullName, r.Stars, lang)
	}
}

func get(name string) {
	var repo struct {
		FullName    string `json:"full_name"`
		Description string `json:"description"`
		Stars       int    `json:"stars"`
		Language    string `json:"language"`
		OwnerLogin  string `json:"owner_login"`
		HTMLURL     string `json:"html_url"`
	}

	if err := getJSON("/api/repos/"+name, &repo); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(`%s
----------------------------------------
Description: %s
Stars:       %d
Language:    %s
Owner:       %s
URL:         %s
`, repo.FullName, repo.Description, repo.Stars, repo.Language, repo.OwnerLogin, repo.HTMLURL)
}

func sync() {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(baseURL+"/api/sync", "application/json", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		fmt.Println("Sync started. Timer reset.")
	} else {
		fmt.Printf("Error: HTTP %d\n", resp.StatusCode)
	}
}

func getJSON(path string, target any) error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(baseURL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
