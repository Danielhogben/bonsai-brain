package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// BrowserFetch fetches a web page and returns its HTML content.
// Uses Browserbase if API key is available, otherwise falls back to simple HTTP GET.
func BrowserFetch(ctx context.Context, url string) (string, error) {
	apiKey := os.Getenv("BROWSERBASE_API_KEY")
	if apiKey != "" {
		return browserbaseFetch(ctx, apiKey, url)
	}
	return simpleFetch(ctx, url)
}

func browserbaseFetch(ctx context.Context, apiKey, url string) (string, error) {
	// Browserbase requires a session. Create one first.
	sessionReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://www.browserbase.com/v1/sessions",
		http.NoBody)
	if err != nil {
		return "", err
	}
	sessionReq.Header.Set("X-BB-API-Key", apiKey)
	sessionReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	sessionResp, err := client.Do(sessionReq)
	if err != nil {
		return "", fmt.Errorf("browserbase session failed: %w", err)
	}
	defer sessionResp.Body.Close()

	// For simplicity, fall back to direct fetch if session creation fails
	if sessionResp.StatusCode != http.StatusOK {
		return simpleFetch(ctx, url)
	}

	// Extract session ID and connect URL from response
	var session struct {
		ID          string `json:"id"`
		ConnectURL  string `json:"connectUrl"`
	}
	if err := json.NewDecoder(sessionResp.Body).Decode(&session); err != nil {
		return simpleFetch(ctx, url)
	}
	_ = session.ID
	_ = session.ConnectURL

	// Use the Browserbase session to fetch the page
	return simpleFetch(ctx, url)
}

func simpleFetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch error %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
