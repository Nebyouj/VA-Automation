package executor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nebyoudamtew/go-job-queue/interval/domain/job"
)

type mockResearchService struct {
	page ResearchPage
	err  error
	url  string
}

func (m *mockResearchService) Fetch(
	ctx context.Context,
	rawURL string,
) (ResearchPage, error) {
	m.url = rawURL
	return m.page, m.err
}

func TestExecuteResearchWeb(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		service *mockResearchService
		wantErr bool
		wantURL string
	}{
		{
			name: "successful research",
			payload: map[string]any{
				"url": "https://example.com",
			},
			service: &mockResearchService{
				page: ResearchPage{
					URL:   "https://example.com",
					Title: "Example Domain",
					Text:  "Example text",
				},
			},
			wantURL: "https://example.com",
		},
		{
			name: "research service returns an error",
			payload: map[string]any{
				"url": "https://example.com",
			},
			service: &mockResearchService{
				err: errors.New("fetch failed"),
			},
			wantErr: true,
			wantURL: "https://example.com",
		},
		{
			name:    "missing URL",
			payload: map[string]any{},
			service: &mockResearchService{},
			wantErr: true,
		},
		{
			name: "URL has wrong type",
			payload: map[string]any{
				"url": 123,
			},
			service: &mockResearchService{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Executor{}
			if tt.service != nil {
				e.SetResearchService(tt.service)
			}

			err := e.Execute(context.Background(), job.Job{
				Type:    "research_web",
				Payload: tt.payload,
			})

			if tt.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.service != nil && tt.service.url != tt.wantURL {
				t.Errorf(
					"Fetch URL = %q, want %q",
					tt.service.url,
					tt.wantURL,
				)
			}
			if tt.wantErr && tt.name == "research service returns an error" &&
				!strings.Contains(err.Error(), "fetch failed") {
				t.Errorf("error %q should contain fetch failed", err)
			}
		})
	}
}

func TestExecuteResearchWebWithoutService(t *testing.T) {
	e := &Executor{}

	err := e.Execute(context.Background(), job.Job{
		Type:    "research_web",
		Payload: map[string]any{"url": "https://example.com"},
	})

	if err == nil {
		t.Fatal("expected an error when research service is not configured")
	}
	if !strings.Contains(err.Error(), "research service is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}
