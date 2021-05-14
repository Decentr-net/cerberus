// Package service contains business logic of application.
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
	"github.com/sirupsen/logrus"

	logging "github.com/Decentr-net/logrus/context"

	"github.com/Decentr-net/cerberus/internal/blockchain"
	"github.com/Decentr-net/cerberus/internal/crypto"
	"github.com/Decentr-net/cerberus/internal/schema"
	"github.com/Decentr-net/cerberus/internal/storage"
)

//go:generate mockgen -destination=./mock/service.go -package=mock -source=service.go

// ErrNotFound means that requested object is not found.
var ErrNotFound = errors.New("not found")

// RewardMap contains dictionary with PDV types and rewards for them.
type RewardMap map[schema.Type]uint64

// PDVMeta contains info about PDV.
type PDVMeta struct {
	// ObjectTypes represents how much certain meta data meta contains.
	ObjectTypes map[schema.Type]uint16 `json:"object_types"`
	Reward      uint64                 `json:"reward"`
}

// Profile ...
type Profile struct {
	Address   string
	FirstName string
	LastName  string
	Bio       string
	Avatar    string
	Gender    string
	Birthday  time.Time
	UpdatedAt *time.Time
	CreatedAt time.Time
}

// Service interface provides service's logic's methods.
type Service interface {
	// SavePDV sends PDV to storage.
	SavePDV(ctx context.Context, p schema.PDV, owner sdk.AccAddress) (uint64, PDVMeta, error)
	// ListPDV lists PDVs.
	ListPDV(ctx context.Context, owner string, from uint64, limit uint16) ([]uint64, error)
	// ReceivePDV returns slice of bytes of PDV requested by address from storage.
	ReceivePDV(ctx context.Context, owner string, id uint64) ([]byte, error)
	// GetPDVMeta returns PDVs meta.
	GetPDVMeta(ctx context.Context, owner string, id uint64) (PDVMeta, error)

	// GetProfile ...
	GetProfiles(ctx context.Context, owner []string) ([]*Profile, error)

	// GetRewardsMap ...
	GetRewardsMap() RewardMap
}

// service is Service interface implementation.
type service struct {
	c  crypto.Crypto
	is storage.IndexStorage
	fs storage.FileStorage
	b  blockchain.Blockchain

	rewardMap RewardMap
}

// New returns new instance of service.
func New(
	c crypto.Crypto,
	fs storage.FileStorage,
	is storage.IndexStorage,
	b blockchain.Blockchain,
	rewardMap RewardMap,
) Service {
	return &service{
		c:         c,
		fs:        fs,
		is:        is,
		b:         b,
		rewardMap: rewardMap,
	}
}

// SavePDV sends PDV to storage.
func (s *service) SavePDV(ctx context.Context, p schema.PDV, owner sdk.AccAddress) (uint64, PDVMeta, error) {
	log := logging.GetLogger(ctx)

	id := uint64(time.Now().Unix())
	filepath := getPDVFilePath(owner.String(), id)

	meta, err := s.processPDV(ctx, owner, p)
	if err != nil {
		return 0, PDVMeta{}, fmt.Errorf("failed to process meta: %w", err)
	}

	data, err := json.Marshal(p)
	if err != nil {
		return 0, PDVMeta{}, fmt.Errorf("failed to marshal meta: %w", err)
	}

	log.Debug("encrypting meta")
	enc, size, err := s.c.Encrypt(bytes.NewReader(data))
	if err != nil {
		return 0, PDVMeta{}, fmt.Errorf("failed to create encrypting reader: %w", err)
	}

	log.WithField("filepath", filepath).Debug("writing meta into the storage")
	if err := s.fs.Write(ctx, enc, size, filepath); err != nil {
		return 0, PDVMeta{}, fmt.Errorf("failed to write meta data to storage: %w", err)
	}

	data, err = json.Marshal(meta)
	if err != nil {
		return 0, PDVMeta{}, fmt.Errorf("failed to marshal meta meta: %w", err)
	}

	log.WithField("filepath", filepath).Debug("writing meta into the storage")
	mr := bytes.NewReader(data)
	if err := s.fs.Write(ctx, mr, mr.Size(), getMetaFilePath(owner.String(), id)); err != nil {
		return 0, PDVMeta{}, fmt.Errorf("failed to write meta meta to storage: %w", err)
	}

	if meta.Reward > 0 {
		log.WithFields(logrus.Fields{
			"owner":  owner.String(),
			"meta":   id,
			"amount": meta.Reward,
		}).Debug("distributing reward")
		if err := s.b.DistributeReward(owner, id, meta.Reward); err != nil {
			return 0, PDVMeta{}, fmt.Errorf("failed to send DistributeReward message to decentr: %w", err)
		}
	}

	return id, meta, nil
}

// ListPDV lists PDVs.
func (s *service) ListPDV(ctx context.Context, owner string, from uint64, limit uint16) ([]uint64, error) {
	logging.GetLogger(ctx).WithFields(logrus.Fields{
		"prefix": getMetaOwnerPrefix(owner),
		"from":   from,
		"limit":  limit,
	}).Debug("trying to list meta")
	files, err := s.fs.List(ctx, getMetaOwnerPrefix(owner), from, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	out := make([]uint64, len(files))
	for i, v := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

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
	log := logging.GetLogger(ctx)

	log.WithField("filepath", getPDVFilePath(owner, id)).Debug("reading meta from storage")
	r, err := s.fs.Read(ctx, getPDVFilePath(owner, id))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get data from storage: %w", err)
	}
	defer r.Close() // nolint

	log.Debug("decrypting meta")
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

// GetPDVMeta returns meta meta.
func (s *service) GetPDVMeta(ctx context.Context, owner string, id uint64) (PDVMeta, error) {
	logging.GetLogger(ctx).WithField("filepath", getMetaFilePath(owner, id)).Debug("reading meta from storage")
	r, err := s.fs.Read(ctx, getMetaFilePath(owner, id))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return PDVMeta{}, ErrNotFound
		}
		return PDVMeta{}, fmt.Errorf("failed to get meta from storage: %w", err)
	}

	var m PDVMeta
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return PDVMeta{}, fmt.Errorf("failed to unmarshal meta: %w", err)
	}

	return m, nil
}

// GetProfiles ...
func (s *service) GetProfiles(ctx context.Context, owner []string) ([]*Profile, error) {
	pp, err := s.is.GetProfiles(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles: %w", err)
	}

	out := make([]*Profile, len(pp))
	for i, v := range pp {
		out[i] = (*Profile)(v)
	}

	return out, nil
}

// GetRewardsConfig ...
func (s *service) GetRewardsMap() RewardMap {
	return s.rewardMap
}

func (s *service) getMeta(ctx context.Context, owner sdk.AccAddress, p schema.PDV) (PDVMeta, error) {
	t := make(map[schema.Type]uint16)
	var reward uint64

	for _, d := range p.Data() {
		t[d.Type()] = t[d.Type()] + 1

		switch d.Type() {
		case schema.PDVProfileType:
			if _, err := s.is.GetProfile(ctx, owner.String()); err == nil {
				continue // we want reward user only for initial profile
			} else if err != storage.ErrNotFound {
				return PDVMeta{}, fmt.Errorf("failed to check profile: %w", err)
			}
		default:
		}

		reward += s.rewardMap[d.Type()]
	}

	return PDVMeta{
		ObjectTypes: t,
		Reward:      reward,
	}, nil
}

func (s *service) processPDV(ctx context.Context, owner sdk.AccAddress, p schema.PDV) (PDVMeta, error) {
	meta, err := s.getMeta(ctx, owner, p)
	if err != nil {
		return PDVMeta{}, fmt.Errorf("failed to get meta: %w", err)
	}

	for _, d := range p.Data() {
		switch d.Type() {
		case schema.PDVProfileType:
			if err := s.is.SetProfile(ctx, getSetProfileParams(owner, d.(schema.V1Profile))); err != nil {
				return PDVMeta{}, fmt.Errorf("failed to set profile: %w", err)
			}
		default:
		}
	}

	return meta, nil
}

func getSetProfileParams(owner sdk.AccAddress, p schema.V1Profile) *storage.SetProfileParams { // nolint:gocritic
	return &storage.SetProfileParams{
		Address:   owner.String(),
		FirstName: p.FirstName,
		LastName:  p.LastName,
		Bio:       p.Bio,
		Avatar:    p.Avatar,
		Gender:    string(p.Gender),
		Birthday:  p.Birthday.Time,
	}
}

func getPDVOwnerPrefix(owner string) string {
	return fmt.Sprintf("%s/pdv", owner)
}

func getMetaOwnerPrefix(owner string) string {
	return fmt.Sprintf("%s/meta", owner)
}

func getPDVFilePath(owner string, id uint64) string {
	// we need to have descending sort on s3 side, that's why we revert id and print it to hex
	return fmt.Sprintf("%s/%016x", getPDVOwnerPrefix(owner), math.MaxUint64-id)
}

func getMetaFilePath(owner string, id uint64) string {
	return fmt.Sprintf("%s/%016x", getMetaOwnerPrefix(owner), math.MaxUint64-id)
}

func getIDFromFilename(s string) (uint64, error) {
	v, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0, err
	}

	return math.MaxUint64 - v, nil
}
