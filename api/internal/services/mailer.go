package services

import (
	"fmt"
	"net/smtp"
)

type Mailer struct {
	host string
	port string
	from string
}

func NewMailer(host, port, from string) *Mailer {
	return &Mailer{host: host, port: port, from: from}
}

func (m *Mailer) Send(to, subject, body string) error {
	if to == "" || m.host == "" || m.port == "" || m.from == "" {
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
	return smtp.SendMail(addr, nil, m.from, []string{to}, msg)
}
