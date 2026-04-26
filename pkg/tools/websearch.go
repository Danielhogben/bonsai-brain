// Package tools provides pre-built integrations for common agent capabilities.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// TavilySearch performs a web search using the Tavily API.
// Returns a summary of search results.
func TavilySearch(ctx context.Context, query string, maxResults int) (string, error) {
	apiKey := os.Getenv("TAVILY_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("TAVILY_API_KEY not set")
	}
	if maxResults <= 0 {
		maxResults = 5
	}

	body := map[string]any{
		"query":          query,
		"max_results":    maxResults,
		"search_depth":   "basic",
		"include_answer": true,
	}
	b, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.tavily.com/search", strings.NewReader(string(b)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tavily request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Answer   string `json:"answer"`
		Results  []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("tavily decode error: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("tavily error: %s", result.Error)
	}

	var out strings.Builder
	if result.Answer != "" {
		out.WriteString("Answer: ")
		out.WriteString(result.Answer)
		out.WriteString("\n\n")
	}
	out.WriteString("Sources:\n")
	for _, r := range result.Results {
		out.WriteString(fmt.Sprintf("- %s: %s\n  %s\n", r.Title, r.URL, r.Content))
	}
	return out.String(), nil
}
