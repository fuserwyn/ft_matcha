# Matcha — Project Status & Evaluation Checklist

Based on subject `en.subject-8.pdf` (v6.1) and the official evaluation scale.
Last updated: 2026-03-03.

Legend: ✅ Done · ⚠️ Partial / needs attention · ❌ Missing

---

## Preliminaries

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| P1 | All credentials/API keys stored in `.env`, excluded from git | ✅ | All config via env vars; `.gitignore` covers `.env` |
| P2 | Micro-framework only (router + optional templating, no ORM/validators/user account manager) | ✅ | Gin (Go) — router only; manual SQL queries throughout |
| P3 | No NoSQL database as primary store | ✅ | PostgreSQL (primary) + Neo4j removed, Redis only for token store |
| P4 | No error-tracking libraries (Raven, Sentry, etc.) | ✅ | None present |
| P5 | No 500-range errors exposed to client | ✅ | Handlers return generic messages; stack traces never sent |

---

## Installation & Seeding

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| S1 | Single-command install / startup | ✅ | `docker compose up --build` |
| S2 | Database seeded with ≥ 500 distinct profiles | ✅ | `SeedService.EnsureMinimumUsers` targets 500 (250 M + 250 F); `SEED_USERS_ENABLED`, `MIN_USERS_COUNT` env vars |
| S3 | Seed photos assigned to profiles | ✅ | 1–2 photos per seed user from picsum.photos |
| S4 | Seed users are email-verified and immediately usable | ✅ | `email_verified_at` set during seed |

---

## Security

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| SEC1 | Passwords hashed in database (not plain text) | ✅ | bcrypt with `DefaultCost` |
| SEC2 | SQL injection not possible | ✅ | All queries use `$1/$2` parameterized statements |
| SEC3 | File uploads validated (type, size, content) | ✅ | Magic-byte detection + extension whitelist + 10 MB limit |
| SEC4 | XSS protected | ✅ | React escapes output by default; no `dangerouslySetInnerHTML` |
| SEC5 | No credentials in git repository | ✅ | Env vars only |

---

## Features

### Simple Start

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| F0 | App launches with no visible errors | ✅ | `docker compose up --build` |

---

### User Account Management (Registration & Signing-in)

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| A1 | Register with email, username, last name, first name, password | ✅ | All fields required in `RegisterReq` |
| A2 | Password not a common dictionary word | ✅ | Blocklist of 22 common passwords in `validation/auth.go` |
| A3 | Confirmation email with unique clickable link after registration | ✅ | Redis token (24 h TTL) + Mailhog in dev |
| A4 | Account only usable after clicking verification link | ✅ | `email_verified_at` checked on login |
| A5 | Login with username + password | ✅ | JWT issued on success |
| A6 | Forgot password / reset via email | ✅ | `/auth/forgot-password` → `/auth/reset-password` |
| A7 | Logout from any page with one click | ✅ | Header "Logout" link in `Layout.jsx` |
| A8 | Update first name, last name, email from profile | ✅ | `PATCH /auth/me` endpoint |
| A9 | Update all extended profile fields at any time | ✅ | `PUT /profile/me` endpoint |

---

### Extended Profile (User Profile)

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| UP1 | Gender field | ✅ | `male / female / non-binary / other` |
| UP2 | Sexual orientation / preferences field | ✅ | `male / female / both / other` |
| UP3 | Biography (short bio) | ✅ | Max 500 chars |
| UP4 | Interest tags (hashtag format, reusable across users) | ✅ | Shared `tags` table + `user_tags` join table |
| UP5 | Tag suggestions (autocomplete / top-trending) | ✅ | `GET /profile/tags/suggestions` returns top 20 tags by usage |
| UP6 | Up to 5 photos including one profile picture | ✅ | `maxPhotosPerUser = 5`, `is_primary` flag |
| UP7 | Fame rating (public, consistent formula) | ✅ | `fame_rating = likes_count × 5 + views_count` |
| UP8 | View history of profile visits | ✅ | `GET /profile/me/views` + `profile_views` table |
| UP9 | List of people who liked my profile | ✅ | `GET /likes/me` |
| UP10 | GPS location (with explicit consent) | ⚠️ | Lat/lng stored and editable; **browser GPS request not implemented in frontend** — user must type coordinates manually |
| UP11 | Fallback manual location entry if GPS refused | ✅ | City + lat/lng fields in profile form |
| UP12 | Location modifiable at any time | ✅ | Part of `PUT /profile/me` |

---

### Consultations (Profile Views & Likes History)

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| C1 | View history of profile visits | ✅ | `Views` page, `GET /profile/me/views` |
| C2 | List of people who liked my profile | ✅ | `GET /likes/me` displayed in frontend |

---

### Profile Suggestion (Browsing / Discovery)

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| BR1 | Suggestions filtered by sexual orientation | ✅ | Elasticsearch query filters by gender matching preference |
| BR2 | Bisexual default when orientation not set | ✅ | Defaults to `"both"` → sees all genders |
| BR3 | Weighted by geographic proximity | ✅ | `max_distance_km` param + coordinate-based scoring |
| BR4 | Weighted by maximum common tags | ✅ | Tag overlap calculated in scoring |
| BR5 | Weighted by maximum fame rating | ✅ | Fame rating included in ranking |
| BR6 | Sortable by age, location, fame rating, common tags | ✅ | `sort_by` / `sort_order` query params |
| BR7 | Filterable by age, location, fame rating, common tags | ✅ | `min_age`, `max_age`, `min_fame`, `max_fame`, `city`, `tags`, `max_distance_km` |
| BR8 | Blocked users excluded from results | ✅ | Block list checked in `discoveryRepo.Search` |

---

### Research (Advanced Search)

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| SR1 | Age range filter | ✅ | `min_age` / `max_age` |
| SR2 | Fame rating range filter | ✅ | `min_fame` / `max_fame` |
| SR3 | Location filter | ✅ | `city` + `max_distance_km` |
| SR4 | One or multiple interest tags filter | ✅ | `tags` (comma-separated) |
| SR5 | Results sortable and filterable same as suggestions | ✅ | Same Search endpoint |

---

### Profile of Other Users (Profile View)

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| PV1 | View other user's full profile (no email/password) | ✅ | `GET /users/:id` |
| PV2 | Profile visit recorded in history | ✅ | `AddProfileView` called on `GetByID` |
| PV3 | Like another user's profile | ✅ | `POST /users/:id/like` |
| PV4 | Cannot like if current user has no profile picture | ✅ | `GetPrimaryByUser` check before like |
| PV5 | Unlike / remove a previously given like | ✅ | `DELETE /users/:id/like` |
| PV6 | Fame rating visible on profile | ✅ | Included in profile response |
| PV7 | Online/offline status visible | ✅ | `GET /presence/:id`, WebSocket hub tracks live connections |
| PV8 | Last connection date/time if offline | ✅ | `last_seen` from `user_presence` table |
| PV9 | Report user as fake account | ✅ | `POST /users/:id/report` |
| PV10 | Block user (excluded from search, suggestions, notifications, chat) | ✅ | `POST /users/:id/block` |
| PV11 | Show if viewed user has liked current user | ✅ | `likedMe` boolean in response |
| PV12 | Show if already connected (mutual like) | ✅ | `isMatch` boolean in response |
| PV13 | Option to unlike / disconnect from profile being viewed | ✅ | Unlike button in `UserProfile.jsx` |

---

### Connection Between Users (Likes & Matching)

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| LK1 | Like / unlike another user | ✅ | |
| LK2 | Mutual like → "connected", chat enabled | ✅ | Match check gates message sending |
| LK3 | User without profile picture cannot like | ✅ | |
| LK4 | Profile shows like status and connection status | ✅ | |

---

### Report & Blocking

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| RB1 | Report a profile as fake account | ✅ | Multiple reasons: `fake_account`, `spam`, `harassment`, etc. |
| RB2 | Block a user | ✅ | |
| RB3 | Blocked user disappears from search / suggestions | ✅ | |
| RB4 | Blocked user generates no further notifications | ✅ | Block check in notification creation |
| RB5 | Chat disabled with blocked user | ✅ | `isBlocked` check in `SendMessage` |

---

### Chat

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| CH1 | Real-time chat between matched users (≤ 10 s delay) | ✅ | WebSocket hub, Go channels |
| CH2 | New message indicator visible from any page | ✅ | Unread count badge in header via `NotificationsContext` |
| CH3 | Chat disabled when users unmatch or block | ✅ | |

---

### Notifications (real-time, ≤ 10 s delay)

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| N1 | Received a like | ✅ | Type `"like"` |
| N2 | Profile was viewed | ✅ | Type `"visit"` |
| N3 | Received a message | ✅ | Type `"message"` |
| N4 | Liked user liked back (match) | ✅ | Type `"match"` |
| N5 | Connected user unliked (unmatch) | ✅ | Type `"unlike"` |
| N6 | Unread notification indicator visible on all pages | ✅ | Badge in header |
| N7 | Real-time delivery (WebSocket) | ✅ | `NotificationsContext` + WS hub push |

---

### Good Practice

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| GP1 | Compatible with latest Firefox and Chrome | ✅ | Standard HTML5/React, no browser-specific APIs |
| GP2 | Mobile-friendly / usable on small screens | ✅ | Tailwind CSS responsive classes throughout |
| GP3 | Header present | ✅ | Nav bar in `Layout.jsx` |
| GP4 | Main section present | ✅ | |
| GP5 | Footer present | ✅ | Added in v1.2.0 — site name, tagline, year |

---

## Items to Fix Before Evaluation

### ✅ All critical and important items resolved

| Item | Status | Fixed in |
|------|--------|---------|
| No footer | ✅ Fixed | v1.2.0 |
| Browser GPS | ✅ Already implemented | — |
| Password blocklist too short | ✅ Fixed (18 → 100+ entries) | v1.2.0 |

### ℹ️ Minor / Nice-to-have

| Item | Notes |
|------|-------|
| No explicit backend logout endpoint | JWT is stateless; client-side clear is acceptable. Token blacklisting via Redis is a possible bonus hardening. |
| Notification polling interval is 30 s | Subject allows 10 s delay; real-time WS push covers it, polling is a fallback only. |
| Seed photo URLs from picsum.photos | Requires internet access at seed time; if evaluator is offline, seed users will have broken photo URLs. |

---

## Bonus Features (subject §V — only evaluated if mandatory is perfect)

| Bonus | Status | Notes |
|-------|--------|-------|
| OmniAuth (OAuth login) | ❌ Not implemented | |
| Photo gallery with drag-and-drop + image editing | ❌ Not implemented | Upload works but no drag-and-drop or editing |
| Interactive user map (precise GPS via JS) | ❌ Not implemented | |
| Video / audio chat | ❌ Not implemented | |
| Schedule real-life dates / events | ❌ Not implemented | |

---

## Tech Stack Summary

| Layer | Technology |
|-------|-----------|
| Backend | Go + Gin (micro-framework) |
| Database | PostgreSQL (primary), Redis (token store) |
| Search / Ranking | Elasticsearch 8 |
| File storage | MinIO (S3-compatible) |
| Real-time | WebSocket (native Go, hub pattern) |
| Frontend | React + Vite + Tailwind CSS |
| Web server | nginx (frontend), Gin built-in (API) |
| Container | Docker Compose |
| Mail (dev) | Mailhog |
