package jina

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var urlRegex = regexp.MustCompile(`https?://\S+`)

type Client struct {
	baseURL    string
	httpClient *http.Client
	maxLength  int
}

func NewClient() *Client {
	return &Client{
		baseURL:    "https://r.jina.ai",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		maxLength:  8000,
	}
}

func (c *Client) ExtractURLs(text string) []string {
	matches := urlRegex.FindAllString(text, -1)
	var urls []string
	for _, u := range matches {
		u = strings.TrimRight(u, ".,;:!?)")
		if !strings.Contains(u, "r.jina.ai") && !strings.Contains(u, "s.jina.ai") {
			urls = append(urls, u)
		}
	}
	return urls
}

func (c *Client) FetchURL(rawURL string) (string, error) {
	jinaURL := c.baseURL + "/" + rawURL

	req, err := http.NewRequest("GET", jinaURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "text/markdown")
	req.Header.Set("X-Respond-With", "markdown")
	req.Header.Set("X-Max-Tokens", fmt.Sprintf("%d", c.maxLength))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	content := string(body)
	if len(content) > c.maxLength {
		content = content[:c.maxLength] + "\n\n[... truncated]"
	}

	return content, nil
}

func (c *Client) FetchURLs(urls []string) map[string]string {
	results := make(map[string]string)
	for _, u := range urls {
		if len(results) >= 3 {
			break
		}
		content, err := c.FetchURL(u)
		if err != nil {
			continue
		}
		results[u] = content
	}
	return results
}

func (c *Client) Search(query string) (string, error) {
	encodedQuery := url.QueryEscape(query)
	searchURL := "https://s.jina.ai/" + encodedQuery

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "text/markdown")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	content := string(body)
	if len(content) > c.maxLength {
		content = content[:c.maxLength] + "\n\n[... truncated]"
	}

	return content, nil
}

func (c *Client) BuildURLContext(urls []string) string {
	if len(urls) == 0 {
		return ""
	}

	results := c.FetchURLs(urls)
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n--- Referenced URLs ---\n")
	for u, content := range results {
		sb.WriteString(fmt.Sprintf("\n## %s\n%s\n", u, content))
	}

	return sb.String()
}
