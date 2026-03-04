package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
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
	_ = postPhoto(t, &httpClient{base: public.base, token: userA.Token, c: public.c}, "/api/v1/photos", "a.png", tinyPNG())
	_ = postPhoto(t, &httpClient{base: public.base, token: userB.Token, c: public.c}, "/api/v1/photos", "b.png", tinyPNG())

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
	_ = postPhoto(t, &httpClient{base: public.base, token: userC.Token, c: public.c}, "/api/v1/photos", "c.png", tinyPNG())
	_ = postPhoto(t, &httpClient{base: public.base, token: userD.Token, c: public.c}, "/api/v1/photos", "d.png", tinyPNG())
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

func TestPhotoUploadAndEmailLikeE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run e2e tests against a running API")
	}

	base := os.Getenv("E2E_API_BASE")
	if base == "" {
		base = "http://localhost:8080"
	}
	mailhogBase := os.Getenv("E2E_MAILHOG_BASE")
	if mailhogBase == "" {
		mailhogBase = "http://localhost:8025"
	}

	public := &httpClient{
		base: strings.TrimRight(base, "/"),
		c:    &http.Client{Timeout: 10 * time.Second},
	}

	userA := registerUser(t, public, "pa")
	userB := registerUser(t, public, "pb")
	_ = postPhoto(t, &httpClient{base: public.base, token: userB.Token, c: public.c}, "/api/v1/photos", "b.png", tinyPNG())
	a := &httpClient{base: public.base, token: userA.Token, c: public.c}
	b := &httpClient{base: public.base, token: userB.Token, c: public.c}

	beforeTotal := readMailhogTotal(t, mailhogBase)

	photoResp := postPhoto(t, a, "/api/v1/photos", "avatar.png", tinyPNG())
	photoID, _ := photoResp["id"].(string)
	if photoID == "" {
		t.Fatalf("expected uploaded photo id, got: %#v", photoResp)
	}
	if isPrimary, _ := photoResp["is_primary"].(bool); !isPrimary {
		t.Fatalf("first uploaded photo must be primary, got: %#v", photoResp)
	}

	myPhotos := getJSON(t, a, "/api/v1/photos/me")
	items, _ := myPhotos["items"].([]any)
	if len(items) == 0 {
		t.Fatalf("expected at least one photo in /photos/me")
	}

	publicProfile := getJSON(t, b, "/api/v1/users/"+userA.ID.String())
	if publicProfile["primary_photo_url"] == nil {
		t.Fatalf("expected primary_photo_url in public profile: %#v", publicProfile)
	}

	postNoBody(t, b, "/api/v1/users/"+userA.ID.String()+"/like")

	waitForMailhogTotalAtLeast(t, mailhogBase, beforeTotal+1, 8*time.Second)
}

func TestBlockSearchVisitAndPresenceE2E(t *testing.T) {
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

	userA := registerUser(t, public, "blk_a")
	userB := registerUser(t, public, "blk_b")
	a := &httpClient{base: public.base, token: userA.Token, c: public.c}
	b := &httpClient{base: public.base, token: userB.Token, c: public.c}

	// Ensure users have photos (like requires primary photo).
	_ = postPhoto(t, a, "/api/v1/photos", "a.png", tinyPNG())
	_ = postPhoto(t, b, "/api/v1/photos", "b.png", tinyPNG())

	// Make a match first, then block should disable interactions.
	postNoBody(t, a, "/api/v1/users/"+userB.ID.String()+"/like")
	postNoBody(t, b, "/api/v1/users/"+userA.ID.String()+"/like")

	wsB := openWS(t, public.base, userB.Token)
	defer wsB.Close()

	_ = getJSON(t, a, "/api/v1/users/"+userB.ID.String()) // visit notification for B
	waitForEvent(t, wsB, "notification", 5*time.Second)

	postNoBody(t, a, "/api/v1/users/"+userB.ID.String()+"/block")

	listA := getJSON(t, a, "/api/v1/users")
	if containsUser(listA, userB.ID.String()) {
		t.Fatalf("blocked user B should not appear in A search results: %#v", listA)
	}
	listB := getJSON(t, b, "/api/v1/users")
	if containsUser(listB, userA.ID.String()) {
		t.Fatalf("user A should not appear in B search results after block: %#v", listB)
	}

	assertStatusJSON(t, b, http.MethodPost, "/api/v1/users/"+userA.ID.String()+"/messages", map[string]any{
		"content": "blocked?",
	}, http.StatusForbidden)
	assertStatusJSON(t, b, http.MethodPost, "/api/v1/users/"+userA.ID.String()+"/like", map[string]any{}, http.StatusForbidden)

	// Presence last_seen should be available after authenticated API activity.
	pres := getJSON(t, b, "/api/v1/presence/"+userA.ID.String())
	if pres["last_seen"] == nil {
		t.Fatalf("expected presence last_seen after authenticated calls, got: %#v", pres)
	}
}

func TestAuthAndProfileE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run e2e tests")
	}
	base := os.Getenv("E2E_API_BASE")
	if base == "" {
		base = "http://localhost:8080"
	}
	mailhogBase := os.Getenv("E2E_MAILHOG_BASE")
	if mailhogBase == "" {
		mailhogBase = "http://localhost:8025"
	}
	public := &httpClient{base: strings.TrimRight(base, "/"), c: &http.Client{Timeout: 10 * time.Second}}

	u := registerUser(t, public, "auth")
	verifyEmailFromMailhog(t, mailhogBase, public.base, u.Username+"@example.com")
	client := &httpClient{base: public.base, token: u.Token, c: public.c}

	// Login (verify login works)
	loginOut := postJSON(t, public, "/api/v1/auth/login", map[string]any{
		"username": u.Username,
		"password": "password123",
	})
	if loginOut["token"] == nil {
		t.Fatalf("login failed: %#v", loginOut)
	}
	me := getJSON(t, client, "/api/v1/auth/me")
	if me["id"] == nil {
		t.Fatalf("auth/me failed: %#v", me)
	}

	// Update profile
	putJSON(t, client, "/api/v1/profile/me", map[string]any{
		"bio":               "e2e bio",
		"gender":            "male",
		"sexual_preference": "female",
		"birth_date":        "1995-01-15",
		"city":              "Paris",
		"latitude":          48.8566,
		"longitude":         2.3522,
	})
	profile := getJSON(t, client, "/api/v1/profile/me")
	if bio, _ := profile["bio"].(string); bio != "e2e bio" {
		t.Fatalf("profile bio not updated: %#v", profile)
	}

	// Tags
	putJSON(t, client, "/api/v1/profile/me/tags", map[string]any{"tags": []string{"music", "travel"}})
	tagsResp := getJSON(t, client, "/api/v1/profile/me/tags")
	if tags, _ := tagsResp["tags"].([]any); len(tags) != 2 {
		t.Fatalf("tags not updated: %#v", tagsResp)
	}

	// Tag suggestions
	sugg := getJSON(t, client, "/api/v1/profile/tags/suggestions")
	if sugg["tags"] == nil {
		t.Fatalf("tag suggestions empty: %#v", sugg)
	}
}

func TestLikeWithoutPhotoE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run e2e tests")
	}
	base := os.Getenv("E2E_API_BASE")
	if base == "" {
		base = "http://localhost:8080"
	}
	public := &httpClient{base: strings.TrimRight(base, "/"), c: &http.Client{Timeout: 10 * time.Second}}

	userA := registerUser(t, public, "nophoto")
	userB := registerUser(t, public, "target")
	_ = postPhoto(t, &httpClient{base: public.base, token: userB.Token, c: public.c}, "/api/v1/photos", "b.png", tinyPNG())

	a := &httpClient{base: public.base, token: userA.Token, c: public.c}
	assertStatusJSON(t, a, http.MethodPost, "/api/v1/users/"+userB.ID.String()+"/like", map[string]any{}, http.StatusBadRequest)
}

func TestUnlikeAndReportE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run e2e tests")
	}
	base := os.Getenv("E2E_API_BASE")
	if base == "" {
		base = "http://localhost:8080"
	}
	public := &httpClient{base: strings.TrimRight(base, "/"), c: &http.Client{Timeout: 10 * time.Second}}

	userA := registerUser(t, public, "unlike_a")
	userB := registerUser(t, public, "unlike_b")
	_ = postPhoto(t, &httpClient{base: public.base, token: userA.Token, c: public.c}, "/api/v1/photos", "a.png", tinyPNG())
	_ = postPhoto(t, &httpClient{base: public.base, token: userB.Token, c: public.c}, "/api/v1/photos", "b.png", tinyPNG())

	a := &httpClient{base: public.base, token: userA.Token, c: public.c}

	postNoBody(t, a, "/api/v1/users/"+userB.ID.String()+"/like")
	liked := getJSON(t, a, "/api/v1/likes")
	if !containsUser(liked, userB.ID.String()) {
		t.Fatalf("expected B in A likes: %#v", liked)
	}

	deleteNoBody(t, a, "/api/v1/users/"+userB.ID.String()+"/like")
	likedAfter := getJSON(t, a, "/api/v1/likes")
	if containsUser(likedAfter, userB.ID.String()) {
		t.Fatalf("unlike failed, B still in likes: %#v", likedAfter)
	}

	postJSON(t, a, "/api/v1/users/"+userB.ID.String()+"/report", map[string]any{
		"reason":  "fake_account",
		"comment": "e2e report",
	})
	reports := getJSON(t, a, "/api/v1/reports/me")
	if items, _ := reports["items"].([]any); len(items) < 1 {
		t.Fatalf("report not listed: %#v", reports)
	}
}

func TestUnblockE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run e2e tests")
	}
	base := os.Getenv("E2E_API_BASE")
	if base == "" {
		base = "http://localhost:8080"
	}
	public := &httpClient{base: strings.TrimRight(base, "/"), c: &http.Client{Timeout: 10 * time.Second}}

	userA := registerUser(t, public, "unblk_a")
	userB := registerUser(t, public, "unblk_b")
	_ = postPhoto(t, &httpClient{base: public.base, token: userA.Token, c: public.c}, "/api/v1/photos", "a.png", tinyPNG())
	_ = postPhoto(t, &httpClient{base: public.base, token: userB.Token, c: public.c}, "/api/v1/photos", "b.png", tinyPNG())

	a := &httpClient{base: public.base, token: userA.Token, c: public.c}
	postNoBody(t, a, "/api/v1/users/"+userB.ID.String()+"/block")
	list := getJSON(t, a, "/api/v1/users")
	if containsUser(list, userB.ID.String()) {
		t.Fatalf("B should not appear after block: %#v", list)
	}

	deleteNoBody(t, a, "/api/v1/users/"+userB.ID.String()+"/block")
	listAfter := getJSON(t, a, "/api/v1/users")
	if !containsUser(listAfter, userB.ID.String()) {
		t.Fatalf("B should appear after unblock: %#v", listAfter)
	}
}

func TestPhotosAndDiscoveryE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run e2e tests")
	}
	base := os.Getenv("E2E_API_BASE")
	if base == "" {
		base = "http://localhost:8080"
	}
	public := &httpClient{base: strings.TrimRight(base, "/"), c: &http.Client{Timeout: 10 * time.Second}}

	userA := registerUser(t, public, "photo_a")
	a := &httpClient{base: public.base, token: userA.Token, c: public.c}

	p1 := postPhoto(t, a, "/api/v1/photos", "1.png", tinyPNG())
	p2 := postPhoto(t, a, "/api/v1/photos", "2.png", tinyPNG())
	pid1, _ := p1["id"].(string)
	pid2, _ := p2["id"].(string)
	if pid1 == "" || pid2 == "" {
		t.Fatalf("photo ids missing: %#v %#v", p1, p2)
	}

	patchNoBody(t, a, "/api/v1/photos/"+pid2+"/primary")
	photos := getJSON(t, a, "/api/v1/photos/me")
	if items, ok := photos["items"].([]any); ok {
		for _, it := range items {
			row, _ := it.(map[string]any)
			if row["id"] == pid2 && !row["is_primary"].(bool) {
				t.Fatalf("photo 2 should be primary: %#v", items)
			}
		}
	}

	deleteNoBody(t, a, "/api/v1/photos/"+pid1)
	photosAfter := getJSON(t, a, "/api/v1/photos/me")
	if items, ok := photosAfter["items"].([]any); ok {
		for _, it := range items {
			if it.(map[string]any)["id"] == pid1 {
				t.Fatalf("photo 1 should be deleted: %#v", items)
			}
		}
	}

	// Discovery search with filters
	search := getJSON(t, a, "/api/v1/users?gender=female&min_age=18&max_age=99&city=Paris&limit=5")
	if search["items"] == nil {
		t.Fatalf("search returned no items key: %#v", search)
	}
}

func TestRESTChatAndViewsE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run e2e tests")
	}
	base := os.Getenv("E2E_API_BASE")
	if base == "" {
		base = "http://localhost:8080"
	}
	public := &httpClient{base: strings.TrimRight(base, "/"), c: &http.Client{Timeout: 10 * time.Second}}

	userA := registerUser(t, public, "chat_a")
	userB := registerUser(t, public, "chat_b")
	_ = postPhoto(t, &httpClient{base: public.base, token: userA.Token, c: public.c}, "/api/v1/photos", "a.png", tinyPNG())
	_ = postPhoto(t, &httpClient{base: public.base, token: userB.Token, c: public.c}, "/api/v1/photos", "b.png", tinyPNG())

	a := &httpClient{base: public.base, token: userA.Token, c: public.c}
	b := &httpClient{base: public.base, token: userB.Token, c: public.c}

	postNoBody(t, a, "/api/v1/users/"+userB.ID.String()+"/like")
	postNoBody(t, b, "/api/v1/users/"+userA.ID.String()+"/like")

	postJSON(t, a, "/api/v1/users/"+userB.ID.String()+"/messages", map[string]any{"content": "hello rest"})
	msgs := getJSON(t, a, "/api/v1/users/"+userB.ID.String()+"/messages")
	if items, _ := msgs["items"].([]any); len(items) < 1 {
		t.Fatalf("messages empty: %#v", msgs)
	}

	patchNoBody(t, b, "/api/v1/users/"+userA.ID.String()+"/messages/read")
	msgsAfter := getJSON(t, b, "/api/v1/users/"+userA.ID.String()+"/messages")
	if containsUnreadFrom(msgsAfter, userA.ID.String()) {
		t.Fatalf("messages should be read: %#v", msgsAfter)
	}

	_ = getJSON(t, a, "/api/v1/users/"+userB.ID.String())
	views := getJSON(t, a, "/api/v1/profile/me/views")
	if items, _ := views["items"].([]any); len(items) < 1 {
		t.Fatalf("views empty after visiting B: %#v", views)
	}

	notifs := getJSON(t, b, "/api/v1/notifications")
	patchNoBody(t, b, "/api/v1/notifications/read-all")
	if notifs["items"] != nil {
		notifsAfter := getJSON(t, b, "/api/v1/notifications?unread_only=true")
		if items, _ := notifsAfter["items"].([]any); len(items) != 0 {
			t.Fatalf("notifications should be read: %#v", notifsAfter)
		}
	}
}

type registeredUser struct {
	ID       uuid.UUID
	Token    string
	Username string
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
	return registeredUser{ID: id, Token: token, Username: username}
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

func (c *httpClient) doWithStatus(method, path string, body any) (int, map[string]any, string) {
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
	if len(data) == 0 {
		return res.StatusCode, map[string]any{}, ""
	}
	var out map[string]any
	if data[0] == '[' {
		var arr []any
		_ = json.Unmarshal(data, &arr)
		return res.StatusCode, map[string]any{"items": arr}, string(data)
	}
	_ = json.Unmarshal(data, &out)
	return res.StatusCode, out, string(data)
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

func deleteNoBody(t *testing.T, c *httpClient, path string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("deleteNoBody panic: %v", r)
		}
	}()
	_ = c.do(http.MethodDelete, path, nil)
}

func putJSON(t *testing.T, c *httpClient, path string, body any) map[string]any {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("putJSON panic: %v", r)
		}
	}()
	return c.do(http.MethodPut, path, body)
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

func assertStatusJSON(t *testing.T, c *httpClient, method, path string, body any, expected int) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("assertStatusJSON panic: %v", r)
		}
	}()
	status, _, raw := c.doWithStatus(method, path, body)
	if status != expected {
		t.Fatalf("expected status=%d got=%d path=%s body=%s", expected, status, path, raw)
	}
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

func postPhoto(t *testing.T, c *httpClient, path, filename string, data []byte) map[string]any {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", "image/png")
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("create multipart part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write multipart data: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.base+path, &body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	res, err := c.c.Do(req)
	if err != nil {
		t.Fatalf("post photo request failed: %v", err)
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		t.Fatalf("post photo failed: status=%d body=%s", res.StatusCode, string(raw))
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode photo response: %v body=%s", err, string(raw))
	}
	return out
}

func tinyPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0xF8, 0xCF, 0x00, 0x00,
		0x02, 0x05, 0x01, 0x02, 0xA2, 0x5D, 0xC6, 0x9B,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44,
		0xAE, 0x42, 0x60, 0x82,
	}
}

func readMailhogTotal(t *testing.T, mailhogBase string) int {
	t.Helper()
	resp, err := http.Get(strings.TrimRight(mailhogBase, "/") + "/api/v2/messages")
	if err != nil {
		t.Fatalf("mailhog request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("mailhog status=%d body=%s", resp.StatusCode, string(raw))
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode mailhog response: %v", err)
	}
	switch v := out["total"].(type) {
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		return 0
	}
}

func waitForMailhogTotalAtLeast(t *testing.T, mailhogBase string, minTotal int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if total := readMailhogTotal(t, mailhogBase); total >= minTotal {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("mailhog total did not reach %d within %s", minTotal, timeout)
}

func verifyEmailFromMailhog(t *testing.T, mailhogBase, apiBase, toEmail string) {
	t.Helper()
	mailhogBase = strings.TrimRight(mailhogBase, "/")
	apiBase = strings.TrimRight(apiBase, "/")

	// Wait for verification email to arrive (async send)
	for attempt := 0; attempt < 10; attempt++ {
		if tryVerifyFromMailhog(t, mailhogBase, apiBase, toEmail) {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("no verification email found for %s in MailHog after retries", toEmail)
}

func tryVerifyFromMailhog(t *testing.T, mailhogBase, apiBase, toEmail string) bool {
	t.Helper()
	// Get recent messages from v2 API (returns "items")
	resp, err := http.Get(mailhogBase + "/api/v2/messages?limit=50")
	if err != nil {
		t.Fatalf("mailhog messages: %v", err)
	}
	defer resp.Body.Close()
	var out struct {
		Items []struct {
			ID      string `json:"ID"`
			To      []struct {
				Mailbox string `json:"Mailbox"`
				Domain string `json:"Domain"`
			} `json:"To"`
			Content struct {
				Body string `json:"Body"`
			} `json:"Content"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("mailhog messages decode: %v", err)
	}
	// Find message to this email with verification link
	for i := range out.Items {
		m := &out.Items[i]
		for _, to := range m.To {
			addr := to.Mailbox + "@" + to.Domain
			if addr == toEmail && strings.Contains(m.Content.Body, "verify-email?token=") {
				extractAndVerify(t, apiBase, m.Content.Body)
				return true
			}
		}
	}
	return false
}

func extractAndVerify(t *testing.T, apiBase, body string) {
	t.Helper()
	// Extract token from verify-email?token=XXX
	idx := strings.Index(body, "verify-email?token=")
	if idx < 0 {
		t.Fatalf("no verify-email link in body")
	}
	idx += len("verify-email?token=")
	end := strings.IndexAny(body[idx:], " \n\r\t")
	token := body[idx:]
	if end >= 0 {
		token = body[idx : idx+end]
	}
	token = strings.TrimSpace(token)
	if token == "" {
		t.Fatalf("empty token in verify link")
	}
	verifyURL := apiBase + "/api/v1/auth/verify-email?token=" + token
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp2, err := client.Get(verifyURL)
	if err != nil {
		t.Fatalf("verify request: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusFound {
		raw, _ := io.ReadAll(resp2.Body)
		t.Fatalf("verify failed: status=%d body=%s", resp2.StatusCode, string(raw))
	}
	loc := resp2.Header.Get("Location")
	if loc == "" || !strings.Contains(loc, "/matches") {
		t.Fatalf("verify redirect missing /matches: Location=%q", loc)
	}
}
