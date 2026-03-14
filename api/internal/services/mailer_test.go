package services

import (
	"testing"
)

func TestMailer_SendSkipsWhenEmpty(t *testing.T) {
	m := NewMailer("smtp.example.com", "587", "from@example.com", 0)

	err := m.Send("", "sub", "body")
	if err != nil {
		t.Errorf("Send() with empty to should skip and return nil, got err=%v", err)
	}
}

func TestMailer_SendSkipsWithEmptyParams(t *testing.T) {
	tests := []struct {
		name  string
		mailer *Mailer
		to    string
	}{
		{"empty host", NewMailer("", "587", "from@x.com", 0), "to@x.com"},
		{"empty port", NewMailer("x.com", "", "from@x.com", 0), "to@x.com"},
		{"empty from", NewMailer("x.com", "587", "", 0), "to@x.com"},
		{"empty to", NewMailer("x.com", "587", "from@x.com", 0), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mailer.Send(tt.to, "sub", "body")
			if err != nil {
				t.Errorf("Send() should skip and return nil, got err=%v", err)
			}
		})
	}
}

func TestMailer_CooldownZeroAttemptsSend(t *testing.T) {
	m := NewMailer("127.0.0.1", "15999", "from@example.com", 0)
	err := m.Send("to@example.com", "sub", "body")
	if err == nil {
		t.Error("Send to closed port should return error, got nil")
	}
}
