package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sirupsen/logrus"

	"github.com/Decentr-net/cerberus/internal/producer"
	"github.com/Decentr-net/cerberus/internal/producer/sqs"
)

type SQSOpts struct {
	SQSRegion         string `long:"sqs.region" env:"SQS_REGION" default:"" description:"sqs region"`
	SQSAccessKeyID    string `long:"sqs.access-key-id" env:"SQS_ACCESS_KEY_ID" description:"access key id for SQS"`
	SQSecretAccessKey string `long:"sqs.secret-access-key" env:"SQS_SECRET_ACCESS_KEY" description:"secret access key for SQS"`
	SQSQueue          string `long:"sqs.queue" env:"SQS_QUEUE" default:"testnet" description:"SQS queue name"`
}

func mustGetProducer() producer.Producer {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(opts.SQSRegion),
		Credentials: credentials.NewStaticCredentials(opts.SQSAccessKeyID, opts.SQSecretAccessKey, ""),
	}))

	c := awssqs.New(sess)
	queue, err := c.GetQueueUrl(&awssqs.GetQueueUrlInput{
		QueueName: &opts.SQSQueue,
	})
	if err != nil {
		logrus.WithError(err).Fatal("failed to get queue url")
	}

	return sqs.New(awssqs.New(sess), *queue.QueueUrl)
}
