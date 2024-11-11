package service

import (
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type Mailer interface {
	SendEmail(to, subject, body string) error
}

type MyMailer struct {
	Config
	l      *zap.Logger
	dialer *gomail.Dialer
}

type Config struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"user"`
	Password string `yaml:"password"`
	Sender   string `yaml:"sender"`
}

func New(log *zap.Logger, cfg Config) *MyMailer {
	dialer := gomail.NewDialer(
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
	)

	// Настраиваем TLSConfig
	//dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	return &MyMailer{
		dialer: dialer,
		Config: cfg,
		l:      log,
	}
}

// SendEmail отправляет письмо с заданными параметрами
func (m *MyMailer) SendEmail(to, subject, body string) error {
	m.l.Debug("Sending email", zap.String("from", m.Config.Sender), zap.String("to", to))

	// Создание нового письма
	message := gomail.NewMessage()
	message.SetHeader("From", m.Config.Sender)
	message.SetHeader("To", to)
	message.SetHeader("Subject", subject)
	message.SetBody("text/plain", body)

	// Если нужно отправить HTML, можно раскомментировать
	// message.SetBody("text/html", "<h1>Hello!</h1><p>This is a test email.</p>")

	// Если необходимо прикрепить файл, можно добавить:
	// message.Attach("/path/to/file")

	// Отправка письма с помощью dialer
	if err := m.dialer.DialAndSend(message); err != nil {
		m.l.Error("Failed to send email", zap.Error(err))
		return err
	}

	m.l.Debug("Email sent successfully", zap.String("to", to))
	return nil
}
