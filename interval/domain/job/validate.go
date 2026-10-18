package job

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

func Validate(j Job) error {
	if j.RepeatEverySeconds > 0 && strings.TrimSpace(j.CronExpression) != "" {
		return fmt.Errorf(
			"%w: repeat_every_seconds and cron_expression cannot both be set",
			ErrInvalidJob,
		)
	}

	if strings.TrimSpace(j.CronExpression) != "" {
		if _, err := cron.ParseStandard(j.CronExpression); err != nil {
			return fmt.Errorf(
				"%w: invalid cron_expression: %v",
				ErrInvalidJob,
				err,
			)
		}
	}

	if j.RepeatEverySeconds < 0 {
		return fmt.Errorf(
			"%w: repeat_every_seconds cannot be negative",
			ErrInvalidJob,
		)
	}

	switch j.Type {
	case "send_email":
		return validateSendEmail(j)

	case "generate_report":
		return validateGenerateReport(j)

	case "process_file":
		return validateProcessFile(j)

	case "create_calendar_event":
		return validateCreateCalendarEvent(j)

	case "list_calendar_events":
		return validateListCalendarEvents(j)

	case "read_sheet":
		return validateReadSheet(j)

	case "append_sheet_rows", "update_sheet_values":
		return validateWriteSheet(j)

	case "failing_job":
		return nil

	case "research_web":
		return validateResearchWeb(j)

	default:
		return fmt.Errorf("unknown job type: %s", j.Type)
	}
}

var ErrInvalidJob = errors.New("invalid job")

func validateSendEmail(j Job) error {
	to, ok := j.Payload["to"].(string)
	if !ok || to == "" {
		return fmt.Errorf("email recipient is required")
	}

	return nil
}

func validateGenerateReport(j Job) error {
	reportType, ok := j.Payload["report_type"].(string)
	if !ok || reportType == "" {
		return fmt.Errorf("report_type is required")
	}

	userID, ok := j.Payload["user_id"].(float64)
	if !ok {
		return fmt.Errorf("user_id is required")
	}

	if userID <= 0 {
		return fmt.Errorf("user_id must be greater than 0")
	}

	_ = reportType

	return nil
}

func validateProcessFile(j Job) error {
	fileName, ok := j.Payload["file_name"].(string)
	if !ok || strings.TrimSpace(fileName) == "" {
		return fmt.Errorf("%w: file_name is required", ErrInvalidJob)
	}

	operation, ok := j.Payload["operation"].(string)
	if !ok || strings.TrimSpace(operation) == "" {
		return fmt.Errorf("%w: operation is required", ErrInvalidJob)
	}

	switch operation {
	case "read_text", "text_stats", "copy":
		// These operations read a file from the input directory.

	case "write_text":
		content, ok := j.Payload["content"].(string)
		if !ok {
			return fmt.Errorf(
				"%w: content must be a string for write_text",
				ErrInvalidJob,
			)
		}

		if len(content) > 10*1024*1024 {
			return fmt.Errorf(
				"%w: content exceeds the 10 MB size limit",
				ErrInvalidJob,
			)
		}

	default:
		return fmt.Errorf(
			"%w: unsupported file operation %q",
			ErrInvalidJob,
			operation,
		)
	}

	return nil
}

func validateCreateCalendarEvent(j Job) error {
	summary, ok := j.Payload["summary"].(string)
	if !ok || strings.TrimSpace(summary) == "" {
		return fmt.Errorf("%w: event summary is required", ErrInvalidJob)
	}

	startTime, ok := j.Payload["start_time"].(string)
	if !ok || startTime == "" {
		return fmt.Errorf("%w: event start_time is required", ErrInvalidJob)
	}

	endTime, ok := j.Payload["end_time"].(string)
	if !ok || endTime == "" {
		return fmt.Errorf("%w: event end_time is required", ErrInvalidJob)
	}

	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return fmt.Errorf("%w: start_time must be RFC3339", ErrInvalidJob)
	}

	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return fmt.Errorf("%w: end_time must be RFC3339", ErrInvalidJob)
	}

	if !end.After(start) {
		return fmt.Errorf("%w: end_time must be after start_time", ErrInvalidJob)
	}

	return nil
}

func validateListCalendarEvents(j Job) error {
	for _, field := range []string{"time_min", "time_max"} {
		value, exists := j.Payload[field]
		if !exists {
			continue
		}

		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("%w: %s must be a string", ErrInvalidJob, field)
		}

		if s != "" {
			if _, err := time.Parse(time.RFC3339, s); err != nil {
				return fmt.Errorf(
					"%w: %s must be RFC3339",
					ErrInvalidJob,
					field,
				)
			}
		}
	}

	if min, ok := j.Payload["time_min"].(string); ok && min != "" {
		if max, ok := j.Payload["time_max"].(string); ok && max != "" {
			start, _ := time.Parse(time.RFC3339, min)
			end, _ := time.Parse(time.RFC3339, max)

			if !end.After(start) {
				return fmt.Errorf(
					"%w: time_max must be after time_min",
					ErrInvalidJob,
				)
			}
		}
	}

	if raw, exists := j.Payload["max_results"]; exists {
		n, ok := raw.(float64)
		if !ok || n != float64(int(n)) || n < 1 || n > 250 {
			return fmt.Errorf(
				"%w: max_results must be an integer between 1 and 250",
				ErrInvalidJob,
			)
		}
	}

	return nil
}

func validateReadSheet(j Job) error {
	if err := validateSpreadsheetID(j); err != nil {
		return err
	}

	readRange, ok := j.Payload["range"].(string)
	if !ok || strings.TrimSpace(readRange) == "" {
		return fmt.Errorf("%w: range is required for read_sheet", ErrInvalidJob)
	}

	return nil
}

func validateWriteSheet(j Job) error {
	if err := validateSpreadsheetID(j); err != nil {
		return err
	}

	writeRange, ok := j.Payload["range"].(string)
	if !ok || strings.TrimSpace(writeRange) == "" {
		return fmt.Errorf("%w: range is required for spreadsheet write", ErrInvalidJob)
	}

	rows, ok := j.Payload["values"].([]any)
	if !ok || len(rows) == 0 {
		return fmt.Errorf("%w: values must contain at least one row", ErrInvalidJob)
	}

	for i, rawRow := range rows {
		row, ok := rawRow.([]any)
		if !ok || len(row) == 0 {
			return fmt.Errorf("%w: row %d must be a non-empty array", ErrInvalidJob, i)
		}
	}

	return nil
}

func validateSpreadsheetID(j Job) error {
	spreadsheetID, ok := j.Payload["spreadsheet_id"].(string)
	if !ok || strings.TrimSpace(spreadsheetID) == "" {
		return fmt.Errorf("%w: spreadsheet_id is required", ErrInvalidJob)
	}

	return nil
}

func validateResearchWeb(j Job) error {
	rawURL, ok := j.Payload["url"].(string)
	if !ok || strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("research URL is required")
	}

	parsedURL, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("invalid research URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("research URL must use HTTP or HTTPS")
	}

	if parsedURL.Hostname() == "" || parsedURL.User != nil {
		return fmt.Errorf("research URL must have a hostname and cannot contain credentials")
	}

	return nil
}
