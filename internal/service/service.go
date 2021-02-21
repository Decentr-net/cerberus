package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/Decentr-net/cerberus/internal/blockchain"
	"github.com/Decentr-net/cerberus/internal/crypto"
	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/pkg/api"
	"github.com/Decentr-net/cerberus/pkg/schema"
)

//go:generate mockgen -destination=./service_mock.go -package=service -source=service.go

// ErrNotFound means that requested object is not found.
var ErrNotFound = errors.New("not found")

// RewardMap contains dictionary with PDV types and rewards for them.
type RewardMap map[schema.PDVType]uint64

// Service interface provides service's logic's methods.
type Service interface {
	// SavePDV sends PDV to storage.
	SavePDV(ctx context.Context, p schema.PDV, owner sdk.AccAddress) (uint64, api.PDVMeta, error)
	// ListPDV lists PDVs.
	ListPDV(ctx context.Context, owner string, from uint64, limit uint16) ([]uint64, error)
	// ReceivePDV returns slice of bytes of PDV requested by address from storage.
	ReceivePDV(ctx context.Context, owner string, id uint64) ([]byte, error)
	// GetPDVMeta returns PDVs meta.
	GetPDVMeta(ctx context.Context, owner string, id uint64) (api.PDVMeta, error)
}

// service is Service interface implementation.
type service struct {
	c crypto.Crypto
	s storage.Storage
	b blockchain.Blockchain

	rewardMap RewardMap
}

// New returns new instance of service.
func New(c crypto.Crypto, s storage.Storage, b blockchain.Blockchain, rewardMap RewardMap) Service {
	return &service{
		c:         c,
		s:         s,
		b:         b,
		rewardMap: rewardMap,
	}
}

// SavePDV sends PDV to storage.
func (s *service) SavePDV(ctx context.Context, p schema.PDV, owner sdk.AccAddress) (uint64, api.PDVMeta, error) {
	id := uint64(time.Now().Unix())

	filepath := getPDVFilePath(owner.String(), id)

	meta := s.getMeta(p)

	data, err := json.Marshal(p)
	if err != nil {
		return 0, api.PDVMeta{}, fmt.Errorf("failed to marshal pdv: %w", err)
	}

	enc, size, err := s.c.Encrypt(bytes.NewReader(data))
	if err != nil {
		return 0, api.PDVMeta{}, fmt.Errorf("failed to create encrypting reader: %w", err)
	}

	if err := s.s.Write(ctx, enc, size, filepath); err != nil {
		return 0, api.PDVMeta{}, fmt.Errorf("failed to write pdv data to storage: %w", err)
	}

	data, err = json.Marshal(meta)
	if err != nil {
		return 0, api.PDVMeta{}, fmt.Errorf("failed to marshal pdv meta: %w", err)
	}

	mr := bytes.NewReader(data)
	if err := s.s.Write(ctx, mr, mr.Size(), getMetaFilePath(owner.String(), id)); err != nil {
		return 0, api.PDVMeta{}, fmt.Errorf("failed to write pdv meta to storage: %w", err)
	}

	if err := s.b.DistributeReward(owner, id, meta.Reward); err != nil {
		return 0, api.PDVMeta{}, fmt.Errorf("failed to send DistributeReward message to decentr: %w", err)
	}

	return id, meta, nil
}

// ListPDV lists PDVs.
func (s *service) ListPDV(ctx context.Context, owner string, from uint64, limit uint16) ([]uint64, error) {
	files, err := s.s.List(ctx, getPDVOwnerPrefix(owner), from, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	out := make([]uint64, len(files))
	for i, v := range files {
		id, err := getIDFromFilename(v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse id: %w", err)
		}
		out[i] = id
	}
	return out, nil
}

// ReceivePDV returns slice of bytes of PDV requested by address from storage.
func (s *service) ReceivePDV(ctx context.Context, owner string, id uint64) ([]byte, error) {
	r, err := s.s.Read(ctx, getPDVFilePath(owner, id))
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
func (s *service) GetPDVMeta(ctx context.Context, owner string, id uint64) (api.PDVMeta, error) {
	r, err := s.s.Read(ctx, getMetaFilePath(owner, id))
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

func getPDVOwnerPrefix(owner string) string {
	return fmt.Sprintf("pdv/%s", owner)
}

func getPDVFilePath(owner string, id uint64) string {
	// we need to have descending sort on s3 side, that's why we revert id and print it to hex
	return fmt.Sprintf("%s/%016x", getPDVOwnerPrefix(owner), math.MaxUint64-id)
}

func getMetaFilePath(owner string, id uint64) string {
	return fmt.Sprintf("meta/%s/%016x", owner, math.MaxUint64-id)
}

func getIDFromFilename(s string) (uint64, error) {
	v, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0, err
	}

	return math.MaxUint64 - v, nil
}
