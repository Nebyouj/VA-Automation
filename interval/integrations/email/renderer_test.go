package email

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateRenderer(t *testing.T) {
	dir := t.TempDir()

	html := `
		<h1>Hello {{name}}</h1>
		<p>Meeting with {{company}}</p>
	`

	text := `
		Hello {{name}}

		Meeting with {{company}}
	`

	if err := os.WriteFile(
		filepath.Join(dir, "test.html"),
		[]byte(html),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(dir, "test.txt"),
		[]byte(text),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	renderer := NewTemplateRenderer(dir)

	renderedText, renderedHTML, err := renderer.Render(
		"test",
		TemplateData{
			"name":    "Nebyou",
			"company": "Acme",
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(renderedHTML, "Nebyou") {
		t.Fatal("name was not rendered in HTML")
	}

	if !strings.Contains(renderedHTML, "Acme") {
		t.Fatal("company was not rendered in HTML")
	}

	if !strings.Contains(renderedText, "Nebyou") {
		t.Fatal("name was not rendered in text")
	}

	if !strings.Contains(renderedText, "Acme") {
		t.Fatal("company was not rendered in text")
	}
}
