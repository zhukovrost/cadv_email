package delivery

import (
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zhukovrost/cadv_email/internal/models"
	"github.com/zhukovrost/cadv_email/internal/service"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

type Consumer struct {
	Config
	l            *zap.Logger
	conn         *amqp.Connection
	channel      *amqp.Channel
	done         chan error
	emailService service.Mailer // Используем интерфейс Mailer
	shutdown     chan struct{}
}

type Config struct {
	URL          string `yaml:"url" envconfig:"RABBITMQ_URL"`
	Exchange     string `yaml:"exchange"`
	ExchangeType string `yaml:"exchange_type"`
	Queue        string `yaml:"queue"`
	ConsumerTag  string `yaml:"consumer_tag"`
}

func NewConsumer(cfg Config, emailService service.Mailer, logger *zap.Logger) (*Consumer, error) {
	c := &Consumer{
		Config:       cfg,
		l:            logger,
		done:         make(chan error),
		emailService: emailService,
		shutdown:     make(chan struct{}),
	}

	var err error

	config := amqp.Config{Properties: amqp.NewConnectionProperties()}
	config.Properties.SetClientConnectionName(c.ConsumerTag)
	logger.Debug("dialing RabbitMQ", zap.String("url", cfg.URL))
	c.conn, err = amqp.DialConfig(cfg.URL, config)
	if err != nil {
		return nil, fmt.Errorf("dial: %s", err)
	}

	c.l.Info("got RabbitMQ connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("channel: %s", err)
	}

	c.l.Info("got Channel, declaring Exchange", zap.String("exchange", c.Exchange))
	if err = c.channel.ExchangeDeclare(
		c.Exchange,     // name of the exchange
		c.ExchangeType, // type
		true,           // durable
		false,          // delete when complete
		false,          // internal
		false,          // noWait
		nil,            // arguments
	); err != nil {
		return nil, fmt.Errorf("exchange Declare: %s", err)
	}

	c.l.Info("declared Exchange, declaring Queue", zap.String("queue", c.Queue))
	queue, err := c.channel.QueueDeclare(
		c.Queue, // name of the queue
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // noWait
		nil,     // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("queue Declare: %s", err)
	}

	key := "" // Здесь добавь ключ биндинга, если он у тебя используется
	c.l.Info("declared Queue, binding to Exchange", zap.String("queue", queue.Name), zap.String("key", key), zap.String("exchange", c.Exchange))

	if err = c.channel.QueueBind(
		queue.Name, // name of the queue
		key,        // bindingKey
		c.Exchange, // sourceExchange
		false,      // noWait
		nil,        // arguments
	); err != nil {
		return nil, fmt.Errorf("queue Bind: %s", err)
	}

	c.l.Info("Queue bound to Exchange")

	return c, nil
}

func (c *Consumer) Run() {
	deliveries, err := c.channel.Consume(
		c.Queue,       // name
		c.ConsumerTag, // consumerTag,
		true,          // autoAck
		false,         // exclusive
		false,         // noLocal
		false,         // noWait
		nil,           // arguments
	)
	if err != nil {
		c.l.Fatal("queue Consume error", zap.Error(err))
	}

	go c.Handle(deliveries, c.done)
	c.SetupCloseHandler()
}

func (c *Consumer) Handle(deliveries <-chan amqp.Delivery, done chan error) {
	cleanup := func() {
		c.l.Info("handle: deliveries channel closed")
		close(done)
	}

	defer cleanup()

	for d := range deliveries {
		c.l.Debug("got new delivery", zap.Uint64("delivery_tag", d.DeliveryTag))

		// Предполагается, что Email — это твоя структура, например:
		var msg models.Email // предполагается, что у тебя есть структура Email с полями to, subject и body

		err := json.Unmarshal(d.Body, &msg)
		if err != nil {
			c.l.Error("error decoding JSON", zap.Error(err))
			continue
		}

		// Отправляем email через интерфейс Mailer
		err = c.emailService.SendEmail(msg.To, msg.Subject, msg.Body)
		if err != nil {
			c.l.Error("error sending email", zap.Error(err))
			continue
		}

		c.l.Info("successfully sent email", zap.String("to", msg.To))
	}
}

func (c *Consumer) SetupCloseHandler() {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		c.l.Info("rabbitmq stopped")
		if err := c.Shutdown(); err != nil {
			c.l.Fatal("error during shutdown", zap.Error(err))
		}
		os.Exit(0)
	}()
}

func (c *Consumer) Shutdown() error {
	select {
	case <-c.shutdown:
		// Already shutting down, return
		return nil
	default:
	}

	if c.channel != nil {
		if err := c.channel.Cancel(c.ConsumerTag, true); err != nil {
			c.l.Error("consumer cancel failed", zap.Error(err))
		}

		if err := c.channel.Close(); err != nil {
			c.l.Error("AMQP channel close error", zap.Error(err))
		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.l.Error("AMQP connection close error", zap.Error(err))
		}
	}

	c.l.Info("AMQP shutdown OK")
	return <-c.done
}
