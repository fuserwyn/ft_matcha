package handlers

import (
	"testing"
	"time"
)

func TestEffectiveOnline(t *testing.T) {
	now := time.Now().UTC()
	fresh := now.Add(-30 * time.Second)
	stale := now.Add(-2 * time.Minute)

	tests := []struct {
		name      string
		hubOnline bool
		lastSeen  *time.Time
		want      bool
	}{
		{"offline hub", false, &fresh, false},
		{"online fresh", true, &fresh, true},
		{"online stale", true, &stale, false},
		{"online nil last seen", true, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveOnline(tt.hubOnline, tt.lastSeen, now)
			if got != tt.want {
				t.Fatalf("effectiveOnline(...) = %v, want %v", got, tt.want)
			}
		})
	}
}
