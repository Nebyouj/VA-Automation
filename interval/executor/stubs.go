package executor

import (
	"context"
	"fmt"
	"time"
)

// StubEmailService is a placeholder implementation that simulates sending email.
// Replace with a real implementation (e.g., Gmail, SMTP) in the future.
type StubEmailService struct{}

func (s *StubEmailService) SendEmail(ctx context.Context, to, subject string, body string, html string, jobID int) error {
	fmt.Printf(
		"Sending email to %s with subject %q (job %d)\n",
		to,
		subject,
		jobID,
	)

	select {
	case <-time.After(2 * time.Second):
		fmt.Printf(
			"Email sent to %s (job %d)\n",
			to,
			jobID,
		)
		return nil

	case <-ctx.Done():
		fmt.Printf(
			"Email job %d cancelled...\n",
			jobID,
		)
		return ctx.Err()
	}
}

// StubReportGenerator is a placeholder implementation that simulates report generation.
type StubReportGenerator struct{}

func (s *StubReportGenerator) GenerateReport(ctx context.Context, reportType string, userID float64, jobID int) error {
	fmt.Printf(
		"Generating %s report for user %.0f (job %d)\n",
		reportType,
		userID,
		jobID,
	)

	select {
	case <-time.After(5 * time.Second):
		fmt.Printf(
			"Report generated for user %.0f (job %d)\n",
			userID,
			jobID,
		)
		return nil

	case <-ctx.Done():
		fmt.Printf(
			"Report job %d cancelled...\n",
			jobID,
		)
		return ctx.Err()
	}
}

// StubFileProcessor is a placeholder implementation that simulates file processing.
type StubFileProcessor struct{}

func (s *StubFileProcessor) ProcessFile(
	ctx context.Context,
	fileName, operation, content string,
	jobID int,
) error {
	fmt.Printf(
		"Processing file %s with operation %q (job %d)\n",
		fileName,
		operation,
		jobID,
	)

	select {
	case <-time.After(3 * time.Second):
		fmt.Printf(
			"File %s processed successfully (job %d)\n",
			fileName,
			jobID,
		)
		return nil

	case <-ctx.Done():
		fmt.Printf(
			"File job %d cancelled...\n",
			jobID,
		)
		return ctx.Err()
	}
}
