package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/caarlos0/env/v6"
	"github.com/forsam-education/hermes/mailmessage"
	"github.com/forsam-education/hermes/storage"
	"github.com/forsam-education/redriver"
	"github.com/forsam-education/simplelogger"
	"gopkg.in/gomail.v2"
)

type config struct {
	TemplateBucket    string `env:"TEMPLATE_BUCKET"`
	AttachementBucket string `env:"ATTACHEMENT_BUCKET"`
	SMTPHost          string `env:"SMTP_HOST"`
	SMTPPort          int    `env:"SMTP_PORT" envDefault:"465"`
	SMTPUserName      string `env:"SMTP_USER"`
	SMTPPassword      string `env:"SMTP_PASS"`
	AWSRegion         string `env:"AWS_REGION_CODE"`
	QueueURL          string `env:"SQS_QUEUE"`
}

// HandleRequest is the main handler function used by the lambda runtime for the incomming event.
func HandleRequest(_ context.Context, event events.SQSEvent) error {
	simplelogger.GlobalLogger = simplelogger.NewDefaultLogger(simplelogger.DEBUG)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		err = fmt.Errorf("unable to parse configuration: %s", err.Error())
		simplelogger.GlobalLogger.StdError(err, nil)
		return err
	}
	smtpTransport := gomail.NewDialer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUserName, cfg.SMTPPassword)
	templateConnector, err := storage.NewS3(cfg.TemplateBucket, cfg.AWSRegion)
	if err != nil {
		err = fmt.Errorf("unable to instantiate template connector: %s", err.Error())
		simplelogger.GlobalLogger.StdError(err, nil)
		return err
	}

	attachementWriter, err := storage.NewS3(cfg.AttachementBucket, cfg.AWSRegion)
	if err != nil {
		err = fmt.Errorf("unable to instantiate attachement writer: %s", err.Error())
		simplelogger.GlobalLogger.StdError(err, nil)
		return err
	}

	messageRedriver := redriver.Redriver{Retries: 3, ConsumedQueueURL: cfg.QueueURL}

	err = messageRedriver.HandleMessages(event.Records, func(event events.SQSMessage) error {
		return mailmessage.SendMail(templateConnector, attachementWriter, smtpTransport, event.Body)
	})
	if err != nil {
		simplelogger.GlobalLogger.StdError(err, nil)
	}

	return err
}

func main() {
	lambda.Start(HandleRequest)
}
