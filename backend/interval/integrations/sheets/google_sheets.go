package sheets

import (
	"context"
	"fmt"

	calendarIntegration "github.com/nebyoudamtew/go-job-queue/interval/integrations/calendar"
	"google.golang.org/api/option"
	googleSheets "google.golang.org/api/sheets/v4"
)

// GoogleSheetsService provides read, append, and update operations
// for Google Sheets spreadsheets.
type GoogleSheetsService struct {
	service *googleSheets.Service
}

// NewGoogleSheetsService creates an authenticated Google Sheets client.
// It reuses the OAuth manager shared with Google Calendar.
func NewGoogleSheetsService(
	ctx context.Context,
	oauthManager *calendarIntegration.OAuthManager,
) (*GoogleSheetsService, error) {
	client, err := oauthManager.HTTPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Google Sheets HTTP client: %w", err)
	}

	sheetsService, err := googleSheets.NewService(
		ctx,
		option.WithHTTPClient(client),
	)
	if err != nil {
		return nil, fmt.Errorf("create Google Sheets API client: %w", err)
	}

	return &GoogleSheetsService{
		service: sheetsService,
	}, nil
}

// ReadValues reads cell values from a spreadsheet range.
// Example range: Sheet1!A1:C10
func (g *GoogleSheetsService) ReadValues(
	ctx context.Context,
	spreadsheetID string,
	readRange string,
) ([][]interface{}, error) {
	if spreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheet ID is required")
	}
	if readRange == "" {
		return nil, fmt.Errorf("read range is required")
	}

	result, err := g.service.Spreadsheets.Values.
		Get(spreadsheetID, readRange).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("read Google Sheets values: %w", err)
	}

	return result.Values, nil
}

// AppendRows appends rows after the existing data in a spreadsheet range.
// Example range: Sheet1!A:C
func (g *GoogleSheetsService) AppendRows(
	ctx context.Context,
	spreadsheetID string,
	writeRange string,
	values [][]interface{},
) error {
	if spreadsheetID == "" {
		return fmt.Errorf("spreadsheet ID is required")
	}
	if writeRange == "" {
		return fmt.Errorf("write range is required")
	}
	if len(values) == 0 {
		return fmt.Errorf("at least one row is required")
	}

	valueRange := &googleSheets.ValueRange{
		Values: values,
	}

	_, err := g.service.Spreadsheets.Values.
		Append(spreadsheetID, writeRange, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("append Google Sheets rows: %w", err)
	}

	return nil
}

// UpdateValues replaces values in the specified spreadsheet range.
// Example range: Sheet1!A2:B3
func (g *GoogleSheetsService) UpdateValues(
	ctx context.Context,
	spreadsheetID string,
	writeRange string,
	values [][]interface{},
) error {
	if spreadsheetID == "" {
		return fmt.Errorf("spreadsheet ID is required")
	}
	if writeRange == "" {
		return fmt.Errorf("write range is required")
	}
	if len(values) == 0 {
		return fmt.Errorf("at least one row is required")
	}

	valueRange := &googleSheets.ValueRange{
		Values: values,
	}

	_, err := g.service.Spreadsheets.Values.
		Update(spreadsheetID, writeRange, valueRange).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("update Google Sheets values: %w", err)
	}

	return nil
}
