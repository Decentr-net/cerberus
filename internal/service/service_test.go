package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Decentr-net/cerberus/internal/crypto"
	"github.com/Decentr-net/cerberus/internal/storage"
)

var ctx = context.Background()
var testFilename = "test"
var testData = []byte("data")
var testEncryptedData = []byte("data_encrypted")
var errTest = errors.New("test")

func TestService_SavePDV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	cr.EXPECT().Encrypt(gomock.Any()).DoAndReturn(func(r io.Reader) (io.Reader, error) {
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, testData, data)

		return bytes.NewReader(testEncryptedData), nil
	})

	st.EXPECT().Write(ctx, gomock.Any(), testFilename).DoAndReturn(func(_ context.Context, r io.Reader, _ string) error {
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, testEncryptedData, data)

		return nil
	})

	err := s.SavePDV(ctx, testData, testFilename)
	require.NoError(t, err)
}

func TestService_SavePDV_EncryptError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	cr.EXPECT().Encrypt(gomock.Any()).Return(nil, errTest)

	err := s.SavePDV(ctx, testData, testFilename)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_SavePDV_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	cr.EXPECT().Encrypt(gomock.Any()).Return(bytes.NewReader(testEncryptedData), nil)

	st.EXPECT().Write(ctx, gomock.Any(), testFilename).Return(errTest)

	err := s.SavePDV(ctx, testData, testFilename)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_ReceivePDV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	st.EXPECT().Read(ctx, testFilename).Return(ioutil.NopCloser(bytes.NewReader(testEncryptedData)), nil)

	cr.EXPECT().Decrypt(gomock.Any()).DoAndReturn(func(r io.Reader) (io.Reader, error) {
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, testEncryptedData, data)

		return bytes.NewReader(testData), nil
	})

	data, err := s.ReceivePDV(ctx, testFilename)
	require.NoError(t, err)
	assert.Equal(t, testData, data)
}

func TestService_ReceivePDV_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	st.EXPECT().Read(ctx, testFilename).Return(nil, errTest)

	data, err := s.ReceivePDV(ctx, testFilename)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
	assert.Nil(t, data)
}

func TestService_ReceivePDV_StorageError_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	st.EXPECT().Read(ctx, testFilename).Return(nil, storage.ErrNotFound)

	data, err := s.ReceivePDV(ctx, testFilename)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
	assert.Nil(t, data)
}

func TestService_ReceivePDV_DecryptError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	st.EXPECT().Read(ctx, testFilename).Return(ioutil.NopCloser(bytes.NewReader(testEncryptedData)), nil)

	cr.EXPECT().Decrypt(gomock.Any()).Return(nil, errTest)

	data, err := s.ReceivePDV(ctx, testFilename)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
	assert.Nil(t, data)
}

func TestService_DoesPDVExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	st.EXPECT().DoesExist(ctx, testFilename).Return(true, nil)

	exists, err := s.DoesPDVExist(ctx, testFilename)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestMockService_DoesPDVExist_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	st.EXPECT().DoesExist(ctx, testFilename).Return(false, errTest)

	exists, err := s.DoesPDVExist(ctx, testFilename)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
	assert.False(t, exists)
}

func TestMockService_DoesPDVExist_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	st.EXPECT().DoesExist(ctx, testFilename).Return(false, nil)

	exists, err := s.DoesPDVExist(ctx, testFilename)
	require.NoError(t, err)
	assert.False(t, exists)
}
