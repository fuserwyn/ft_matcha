# Matcha — Frontend Documentation

## Tech Stack

| Tool | Version | Role |
|------|---------|------|
| React | 18 | UI framework (component-based, virtual DOM) |
| Vite | 5 | Dev server + bundler (replaces CRA, much faster HMR) |
| Tailwind CSS | 4 | Utility-first CSS — no separate stylesheet files |
| React Router | 6 | Client-side routing (SPA — no full page reloads) |
| react-easy-crop | 5 | Photo cropping library used before uploading images |

---

## Directory Structure

```
frontend/src/
├── main.jsx                    Bootstrap — mounts React into the DOM
├── App.jsx                     Router — defines all page routes
├── index.css                   Global styles (Tailwind base + custom)
│
├── api/
│   └── client.js               All HTTP calls to the backend API
│
├── context/
│   ├── AuthContext.jsx          Global auth state (current user, token)
│   └── NotificationsContext.jsx Global unread notification count
│
├── components/
│   ├── Layout.jsx               Navbar + footer wrapper for all pages
│   ├── ProtectedRoute.jsx       Redirects unauthenticated users to /login
│   ├── ProfileModal.jsx         Popup profile card in Discovery
│   ├── PhotoCropper.jsx         Crop overlay before photo upload
│   └── CityInput.jsx            City autocomplete using Photon geocoding API
│
└── pages/
    ├── Home.jsx                 Landing page
    ├── Login.jsx                Login form
    ├── Register.jsx             Registration form
    ├── Profile.jsx              My profile editor
    ├── Discovery.jsx            Browse and filter other users
    ├── UserProfile.jsx          Full page view of another user's profile
    ├── Matches.jsx              List of mutual matches
    ├── Likes.jsx                Who I liked / who liked me
    ├── Views.jsx                Profile view history
    ├── Notifications.jsx        Notification center
    └── Chat.jsx                 Real-time chat + video/voice calls
```

---

## Entry Points

### `main.jsx`
The very first file React runs. It finds the `<div id="root">` in `index.html` and renders the entire application into it.

### `App.jsx`
Wraps the whole app in two context providers (`AuthProvider`, `NotificationsProvider`) so any component can access auth state and notification count. Then it defines all the routes using React Router:

```
/              → Home.jsx
/login         → Login.jsx
/register      → Register.jsx
/profile       → Profile.jsx          (protected)
/discover      → Discovery.jsx        (protected)
/users/:id     → UserProfile.jsx      (protected)
/matches       → Matches.jsx          (protected)
/likes         → Likes.jsx            (protected)
/views         → Views.jsx            (protected)
/notifications → Notifications.jsx    (protected)
/chat/:id      → Chat.jsx             (protected)
```

"Protected" routes are wrapped in `ProtectedRoute` — unauthenticated users are sent to `/login`.

---

## API Client — `api/client.js`

Central place for **all** communication with the backend. Nothing else calls `fetch()` directly.

### How it works
```
Component → api module (e.g. users.like()) → api() helper → fetch(/api/...) → Backend
```

The `api()` helper automatically:
- Reads the JWT token from `localStorage`
- Adds the `Authorization: Bearer <token>` header
- Sets `Content-Type: application/json` for JSON bodies
- Parses errors from the response and throws them

### API Modules

| Module | What it handles |
|--------|----------------|
| `auth` | register, login, get current user (`me`), update account |
| `profile` | get/update profile, tags, city suggestions, view history |
| `users` | search/filter users, get by ID, like/unlike, block, report |
| `matches` | list mutual matches |
| `likes` | list who I liked, list who liked me |
| `chat` | list messages, send text, send voice message, mark as read |
| `notifications` | list all, mark all read |
| `presence` | get online/last-seen status for a user |
| `photos` | upload, delete, set primary, list my photos |

### WebSocket
`wsChatUrl()` builds a `ws://` or `wss://` URL and appends the JWT token as a query parameter so the server can authenticate the connection:
```
ws://localhost/ws/chat?token=<jwt>
```

---

## Global State — Context Providers

React Context = global state without prop drilling. Any component can call `useAuth()` or `useNotifications()` to read/update the state.

### `AuthContext.jsx`

Stores who is currently logged in.

**State:**
- `user` — the full user object (`id`, `username`, `first_name`, etc.) or `null`
- `loading` — `true` while checking the token on first app load

**How login works:**
1. `login(token, userData)` is called after a successful API login
2. Token is saved to `localStorage`
3. `user` state is updated → all components re-render with the new user

**On page refresh:**
1. App reads token from `localStorage`
2. Calls `auth.me()` to verify it's still valid
3. If valid → sets `user`; if expired/invalid → clears and stays logged out

### `NotificationsContext.jsx`

Tracks the unread notification count shown as a badge on the nav.

- Polls the API every **30 seconds** to refresh the count
- Also listens on a WebSocket for `notification` events to update instantly
- `refreshUnread()` can be called from any component to force a refresh

---

## Layout & Navigation

### `Layout.jsx`

Every page is wrapped in `Layout`. It renders:
1. **Sticky navbar** — always visible at the top. Shows nav links when logged in, Login/Sign Up when logged out. Collapses to a hamburger menu on mobile.
2. **`<main>`** — where the current page renders
3. **Footer** — "Because love, too, can be industrialized."

The navbar notification badge is driven by `useNotifications()`.

### `ProtectedRoute.jsx`

```jsx
// If not logged in → redirect to /login
// If still loading auth state → show spinner
// If logged in → render the page
```

This wraps all routes that require authentication.

---

## Components

### `CityInput.jsx`

A standard `<input>` with autocomplete powered by the **Photon geocoding API** (Komoot, free, no API key needed).

**Flow:**
1. User types ≥ 2 characters
2. After 250ms debounce, fetches `https://photon.komoot.io/api/?q=...&layer=city`
3. Results shown as a dropdown below the input
4. Keyboard navigation: `↑↓` to move, `Enter` to select, `Escape` to close
5. On select, calls `onChange(cityName)` to update the parent

**Key detail:** Uses `isMount` ref to skip the initial fetch when the input already has a value (prevents the dropdown from opening when you navigate to a page with pre-filled city).

### `PhotoCropper.jsx`

Uses the `react-easy-crop` library to show an interactive crop overlay on top of an uploaded image.

- Aspect ratio locked to **3:4** (portrait) to match the profile card style
- User can pan and zoom the image
- On confirm: uses the `cropArea` coordinates to draw the cropped region onto a `<canvas>`, then exports as a JPEG `Blob`
- The blob is then uploaded to the backend

### `ProfileModal.jsx`

Opens another user's profile as a **modal overlay** on top of the Discovery page instead of navigating away. This preserves the scroll position and filter state in Discovery.

**Contents:**
- Photo gallery with lightbox (arrow keys / swipe to switch photos)
- Full profile info (bio, tags, age, city, fame rating)
- Action buttons: Like/Unlike, Chat, Block, Report
- Report modal with 6 reason options
- "View full profile →" link for the dedicated page

**Lightbox:** Supports keyboard (`←→` arrows, `Escape`), mouse click outside to close, and touch swipe gestures on mobile. Locks body scroll while open.

---

## Pages

### `Home.jsx`
Landing page. Shows login/register buttons to guests, and quick-nav links to Discover/Matches/Profile for logged-in users.

### `Login.jsx`
Standard login form. On success:
1. Saves JWT to `localStorage`
2. Calls `login(token, user)` from AuthContext
3. Navigates to the page the user was trying to reach (or home)

Handles URL params for status messages: `?verified=1`, `?error=`, `?already=1` (set by the backend after email verification redirect).

### `Register.jsx`
Registration form with client-side validation (name, username 3-50 chars, valid email, password 8-72 chars). On success shows a "check your email" message instead of auto-logging in, since the backend requires email verification.

### `Profile.jsx`

The user's own profile editor. Two-column layout:

**Left column:**
- **Account info** (username, email, first/last name) — saved separately
- **Interests/Tags** — comma-separated, with tag suggestions from the API
- **Photos** — upload (max 5), crop via `PhotoCropper`, set primary, delete

**Right column:**
- **Bio** (textarea, max 500 chars)
- **Gender and sexual preference** (dropdowns)
- **Birth date** (date picker, enforces 18+ minimum age)
- **Location** — `CityInput` component + a GPS button

**GPS location flow:**
1. Click GPS button → calls `navigator.geolocation.getCurrentPosition()`
2. If denied/unsupported → falls back to `ipapi.co` IP geolocation
3. Updates `latitude`, `longitude`, and `city` fields

### `Discovery.jsx`

The main feature page. Users browse potential matches with real-time filtering powered by Elasticsearch on the backend.

**Sidebar filters:**
- Gender (checkboxes: male, female, other)
- Their preference (checkboxes: male, female, both)
- Age range (dual slider, 18-99)
- Fame rating range (dual slider, 0-100)
- City (CityInput autocomplete)
- Tags (comma-separated)
- Max distance (km input)
- Sort by (relevance / age / location / fame / shared tags) + order (asc/desc)

**Results:** CSS grid of user cards (2-4 columns). Each card shows the primary photo with a gradient overlay, name, age, city, tags, and fame rating stars.

**Clicking a card** opens `ProfileModal` — Discovery stays mounted, no navigation happens, scroll position is preserved.

**Module-level cache:** Filter state and scroll position are stored in a module-level variable (outside the component) so returning from a profile view restores exactly where you were.

### `UserProfile.jsx`

Full dedicated page for viewing another user's profile at `/users/:id`. Reached via "View full profile →" in the modal or direct links.

- Same info and actions as `ProfileModal` but as a full page
- Back button returns to previous location
- Photo lightbox with keyboard and swipe support

### `Matches.jsx`

Shows all mutual matches (both users liked each other). Grid of cards with "View Profile" and "Chat" buttons. The Chat button links to `/chat/:id`.

### `Likes.jsx`

Two tabs:
- **"I liked"** — users I've liked, with an "Unlike" button
- **"Liked me"** — users who liked me, with a "Like back" button

Switching tabs fetches the appropriate data.

### `Views.jsx`

Two tabs showing profile view history:
- **"I viewed"** — profiles I've visited
- **"Viewed me"** — who visited my profile

Shows the timestamp of the view.

### `Notifications.jsx`

A list of all notifications (new match, new like, new message, profile view, unlike, block). Features:
- Toggle to show only unread
- "Mark all read" button
- Unread items highlighted with rose background
- Syncs with `NotificationsContext` to clear the nav badge

### `Chat.jsx`

The most complex page (~550 lines of logic). Handles real-time text messaging and peer-to-peer video/audio calls.

---

## Chat Page — Deep Dive

### Layout (top → bottom)
```
┌─────────────────────────────────────────┐
│  ← [Avatar] Name • Online      📞 📹   │  ← sticky header
├─────────────────────────────────────────┤
│  [Video panel — hidden when no call]    │  ← collapses to 0px
├─────────────────────────────────────────┤
│                                         │
│  Message bubbles (scrollable)           │  ← flex-1 overflow-y-auto
│                                         │
├─────────────────────────────────────────┤
│  🎤  [Type a message…]           ➤     │  ← sticky input bar
└─────────────────────────────────────────┘
```

### Real-time Messaging (WebSocket)

```
Browser WebSocket ←──────────── Backend WebSocket Hub
     │                                    │
     ├─ sends: { to_user_id, content }    │
     ├─ receives: { type: "message" }     │
     ├─ receives: { type: "message_read" }│
     └─ receives: { type: "call_*" }      │
```

- Connects on mount, auto-reconnects every 2 seconds if disconnected
- If WebSocket is offline, falls back to polling the REST API every 5 seconds
- Presence (online/offline) polled every 15 seconds via REST API

### Voice Message Recording

Uses the browser's **MediaRecorder API**:
1. Click mic button → `getUserMedia({ audio: true })` → asks permission
2. Recording starts → audio chunks collected in `mediaChunksRef`
3. Click stop → `recorder.stop()` → `ondataavailable` fires
4. Chunks assembled into a `Blob` → wrapped in a `File`
5. Uploaded to backend via `chat.sendVoiceMessage()`
6. New message appears in the list with an `<audio>` player

### Video/Audio Calls (WebRTC)

WebRTC = direct peer-to-peer connection for audio/video. The backend WebSocket is only used to exchange signaling messages (offers, answers, ICE candidates) — the actual media stream goes directly browser-to-browser.

**Call initiation (caller side):**
```
1. startCall('video' or 'audio')
2. getUserMedia() → get local camera/mic stream
3. new RTCPeerConnection() → create peer connection
4. Add local stream tracks to peer connection
5. createOffer() → generate SDP (Session Description Protocol)
6. setLocalDescription(offer)
7. Send { type: 'call_invite', sdp, mode } via WebSocket
```

**Answering (callee side):**
```
1. Receive 'call_invite' via WebSocket → show incoming call banner
2. User clicks Accept → getUserMedia() → get local stream
3. new RTCPeerConnection()
4. setRemoteDescription(offer.sdp)
5. createAnswer() → generate SDP answer
6. setLocalDescription(answer)
7. Send { type: 'call_accept', sdp } via WebSocket
```

**ICE Candidates (both sides):**
- As ICE candidates are discovered, send them via WebSocket (`call_ice`)
- Each side adds the other's candidates to the peer connection
- When both sides exchange enough candidates, the direct connection is established

**Connection states:**
- `idle` → no call
- `calling` → sent invite, waiting for answer
- `incoming` → received invite, showing banner
- `connecting` → exchanging ICE candidates
- `in_call` → connected, media flowing

**Audio-only mode:** When `mode === 'audio'`, video tracks are not requested from `getUserMedia`. The video panel shows the other user's avatar instead of a video element.

**STUN server:** `stun:stun.l.google.com:19302` — used to discover the public IP/port for NAT traversal (free public server from Google).

**Error handling:** Specific user-friendly messages for:
- Permission denied (`NotAllowedError`)
- No camera/mic found (`NotFoundError`)
- Camera busy in another app (`NotReadableError`)
- Insecure context (HTTPS required for camera access on mobile)

---

## Data Flow Summary

### Authentication Flow
```
User types credentials
→ POST /api/auth/login
→ Backend returns JWT
→ Saved to localStorage
→ AuthContext.user set
→ All protected routes accessible
→ Token sent as Authorization header on every request
```

### Discovery Flow
```
User sets filters in sidebar
→ Debounced call to users.search() with filter params
→ Backend queries Elasticsearch
→ Elasticsearch filters by gender/age/tags/distance/fame
→ Returns sorted, paginated user list
→ Cards rendered in grid
→ Card click → ProfileModal opens
```

### Like Flow
```
User clicks Like on a profile
→ POST /api/users/:id/like
→ Backend creates like record in PostgreSQL
→ Backend checks if other user also liked current user
→ If yes: creates a match, sends notification to both users
→ Notification arrives via WebSocket → badge increments
```

### Real-time Notification Flow
```
Any event (like, match, message, view)
→ Backend creates notification in DB
→ Backend sends WebSocket event to recipient
→ NotificationsContext receives it
→ unreadCount increments
→ Navbar badge updates
```

---

## Key Technical Decisions

**Why module-level cache in Discovery?**
React unmounts components on navigation. When returning to Discovery from a profile, all state would be lost. Storing it in a module-level variable (outside the component) survives unmount/remount cycles within the same browser session.

**Why ProfileModal instead of navigating to UserProfile?**
Navigation would unmount Discovery → lose scroll position and filter state. A modal keeps Discovery mounted underneath, so position is preserved.

**Why `sticky top-14` on the Chat header?**
The app navbar is `h-14` (56px). `sticky top-14` pins the chat header exactly below the navbar so it never scrolls away with messages.

**Why negative margins (`-mt-6 sm:-mt-8`) on Chat?**
The `<main>` layout wrapper has `py-6 sm:py-8` padding. To make the chat fill the full viewport height (not just the padded content area), the negative margins cancel that padding, and `height: calc(100vh - 3.5rem)` fills from just below the navbar to the bottom of the screen.

**Why JWT in query param for WebSocket?**
The browser's `WebSocket` constructor doesn't support custom headers. The token must be passed in the URL so the backend can authenticate the connection.

**Why `react-easy-crop` + canvas for photo upload?**
The subject requires specific photo handling. Cropping client-side avoids sending a large raw image to the server; only the final cropped region (as a compressed JPEG) is uploaded.
