package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/Decentr-net/cerberus/internal/crypto"
	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/pkg/api"
	"github.com/Decentr-net/cerberus/pkg/schema"
)

//go:generate mockgen -destination=./service_mock.go -package=service -source=service.go

// ErrNotFound means that requested object is not found.
var ErrNotFound = errors.New("not found")

const metaFilepathTpl = "%s/meta.json"

// RewardMap contains dictionary with PDV types and rewards for them.
type RewardMap map[schema.PDVType]uint64

// Service interface provides service's logic's methods.
type Service interface {
	// SavePDV sends PDV to storage.
	SavePDV(ctx context.Context, p schema.PDV, filename string) error
	// ReceivePDV returns slice of bytes of PDV requested by address from storage.
	ReceivePDV(ctx context.Context, address string) ([]byte, error)
	// GetPDVMeta returns PDVs meta.
	GetPDVMeta(ctx context.Context, address string) (api.PDVMeta, error)
}

// service is Service interface implementation.
type service struct {
	c crypto.Crypto
	s storage.Storage

	rewardMap RewardMap
}

// New returns new instance of service.
func New(c crypto.Crypto, s storage.Storage, rewardMap RewardMap) Service {
	return &service{
		c:         c,
		s:         s,
		rewardMap: rewardMap,
	}
}

// SavePDV sends PDV to storage.
func (s *service) SavePDV(ctx context.Context, p schema.PDV, filepath string) error {
	meta, err := json.Marshal(s.getMeta(p))
	if err != nil {
		return fmt.Errorf("failed to marshal pdv meta: %w", err)
	}

	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal pdv: %w", err)
	}

	enc, size, err := s.c.Encrypt(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create encrypting reader: %w", err)
	}

	if err := s.s.Write(ctx, enc, size, filepath); err != nil {
		return fmt.Errorf("failed to write pdv data to storage: %w", err)
	}

	mr := bytes.NewReader(meta)
	if err := s.s.Write(ctx, mr, mr.Size(), fmt.Sprintf(metaFilepathTpl, filepath)); err != nil {
		return fmt.Errorf("failed to write pdv meta to storage: %w", err)
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
	defer r.Close() // nolint

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

// GetPDVMeta returns pdv meta.
func (s *service) GetPDVMeta(ctx context.Context, address string) (api.PDVMeta, error) {
	r, err := s.s.Read(ctx, fmt.Sprintf(metaFilepathTpl, address))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return api.PDVMeta{}, ErrNotFound
		}
		return api.PDVMeta{}, fmt.Errorf("failed to get meta from storage: %w", err)
	}

	var m api.PDVMeta
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return api.PDVMeta{}, fmt.Errorf("failed to unmarshal meta: %w", err)
	}

	return m, nil
}

func (s *service) getMeta(p schema.PDV) api.PDVMeta {
	t := make(map[schema.PDVType]uint16)
	var reward uint64

	for _, v := range p.PDV {
		for _, d := range v.GetData() {
			t[d.Type()] = t[d.Type()] + 1
			reward += s.rewardMap[d.Type()]
		}
	}

	return api.PDVMeta{
		ObjectTypes: t,
		Reward:      reward,
	}
}
