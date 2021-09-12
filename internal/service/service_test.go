package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/disintegration/imaging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	blockchainmock "github.com/Decentr-net/cerberus/internal/blockchain/mock"
	cryptomock "github.com/Decentr-net/cerberus/internal/crypto/mock"
	"github.com/Decentr-net/cerberus/internal/schema"
	"github.com/Decentr-net/cerberus/internal/schema/types"
	v1 "github.com/Decentr-net/cerberus/internal/schema/v1"
	"github.com/Decentr-net/cerberus/internal/storage"
	storagemock "github.com/Decentr-net/cerberus/internal/storage/mock"
)

var ctx = context.Background()
var testOwner = "decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz"
var testOwnerSdkAddr, _ = sdk.AccAddressFromBech32(testOwner)
var testID = uint64(1)
var testData = []byte("data")
var testEncryptedData = []byte("data_encrypted")
var errTest = errors.New("test")
var rewardsMap = RewardMap{
	schema.PDVCookieType:   2,
	schema.PDVLocationType: 4,
	schema.PDVProfileType:  6,
}

var pdv = v1.PDV{
	&v1.Cookie{
		Source: schema.Source{
			Host: "decentr.net",
			Path: "/",
		},
		Name:           "my cookie",
		Value:          "some value",
		Domain:         "*",
		HostOnly:       true,
		Path:           "*",
		Secure:         true,
		SameSite:       "None",
		ExpirationDate: 1861920000,
	},
	&v1.Location{
		Latitude:  1,
		Longitude: -1,
	},
}

func TestService_SaveImage(t *testing.T) {
	tt := []struct {
		file           string
		x1, y1, x2, y2 int
	}{
		{"1920x1080.png", 1920, 1080, 480, 270},
		{"100x100.png", 100, 100, 100, 100},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.file, func(t *testing.T) {
			t.Parallel()

			body, err := ioutil.ReadFile("testdata/" + tc.file)
			require.NoError(t, err)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			fs := storagemock.NewMockFileStorage(ctrl)
			fs.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r io.Reader, size int64, filepath, contentType string) (string, error) {
				require.NotZero(t, size)
				img, err := imaging.Decode(r)
				require.NoError(t, err)

				bonds := img.Bounds()
				if !strings.Contains(filepath, "thumb") {
					require.Equal(t, bonds.Size(), image.Pt(tc.x1, tc.y1))
				} else {
					require.Equal(t, bonds.Size(), image.Pt(tc.x2, tc.y2))
				}

				return filepath, nil
			}).Times(2)

			s := service{
				fs: fs,
			}

			hd, thumb, err := s.SaveImage(context.Background(), bytes.NewReader(body), "owner")
			require.NoError(t, err)
			require.NotEmpty(t, hd)
			require.NotEmpty(t, thumb)
			require.True(t, strings.HasPrefix(thumb, hd))
		})
	}
}

func TestService_SavePDV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	expectedID := uint64(time.Now().Unix())

	cr.EXPECT().Encrypt(gomock.Any()).DoAndReturn(func(r io.Reader) (io.Reader, int64, error) {
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		return bytes.NewReader(testEncryptedData), int64(len(testEncryptedData)), nil
	})

	fs.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r io.Reader, size int64, filepath, contentType string) (string, error) {
		require.Equal(t, getPDVFilePath(testOwner, expectedID), filepath)

		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, testEncryptedData, data)
		require.EqualValues(t, len(data), size)

		return "", nil
	})

	expectedMeta := PDVMeta{
		ObjectTypes: map[schema.Type]uint16{
			schema.PDVCookieType:   1,
			schema.PDVLocationType: 1,
		},
		Reward: 6,
	}

	fs.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r io.Reader, size int64, filepath, contentType string) (string, error) {
		require.Equal(t, getMetaFilePath(testOwner, expectedID), filepath)

		var m PDVMeta
		require.NoError(t, json.NewDecoder(r).Decode(&m))

		require.Equal(t, expectedMeta, m)

		return "", nil
	})

	b.EXPECT().DistributeReward(testOwnerSdkAddr, expectedID, expectedMeta.Reward).Return(nil)

	id, meta, err := s.SavePDV(ctx, pdv, testOwnerSdkAddr)
	require.Equal(t, expectedID, id)
	require.Equal(t, expectedMeta, meta)
	require.NoError(t, err)
}

func TestService_SavePDV_Profile(t *testing.T) {
	// nolint:govet
	pdv := v1.PDV{
		&v1.Profile{
			FirstName: "first",
			LastName:  "last",
			Emails:    []string{"email1", "email2"},
			Bio:       "bio",
			Gender:    "male",
			Avatar:    "avatar",
			Birthday:  mustDate("2020-02-01"),
		},
	}

	tt := []struct {
		name  string
		exist bool
		meta  PDVMeta
	}{
		{
			name:  "exist",
			exist: true,
			meta: PDVMeta{
				ObjectTypes: map[schema.Type]uint16{
					schema.PDVProfileType: 1,
				},
				Reward: 0,
			},
		},
		{
			name:  "not_exist",
			exist: false,
			meta: PDVMeta{
				ObjectTypes: map[schema.Type]uint16{
					schema.PDVProfileType: 1,
				},
				Reward: 6,
			},
		},
	}

	for i := range tt {
		tc := tt[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			fs := storagemock.NewMockFileStorage(ctrl)
			is := storagemock.NewMockIndexStorage(ctrl)
			cr := cryptomock.NewMockCrypto(ctrl)
			b := blockchainmock.NewMockBlockchain(ctrl)

			s := New(cr, fs, is, b, rewardsMap)

			expectedID := uint64(time.Now().Unix())

			cr.EXPECT().Encrypt(gomock.Any()).DoAndReturn(func(r io.Reader) (io.Reader, int64, error) {
				data, err := ioutil.ReadAll(r)
				require.NoError(t, err)
				require.NotEmpty(t, data)

				return bytes.NewReader(testEncryptedData), int64(len(testEncryptedData)), nil
			})

			fs.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r io.Reader, size int64, filepath, contentType string) (string, error) {
				require.Equal(t, getPDVFilePath(testOwner, expectedID), filepath)

				data, err := ioutil.ReadAll(r)
				require.NoError(t, err)
				require.Equal(t, testEncryptedData, data)
				require.EqualValues(t, len(data), size)

				return "", nil
			})

			is.EXPECT().GetProfile(ctx, testOwner).DoAndReturn(func(_ context.Context, _ string) (*storage.Profile, error) {
				if tc.exist {
					return nil, nil
				}

				return nil, storage.ErrNotFound
			})

			is.EXPECT().SetProfile(ctx, gomock.Eq(&storage.SetProfileParams{
				Address:   "decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz",
				FirstName: "first",
				LastName:  "last",
				Emails:    []string{"email1", "email2"},
				Bio:       "bio",
				Avatar:    "avatar",
				Gender:    "male",
				Birthday:  pdv[0].(*schema.V1Profile).Birthday.Time,
			})).Return(nil)

			fs.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r io.Reader, size int64, filepath, contentType string) (string, error) {
				require.Equal(t, getMetaFilePath(testOwner, expectedID), filepath)

				var m PDVMeta
				require.NoError(t, json.NewDecoder(r).Decode(&m))

				require.Equal(t, tc.meta, m)

				return "", nil
			})

			if tc.meta.Reward > 0 {
				b.EXPECT().DistributeReward(testOwnerSdkAddr, expectedID, tc.meta.Reward).Return(nil)
			}

			id, meta, err := s.SavePDV(ctx, pdv, testOwnerSdkAddr)
			require.Equal(t, expectedID, id)
			require.Equal(t, tc.meta, meta)
			require.NoError(t, err)
		})
	}
}

func TestService_SavePDV_EncryptError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	cr.EXPECT().Encrypt(gomock.Any()).Return(nil, int64(0), errTest)

	_, _, err := s.SavePDV(ctx, pdv, testOwnerSdkAddr)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_SavePDV_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	cr.EXPECT().Encrypt(gomock.Any()).Return(bytes.NewReader(testEncryptedData), int64(len(testEncryptedData)), nil)

	fs.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", errTest)

	_, _, err := s.SavePDV(ctx, pdv, testOwnerSdkAddr)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_SavePDV_BlockchainError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	cr.EXPECT().Encrypt(gomock.Any()).Return(bytes.NewReader(testEncryptedData), int64(len(testEncryptedData)), nil)
	fs.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil)
	fs.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil)

	b.EXPECT().DistributeReward(testOwnerSdkAddr, gomock.Any(), gomock.Any()).Return(errTest)

	_, _, err := s.SavePDV(ctx, pdv, testOwnerSdkAddr)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_ReceivePDV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	fs.EXPECT().Read(ctx, getPDVFilePath(testOwner, testID)).Return(ioutil.NopCloser(bytes.NewReader(testEncryptedData)), nil)

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

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	fs.EXPECT().Read(ctx, getPDVFilePath(testOwner, testID)).Return(nil, errTest)

	data, err := s.ReceivePDV(ctx, testOwner, testID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
	assert.Nil(t, data)
}

func TestService_ReceivePDV_StorageError_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	fs.EXPECT().Read(ctx, getPDVFilePath(testOwner, testID)).Return(nil, storage.ErrNotFound)

	data, err := s.ReceivePDV(ctx, testOwner, testID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
	assert.Nil(t, data)
}

func TestService_ReceivePDV_DecryptError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	fs.EXPECT().Read(ctx, getPDVFilePath(testOwner, testID)).Return(ioutil.NopCloser(bytes.NewReader(testEncryptedData)), nil)

	cr.EXPECT().Decrypt(gomock.Any()).Return(nil, errTest)

	data, err := s.ReceivePDV(ctx, testOwner, testID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
	assert.Nil(t, data)
}

func TestService_GetPDVMeta(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	r := ioutil.NopCloser(bytes.NewBufferString(`{"object_types":{"cookie": 1, "location": 2}, "reward": 10}`))
	fs.EXPECT().Read(ctx, getMetaFilePath(testOwner, testID)).Return(r, nil)

	meta, err := s.GetPDVMeta(ctx, testOwner, testID)
	require.NoError(t, err)
	require.Equal(t, PDVMeta{
		ObjectTypes: map[schema.Type]uint16{
			schema.PDVCookieType:   1,
			schema.PDVLocationType: 2,
		},
		Reward: 10,
	}, meta)
}

func TestService_GetPDVMeta_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	fs.EXPECT().Read(ctx, getMetaFilePath(testOwner, testID)).Return(nil, errTest)

	_, err := s.GetPDVMeta(ctx, testOwner, testID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_GetPDVMeta_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	fs.EXPECT().Read(ctx, getMetaFilePath(testOwner, testID)).Return(nil, storage.ErrNotFound)

	_, err := s.GetPDVMeta(ctx, testOwner, testID)
	require.EqualError(t, err, ErrNotFound.Error())
}

func TestService_getFilePath(t *testing.T) {
	// we want to sort it for list on s3 side
	require.Equal(t, "test/pdv/fffffffffffffffe", getPDVFilePath("test", 1))
}

func TestService_getMetaFilePath(t *testing.T) {
	require.Equal(t, "test/meta/fffffffffffffffe", getMetaFilePath("test", 1))
}

func TestService_ListPDV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	res := []string{"fffffffffffffffe", "fffffffffffffffd", "fffffffffffffffc"}

	fs.EXPECT().List(ctx, "owner/meta", uint64(5), uint16(10)).Return(res, nil)

	l, err := s.ListPDV(ctx, "owner", 5, 10)
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2, 3}, l)
}

func TestService_GetProfiles(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	b := blockchainmock.NewMockBlockchain(ctrl)

	s := New(cr, fs, is, b, rewardsMap)

	is.EXPECT().GetProfiles(ctx, []string{"1", "2"}).Return([]*storage.Profile{
		{
			Address:   "1",
			FirstName: "2",
			LastName:  "3",
			Emails:    []string{"email1", "email2"},
			Bio:       "4",
			Avatar:    "5",
			Gender:    "6",
			Birthday:  time.Unix(1, 0),
			CreatedAt: time.Unix(2, 0),
		},
		{
			Address:   "2",
			FirstName: "3",
			LastName:  "4",
			Emails:    []string{"email3"},
			Bio:       "5",
			Avatar:    "6",
			Gender:    "7",
			Birthday:  time.Unix(2, 0),
			CreatedAt: time.Unix(3, 0),
		},
	}, nil)

	pp, err := s.GetProfiles(ctx, []string{"1", "2"})
	require.NoError(t, err)
	assert.Equal(t, []*Profile{
		{
			Address:   "1",
			FirstName: "2",
			LastName:  "3",
			Emails:    []string{"email1", "email2"},
			Bio:       "4",
			Avatar:    "5",
			Gender:    "6",
			Birthday:  time.Unix(1, 0),
			CreatedAt: time.Unix(2, 0),
		},
		{
			Address:   "2",
			FirstName: "3",
			LastName:  "4",
			Emails:    []string{"email3"},
			Bio:       "5",
			Avatar:    "6",
			Gender:    "7",
			Birthday:  time.Unix(2, 0),
			CreatedAt: time.Unix(3, 0),
		},
	}, pp)
}

func TestService_GetRewardsMap(t *testing.T) {
	rm := RewardMap{
		"m": 1,
		"t": 2,
	}
	s := service{rewardMap: rm}

	require.EqualValues(t, rm, s.GetRewardsMap())
}

func mustDate(s string) types.Date {
	var d types.Date

	if err := d.UnmarshalJSON([]byte(s)); err != nil {
		panic(err)
	}

	return d
}
