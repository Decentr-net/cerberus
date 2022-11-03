package service

import (
	"bytes"
	"context"
	"encoding/base64"
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

	_ "github.com/Decentr-net/cerberus/internal/blockchain"
	cryptomock "github.com/Decentr-net/cerberus/internal/crypto/mock"
	"github.com/Decentr-net/cerberus/internal/entities"
	hadesclient "github.com/Decentr-net/cerberus/internal/hades"
	hadesmock "github.com/Decentr-net/cerberus/internal/hades/mock"
	"github.com/Decentr-net/cerberus/internal/producer"
	producermock "github.com/Decentr-net/cerberus/internal/producer/mock"
	"github.com/Decentr-net/cerberus/internal/storage"
	storagemock "github.com/Decentr-net/cerberus/internal/storage/mock"
	"github.com/Decentr-net/cerberus/pkg/schema"
	"github.com/Decentr-net/cerberus/pkg/schema/types"
	v1 "github.com/Decentr-net/cerberus/pkg/schema/v1"
)

var (
	ctx                 = context.Background()
	testOwner           = "decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz"
	testOwnerSdkAddr, _ = sdk.AccAddressFromBech32(testOwner)
	testDevice          = "ios"
	testID              = uint64(1)
	testData            = []byte("data")
	testEncryptedData   = []byte("data_encrypted")
	errTest             = errors.New("test")
	pdvRewardsInterval  = time.Hour
	rewardsMap          = RewardMap{
		schema.PDVCookieType:   sdk.NewDecWithPrec(2, 6),
		schema.PDVLocationType: sdk.NewDecWithPrec(4, 6),
		schema.PDVProfileType:  sdk.NewDecWithPrec(6, 6),
	}
)

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
		{"400x400.jpeg", 400, 400, 270, 270},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.file, func(t *testing.T) {
			t.Parallel()

			body, err := ioutil.ReadFile("testdata/" + tc.file)
			require.NoError(t, err)

			dataImage := "data:image/png;base64," + base64.StdEncoding.EncodeToString(body)
			if strings.HasSuffix(tc.file, ".jpeg") {
				dataImage = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(body)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			fs := storagemock.NewMockFileStorage(ctrl)
			fs.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).DoAndReturn(func(_ context.Context, r io.Reader, size int64, filepath, contentType string, isPublicRead bool) (string, error) {
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

			hd, thumb, err := s.SaveImage(context.Background(), strings.NewReader(dataImage), "owner")
			require.NoError(t, err)
			require.NotEmpty(t, hd)
			require.NotEmpty(t, thumb)
			require.True(t, strings.HasPrefix(thumb, hd))
		})
	}
}

func TestService_SavePDV(t *testing.T) {
	t.Skip()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

	expectedID := uint64(time.Now().Unix())

	cr.EXPECT().Encrypt(gomock.Any()).Return(testEncryptedData, nil)

	expectedMeta := &entities.PDVMeta{
		ObjectTypes: map[schema.Type]uint16{
			schema.PDVCookieType:   1,
			schema.PDVLocationType: 1,
		},
		Reward: sdk.NewDecWithPrec(6, 6),
	}

	is.EXPECT().IsProfileBanned(gomock.Any(), testOwnerSdkAddr.String()).Return(false, nil)

	hades.EXPECT().AntiFraud(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, req *hadesclient.AntiFraudRequest) (*hadesclient.AntiFraudResponse, error) {
		require.Equal(t, expectedID, req.ID)
		require.Equal(t, testOwner, req.Address)
		require.Equal(t, testDevice, req.Data.Device)
		return &hadesclient.AntiFraudResponse{IsFraud: false}, nil
	})

	p.EXPECT().Produce(ctx, gomock.Eq(&producer.PDVMessage{
		ID:      expectedID,
		Address: testOwner,
		Meta:    expectedMeta,
		Device:  testDevice,
		Data:    testEncryptedData,
	}))

	id, meta, err := s.SavePDV(ctx, schema.NewPDVWrapper(testDevice, pdv), testOwnerSdkAddr)
	require.Equal(t, expectedID, id)
	require.Equal(t, expectedMeta, meta)
	require.NoError(t, err)
}

func TestService_SavePDV_Blacklist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

	expectedID := uint64(time.Now().Unix())

	cr.EXPECT().Encrypt(gomock.Any()).Return(testEncryptedData, nil)

	expectedMeta := &entities.PDVMeta{
		ObjectTypes: map[schema.Type]uint16{
			schema.PDVCookieType: 1,
		},
		Reward: sdk.ZeroDec(),
	}

	is.EXPECT().IsProfileBanned(gomock.Any(), testOwnerSdkAddr.String()).Return(false, nil)

	hades.EXPECT().AntiFraud(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, req *hadesclient.AntiFraudRequest) (*hadesclient.AntiFraudResponse, error) {
		require.Equal(t, expectedID, req.ID)
		require.Equal(t, testOwner, req.Address)
		require.Equal(t, testDevice, req.Data.Device)
		return &hadesclient.AntiFraudResponse{IsFraud: false}, nil
	})

	p.EXPECT().Produce(ctx, gomock.Eq(&producer.PDVMessage{
		ID:      expectedID,
		Address: testOwner,
		Meta:    expectedMeta,
		Device:  testDevice,
		Data:    testEncryptedData,
	}))

	id, meta, err := s.SavePDV(ctx, schema.NewPDVWrapper(testDevice, v1.PDV{
		&v1.Cookie{
			Source: schema.Source{
				Host: "youtube.com",
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
	}), testOwnerSdkAddr)
	require.Equal(t, expectedID, id)
	require.Equal(t, expectedMeta, meta)
	require.NoError(t, err)
}

func TestFloat64ToDecimal(t *testing.T) {
	d, err := float64ToDecimal(0.5)
	require.NoError(t, err)
	require.Equal(t, "0.500000000000000000", d.String())

	d, err = float64ToDecimal(201.53)
	require.NoError(t, err)
	require.Equal(t, "201.530000000000000000", d.String())
}

func TestService_SavePDV_Profile(t *testing.T) {
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
		meta  *entities.PDVMeta
	}{
		{
			name:  "exist",
			exist: true,
			meta: &entities.PDVMeta{
				ObjectTypes: map[schema.Type]uint16{
					schema.PDVProfileType: 1,
				},
				Reward: sdk.ZeroDec(),
			},
		},
		{
			name:  "not_exist",
			exist: false,
			meta: &entities.PDVMeta{
				ObjectTypes: map[schema.Type]uint16{
					schema.PDVProfileType: 1,
				},
				Reward: sdk.NewDecWithPrec(6, 6),
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
			p := producermock.NewMockProducer(ctrl)
			hades := hadesmock.NewMockHades(ctrl)

			s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

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
				Birthday:  &pdv[0].(*schema.V1Profile).Birthday.Time,
			})).Return(nil)

			is.EXPECT().IsProfileBanned(gomock.Any(), testOwnerSdkAddr.String()).Return(false, nil)

			expectedID := uint64(time.Now().Unix())

			cr.EXPECT().Encrypt(gomock.Any()).Return(testEncryptedData, nil)

			hades.EXPECT().AntiFraud(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, req *hadesclient.AntiFraudRequest) (*hadesclient.AntiFraudResponse, error) {
				require.Equal(t, expectedID, req.ID)
				require.Equal(t, testOwner, req.Address)
				require.Equal(t, testDevice, req.Data.Device)
				return &hadesclient.AntiFraudResponse{IsFraud: false}, nil
			})

			p.EXPECT().Produce(ctx, &producer.PDVMessage{
				ID:      expectedID,
				Address: testOwner,
				Meta:    tc.meta,
				Device:  testDevice,
				Data:    testEncryptedData,
			}).Return(nil)

			id, meta, err := s.SavePDV(ctx, schema.NewPDVWrapper(testDevice, pdv), testOwnerSdkAddr)
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
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

	is.EXPECT().IsProfileBanned(gomock.Any(), testOwnerSdkAddr.String()).Return(false, nil)

	cr.EXPECT().Encrypt(gomock.Any()).Return(nil, errTest)

	_, _, err := s.SavePDV(ctx, schema.NewPDVWrapper(testDevice, pdv), testOwnerSdkAddr)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_SavePDV_Fraud(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

	expectedID := uint64(time.Now().Unix())

	cr.EXPECT().Encrypt(gomock.Any()).Return(testEncryptedData, nil)

	hades.EXPECT().AntiFraud(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, req *hadesclient.AntiFraudRequest) (*hadesclient.AntiFraudResponse, error) {
		require.Equal(t, expectedID, req.ID)
		require.Equal(t, testOwner, req.Address)
		require.Equal(t, testDevice, req.Data.Device)
		return &hadesclient.AntiFraudResponse{IsFraud: true}, nil
	})

	is.EXPECT().SetProfileBanned(gomock.Any(), testOwnerSdkAddr.String()).Return(nil)
	is.EXPECT().IsProfileBanned(gomock.Any(), testOwnerSdkAddr.String()).Return(false, nil)

	id, meta, err := s.SavePDV(ctx, schema.NewPDVWrapper(testDevice, pdv), testOwnerSdkAddr)
	require.Equal(t, expectedID, id)
	require.Nil(t, meta)
	require.Equal(t, err, ErrPDVFraud)
}

func TestService_SavePDV_ProducerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

	is.EXPECT().IsProfileBanned(gomock.Any(), testOwnerSdkAddr.String()).Return(false, nil)

	cr.EXPECT().Encrypt(gomock.Any()).Return(testEncryptedData, nil)

	hades.EXPECT().AntiFraud(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, req *hadesclient.AntiFraudRequest) (*hadesclient.AntiFraudResponse, error) {
		require.Equal(t, testOwner, req.Address)
		require.Equal(t, testDevice, req.Data.Device)
		return &hadesclient.AntiFraudResponse{IsFraud: false}, nil
	})

	p.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(errTest)

	_, _, err := s.SavePDV(ctx, schema.NewPDVWrapper(testDevice, pdv), testOwnerSdkAddr)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTest))
}

func TestService_ReceivePDV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

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
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

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
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

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
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

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
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

	exp := &entities.PDVMeta{
		ObjectTypes: map[schema.Type]uint16{
			schema.PDVCookieType:   1,
			schema.PDVLocationType: 2,
		},
		Reward: sdk.NewDecWithPrec(10, 6),
	}
	is.EXPECT().GetPDVMeta(gomock.Any(), testOwner, testID).Return(exp, nil)

	act, err := s.GetPDVMeta(ctx, testOwner, testID)
	require.NoError(t, err)
	require.Equal(t, exp, act)
}

func TestService_GetPDVMeta_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

	is.EXPECT().GetPDVMeta(gomock.Any(), testOwner, testID).Return(nil, errTest)

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
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

	is.EXPECT().GetPDVMeta(gomock.Any(), testOwner, testID).Return(nil, storage.ErrNotFound)

	_, err := s.GetPDVMeta(ctx, testOwner, testID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestService_getFilePath(t *testing.T) {
	// we want to sort it for list on s3 side
	require.Equal(t, "test/pdv/fffffffffffffffe", getPDVFilePath("test", 1))
}

func TestService_ListPDV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := storagemock.NewMockFileStorage(ctrl)
	is := storagemock.NewMockIndexStorage(ctrl)
	cr := cryptomock.NewMockCrypto(ctrl)
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

	is.EXPECT().ListPDV(gomock.Any(), "owner", uint64(5), uint16(10)).Return([]uint64{1, 2, 3}, nil)

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
	p := producermock.NewMockProducer(ctrl)
	hades := hadesmock.NewMockHades(ctrl)

	s := New(cr, fs, is, p, hades, rewardsMap, pdvRewardsInterval)

	is.EXPECT().GetProfiles(ctx, []string{"1", "2"}).Return([]*storage.Profile{
		{
			Address:   "1",
			FirstName: "2",
			LastName:  "3",
			Emails:    []string{"email1", "email2"},
			Bio:       "4",
			Avatar:    "5",
			Gender:    "6",
			Birthday:  toTimePrt(time.Unix(1, 0)),
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
			Birthday:  toTimePrt(time.Unix(2, 0)),
			CreatedAt: time.Unix(3, 0),
		},
	}, nil)

	pp, err := s.GetProfiles(ctx, []string{"1", "2"})
	require.NoError(t, err)
	assert.Equal(t, []*entities.Profile{
		{
			Address:   "1",
			FirstName: "2",
			LastName:  "3",
			Emails:    []string{"email1", "email2"},
			Bio:       "4",
			Avatar:    "5",
			Gender:    "6",
			Birthday:  toTimePrt(time.Unix(1, 0)),
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
			Birthday:  toTimePrt(time.Unix(2, 0)),
			CreatedAt: time.Unix(3, 0),
		},
	}, pp)
}

func TestService_GetRewardsMap(t *testing.T) {
	rm := RewardMap{
		"m": sdk.NewDecWithPrec(1, 6),
		"t": sdk.NewDecWithPrec(2, 6),
	}
	s := service{rewardMap: rm}

	require.EqualValues(t, rm, s.GetRewardsMap())
}

func mustDate(s string) *types.Date {
	var d types.Date

	if err := d.UnmarshalJSON([]byte(s)); err != nil {
		panic(err)
	}

	return &d
}

func toTimePrt(t time.Time) *time.Time {
	return &t
}
