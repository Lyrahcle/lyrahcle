package mailer

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type Config struct {
	Host       string
	Port       string
	Username   string
	Password   string
	From       string
	ServerName string
}

type Client struct {
	cfg Config
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Enabled() bool {
	return c != nil && c.cfg.Host != "" && c.cfg.Port != "" && c.cfg.From != "" && c.cfg.ServerName != ""
}

func (c *Client) SendPasswordReset(to, resetURL string) error {
	if !c.Enabled() {
		return errors.New("mailer disabled")
	}
	address := net.JoinHostPort(c.cfg.Host, c.cfg.Port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: c.cfg.ServerName})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, c.cfg.ServerName)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()
	if c.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.ServerName)); err != nil {
			return err
		}
	}
	if err := client.Mail(c.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	subject := "Lyrahcle password recovery"
	body := "A password reset was requested for your Lyrahcle account.\r\n\r\nOpen this link within 15 minutes:\r\n" + resetURL + "\r\n\r\nIf you did not request this, ignore this message."
	message := strings.Join([]string{
		"From: " + c.cfg.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	if _, err := fmt.Fprint(writer, message); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
