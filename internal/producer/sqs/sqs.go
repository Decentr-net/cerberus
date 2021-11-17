// Package sqs is an aws sqs implementation of producer
package sqs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/Decentr-net/cerberus/internal/producer"
)

var _ producer.Producer = &impl{}

type impl struct {
	queueURL string
	sqs      *sqs.SQS
}

// New returns new instance of impl.
func New(sqs *sqs.SQS, queueURL string) *impl { // nolint:golint
	return &impl{
		sqs:      sqs,
		queueURL: queueURL,
	}
}

// Produce sends message to SQS.
func (i impl) Produce(ctx context.Context, m *producer.PDVMessage) error {
	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to convert to base64: %w", err)
	}

	if _, err := i.sqs.SendMessageWithContext(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(body)),
		QueueUrl:    &i.queueURL,
	}); err != nil {
		return fmt.Errorf("failed to send sqs message: %w", err)
	}

	return nil
}
