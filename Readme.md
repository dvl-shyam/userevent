Task Overview:
You will create a simple event-driven application where a producer service generates events (e.g., user actions), and a consumer service processes these events. MongoDB will be used to store events, manage retries, and handle dead-letter queues.

System Overview

Producer Service:
Emits events related to user actions, such as user registration.
Stores these events in MongoDB for processing by the consumer service.
The producer service emits events whenever a user registers.
For each new user, the service creates a new document in the Events collection.
The event is marked as pending and includes relevant user data.
Event Types:

User Registration Event: Contains user details needed to send a welcome email.

Consumer Service:
Listens for new events in MongoDB.
Processes each event, such as sending a welcome email to the user.
Implements retry logic for failed events and manages a dead-letter queue.

Event Consumption:
The consumer service continuously polls the Events collection for new pending events.
When an event is found, the service processes it (e.g., sending a welcome email).\

Event Processing:
Process the event based on its type (e.g., if it's a user_registration event, send an email).
Update the event’s status to processed if successful.

Retry Logic:
If the event processing fails (e.g., email service is down), increment the retry_count.
If the retry_count exceeds a certain threshold (e.g., 5 retries), move the event to the Dead-Letter Queue collection with the failure reason.

Event Document Collection
{
  "_id": "event_id",
  "event_type": "user_registration",
  "payload": {
    "user_id": "12345",
    "email": "user@example.com",
    "name": "John Doe"
  },
  "status": "pending",
  "retry_count": 0,
  "created_at": "2024-08-21T15:00:00Z",
  "updated_at": "2024-08-21T15:00:00Z"
}

Dead Event Queue Collection
{
  "_id": "event_id",
  "event_type": "user_registration",
  "payload": {
    "user_id": "12345",
    "email": "user@example.com",
    "name": "John Doe"
  },
  "failure_reason": "Email service not available",
  "created_at": "2024-08-21T15:00:00Z",
  "failed_at": "2024-08-21T15:30:00Z"
}



Register
```bash
curl --location --request POST 'http://localhost:{PORT}/register
```

Go To Login 
```bash
curl --location --request GET 'http://localhost:{PORT}/login
```

