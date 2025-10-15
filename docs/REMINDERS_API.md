# Reminders API Documentation

## Overview
The Reminders API provides comprehensive functionality for managing user reminders with support for different reminder types, scheduling, and notification processing.

## Features
- ✅ Create, read, update, delete reminders
- ✅ Support for multiple reminder types (once, daily, weekly, monthly)
- ✅ User-specific reminder management
- ✅ Automatic notification scheduling
- ✅ Backend notification processing
- ✅ Graceful error handling
- ✅ Database persistence with SQLite

## Database Schema

### Reminders Table
```sql
CREATE TABLE reminders (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    reminder_type TEXT NOT NULL DEFAULT 'once',
    scheduled_at DATETIME NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    is_completed BOOLEAN DEFAULT FALSE,
    notification_sent BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

## API Endpoints

### Base URL
```
/api/v1/reminders
```

### Authentication
All endpoints require authentication via JWT token in the Authorization header:
```
Authorization: Bearer <jwt_token>
```

### 1. Create Reminder
**POST** `/api/v1/reminders/`

**Request Body:**
```json
{
  "title": "Meeting with team",
  "description": "Weekly team standup meeting",
  "reminderType": "weekly",
  "scheduledAt": "2024-01-15T10:00:00Z",
  "isEnabled": true
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "reminder-uuid",
    "userId": "user-uuid",
    "title": "Meeting with team",
    "description": "Weekly team standup meeting",
    "reminderType": "weekly",
    "scheduledAt": "2024-01-15T10:00:00Z",
    "isEnabled": true,
    "isCompleted": false,
    "notificationSent": false,
    "createdAt": "2024-01-01T12:00:00Z",
    "updatedAt": "2024-01-01T12:00:00Z"
  },
  "message": "Reminder created successfully"
}
```

### 2. Get User Reminders
**GET** `/api/v1/reminders/`

**Query Parameters:**
- `limit` (optional): Maximum number of reminders to return (default: 50, max: 100)
- `offset` (optional): Number of reminders to skip (default: 0)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "reminder-uuid",
      "userId": "user-uuid",
      "title": "Meeting with team",
      "reminderType": "weekly",
      "scheduledAt": "2024-01-15T10:00:00Z",
      "isEnabled": true,
      "isCompleted": false
    }
  ],
  "count": 1
}
```

### 3. Get Specific Reminder
**GET** `/api/v1/reminders/:id`

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "reminder-uuid",
    "title": "Meeting with team",
    "reminderType": "weekly",
    "scheduledAt": "2024-01-15T10:00:00Z",
    "isEnabled": true
  }
}
```

### 4. Update Reminder
**PUT** `/api/v1/reminders/:id`

**Request Body (all fields optional):**
```json
{
  "title": "Updated meeting title",
  "description": "Updated description",
  "reminderType": "daily",
  "scheduledAt": "2024-01-16T10:00:00Z",
  "isEnabled": false,
  "isCompleted": true
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "reminder-uuid",
    "title": "Updated meeting title",
    "isEnabled": false
  },
  "message": "Reminder updated successfully"
}
```

### 5. Delete Reminder
**DELETE** `/api/v1/reminders/:id`

**Response:**
```json
{
  "success": true,
  "message": "Reminder deleted successfully"
}
```

### 6. Toggle Reminder
**POST** `/api/v1/reminders/:id/toggle`

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "reminder-uuid",
    "isEnabled": false
  },
  "message": "Reminder toggled successfully"
}
```

## Reminder Types

### Supported Types
- `once`: One-time reminder
- `daily`: Daily recurring reminder
- `weekly`: Weekly recurring reminder
- `monthly`: Monthly recurring reminder

### Scheduling Logic
- **Once**: Notification sent at the exact scheduled time
- **Daily**: Notification sent daily at the scheduled time
- **Weekly**: Notification sent weekly on the same day and time
- **Monthly**: Notification sent monthly on the same date and time

## Notification Processing

### Automatic Scheduler
The backend includes an automatic notification scheduler that:
- Runs every minute to check for pending notifications
- Processes reminders that are due for notification
- Handles recurring reminder logic
- Marks one-time reminders as notification sent

### Notification Flow
1. Scheduler checks for pending reminders
2. Determines if notification should be sent based on type and timing
3. Processes notification (logs for now, can be extended)
4. Updates reminder status as needed

## Error Handling

### Common Error Responses
```json
{
  "success": false,
  "error": "Error message description"
}
```

### Error Codes
- `400 Bad Request`: Invalid request data or past scheduled time
- `401 Unauthorized`: Missing or invalid authentication
- `404 Not Found`: Reminder not found
- `500 Internal Server Error`: Server error

## Validation Rules

### Create/Update Reminder
- `title`: Required, 1-100 characters
- `description`: Optional, max 500 characters
- `reminderType`: Required, must be one of: once, daily, weekly, monthly
- `scheduledAt`: Required, must be a future date/time
- `isEnabled`: Optional boolean

## Integration with Mobile App

### Service Layer
The mobile app includes `ReminderService` that:
- Handles API communication
- Provides fallback to local storage
- Manages data format conversion
- Handles migration from local to backend storage

### Offline Support
- App works offline using local storage
- Syncs with backend when connection is available
- Migrates local reminders to backend automatically

## Testing

### Running Tests
```bash
cd apps/backend
go test ./test/reminder_test.go -v
```

### Test Coverage
- Create reminder with valid data
- Create reminder with invalid data (past date, invalid type)
- Get user reminders with pagination
- Update reminder fields
- Delete reminder
- Toggle reminder status

## Future Enhancements

### Planned Features
- Push notification integration (FCM/APNs)
- Email notification support
- SMS notification support
- Reminder categories/tags
- Snooze functionality
- Reminder templates
- Analytics and insights

### Integration Points
- WebSocket notifications for real-time updates
- Email service integration
- SMS service integration (Twilio)
- Push notification services
