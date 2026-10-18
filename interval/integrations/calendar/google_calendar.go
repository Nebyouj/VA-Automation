package calendar

import (
	"context"
	"fmt"
	"net/http"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/nebyoudamtew/go-job-queue/interval/executor"
)

// GoogleCalendarService implements the calendar service used by the job executor.
type GoogleCalendarService struct {
	service *calendar.Service
}

// NewGoogleCalendarService creates an authenticated Google Calendar client.
func NewGoogleCalendarService(
	ctx context.Context,
	oauthManager *OAuthManager,
) (*GoogleCalendarService, error) {
	client, err := oauthManager.HTTPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Google Calendar HTTP client: %w", err)
	}

	googleService, err := calendar.NewService(
		ctx,
		option.WithHTTPClient(client),
	)
	if err != nil {
		return nil, fmt.Errorf("create Google Calendar service: %w", err)
	}

	return &GoogleCalendarService{
		service: googleService,
	}, nil
}

// CreateEvent creates an event on the user's primary calendar.
func (g *GoogleCalendarService) CreateEvent(
	ctx context.Context,
	event executor.CalendarEvent,
	jobID int,
) (string, error) {
	googleEvent := &calendar.Event{
		Summary:     event.Summary,
		Description: event.Description,
		Start: &calendar.EventDateTime{
			DateTime: event.StartTime,
			TimeZone: event.TimeZone,
		},
		End: &calendar.EventDateTime{
			DateTime: event.EndTime,
			TimeZone: event.TimeZone,
		},
	}

	created, err := g.service.Events.
		Insert("primary", googleEvent).
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf(
			"job %d: create Google Calendar event: %w",
			jobID,
			err,
		)
	}

	return created.Id, nil
}

// ListEvents lists events on the user's primary calendar.
func (g *GoogleCalendarService) ListEvents(
	ctx context.Context,
	timeMin string,
	timeMax string,
	maxResults int,
	jobID int,
) ([]executor.CalendarEvent, error) {
	call := g.service.Events.List("primary").
		Context(ctx).
		SingleEvents(true).
		OrderBy("startTime")

	if timeMin != "" {
		call = call.TimeMin(timeMin)
	}

	if timeMax != "" {
		call = call.TimeMax(timeMax)
	}

	if maxResults > 0 {
		call = call.MaxResults(int64(maxResults))
	}

	result, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf(
			"job %d: list Google Calendar events: %w",
			jobID,
			err,
		)
	}

	events := make([]executor.CalendarEvent, 0, len(result.Items))

	for _, item := range result.Items {
		startTime := ""
		endTime := ""
		timeZone := ""

		if item.Start != nil {
			startTime = item.Start.DateTime
			if startTime == "" {
				startTime = item.Start.Date
			}
			timeZone = item.Start.TimeZone
		}

		if item.End != nil {
			endTime = item.End.DateTime
			if endTime == "" {
				endTime = item.End.Date
			}
		}

		events = append(events, executor.CalendarEvent{
			ID:          item.Id,
			Summary:     item.Summary,
			Description: item.Description,
			StartTime:   startTime,
			EndTime:     endTime,
			TimeZone:    timeZone,
		})
	}

	return events, nil
}

// HTTPClient loads the saved token and creates a client that refreshes
// expired access tokens when Google provides a refresh token.
func (m *OAuthManager) HTTPClient(
	ctx context.Context,
) (*http.Client, error) {
	token, err := m.LoadToken()
	if err != nil {
		return nil, fmt.Errorf("load Google Calendar token: %w", err)
	}

	return m.config.Client(ctx, token), nil
}
