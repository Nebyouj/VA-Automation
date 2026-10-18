package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nebyoudamtew/go-job-queue/interval/domain/job"
	"github.com/nebyoudamtew/go-job-queue/interval/integrations/email"
)

type EmailService interface {
	SendEmail(ctx context.Context, to, subject string, bodt string, html string, jobID int) error
}

type ReportGenerator interface {
	GenerateReport(ctx context.Context, reportType string, userID float64, jobID int) error
}

type FileProcessor interface {
	ProcessFile(ctx context.Context, fileName, operation string, content string, jobID int) error
}

type CalendarEvent struct {
	ID          string `json:"id,omitempty"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	TimeZone    string `json:"time_zone,omitempty"`
}

type CalendarService interface {
	CreateEvent(
		ctx context.Context,
		event CalendarEvent,
		jobID int,
	) (string, error)

	ListEvents(
		ctx context.Context,
		timeMin string,
		timeMaz string,
		maxResults int,
		jobID int,
	) ([]CalendarEvent, error)
}

type SheetsService interface {
	ReadValues(
		ctx context.Context,
		spreadsheetID string,
		readRange string,
	) ([][]interface{}, error)

	AppendRows(
		ctx context.Context,
		spreadsheetID string,
		writeRange string,
		values [][]interface{},
	) error

	UpdateValues(
		ctx context.Context,
		spreadsheetID string,
		writeRange string,
		values [][]interface{},
	) error
}

type Executor struct {
	EmailService    EmailService
	ReportGenerator ReportGenerator
	FileProcessor   FileProcessor
	CalendarService CalendarService
	SheetsService   SheetsService
	EmailRenderer   *email.TemplateRenderer
	ResearchService ResearchService
}

func NewExecutor(
	email EmailService,
	report ReportGenerator,
	file FileProcessor,
	calendar CalendarService,
	sheets SheetsService,
	emailRenderer *email.TemplateRenderer,
) *Executor {
	return &Executor{
		EmailService:    email,
		ReportGenerator: report,
		FileProcessor:   file,
		CalendarService: calendar,
		SheetsService:   sheets,
		EmailRenderer:   emailRenderer,
	}
}

func (e *Executor) SetResearchService(service ResearchService) {
	e.ResearchService = service
}

func (e *Executor) Execute(
	ctx context.Context,
	j job.Job,
) error {
	switch j.Type {
	case "send_email":
		return e.sendEmail(ctx, j)
	case "generate_report":
		return e.generateReport(ctx, j)
	case "process_file":
		return e.processFile(ctx, j)
	case "failing_job":
		return e.failingJob(ctx, j)
	case "create_calendar_event":
		return e.createCalendarEvent(ctx, j)
	case "list_calendar_events":
		return e.listCalendarEvents(ctx, j)
	case "read_sheet":
		return e.readSheet(ctx, j)
	case "append_sheet_rows":
		return e.appendSheetRows(ctx, j)
	case "update_sheet_values":
		return e.updateSheetValues(ctx, j)
	case "research_web":
		if e.ResearchService == nil {
			return fmt.Errorf("research service is not configured")
		}

		rawURL, ok := j.Payload["url"].(string)
		if !ok || strings.TrimSpace(rawURL) == "" {
			return fmt.Errorf("research URL is required")
		}

		page, err := e.ResearchService.Fetch(ctx, rawURL)
		if err != nil {
			return fmt.Errorf("research webpage: %w", err)
		}

		fmt.Printf(
			"Research completed: %s\nTitle: %s\nExtracted characters: %d\n",
			page.URL,
			page.Title,
			len(page.Text),
		)

		return nil
	default:
		return fmt.Errorf("unknown job type: %s", j.Type)
	}
}

func (e *Executor) sendEmail(
	ctx context.Context,
	j job.Job,
) error {
	to, ok := j.Payload["to"].(string)
	if !ok || to == "" {
		return fmt.Errorf("email recipient is required")
	}

	subject, ok := j.Payload["subject"].(string)
	if !ok || subject == "" {
		subject = "No Subject"
	}

	// Check whether this job uses a template.
	templateName, hasTemplate := j.Payload["template"].(string)

	if hasTemplate && templateName != "" {
		if e.EmailRenderer == nil {
			return fmt.Errorf("email template renderer is not configured")
		}

		variables := email.TemplateData{}

		if rawVariables, ok := j.Payload["variables"].(map[string]any); ok {
			for key, value := range rawVariables {
				variables[key] = fmt.Sprint(value)
			}
		}

		body, html, err := e.EmailRenderer.Render(
			templateName,
			variables,
		)
		if err != nil {
			return fmt.Errorf(
				"failed to render email template: %w",
				err,
			)
		}

		if e.EmailService == nil {
			return fmt.Errorf("email service not configured")
		}

		return e.EmailService.SendEmail(
			ctx,
			to,
			subject,
			body,
			html,
			j.ID,
		)
	}

	body, ok := j.Payload["body"].(string)
	if !ok || body == "" {
		body = ""
	}

	html, ok := j.Payload["html"].(string)
	if !ok {
		html = ""
	}

	if e.EmailService != nil {
		return e.EmailService.SendEmail(ctx, to, subject, body, html, j.ID)
	}

	return fmt.Errorf("email service not configured")
}

func (e *Executor) generateReport(
	ctx context.Context,
	j job.Job,
) error {
	reportType, ok := j.Payload["report_type"].(string)
	if !ok || reportType == "" {
		reportType = "general"
	}

	userID, ok := j.Payload["user_id"].(float64)
	if !ok {
		userID = 0
	}

	if e.ReportGenerator != nil {
		return e.ReportGenerator.GenerateReport(ctx, reportType, userID, j.ID)
	}

	return fmt.Errorf("report generator not configured")
}

func (e *Executor) processFile(
	ctx context.Context,
	j job.Job,
) error {
	fileName, ok := j.Payload["file_name"].(string)
	if !ok || fileName == "" {
		return fmt.Errorf("file_name is required")
	}

	operation, ok := j.Payload["operation"].(string)
	if !ok || operation == "" {
		operation = "process"
	}

	content, ok := j.Payload["content"].(string)
	if !ok || operation == "" {
		operation = "process"
	}

	if e.FileProcessor != nil {
		return e.FileProcessor.ProcessFile(ctx, fileName, operation, content, j.ID)
	}

	return fmt.Errorf("file processor not configured")
}

func (e *Executor) failingJob(
	ctx context.Context,
	j job.Job,
) error {
	fmt.Printf(
		"Executing failing job %d ... \n",
		j.ID,
	)

	select {
	case <-time.After(1 * time.Second):
		return fmt.Errorf("simulated job failure")

	case <-ctx.Done():
		fmt.Printf(
			"failing job %d cancelled... \n",
			j.ID,
		)

		return ctx.Err()
	}
}

func (e *Executor) createCalendarEvent(
	ctx context.Context,
	j job.Job,
) error {
	if e.CalendarService == nil {
		return fmt.Errorf("calendar service not configured")
	}

	summary, ok := j.Payload["summary"].(string)
	if !ok || summary == "" {
		return fmt.Errorf("event summary is required")
	}

	startTime, ok := j.Payload["start_time"].(string)
	if !ok || startTime == "" {
		return fmt.Errorf("event start_time is required")
	}

	endTime, ok := j.Payload["end_time"].(string)
	if !ok || endTime == "" {
		return fmt.Errorf("event end_time is required")
	}

	description, _ := j.Payload["description"].(string)
	timeZone, _ := j.Payload["time_zone"].(string)

	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return fmt.Errorf("invalid start_time: %w", err)
	}

	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return fmt.Errorf("invalid end_time: %w", err)
	}

	if !end.After(start) {
		return fmt.Errorf("end_time must be after start_time")
	}

	event := CalendarEvent{
		Summary:     summary,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
		TimeZone:    timeZone,
	}

	eventID, err := e.CalendarService.CreateEvent(
		ctx,
		event,
		j.ID,
	)
	if err != nil {
		return fmt.Errorf("create calendar event: %w", err)
	}

	fmt.Printf(
		"Created calendar event %q (event ID %s, job %d)\n",
		summary,
		eventID,
		j.ID,
	)

	return nil
}

func (e *Executor) listCalendarEvents(
	ctx context.Context,
	j job.Job,
) error {
	if e.CalendarService == nil {
		return fmt.Errorf("calendar service not configured")
	}

	timeMin, _ := j.Payload["time_min"].(string)
	timeMax, _ := j.Payload["time_max"].(string)

	maxResults := 10
	if raw, ok := j.Payload["max_results"].(float64); ok {
		if raw != float64(int(raw)) || raw < 1 || raw > 250 {
			return fmt.Errorf("max_results must be an integer between 1 and 250")
		}
		maxResults = int(raw)
	}

	events, err := e.CalendarService.ListEvents(
		ctx,
		timeMin,
		timeMax,
		maxResults,
		j.ID,
	)
	if err != nil {
		return fmt.Errorf("list calendar events: %w", err)
	}

	fmt.Printf(
		"Retrieved %d calendar events for job %d\n",
		len(events),
		j.ID,
	)

	for _, event := range events {
		fmt.Printf(
			"Event: %s | Start: %s | End: %s | ID: %s\n",
			event.Summary,
			event.StartTime,
			event.EndTime,
			event.ID,
		)
	}

	return nil
}

func (e *Executor) readSheet(ctx context.Context, j job.Job) error {
	if e.SheetsService == nil {
		return fmt.Errorf("Google Sheets service is not configured")
	}

	spreadsheetID, ok := j.Payload["spreadsheet_id"].(string)
	if !ok || spreadsheetID == "" {
		return fmt.Errorf("job %d: spreadsheet_id is required", j.ID)
	}

	readRange, ok := j.Payload["range"].(string)
	if !ok || readRange == "" {
		return fmt.Errorf("job %d: range is required", j.ID)
	}

	values, err := e.SheetsService.ReadValues(
		ctx,
		spreadsheetID,
		readRange,
	)
	if err != nil {
		return err
	}

	fmt.Printf(
		"Read %d rows from spreadsheet %s, range %s, job %d\n",
		len(values),
		spreadsheetID,
		readRange,
		j.ID,
	)

	for _, row := range values {
		fmt.Printf("Row: %v\n", row)
	}

	return nil
}

func (e *Executor) appendSheetRows(ctx context.Context, j job.Job) error {
	if e.SheetsService == nil {
		return fmt.Errorf("Google Sheets service is not configured")
	}

	spreadsheetID, ok := j.Payload["spreadsheet_id"].(string)
	if !ok || spreadsheetID == "" {
		return fmt.Errorf("job %d: spreadsheet_id is required", j.ID)
	}

	writeRange, ok := j.Payload["range"].(string)
	if !ok || writeRange == "" {
		return fmt.Errorf("job %d: range is required", j.ID)
	}

	values, err := parseSheetRows(j.Payload["values"])
	if err != nil {
		return fmt.Errorf("job %d: invalid values: %w", j.ID, err)
	}

	if err := e.SheetsService.AppendRows(
		ctx,
		spreadsheetID,
		writeRange,
		values,
	); err != nil {
		return err
	}

	fmt.Printf(
		"Appended %d rows to spreadsheet %s, range %s, job %d\n",
		len(values),
		spreadsheetID,
		writeRange,
		j.ID,
	)

	return nil
}

func (e *Executor) updateSheetValues(ctx context.Context, j job.Job) error {
	if e.SheetsService == nil {
		return fmt.Errorf("Google Sheets service is not configured")
	}

	spreadsheetID, ok := j.Payload["spreadsheet_id"].(string)
	if !ok || spreadsheetID == "" {
		return fmt.Errorf("job %d: spreadsheet_id is required", j.ID)
	}

	writeRange, ok := j.Payload["range"].(string)
	if !ok || writeRange == "" {
		return fmt.Errorf("job %d: range is required", j.ID)
	}

	values, err := parseSheetRows(j.Payload["values"])
	if err != nil {
		return fmt.Errorf("job %d: invalid values: %w", j.ID, err)
	}

	if err := e.SheetsService.UpdateValues(
		ctx,
		spreadsheetID,
		writeRange,
		values,
	); err != nil {
		return err
	}

	fmt.Printf(
		"Updated %d rows in spreadsheet %s, range %s, job %d\n",
		len(values),
		spreadsheetID,
		writeRange,
		j.ID,
	)

	return nil
}

// parseSheetRows converts JSON-decoded payload values into spreadsheet rows.
// JSON arrays decoded into map[string]any are represented as []any.
func parseSheetRows(raw any) ([][]interface{}, error) {
	rows, ok := raw.([]any)
	if !ok || len(rows) == 0 {
		return nil, fmt.Errorf("values must be a non-empty array of rows")
	}

	result := make([][]interface{}, 0, len(rows))

	for i, rawRow := range rows {
		row, ok := rawRow.([]any)
		if !ok || len(row) == 0 {
			return nil, fmt.Errorf("row %d must be a non-empty array", i)
		}

		convertedRow := make([]interface{}, len(row))
		for j, value := range row {
			convertedRow[j] = value
		}

		result = append(result, convertedRow)
	}

	return result, nil
}
