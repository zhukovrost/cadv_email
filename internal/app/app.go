package app

import (
	"github.com/zhukovrost/cadv_email/internal/config"
	"github.com/zhukovrost/cadv_email/internal/delivery"
	"github.com/zhukovrost/cadv_email/internal/service"
	logger "github.com/zhukovrost/cadv_logger"
	"go.uber.org/zap"
)

func Run(cfg *config.Config) {
	l := logger.New("standard", true)
	l.Info("Starting email service")

	emailService := service.New(l, cfg.SMTP)

	l.Info("Starting RabbitMQ")
	consumer, err := delivery.NewConsumer(cfg.RabbitMQ, emailService, l)
	if err != nil {
		l.Fatal("Error starting RabbitMQ consumer", zap.Error(err))
	}
	consumer.Run()
}
