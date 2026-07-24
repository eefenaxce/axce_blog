package utils

import (
	"crypto/tls"
	"fmt"
	"log"
	"mime"
	"net/smtp"

	"github.com/eefenaxce/axce_blog/internal/config"
)

type EmailSender struct {
	cfg config.EmailConfig
}

func NewEmailSender(cfg config.EmailConfig) *EmailSender {
	return &EmailSender{cfg: cfg}
}

func (s *EmailSender) Send(to, subject, body string) error {
	encodedSubject := mime.QEncoding.Encode("UTF-8", subject)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.cfg.From, to, encodedSubject, body)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	// For port 465, we usually need SSL/TLS from the start
	if s.cfg.Port == 465 {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         s.cfg.Host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			log.Printf("Failed to connect to SMTP server via SSL: %v", err)
			return err
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			log.Printf("Failed to create SMTP client: %v", err)
			return err
		}
		defer client.Quit()

		if err = client.Auth(auth); err != nil {
			log.Printf("SMTP auth failed: %v", err)
			return err
		}

		if err = client.Mail(s.cfg.From); err != nil {
			log.Printf("SMTP MAIL command failed: %v", err)
			return err
		}

		if err = client.Rcpt(to); err != nil {
			log.Printf("SMTP RCPT command failed: %v", err)
			return err
		}

		w, err := client.Data()
		if err != nil {
			log.Printf("SMTP DATA command failed: %v", err)
			return err
		}

		_, err = w.Write([]byte(msg))
		if err != nil {
			log.Printf("Failed to write email body: %v", err)
			return err
		}

		err = w.Close()
		if err != nil {
			log.Printf("Failed to close SMTP data writer: %v", err)
			return err
		}

		log.Printf("Successfully sent email to %s via SSL", to)
		return nil
	}

	// For other ports (like 587 or 25), use smtp.SendMail which handles STARTTLS if supported
	err := smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg))
	if err != nil {
		log.Printf("Failed to send email to %s: %v", to, err)
		return err
	}

	log.Printf("Successfully sent email to %s", to)
	return nil
}
