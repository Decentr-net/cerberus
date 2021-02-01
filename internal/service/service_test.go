package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Decentr-net/cerberus/internal/crypto"
	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/pkg/api"
	"github.com/Decentr-net/cerberus/pkg/schema"
)

var ctx = context.Background()
var testFilename = "test"
var testData = []byte("data")
var testEncryptedData = []byte("data_encrypted")
var errTest = errors.New("test")

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
			},
		},
	},
}

func TestService_SavePDV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	cr.EXPECT().Encrypt(gomock.Any()).DoAndReturn(func(r io.Reader) (io.Reader, int64, error) {
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		return bytes.NewReader(testEncryptedData), int64(len(testEncryptedData)), nil
	})

	st.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), testFilename).DoAndReturn(func(_ context.Context, r io.Reader, size int64, _ string) error {
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, testEncryptedData, data)
		require.EqualValues(t, len(data), size)

		return nil
	})

	st.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), fmt.Sprintf(metaFilepathTpl, testFilename)).DoAndReturn(func(_ context.Context, r io.Reader, size int64, _ string) error {
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, []byte{0x7b, 0x22, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x22, 0x3a, 0x7b, 0x22, 0x63, 0x6f, 0x6f, 0x6b, 0x69, 0x65, 0x22, 0x3a, 0x31, 0x7d, 0x7d}, data)
		require.EqualValues(t, len(data), size)

		return nil
	})

	err := s.SavePDV(ctx, pdv, testFilename)
	require.NoError(t, err)
}

func TestService_SavePDV_EncryptError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	cr.EXPECT().Encrypt(gomock.Any()).Return(nil, int64(0), errTest)

	err := s.SavePDV(ctx, pdv, testFilename)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_SavePDV_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	cr.EXPECT().Encrypt(gomock.Any()).Return(bytes.NewReader(testEncryptedData), int64(len(testEncryptedData)), nil)

	st.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), testFilename).Return(errTest)

	err := s.SavePDV(ctx, pdv, testFilename)
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

func TestService_GetPDVMeta(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	r := ioutil.NopCloser(bytes.NewBufferString(`{"object_types":{"cookie": 1, "login_cookie": 2}}`))
	st.EXPECT().Read(ctx, fmt.Sprintf(metaFilepathTpl, testFilename)).Return(r, nil)

	meta, err := s.GetPDVMeta(ctx, testFilename)
	require.NoError(t, err)
	require.Equal(t, api.PDVMeta{ObjectTypes: map[schema.PDVType]uint16{schema.PDVCookieType: 1, schema.PDVLoginCookieType: 2}}, meta)
}

func TestService_GetPDVMeta_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	st.EXPECT().Read(ctx, fmt.Sprintf(metaFilepathTpl, testFilename)).Return(nil, errTest)

	_, err := s.GetPDVMeta(ctx, testFilename)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_GetPDVMeta_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := storage.NewMockStorage(ctrl)
	cr := crypto.NewMockCrypto(ctrl)

	s := New(cr, st)

	st.EXPECT().Read(ctx, fmt.Sprintf(metaFilepathTpl, testFilename)).Return(nil, storage.ErrNotFound)

	_, err := s.GetPDVMeta(ctx, testFilename)
	require.EqualError(t, err, ErrNotFound.Error())
}
