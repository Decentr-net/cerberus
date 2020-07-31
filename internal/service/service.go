package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/Decentr-net/cerberus/internal/crypto"
	"github.com/Decentr-net/cerberus/internal/storage"
)

//go:generate mockgen -destination=./service_mock.go -package=service -source=service.go

// ErrNotFound means that requested object is not found.
var ErrNotFound = errors.New("not found")

// Service interface provides service's logic's methods.
type Service interface {
	// SendPDV sends PDV to storage.
	SendPDV(ctx context.Context, data []byte, filename string) error
	// ReceivePDV returns slice of bytes of PDV requested by address from storage.
	ReceivePDV(ctx context.Context, address string) ([]byte, error)
	// DoesPDVExist checks PDV existence by address in storage.
	DoesPDVExist(ctx context.Context, address string) (bool, error)
}

// service is Service interface implementation.
type service struct {
	c crypto.Crypto
	s storage.Storage
}

// New returns new instance of service.
func New(c crypto.Crypto, s storage.Storage) Service {
	return &service{
		c: c,
		s: s,
	}
}

// SendPDV sends PDV to storage.
func (s *service) SendPDV(ctx context.Context, data []byte, filepath string) error {
	enc, err := s.c.Encrypt(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create encrypting reader: %w", err)
	}

	if err := s.s.Write(ctx, enc, filepath); err != nil {
		return fmt.Errorf("failed to write to storage: %w", err)
	}

	return nil
}

// ReceivePDV returns slice of bytes of PDV requested by address from storage.
func (s *service) ReceivePDV(ctx context.Context, address string) ([]byte, error) {
	r, err := s.s.Read(ctx, address)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get data from storage: %w", err)
	}
	defer r.Close()

	dr, err := s.c.Decrypt(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create decrypting reader: %w", err)
	}

	data, err := ioutil.ReadAll(dr)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from decryping reader: %w", err)
	}

	return data, nil
}

// DoesPDVExist checks PDV existence by address in storage.
func (s *service) DoesPDVExist(ctx context.Context, address string) (bool, error) {
	exists, err := s.s.DoesExist(ctx, address)
	if err != nil {
		return false, fmt.Errorf("failed to check PDV existatnce in storage: %w", err)
	}

	return exists, nil
}
