package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Decentr-net/cerberus/internal/blockchain"
	"github.com/Decentr-net/cerberus/internal/crypto"
	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/pkg/api"
	"github.com/Decentr-net/cerberus/pkg/schema"
)

var ctx = context.Background()
var testOwner = "decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz"
var testOwnerSdkAddr, _ = sdk.AccAddressFromBech32(testOwner)
var testID = uint64(1)
var testData = []byte("data")
var testEncryptedData = []byte("data_encrypted")
var errTest = errors.New("test")
var rewardsMap = RewardMap{
	schema.PDVCookieType:      2,
	schema.PDVLoginCookieType: 4,
}

var pdv = schema.PDV{
	Version: schema.PDVV1,
	PDV: []schema.PDVObject{
		&schema.PDVObjectV1{
			PDVObjectMetaV1: schema.PDVObjectMetaV1{
				Host: "decentr.net",
				Path: "/",
			},
			Data: []schema.PDVData{
				&schema.PDVDataCookie{
					Name:           "my cookie",
					Value:          "some value",
					Domain:         "*",
					HostOnly:       true,
					Path:           "*",
					Secure:         true,
					SameSite:       "None",
					ExpirationDate: 1861920000,
				},
				&schema.PDVDataLoginCookie{
					Name:           "my cookie",
					Value:          "some value",
					Domain:         "*",
					HostOnly:       true,
					Path:           "*",
					Secure:         true,
					SameSite:       "None",
					ExpirationDate: 1861920000,
				},
			},
		},
	},
}

func TestService_SavePDV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	expectedID := uint64(time.Now().Unix())

	cr.EXPECT().Encrypt(gomock.Any()).DoAndReturn(func(r io.Reader) (io.Reader, int64, error) {
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		return bytes.NewReader(testEncryptedData), int64(len(testEncryptedData)), nil
	})

	st.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r io.Reader, size int64, filepath string) error {
		require.Equal(t, getPDVFilePath(testOwner, expectedID), filepath)

		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, testEncryptedData, data)
		require.EqualValues(t, len(data), size)

		return nil
	})

	expectedMeta := api.PDVMeta{
		ObjectTypes: map[schema.PDVType]uint16{
			schema.PDVCookieType:      1,
			schema.PDVLoginCookieType: 1,
		},
		Reward: 6,
	}

	st.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r io.Reader, size int64, filepath string) error {
		require.Equal(t, getMetaFilePath(testOwner, expectedID), filepath)

		var m api.PDVMeta
		require.NoError(t, json.NewDecoder(r).Decode(&m))

		require.Equal(t, expectedMeta, m)

		return nil
	})

	b.EXPECT().DistributeReward(testOwnerSdkAddr, expectedID, expectedMeta.Reward).Return(nil)

	id, meta, err := s.SavePDV(ctx, pdv, testOwnerSdkAddr)
	require.Equal(t, expectedID, id)
	require.Equal(t, expectedMeta, meta)
	require.NoError(t, err)
}

func TestService_SavePDV_EncryptError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	cr.EXPECT().Encrypt(gomock.Any()).Return(nil, int64(0), errTest)

	_, _, err := s.SavePDV(ctx, pdv, testOwnerSdkAddr)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_SavePDV_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	cr.EXPECT().Encrypt(gomock.Any()).Return(bytes.NewReader(testEncryptedData), int64(len(testEncryptedData)), nil)

	st.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(errTest)

	_, _, err := s.SavePDV(ctx, pdv, testOwnerSdkAddr)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_SavePDV_BlockchainError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	cr.EXPECT().Encrypt(gomock.Any()).Return(bytes.NewReader(testEncryptedData), int64(len(testEncryptedData)), nil)
	st.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	st.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	b.EXPECT().DistributeReward(testOwnerSdkAddr, gomock.Any(), gomock.Any()).Return(errTest)

	_, _, err := s.SavePDV(ctx, pdv, testOwnerSdkAddr)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_ReceivePDV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	st.EXPECT().Read(ctx, getPDVFilePath(testOwner, testID)).Return(ioutil.NopCloser(bytes.NewReader(testEncryptedData)), nil)

	cr.EXPECT().Decrypt(gomock.Any()).DoAndReturn(func(r io.Reader) (io.Reader, error) {
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, testEncryptedData, data)

		return bytes.NewReader(testData), nil
	})

	data, err := s.ReceivePDV(ctx, testOwner, testID)
	require.NoError(t, err)
	assert.Equal(t, testData, data)
}

func TestService_ReceivePDV_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	st.EXPECT().Read(ctx, getPDVFilePath(testOwner, testID)).Return(nil, errTest)

	data, err := s.ReceivePDV(ctx, testOwner, testID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
	assert.Nil(t, data)
}

func TestService_ReceivePDV_StorageError_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	st.EXPECT().Read(ctx, getPDVFilePath(testOwner, testID)).Return(nil, storage.ErrNotFound)

	data, err := s.ReceivePDV(ctx, testOwner, testID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
	assert.Nil(t, data)
}

func TestService_ReceivePDV_DecryptError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	st.EXPECT().Read(ctx, getPDVFilePath(testOwner, testID)).Return(ioutil.NopCloser(bytes.NewReader(testEncryptedData)), nil)

	cr.EXPECT().Decrypt(gomock.Any()).Return(nil, errTest)

	data, err := s.ReceivePDV(ctx, testOwner, testID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
	assert.Nil(t, data)
}

func TestService_GetPDVMeta(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	r := ioutil.NopCloser(bytes.NewBufferString(`{"object_types":{"cookie": 1, "login_cookie": 2}, "reward": 10}`))
	st.EXPECT().Read(ctx, getMetaFilePath(testOwner, testID)).Return(r, nil)

	meta, err := s.GetPDVMeta(ctx, testOwner, testID)
	require.NoError(t, err)
	require.Equal(t, api.PDVMeta{
		ObjectTypes: map[schema.PDVType]uint16{
			schema.PDVCookieType:      1,
			schema.PDVLoginCookieType: 2,
		},
		Reward: 10,
	}, meta)
}

func TestService_GetPDVMeta_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	st.EXPECT().Read(ctx, getMetaFilePath(testOwner, testID)).Return(nil, errTest)

	_, err := s.GetPDVMeta(ctx, testOwner, testID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_GetPDVMeta_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	st.EXPECT().Read(ctx, getMetaFilePath(testOwner, testID)).Return(nil, storage.ErrNotFound)

	_, err := s.GetPDVMeta(ctx, testOwner, testID)
	require.EqualError(t, err, ErrNotFound.Error())
}

func TestService_getFilePath(t *testing.T) {
	// we want to sort it for list on s3 side
	require.Equal(t, "pdv/test/fffffffffffffffe", getPDVFilePath("test", 1))
}

func TestService_getMetaFilePath(t *testing.T) {
	require.Equal(t, "meta/test/fffffffffffffffe", getMetaFilePath("test", 1))
}

func TestService_ListPDV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)
	b := blockchain.NewMockBlockchain(ctrl)

	s := New(cr, st, b, rewardsMap)

	res := []string{"fffffffffffffffe", "fffffffffffffffd", "fffffffffffffffc"}

	st.EXPECT().List(ctx, "pdv/owner", uint64(5), uint16(10)).Return(res, nil)

	l, err := s.ListPDV(ctx, "owner", 5, 10)
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2, 3}, l)
}
