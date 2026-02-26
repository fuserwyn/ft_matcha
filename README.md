# Matcha

A dating web application that facilitates connections between users—from registration to meeting. Built with Go, React, and a hybrid database architecture.

## Tech Stack

| Layer | Technologies |
|-------|--------------|
| **Frontend** | React, Vite, Tailwind CSS, React Router |
| **Backend** | Go, Gin |
| **Worker** | Go |
| **Databases** | PostgreSQL, Neo4j |
| **Search** | Elasticsearch |
| **Cache** | Redis |
| **File Storage** | MinIO |
| **Email (dev)** | MailHog |
| **Infrastructure** | Docker Compose |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                              Docker Compose                                              │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                          │
│  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐                        │
│  │   Frontend      │   │   API (Go)       │   │   Worker (Go)    │                        │
│  │   React         │   │   :8080          │   │   background     │                        │
│  │   :3000         │──►│                  │   │                 │                        │
│  │                 │   │  • Auth          │   │  • Sync → ES     │                        │
│  │  Vite + React   │   │  • Profile       │   │  • Sync → Neo4j  │                        │
│  │  Tailwind       │   │  • Search        │   │  • Email        │                        │
│  │  React Router   │   │  • Neo4j client  │   │  • Fame rating   │                        │
│  │  WebSocket      │   │  • WebSocket     │   │                 │                        │
│  └─────────────────┘   └────────┬────────┘   └────────┬────────┘                        │
│                                  │                     │                                 │
│                    ┌─────────────┼─────────────────────┼─────────────────────────────┐  │
│                    ▼             ▼                     ▼             ▼                 ▼  │
│             ┌──────────┐  ┌────────┐  ┌────────┐  ┌─────────┐  ┌───────┐  ┌──────┐     │
│             │PostgreSQL│  │ Neo4j  │  │ Redis  │  │Elastic   │  │MailHog│  │MinIO │     │
│             │  :5432   │  │ :7687  │  │ :6379  │  │ :9200    │  │ :1025 │  │:9000 │     │
│             └──────────┘  └────────┘  └────────┘  └─────────┘  └───────┘  └──────┘     │
│                                                                                          │
└─────────────────────────────────────────────────────────────────────────────────────────┘

                          Browser → http://localhost:3000 (React) → API :8080
```

## Services

| Service | Port | Purpose |
|---------|------|---------|
| **Frontend** | 3000 | React SPA, UI, WebSocket client |
| **API** | 8080 | REST API, WebSocket server |
| **Worker** | — | Sync, email, background jobs |
| **PostgreSQL** | 5432 | Profiles, auth, chat |
| **Neo4j** | 7474, 7687 | Graph (likes, views, blocks) |
| **Redis** | 6379 | Sessions, pub/sub |
| **Elasticsearch** | 9200 | Search, geo, recommendations |
| **MailHog** | 8025, 1025 | SMTP for development |
| **MinIO** | 9000, 9001 | Photo storage (S3-compatible) |

## Data Flow

| Data | PostgreSQL | Neo4j | Elasticsearch |
|------|------------|-------|---------------|
| Users, profiles | ✓ | id (sync) | indexed |
| Likes | — | ✓ | — |
| Views | — | ✓ | — |
| Blocks | — | ✓ | — |
| Tags | ✓ | HAS_TAG | ✓ |
| Chat | ✓ | — | — |
| Search / filters | — | — | ✓ |
| Recommendations | details | graph (ids) | geo + filters |

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
├── worker/                  # Go Worker
│   ├── cmd/
│   ├── internal/
│   ├── go.mod
│   └── Dockerfile
├── frontend/                # React
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── hooks/
│   │   ├── api/
│   │   ├── stores/
│   │   └── App.tsx
│   ├── package.json
│   ├── vite.config.ts
│   └── Dockerfile
└── docs/
    └── STACK.md
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
| **Sessions** | Redis + secure cookie |
| **WebSocket** | gorilla/websocket or nhooyr.io/websocket |

## Getting Started

1. Copy `.env.example` to `.env` and configure environment variables.
2. Start the stack: `make up` (or `docker compose up -d`).
3. Open http://localhost:3000 for the frontend.
4. API runs at http://localhost:8080.
5. Neo4j Browser is available at http://localhost:7474.
6. MailHog UI is available at http://localhost:8025.
7. MinIO console is available at http://localhost:9001.

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

## Security

- Passwords hashed with bcrypt
- Credentials stored in `.env` (excluded from Git)
- Prepared statements to prevent SQL injection
- CSRF protection
- Input validation on all forms
