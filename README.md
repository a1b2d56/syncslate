# SyncSlate — SyncSlate Backend Server

SyncSlate is the high-performance, self-hosted Go backend for the SyncSlate messaging app. It features zero-config hosting via SQLite, real-time WebSocket communication, server-side Ghost Mode enforcement, anti-deletion event propagation (Ayu Spy support), strict 48-hour message editing windows, and rate-limiting from day one.

---

## Technical Features

- **Framework & Router**: Go 1.23+ with `github.com/go-chi/chi/v5`
- **Database**: Zero-dependency SQLite via `modernc.org/sqlite` with WAL mode and concurrent reader connection pooling.
- **Authentication**: JWT Bearer Tokens with bcrypt password hashing and token rotation.
- **Real-Time Gateway**: WebSockets (`github.com/gorilla/websocket`) with 5s auth handshake timeout, ping/pong keepalive, bounded send queues (256 msg limit), and connection rate-limiting.
- **Ghost Mode (Server-Side)**: Strictly suppresses read receipts, typing indicators, and presence status updates at the server layer for privacy-enabled accounts.
- **Delete Propagation**: Soft-deletes messages and broadcasts `message:deleted` events so the frontend's Ayu Spy anti-delete feature can retain messages locally.
- **48-Hour Message Edit Window**: Server-enforced 48-hour maximum edit limit.
- **Rate Limiting**: Built-in per-IP token bucket rate limiting on HTTP endpoints and WebSocket read pumps.
- **Media Engine**: Local file storage with MIME type verification and up to 50MB file upload support.

---

## Directory Layout

```
SyncSlate/
├── cmd/
│   └── syncslate/
│       └── main.go           # Server entry point
├── internal/
│   ├── auth/                 # Auth handler, JWT service, password hashing
│   ├── user/                 # User profiles, discoverability search, Ghost Mode
│   ├── contact/              # Contact list management
│   ├── request/              # Message requests & blocking state machine
│   ├── chat/                 # Chat lists & message history pagination
│   ├── message/              # Messaging logic, 48h edit window, soft-delete
│   ├── media/                # File upload/serving engine
│   ├── group/                # Group and channel administration
│   ├── ws/                   # WebSocket hub, router, client pump & events
│   ├── health/               # Health check endpoint
│   ├── middleware/           # Auth, logging, rate limiting
│   ├── models/               # Core data structures
│   ├── database/             # SQLite connection & migration runner
│   └── config/               # Environment configuration
├── migrations/               # Numbered SQL schema migrations (001 - 010)
├── tests/                    # Unit and integration tests
├── Dockerfile                # Multi-stage production container
├── Makefile                  # Build & test helpers
└── README.md
```

---

## REST API Summary

| Group | Method | Endpoint | Description |
|:---|:---|:---|:---|
| **Health** | `GET` | `/api/v1/health` | Health status and uptime |
| **Auth** | `POST` | `/api/v1/auth/register` | Register new user |
| **Auth** | `POST` | `/api/v1/auth/login` | Login & receive tokens |
| **Auth** | `POST` | `/api/v1/auth/refresh` | Rotate access/refresh tokens |
| **Auth** | `POST` | `/api/v1/auth/logout` | Revoke current session |
| **Auth** | `GET` | `/api/v1/auth/me` | Current user profile |
| **Users** | `GET` | `/api/v1/users/search?q=` | Search discoverable users |
| **Users** | `GET` | `/api/v1/users/{userId}` | Get user profile |
| **Users** | `PUT` | `/api/v1/users/me` | Update profile / Ghost Mode |
| **Contacts** | `GET` | `/api/v1/contacts` | List contacts |
| **Contacts** | `POST` | `/api/v1/contacts` | Add contact |
| **Contacts** | `DELETE`| `/api/v1/contacts/{id}` | Remove contact |
| **Requests** | `GET` | `/api/v1/message-requests` | List pending requests |
| **Requests** | `POST` | `/api/v1/message-requests/{id}/accept` | Accept request & create chat |
| **Requests** | `POST` | `/api/v1/message-requests/{id}/decline` | Decline request / block |
| **Chats** | `GET` | `/api/v1/chats` | List active chats |
| **Chats** | `GET` | `/api/v1/chats/{id}/messages` | Paginated chat history |
| **Messages** | `POST` | `/api/v1/messages` | Send message (HTTP fallback) |
| **Messages** | `PUT` | `/api/v1/messages/{id}` | Edit message (Max 48h limit) |
| **Messages** | `DELETE`| `/api/v1/messages/{id}?for_everyone=true` | Soft-delete & broadcast |
| **Media** | `POST` | `/api/v1/media/upload` | Multipart file upload |
| **Media** | `GET` | `/api/v1/media/{mediaId}` | Stream media file |

---

## WebSocket Gateway (`/ws/v1/chat`)

Send immediate authentication payload upon opening connection:
```json
{
  "type": "auth",
  "payload": {
    "token": "Bearer <your_access_token>"
  }
}
```

---

## Local Quickstart

### Prerequisites
- Go 1.23 or higher

### Build & Run
```bash
make build
make run
```
Or directly:
```bash
go run ./cmd/syncslate
```

The server starts on `http://localhost:8080`.

### Running Tests
```bash
make test
```
