package main

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

func sendEmail(to, subject, body string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")

	auth := smtp.PlainAuth("", from, password, "smtp.gmail.com")

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		from, to, subject, body)

	return smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		from,
		[]string{to},
		[]byte(msg),
	)

}

func parseEmailJob(payload string) (to, subject, body string, err error) {
	// payload format
	parts := strings.SplitN(payload, ":", 4)
	if len(parts) != 4 || parts[0] != "send_email" {
		return "", "", "", fmt.Errorf("invalid email job payload: %s", payload)
	}
	return parts[1], parts[2], parts[3], nil
}
