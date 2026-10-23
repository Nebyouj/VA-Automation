package executor

import "context"

// ResearchPage represents the useful content extracted from a web page.
type ResearchPage struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// ResearchService defines the behavior required by the job executor.
type ResearchService interface {
	Fetch(ctx context.Context, rawURL string) (ResearchPage, error)
}
