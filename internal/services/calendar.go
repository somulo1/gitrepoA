package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// CalendarService handles Google Calendar integration
type CalendarService struct {
	service *calendar.Service
	config  *oauth2.Config
}

// CalendarEvent represents a calendar event
type CalendarEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Location    string    `json:"location"`
	MeetingURL  string    `json:"meetingUrl"`
	Attendees   []string  `json:"attendees"`
}

// NewCalendarService creates a new calendar service instance
func NewCalendarService(credentialsJSON []byte) (*CalendarService, error) {
	config, err := google.ConfigFromJSON(credentialsJSON, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &CalendarService{
		config: config,
	}, nil
}

// InitializeWithToken initializes the service with an OAuth token
func (cs *CalendarService) InitializeWithToken(token *oauth2.Token) error {
	ctx := context.Background()
	client := cs.config.Client(ctx, token)

	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create calendar service: %w", err)
	}

	cs.service = service
	return nil
}

// CreateEvent creates a new calendar event
func (cs *CalendarService) CreateEvent(calendarID string, event *CalendarEvent) (*calendar.Event, error) {
	if cs.service == nil {
		return nil, fmt.Errorf("calendar service not initialized")
	}

	// Convert attendees to calendar attendees
	var attendees []*calendar.EventAttendee
	for _, email := range event.Attendees {
		attendees = append(attendees, &calendar.EventAttendee{
			Email: email,
		})
	}

	// Use East Africa Time (EAT)
	eat, _ := time.LoadLocation("Africa/Nairobi")

	// Create the calendar event
	calendarEvent := &calendar.Event{
		Summary:     event.Title,
		Description: event.Description,
		Location:    event.Location,
		Start: &calendar.EventDateTime{
			DateTime: event.StartTime.In(eat).Format(time.RFC3339),
			TimeZone: "Africa/Nairobi",
		},
		End: &calendar.EventDateTime{
			DateTime: event.EndTime.In(eat).Format(time.RFC3339),
			TimeZone: "Africa/Nairobi",
		},
		Attendees: attendees,
	}

	// Add meeting URL to description if provided
	if event.MeetingURL != "" {
		calendarEvent.Description += fmt.Sprintf("\n\nJoin meeting: %s", event.MeetingURL)

		// Add conference data for Google Meet integration
		calendarEvent.ConferenceData = &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId: fmt.Sprintf("meeting_%d", time.Now().Unix()),
			},
		}
	}

	// Create the event
	createdEvent, err := cs.service.Events.Insert(calendarID, calendarEvent).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar event: %w", err)
	}

	log.Printf("Created calendar event: %s (ID: %s)", createdEvent.Summary, createdEvent.Id)
	return createdEvent, nil
}

// CreateEventWithReminders creates a calendar event with explicit reminder overrides
func (cs *CalendarService) CreateEventWithReminders(calendarID string, event *CalendarEvent, reminderMinutes []int) (*calendar.Event, error) {
	if cs.service == nil {
		return nil, fmt.Errorf("calendar service not initialized")
	}

	// Convert attendees to calendar attendees
	var attendees []*calendar.EventAttendee
	for _, email := range event.Attendees {
		attendees = append(attendees, &calendar.EventAttendee{Email: email})
	}

	// Build reminder overrides
	var overrides []*calendar.EventReminder
	for _, m := range reminderMinutes {
		overrides = append(overrides, &calendar.EventReminder{Method: "popup", Minutes: int64(m)})
	}

	// Use East Africa Time (EAT)
	eat, _ := time.LoadLocation("Africa/Nairobi")

	calendarEvent := &calendar.Event{
		Summary:     event.Title,
		Description: event.Description,
		Location:    event.Location,
		Start:       &calendar.EventDateTime{DateTime: event.StartTime.In(eat).Format(time.RFC3339), TimeZone: "Africa/Nairobi"},
		End:         &calendar.EventDateTime{DateTime: event.EndTime.In(eat).Format(time.RFC3339), TimeZone: "Africa/Nairobi"},
		Attendees:   attendees,
		Reminders: &calendar.EventReminders{
			UseDefault: false,
			Overrides:  overrides,
		},
	}

	if event.MeetingURL != "" {
		calendarEvent.Description += fmt.Sprintf("\n\nJoin meeting: %s", event.MeetingURL)
	}

	createdEvent, err := cs.service.Events.Insert(calendarID, calendarEvent).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar event with reminders: %w", err)
	}
	log.Printf("Created calendar event with reminders: %s (ID: %s)", createdEvent.Summary, createdEvent.Id)
	return createdEvent, nil
}

// UpdateEvent updates an existing calendar event
func (cs *CalendarService) UpdateEvent(calendarID, eventID string, event *CalendarEvent) (*calendar.Event, error) {
	if cs.service == nil {
		return nil, fmt.Errorf("calendar service not initialized")
	}

	// Get the existing event
	existingEvent, err := cs.service.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing event: %w", err)
	}

	// Update the event fields
	existingEvent.Summary = event.Title
	existingEvent.Description = event.Description
	existingEvent.Location = event.Location
	existingEvent.Start = &calendar.EventDateTime{
		DateTime: event.StartTime.Format(time.RFC3339),
		TimeZone: "UTC",
	}
	existingEvent.End = &calendar.EventDateTime{
		DateTime: event.EndTime.Format(time.RFC3339),
		TimeZone: "UTC",
	}

	// Update attendees
	var attendees []*calendar.EventAttendee
	for _, email := range event.Attendees {
		attendees = append(attendees, &calendar.EventAttendee{
			Email: email,
		})
	}
	existingEvent.Attendees = attendees

	// Add meeting URL to description if provided
	if event.MeetingURL != "" {
		existingEvent.Description += fmt.Sprintf("\n\nJoin meeting: %s", event.MeetingURL)
	}

	// Update the event
	updatedEvent, err := cs.service.Events.Update(calendarID, eventID, existingEvent).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update calendar event: %w", err)
	}

	log.Printf("Updated calendar event: %s (ID: %s)", updatedEvent.Summary, updatedEvent.Id)
	return updatedEvent, nil
}

// DeleteEvent deletes a calendar event
func (cs *CalendarService) DeleteEvent(calendarID, eventID string) error {
	if cs.service == nil {
		return fmt.Errorf("calendar service not initialized")
	}

	err := cs.service.Events.Delete(calendarID, eventID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete calendar event: %w", err)
	}

	log.Printf("Deleted calendar event: %s", eventID)
	return nil
}

// GetEvent retrieves a calendar event
func (cs *CalendarService) GetEvent(calendarID, eventID string) (*calendar.Event, error) {
	if cs.service == nil {
		return nil, fmt.Errorf("calendar service not initialized")
	}

	event, err := cs.service.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar event: %w", err)
	}

	return event, nil
}

// ListEvents lists calendar events within a time range
func (cs *CalendarService) ListEvents(calendarID string, timeMin, timeMax time.Time) ([]*calendar.Event, error) {
	if cs.service == nil {
		return nil, fmt.Errorf("calendar service not initialized")
	}

	events, err := cs.service.Events.List(calendarID).
		TimeMin(timeMin.Format(time.RFC3339)).
		TimeMax(timeMax.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendar events: %w", err)
	}

	return events.Items, nil
}

// GetAuthURL returns the OAuth authorization URL
func (cs *CalendarService) GetAuthURL(state string) string {
	return cs.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges an authorization code for a token
func (cs *CalendarService) ExchangeCode(code string) (*oauth2.Token, error) {
	ctx := context.Background()
	token, err := cs.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	return token, nil
}

// TokenToJSON converts a token to JSON string
func (cs *CalendarService) TokenToJSON(token *oauth2.Token) (string, error) {
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token: %w", err)
	}

	return string(tokenJSON), nil
}

// TokenFromJSON creates a token from JSON string
func (cs *CalendarService) TokenFromJSON(tokenJSON string) (*oauth2.Token, error) {
	var token oauth2.Token
	err := json.Unmarshal([]byte(tokenJSON), &token)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// CreateMeetingEvent creates a calendar event for a chama meeting
func (cs *CalendarService) CreateMeetingEvent(calendarID string, meeting *Meeting, attendeeEmails []string) (*calendar.Event, error) {
	event := &CalendarEvent{
		Title:       meeting.Title,
		Description: meeting.Description,
		StartTime:   meeting.ScheduledAt,
		EndTime:     meeting.ScheduledAt.Add(time.Duration(meeting.Duration) * time.Minute),
		Location:    meeting.Location,
		MeetingURL:  meeting.MeetingURL,
		Attendees:   attendeeEmails,
	}

	return cs.CreateEvent(calendarID, event)
}
