package notification

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"

	"github.com/bher20/eratemanager/internal/storage"
	"github.com/google/uuid"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Service struct {
	storage storage.Storage
}

func NewService(s storage.Storage) *Service {
	return &Service{storage: s}
}

func (s *Service) GetConfig(ctx context.Context) (*storage.EmailConfig, error) {
	return s.storage.GetEmailConfig(ctx)
}

func (s *Service) SaveConfig(ctx context.Context, cfg storage.EmailConfig) error {
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	return s.storage.SaveEmailConfig(ctx, cfg)
}

func (s *Service) SendEmail(ctx context.Context, to, subject, body string) error {
	cfg, err := s.storage.GetEmailConfig(ctx)
	if err != nil {
		return err
	}
	if cfg == nil || !cfg.Enabled {
		return errors.New("email not configured or disabled")
	}

	switch cfg.Provider {
	case "smtp", "gmail":
		return s.sendSMTP(cfg, to, subject, body)
	case "sendgrid":
		return s.sendSendgrid(cfg, to, subject, body)
	default:
		return fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}

func (s *Service) TestConfig(ctx context.Context, cfg storage.EmailConfig, to string) error {
	// Use the provided config to send a test email
	switch cfg.Provider {
	case "smtp", "gmail":
		return s.sendSMTP(&cfg, to, "Test Email", "This is a test email from eRateManager.")
	case "sendgrid":
		return s.sendSendgrid(&cfg, to, "Test Email", "This is a test email from eRateManager.")
	default:
		return fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}

func (s *Service) sendSMTP(cfg *storage.EmailConfig, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", to, subject, body))

	if cfg.Encryption == "ssl" {
		// SSL/TLS (Implicit)
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         cfg.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return err
		}
		defer c.Quit()

		if cfg.Username != "" && cfg.Password != "" {
			auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
			if err = c.Auth(auth); err != nil {
				return err
			}
		}

		if err = c.Mail(cfg.FromAddress); err != nil {
			return err
		}
		if err = c.Rcpt(to); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(msg)
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
		return nil
	} else if cfg.Encryption == "tls" {
		// STARTTLS (Explicit)
		c, err := smtp.Dial(addr)
		if err != nil {
			return err
		}
		defer c.Quit()

		if ok, _ := c.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: cfg.Host}
			if err = c.StartTLS(config); err != nil {
				return err
			}
		}

		if cfg.Username != "" && cfg.Password != "" {
			auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
			if err = c.Auth(auth); err != nil {
				return err
			}
		}

		if err = c.Mail(cfg.FromAddress); err != nil {
			return err
		}
		if err = c.Rcpt(to); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(msg)
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
		return nil
	} else {
		// None / Plain
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		return smtp.SendMail(addr, auth, cfg.FromAddress, []string{to}, msg)
	}
}

func (s *Service) sendSendgrid(cfg *storage.EmailConfig, to, subject, body string) error {
	from := mail.NewEmail(cfg.FromName, cfg.FromAddress)
	toEmail := mail.NewEmail("", to)
	message := mail.NewSingleEmail(from, subject, toEmail, body, body)
	client := sendgrid.NewSendClient(cfg.APIKey)
	resp, err := client.Send(message)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error: %d %s", resp.StatusCode, resp.Body)
	}
	return nil
}
