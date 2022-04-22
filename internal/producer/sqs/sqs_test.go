//go:build integration
// +build integration

package sqs

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Decentr-net/cerberus/internal/entities"
	"github.com/Decentr-net/cerberus/internal/producer"
	"github.com/Decentr-net/cerberus/pkg/schema"
)

var (
	ctx = context.Background()
	c   *sqs.SQS
	p   *impl
)

func TestMain(m *testing.M) {
	shutdown := setup()

	code := m.Run()

	shutdown()
	os.Exit(code)
}

func setup() func() {
	req := testcontainers.ContainerRequest{
		Image:        "pafortin/goaws",
		ExposedPorts: []string{"4100/tcp"},
		WaitingFor:   wait.ForListeningPort("4100/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
	})
	if err != nil {
		logrus.WithError(err).Fatalf("failed to create sqs container")
	}

	if err := container.Start(ctx); err != nil {
		logrus.WithError(err).Fatal("failed to start container")
	}

	host, err := container.Host(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("failed to get host")
	}

	port, err := container.MappedPort(ctx, "4100")
	if err != nil {
		logrus.WithError(err).Fatal("failed to map port")
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:    aws.String(fmt.Sprintf("http://%s:%d", host, port.Int())),
		Region:      aws.String("reg"),
		Credentials: credentials.AnonymousCredentials,
	}))

	c = sqs.New(sess)

	queue, err := c.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String("pdv"),
	})
	if err != nil {
		logrus.WithError(err).Fatal("failed to create queue")
	}

	p = New(c, *queue.QueueUrl)

	return func() {
		if container == nil {
			return
		}
		if err := container.Terminate(ctx); err != nil {
			logrus.WithError(err).Error("failed to terminate container")
		}
	}
}

func TestImpl_Produce(t *testing.T) {
	require.NoError(t, p.Produce(ctx, &producer.PDVMessage{
		ID:      1,
		Address: "1",
		Meta: &entities.PDVMeta{
			ObjectTypes: map[schema.Type]uint16{
				schema.PDVCookieType: 1,
			},
			Reward: sdk.NewDecWithPrec(1, 6),
		},
		Data: []byte(`{}`),
	}))

	m, err := c.ReceiveMessage(&sqs.ReceiveMessageInput{
		WaitTimeSeconds: aws.Int64(2),
		QueueUrl:        &p.queueURL,
	})
	require.NoError(t, err)
	require.Len(t, m.Messages, 1)
	require.Equal(t, `{"ID":1,"Address":"1","Meta":{"object_types":{"cookie":1},"reward":"0.000001000000000000"},"Data":"e30="}`, *m.Messages[0].Body)
}
