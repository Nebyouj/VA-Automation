package research

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	maxResponseSize = 2 * 1024 * 1024 // 2 MB
	requestTimeout  = 15 * time.Second
)

// Page contains the useful information extracted from a web page.
type Page struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Crawler fetches and extracts text from public web pages.
type Crawler struct {
	client *http.Client
}

// NewCrawler creates a crawler with a timeout and safe redirect handling.
func NewCrawler() *Crawler {
	client := &http.Client{
		Timeout: requestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			if err := validatePublicURL(req.URL); err != nil {
				return err
			}
			return nil
		},
	}

	return &Crawler{client: client}
}

// Fetch retrieves a public HTTP or HTTPS page and extracts its title and text.
func (c *Crawler) Fetch(ctx context.Context, rawURL string) (Page, error) {
	target, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil {
		return Page{}, fmt.Errorf("invalid URL: %w", err)
	}

	if err := validatePublicURL(target); err != nil {
		return Page{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return Page{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "GoJobQueueResearchBot/1.0")
	req.Header.Set("Accept", "text/html")

	resp, err := c.client.Do(req)
	if err != nil {
		return Page{}, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Page{}, fmt.Errorf("fetch page: unexpected HTTP status %s", resp.Status)
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(contentType, "text/html") {
		return Page{}, fmt.Errorf("unsupported content type %q: expected text/html", contentType)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return Page{}, fmt.Errorf("read page: %w", err)
	}
	if len(body) > maxResponseSize {
		return Page{}, fmt.Errorf("page exceeds the 2 MB response limit")
	}

	if err := ctx.Err(); err != nil {
		return Page{}, err
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return Page{}, fmt.Errorf("parse HTML: %w", err)
	}

	page := Page{URL: resp.Request.URL.String()}
	extractPage(doc, &page)

	return page, nil
}

// validatePublicURL rejects non-HTTP URLs and obvious private or local targets.
func validatePublicURL(u *url.URL) error {
	if u == nil {
		return fmt.Errorf("URL is required")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only HTTP and HTTPS URLs are supported")
	}

	if u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("URL must have a hostname and cannot contain credentials")
	}

	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") {
		return fmt.Errorf("local hostnames are not allowed")
	}

	if ip := net.ParseIP(host); ip != nil {
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() ||
			ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("private or local IP addresses are not allowed")
		}
	}

	return nil
}

// extractPage extracts the title and readable text while skipping non-content elements.
func extractPage(n *html.Node, page *Page) {
	var walk func(*html.Node, bool)
	var text strings.Builder

	walk = func(node *html.Node, inTitle bool) {
		if node.Type == html.ElementNode {
			switch strings.ToLower(node.Data) {
			case "script", "style", "noscript", "svg", "iframe":
				return
			case "title":
				inTitle = true
			}
		}

		if node.Type == html.TextNode {
			value := strings.TrimSpace(node.Data)
			if value != "" {
				if inTitle {
					if page.Title != "" {
						page.Title += " "
					}
					page.Title += value
				} else {
					text.WriteString(value)
					text.WriteByte(' ')
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child, inTitle)
		}
	}

	walk(n, false)
	page.Title = strings.Join(strings.Fields(page.Title), " ")
	page.Text = strings.Join(strings.Fields(text.String()), " ")
}
