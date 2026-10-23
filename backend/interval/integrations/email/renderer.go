package email

import (
	"fmt"
	"os"
	"strings"
)

type TemplateData map[string]string

type TemplateRenderer struct {
	basePath string
}

func NewTemplateRenderer(basePath string) *TemplateRenderer {
	return &TemplateRenderer{
		basePath: basePath,
	}
}

func (r *TemplateRenderer) Render(
	templateName string,
	data TemplateData,
) (string, string, error) {
	htmlPath := fmt.Sprintf(
		"%s/%s.html",
		r.basePath,
		templateName,
	)

	textPath := fmt.Sprintf(
		"%s/%s.txt",
		r.basePath,
		templateName,
	)

	html, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", "", fmt.Errorf(
			"failed to read HTML template: %w",
			err,
		)
	}

	text, err := os.ReadFile(textPath)
	if err != nil {
		return "", "", fmt.Errorf(
			"failed to read text template: %w",
			err,
		)
	}

	renderedHTML := replaceVariables(string(html), data)
	renderedText := replaceVariables(string(text), data)

	return renderedText, renderedHTML, nil
}

func replaceVariables(
	content string,
	data TemplateData,
) string {
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		content = strings.ReplaceAll(
			content,
			placeholder,
			value,
		)
	}

	return content
}
