package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// ---- types - feed ----

type appleFeedRoot struct {
	Feed struct {
		Entry []appleEntry `json:"entry"`
	} `json:"feed"`
}

type labeled struct {
	Label string `json:"label"`
}

type appleEntry struct {
	ID      labeled `json:"id"`
	Updated labeled `json:"updated"`
	Author  struct {
		Name labeled `json:"name"`
	} `json:"author"`
	Rating  labeled `json:"im:rating"`
	Content labeled `json:"content"`
	Title   labeled `json:"title"`
}

// ---- types - errors ----

type HTTPError struct {
	Status int
	Body   string
	URL    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http %d %s: %s", e.Status, e.URL, e.Body)
}

// ---- client with timeout ----

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func feedURL(country, appID string, page int) string {
	return fmt.Sprintf("https://itunes.apple.com/%s/rss/customerreviews/id=%s/sortBy=mostRecent/page=%d/json", country, appID, page)
}

// Fetch a feed page (single attempt)
func fetchPageOnce(ctx context.Context, country, appID string, page int) ([]Review, error) {
	url := feedURL(country, appID, page)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "recent-reviews-backend/1.1")
	req = req.WithContext(ctx)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err // could be net.Error (timeout/temporary)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, &HTTPError{Status: resp.StatusCode, Body: string(body), URL: url}
	}

	var root appleFeedRoot
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	reviews := make([]Review, 0, len(root.Feed.Entry))
	for _, e := range root.Feed.Entry {
		if e.Rating.Label == "" { // App's entry - skip it!
			continue
		}
		t, err := time.Parse(time.RFC3339, e.Updated.Label)
		if err != nil {
			if tt, err2 := time.Parse("2006-01-02T15:04:05Z", e.Updated.Label); err2 == nil {
				t = tt
			} else {
				t = now
			}
		}
		r := Review{
			ID:          e.ID.Label,
			AppID:       appID,
			Country:     country,
			Author:      e.Author.Name.Label,
			Rating:      atoiSafe(e.Rating.Label),
			Title:       e.Title.Label,
			Content:     e.Content.Label,
			SubmittedAt: t.UTC(),
		}
		reviews = append(reviews, r)
	}
	return reviews, nil
}

func atoiSafe(s string) int {
	n := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}

// ---- retry with backoff and jitter ----

func isTimeout(err error) bool {
	if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
		return true
	}
	return false
}

func errorType(err error) string {
	switch e := err.(type) {
	case *HTTPError:
		return fmt.Sprintf("http_status_%d", e.Status)
	default:
		if isTimeout(err) {
			return "network_timeout"
		}
		return "network_error"
	}
}

// Fetch with retry: 3 attempts (1 + 2 retry) with increasing backoff
func FetchPageWithRetry(ctx context.Context, cfg *Config, country, appID string, page int) ([]Review, error) {
	const attempts = 3
	base := 500 * time.Millisecond

	var lastErr error
	for i := 0; i < attempts; i++ {
		revs, err := fetchPageOnce(ctx, country, appID, page)
		if err == nil {
			return revs, nil
		}
		lastErr = err

		// last attempt? exit!
		if i == attempts-1 {
			break
		}

		// backoff with simple jitter
		sleep := time.Duration(1<<i) * base                    // 0:500ms, 1:1s, 2:2s
		jitter := time.Duration(time.Now().UnixNano() % 200e6) // 0-200ms
		select {
		case <-time.After(sleep + jitter):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// all attempts failed: notify webhook (best-effort)
	id := appID + "-" + country
	_ = NotifyWebhook(cfg.WebhookURL, id, errorType(lastErr))
	return nil, lastErr
}
