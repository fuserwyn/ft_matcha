package services

import (
	"fmt"
	"net/smtp"
	"sync"
	"time"
)

type Mailer struct {
	host     string
	port     string
	from     string
	cooldown time.Duration

	mu        sync.Mutex
	lastByKey map[string]time.Time
}

func NewMailer(host, port, from string, cooldownSeconds int) *Mailer {
	return &Mailer{
		host:      host,
		port:      port,
		from:      from,
		cooldown:  time.Duration(cooldownSeconds) * time.Second,
		lastByKey: make(map[string]time.Time),
	}
}

func (m *Mailer) Send(to, subject, body string) error {
	if to == "" || m.host == "" || m.port == "" || m.from == "" {
		return nil
	}
	if m.shouldSkip(to, subject) {
		return nil
	}

	msg := []byte(
		fmt.Sprintf(
			"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
			m.from,
			to,
			subject,
			body,
		),
	)

	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	if err := smtp.SendMail(addr, nil, m.from, []string{to}, msg); err != nil {
		return err
	}
	m.markSent(to, subject)
	return nil
}

func (m *Mailer) shouldSkip(to, subject string) bool {
	if m.cooldown <= 0 {
		return false
	}
	key := to + "|" + subject
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()
	last, ok := m.lastByKey[key]
	return ok && now.Sub(last) < m.cooldown
}

func (m *Mailer) markSent(to, subject string) {
	if m.cooldown <= 0 {
		return
	}
	key := to + "|" + subject

	m.mu.Lock()
	m.lastByKey[key] = time.Now()
	m.mu.Unlock()
}
