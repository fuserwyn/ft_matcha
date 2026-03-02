# Matcha

A dating web application that facilitates connections between users—from registration to meeting. Built with Go, React, and a hybrid database architecture.

## Tech Stack

| Layer | Technologies |
|-------|--------------|
| **Frontend** | React, Vite, Tailwind CSS, React Router |
| **Backend** | Go, Gin |
| **Database** | PostgreSQL |
| **Cache** | Redis (email verify, password reset tokens) |
| **Search** | Elasticsearch |
| **File Storage** | MinIO |
| **Email (dev)** | MailHog |
| **Infrastructure** | Docker Compose |

## Architecture

```
┌──────────────────────────────────────────────────────────────────────────────────────────┐
│                              Docker Compose                                              │
├──────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                          │
│  ┌─────────────────┐   ┌──────────────────┐                                              │
│  │   Frontend      │   │   API (Go)       │                                              │
│  │   React         │   │   :8080          │                                              │
│  │   :3000         │──►│                  │                                              │
│  │                 │   │  • Auth          │                                              │
│  │  Vite + React   │   │  • Profile       │                                              │
│  │  Tailwind       │   │  • Search        │                                              │
│  │  React Router   │   │  • WebSocket     │                                              │
│  │  WebSocket      │   │                  │                                              │
│  └─────────────────┘   └────────┬────────┘                                               │
│                                 │                                                        │
│                   ┌─────────────┴─────────────────────────────────────────────────────┐  │
│                   ▼              ▼           ▼         ▼                              │ |
│             ┌──────────┐  ┌─────────────┐  ┌───────┐  ┌──────┐                        │ |
│             │PostgreSQL│  │Elasticsearch│  │MailHog│  │MinIO │                        │ |
│             │  :5432   │  │   :9200     │  │ :1025 │  │:9000 │                        │ |
│             └──────────┘  └─────────────┘  └───────┘  └──────┘                        │ |
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘

                          Browser → http://localhost:3000 (React) → API :8080
```

## Services

| Service | Port | Purpose |
|---------|------|---------|
| **Frontend** | 3000 | React SPA, UI, WebSocket client |
| **API** | 8080 | REST API, WebSocket server, Swagger at `/swagger/index.html` |
| **PostgreSQL** | 5432 | Profiles, auth, chat, likes, blocks |
| **Elasticsearch** | 9200 | Search, geo, recommendations |
| **MailHog** | 8025, 1025 | SMTP for development |
| **MinIO** | 9000, 9001 | Photo storage (S3-compatible) |

## Data Flow

| Data | PostgreSQL | Elasticsearch |
|------|------------|---------------|
| Users, profiles   | ✓ | indexed |
| Likes, blocks, views | ✓ | — |
| Tags | ✓ | ✓ |
| Chat | ✓ | — |
| Search / filters | — | ✓ |
| Recommendations | details | geo + filters |

## Project Structure

```
matcha/
├── docker-compose.yml
├── .env
├── api/                    # Go API
│   ├── cmd/
│   ├── internal/
│   │   ├── handlers/
│   │   ├── services/
│   │   ├── repository/
│   │   ├── middleware/
│   │   └── websocket/
│   ├── go.mod
│   └── Dockerfile
├── frontend/                # React
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── api/
│   │   └── context/
│   ├── package.json
│   ├── vite.config.ts
│   └── Dockerfile
```

## Frontend Stack (React)

React is a **JavaScript library** for building user interfaces. When we say "writing in React", we mean writing **JavaScript** (or TypeScript) code that uses React's API. React is not a separate language—it provides components and hooks that you use within JavaScript.

| Component | Technology |
|-----------|------------|
| **Framework** | React 18 |
| **Build** | Vite |
| **Routing** | React Router v6 |
| **Styles** | Tailwind CSS |
| **HTTP** | fetch / axios |
| **Real-time** | WebSocket |
| **State** | React Context / Zustand |
| **Forms** | React Hook Form + Zod |

## Backend Stack (Go)

| Component | Technology |
|-----------|------------|
| **Framework** | Gin |
| **Validation** | go-playground/validator |
| **Passwords** | bcrypt |
| **WebSocket** | gorilla/websocket |

## Getting Started

1. Copy `.env.example` to `.env` and configure environment variables.
2. Start the stack: `make up` (or `docker compose up -d`).
3. Open http://localhost:3000 for the frontend.
4. API runs at http://localhost:8080.
5. **Swagger** API docs: http://localhost:8080/swagger/index.html
6. MailHog UI is available at http://localhost:8025.
7. MinIO console is available at http://localhost:9001.

## Development (hot reload)

To avoid rebuilding Docker on every code change:

1. Start infra only: `make dev-infra`
2. In one terminal: `make dev-api` (or `cd api && air`) — API restarts on .go changes
3. In another: `cd frontend && npm run dev` — Vite HMR for frontend

Install air: `go install github.com/air-verse/air@latest`

## Useful Commands

- `make up` - start all services.
- `make rebuild` - rebuild and restart services.
- `make down` - stop the stack.
- `make ps` - show service status.
- `make e2e` - run backend end-to-end tests.
- `make test` - run backend unit/integration tests.

## New Features (MVP)

- Likes and matches with notifications.
- WebSocket chat with heartbeat and anti-spam.
- Read/unread messages and presence (`online` / `last_seen`).
- Photos upload/list/delete/set-primary via MinIO.
- Email notifications (like/match/message) via MailHog SMTP in development.

## Compatibility

- **Browsers**: Latest Chrome and Firefox (browserslist in `frontend/package.json`)
- **Mobile**: Responsive layout with hamburger menu on small screens; viewport meta for proper scaling
- **Small screens**: min-width 280px, overflow-x hidden, safe-area padding for notched devices; images scale with `max-width: 100%`

## Security

- Passwords hashed with bcrypt
- Credentials stored in `.env` (excluded from Git)
- Prepared statements to prevent SQL injection
- CSRF protection
- Input validation on all forms
