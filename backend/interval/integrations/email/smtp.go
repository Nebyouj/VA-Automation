package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

type SMTPEmailService struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func NewSMTPEmailService(
	host string,
	port string,
	username string,
	password string,
	from string,
) *SMTPEmailService {
	return &SMTPEmailService{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
	}
}

func (s *SMTPEmailService) SendEmail(
	ctx context.Context,
	to string,
	subject string,
	body string,
	html string,
	jobID int,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	auth := smtp.PlainAuth(
		"",
		s.Username,
		s.Password,
		s.Host,
	)

	headers := []string{
		"From: Job Queue Service <" + s.From + ">",
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
	}

	var message string

	if html != "" {
		boundary := "boundary-job-queue"

		headers = append(
			headers,
			`Content-Type: multipart/alternative; boundary="`+boundary+`"`,
		)

		message = strings.Join(headers, "\r\n") +
			"\r\n\r\n" +
			"--" + boundary + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n" +
			"\r\n" +
			body +
			"\r\n\r\n" +
			"--" + boundary + "\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n" +
			"\r\n" +
			html +
			"\r\n\r\n" +
			"--" + boundary + "--\r\n"
	} else {
		headers = append(
			headers,
			"Content-Type: text/plain; charset=UTF-8",
		)

		message = strings.Join(headers, "\r\n") +
			"\r\n\r\n" +
			body
	}

	address := s.Host + ":" + s.Port

	err := smtp.SendMail(
		address,
		auth,
		s.From,
		[]string{to},
		[]byte(message),
	)

	if err != nil {
		return fmt.Errorf("failed tp send email: %w", err)
	}

	fmt.Printf(
		"Email sent to %s (job %d)\n",
		to,
		jobID,
	)

	return nil

}
