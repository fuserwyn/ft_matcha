package handlers

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type pageCursor struct {
	Time time.Time
	ID   uuid.UUID
}

func parseCursorLimit(c *gin.Context, defaultLimit, maxLimit int) int {
	limit := defaultLimit
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= maxLimit {
			limit = n
		}
	}
	return limit
}

func parseLimitOffset(c *gin.Context) (limit, offset int) {
	limit = 20
	offset = 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

func parsePageCursor(raw string) (*pageCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cursor")
	}
	nano, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	return &pageCursor{
		Time: time.Unix(0, nano).UTC(),
		ID:   id,
	}, nil
}

func encodePageCursor(ts time.Time, id uuid.UUID) string {
	raw := fmt.Sprintf("%d|%s", ts.UTC().UnixNano(), id.String())
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func cursorTime(c *pageCursor) *time.Time {
	if c == nil {
		return nil
	}
	return &c.Time
}

func cursorID(c *pageCursor) *uuid.UUID {
	if c == nil {
		return nil
	}
	return &c.ID
}
