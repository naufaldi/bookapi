package openlibrary

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	httpClient *http.Client
	userAgent  string
	baseURL    string
	limiter    *rate.Limiter
	maxRetries int
}

func NewClient(userAgent string, rps int, maxRetries int) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		userAgent:  userAgent,
		baseURL:    "https://openlibrary.org",
		limiter:    rate.NewLimiter(rate.Every(time.Second/time.Duration(rps)), 1),
		maxRetries: maxRetries,
	}
}

// SearchResponse matches search.json
type SearchResponse struct {
	NumFound int `json:"numFound"`
	Docs     []struct {
		Key              string   `json:"key"`
		Title            string   `json:"title"`
		AuthorNames      []string `json:"author_name"`
		AuthorKeys       []string `json:"author_key"`
		ISBN             []string `json:"isbn"`
		FirstPublishYear int      `json:"first_publish_year"`
		Language         []string `json:"language"`
	} `json:"docs"`
}

type Publisher struct {
	Name string `json:"name"`
}

// BookDetails matches api/books?jscmd=data
type BookDetails struct {
	Title       string      `json:"title"`
	Subtitle    string      `json:"subtitle"`
	Publishers  []Publisher `json:"publishers"`
	PublishDate string      `json:"publish_date"`
	Cover       struct {
		Large string `json:"large"`
	} `json:"cover"`
	Authors []struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	} `json:"authors"`
	Subjects []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"subjects"`
	NumberOfPages int    `json:"number_of_pages"`
	Notes         string `json:"notes"`
}

// AuthorDetails matches authors/{key}.json
type AuthorDetails struct {
	Name         string      `json:"name"`
	PersonalName string      `json:"personal_name"`
	BirthDate    string      `json:"birth_date"`
	Bio          interface{} `json:"bio"` // Can be string or {type: ..., value: ...}
	Photos       []int       `json:"photos"`
}

func (c *Client) SearchBooks(ctx context.Context, subject string, limit int) (*SearchResponse, error) {
	u := fmt.Sprintf("%s/search.json?q=subject:%s&fields=key,title,author_name,author_key,isbn,first_publish_year,language&limit=%d",
		c.baseURL, url.QueryEscape(subject), limit)

	var res SearchResponse
	if err := c.get(ctx, u, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) GetBooksByISBN(ctx context.Context, isbns []string) (map[string]BookDetails, error) {
	if len(isbns) == 0 {
		return nil, nil
	}

	bibkeys := make([]string, len(isbns))
	for i, isbn := range isbns {
		bibkeys[i] = "ISBN:" + isbn
	}

	u := fmt.Sprintf("%s/api/books?bibkeys=%s&jscmd=data&format=json",
		c.baseURL, strings.Join(bibkeys, ","))

	var res map[string]BookDetails
	if err := c.get(ctx, u, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) GetAuthor(ctx context.Context, authorKey string) (*AuthorDetails, error) {
	// authorKey is usually "/authors/OL..." or just "OL..."
	key := strings.TrimPrefix(authorKey, "/authors/")
	u := fmt.Sprintf("%s/authors/%s.json", c.baseURL, key)

	var res AuthorDetails
	if err := c.get(ctx, u, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) get(ctx context.Context, url string, target interface{}) error {
	var lastErr error
	for i := 0; i <= c.maxRetries; i++ {
		if i > 0 {
			// Backoff: 1s, 2s, 4s...
			backoff := time.Duration(1<<uint(i-1)) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				continue
			}
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return json.NewDecoder(resp.Body).Decode(target)
	}
	return fmt.Errorf("after %d retries: %w", c.maxRetries, lastErr)
}

// RawGet is used for caching the raw JSON
func (c *Client) RawGet(ctx context.Context, url string) ([]byte, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
