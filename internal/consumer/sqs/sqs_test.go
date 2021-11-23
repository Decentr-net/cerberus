// +build integration

package sqs

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Decentr-net/cerberus/internal/blockchain"
	blockchainmock "github.com/Decentr-net/cerberus/internal/blockchain/mock"
	"github.com/Decentr-net/cerberus/internal/entities"
	"github.com/Decentr-net/cerberus/internal/producer"
	sqsproducer "github.com/Decentr-net/cerberus/internal/producer/sqs"
	"github.com/Decentr-net/cerberus/internal/schema"
	"github.com/Decentr-net/cerberus/internal/storage"
	storagemock "github.com/Decentr-net/cerberus/internal/storage/mock"
)

var (
	ctx      = context.Background()
	c        *sqs.SQS
	queueURL string
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

	queueURL = *queue.QueueUrl

	return func() {
		if container == nil {
			return
		}
		if err := container.Terminate(ctx); err != nil {
			logrus.WithError(err).Error("failed to terminate container")
		}
	}
}

func TestImpl_ProcessMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	i := New(fs, is, b, c, queueURL)
	p := sqsproducer.New(c, queueURL)

	addr1, addr2 := "addr1", "addr2"
	ctx, cancel := context.WithCancel(ctx)

	require.NoError(t, p.Produce(ctx, &producer.PDVMessage{
		ID:      1,
		Address: addr1,
		Meta: &entities.PDVMeta{
			ObjectTypes: map[schema.Type]uint16{
				schema.PDVCookieType: 1,
			},
			Reward: 1,
		},
		Data: []byte(`{"id": 1}`),
	}))
	require.NoError(t, p.Produce(ctx, &producer.PDVMessage{
		ID:      2,
		Address: addr2,
		Meta: &entities.PDVMeta{
			ObjectTypes: map[schema.Type]uint16{
				schema.PDVCookieType: 2,
			},
			Reward: 2,
		},
		Data: []byte(`{"id": 2}`),
	}))

	is.EXPECT().InTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, f func(s storage.IndexStorage) error) error {
		defer cancel()
		return f(is)
	})
	is.EXPECT().GetPDVMeta(gomock.Any(), addr1, uint64(1)).Return(&entities.PDVMeta{}, nil)
	is.EXPECT().GetPDVMeta(gomock.Any(), addr2, uint64(2)).Return(nil, storage.ErrNotFound)
	fs.EXPECT().Write(
		gomock.Any(),
		gomock.Any(),
		int64(9),
		"addr2/pdv/fffffffffffffffd",
		"binary/octet-stream",
		false,
	).DoAndReturn(func(_ context.Context, data io.Reader, _ int64, _, _ string, _ bool) (string, error) {
		b, err := io.ReadAll(data)
		require.NoError(t, err)
		require.Equal(t, `{"id": 2}`, string(b))
		return "2", nil
	})
	b.EXPECT().DistributeRewards([]blockchain.Reward{{
		Receiver: addr2,
		ID:       2,
		Reward:   2,
	}}).Return("tx", nil)
	is.EXPECT().SetPDVMeta(gomock.Any(), addr2, uint64(2), "tx", &entities.PDVMeta{
		ObjectTypes: map[schema.Type]uint16{
			schema.PDVCookieType: 2,
		},
		Reward: 2,
	})

	require.ErrorIs(t, i.Run(ctx), context.Canceled)

	attr, err := c.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		AttributeNames: []*string{aws.String(sqs.QueueAttributeNameAll)},
		QueueUrl:       aws.String(queueURL),
	})
	require.NoError(t, err)
	require.Equal(t, "0", *attr.Attributes[sqs.QueueAttributeNameApproximateNumberOfMessages])
	require.Equal(t, "0", *attr.Attributes[sqs.QueueAttributeNameApproximateNumberOfMessagesNotVisible])
}
