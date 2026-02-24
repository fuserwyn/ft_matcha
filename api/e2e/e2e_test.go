package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type httpClient struct {
	base  string
	token string
	c     *http.Client
}

func TestChatPresenceNotificationsE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run e2e tests against a running API")
	}

	base := os.Getenv("E2E_API_BASE")
	if base == "" {
		base = "http://localhost:8080"
	}

	public := &httpClient{
		base: strings.TrimRight(base, "/"),
		c:    &http.Client{Timeout: 10 * time.Second},
	}

	// A/B: full happy path (match + ws + notifications + read/unread + presence)
	userA := registerUser(t, public, "a")
	userB := registerUser(t, public, "b")

	a := &httpClient{base: public.base, token: userA.Token, c: public.c}
	b := &httpClient{base: public.base, token: userB.Token, c: public.c}

	postNoBody(t, a, "/api/v1/users/"+userB.ID.String()+"/like")
	postNoBody(t, b, "/api/v1/users/"+userA.ID.String()+"/like")

	matches := getJSON(t, a, "/api/v1/matches")
	if !containsUser(matches, userB.ID.String()) {
		t.Fatalf("expected user B in A matches: %#v", matches)
	}

	wsA := openWS(t, public.base, userA.Token)
	defer wsA.Close()

	p1 := getJSON(t, b, "/api/v1/presence/"+userA.ID.String())
	if online, _ := p1["is_online"].(bool); !online {
		t.Fatalf("expected user A online, got: %#v", p1)
	}

	msgPayload := map[string]any{
		"to_user_id": userB.ID.String(),
		"content":    "hello from e2e ws",
	}
	if err := wsA.WriteJSON(msgPayload); err != nil {
		t.Fatalf("ws write: %v", err)
	}

	waitForEvent(t, wsA, "message", 5*time.Second)

	notifs := getJSON(t, b, "/api/v1/notifications?unread_only=true")
	if !containsNotificationType(notifs, "message") {
		t.Fatalf("expected message notification: %#v", notifs)
	}

	msgsBefore := getJSON(t, b, "/api/v1/users/"+userA.ID.String()+"/messages")
	if !containsUnreadFrom(msgsBefore, userA.ID.String()) {
		t.Fatalf("expected unread message from A: %#v", msgsBefore)
	}

	patchNoBody(t, b, "/api/v1/users/"+userA.ID.String()+"/messages/read")
	msgsAfter := getJSON(t, b, "/api/v1/users/"+userA.ID.String()+"/messages")
	if containsUnreadFrom(msgsAfter, userA.ID.String()) {
		t.Fatalf("expected all messages from A marked read: %#v", msgsAfter)
	}

	_ = wsA.Close()
	time.Sleep(300 * time.Millisecond)
	p2 := getJSON(t, b, "/api/v1/presence/"+userA.ID.String())
	if online, _ := p2["is_online"].(bool); online {
		t.Fatalf("expected user A offline after disconnect: %#v", p2)
	}
	if p2["last_seen"] == nil {
		t.Fatalf("expected last_seen present after disconnect: %#v", p2)
	}

	// C/D: no-match blocked flow
	userC := registerUser(t, public, "c")
	userD := registerUser(t, public, "d")
	cu := &httpClient{base: public.base, token: userC.Token, c: public.c}
	postNoBody(t, cu, "/api/v1/users/"+userD.ID.String()+"/like")

	wsC := openWS(t, public.base, userC.Token)
	defer wsC.Close()
	if err := wsC.WriteJSON(map[string]any{
		"to_user_id": userD.ID.String(),
		"content":    "this should fail",
	}); err != nil {
		t.Fatalf("ws write no-match: %v", err)
	}
	waitForErrorContains(t, wsC, "can only message matches", 5*time.Second)
}

type registeredUser struct {
	ID    uuid.UUID
	Token string
}

func registerUser(t *testing.T, c *httpClient, prefix string) registeredUser {
	t.Helper()
	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	username := fmt.Sprintf("%s_%s", prefix, suffix)
	email := fmt.Sprintf("%s@example.com", username)

	payload := map[string]any{
		"username":   username,
		"email":      email,
		"password":   "password123",
		"first_name": "First",
		"last_name":  "Last",
	}
	out := postJSON(t, c, "/api/v1/auth/register", payload)
	token, _ := out["token"].(string)
	user, _ := out["user"].(map[string]any)
	idStr, _ := user["id"].(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}
	return registeredUser{ID: id, Token: token}
}

func openWS(t *testing.T, apiBase, token string) *websocket.Conn {
	t.Helper()
	wsBase := strings.TrimRight(apiBase, "/")
	wsBase = strings.Replace(wsBase, "http://", "ws://", 1)
	wsBase = strings.Replace(wsBase, "https://", "wss://", 1)
	u := wsBase + "/api/v1/ws/chat?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	return conn
}

func (c *httpClient) do(method, path string, body any) map[string]any {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, r)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.c.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		panic(fmt.Sprintf("request %s %s failed: %d body=%s", method, path, res.StatusCode, string(data)))
	}
	if len(data) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if data[0] == '[' {
		var arr []any
		_ = json.Unmarshal(data, &arr)
		return map[string]any{"items": arr}
	}
	_ = json.Unmarshal(data, &out)
	return out
}

func postJSON(t *testing.T, c *httpClient, path string, body any) map[string]any {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("postJSON panic: %v", r)
		}
	}()
	return c.do(http.MethodPost, path, body)
}

func postNoBody(t *testing.T, c *httpClient, path string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("postNoBody panic: %v", r)
		}
	}()
	_ = c.do(http.MethodPost, path, map[string]any{})
}

func patchNoBody(t *testing.T, c *httpClient, path string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("patchNoBody panic: %v", r)
		}
	}()
	_ = c.do(http.MethodPatch, path, map[string]any{})
}

func getJSON(t *testing.T, c *httpClient, path string) map[string]any {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("getJSON panic: %v", r)
		}
	}()
	return c.do(http.MethodGet, path, nil)
}

func containsUser(resp map[string]any, id string) bool {
	items, _ := resp["items"].([]any)
	for _, it := range items {
		row, _ := it.(map[string]any)
		if row["id"] == id {
			return true
		}
	}
	return false
}

func containsNotificationType(resp map[string]any, typ string) bool {
	items, _ := resp["items"].([]any)
	for _, it := range items {
		row, _ := it.(map[string]any)
		if row["type"] == typ {
			return true
		}
	}
	return false
}

func containsUnreadFrom(resp map[string]any, senderID string) bool {
	items, _ := resp["items"].([]any)
	for _, it := range items {
		row, _ := it.(map[string]any)
		if row["sender_id"] == senderID {
			if isRead, ok := row["is_read"].(bool); ok && !isRead {
				return true
			}
		}
	}
	return false
}

func waitForEvent(t *testing.T, conn *websocket.Conn, typ string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var msg map[string]any
		if err := conn.ReadJSON(&msg); err != nil {
			continue
		}
		if msg["type"] == typ {
			return
		}
	}
	t.Fatalf("did not receive event type %q in time", typ)
}

func waitForErrorContains(t *testing.T, conn *websocket.Conn, contains string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var msg map[string]any
		if err := conn.ReadJSON(&msg); err != nil {
			continue
		}
		if msg["type"] == "error" {
			if errText, _ := msg["error"].(string); strings.Contains(errText, contains) {
				return
			}
		}
	}
	t.Fatalf("did not receive expected error containing %q", contains)
}
