package email

import (
	"gopkg.in/gomail.v2"
)

type Sender struct {
	host string
	port int
	user string
	pass string
	from string
}

func New(host string, port int, user, pass, from string) *Sender {
	return &Sender{host, port, user, pass, from}
}

func (s *Sender) Send(to, subject, html string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", html)

	d := gomail.NewDialer(s.host, s.port, s.user, s.pass)
	return d.DialAndSend(m)
}
