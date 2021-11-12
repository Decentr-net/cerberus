package blockchain

import (
	"context"
	"errors"
	"testing"
	"time"

	ctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/Decentr-net/ariadne"
	ariadnemock "github.com/Decentr-net/ariadne/mock"
	operationstypes "github.com/Decentr-net/decentr/x/operations/types"

	"github.com/Decentr-net/cerberus/internal/storage"
	storagemock "github.com/Decentr-net/cerberus/internal/storage/mock"
)

var errTest = errors.New("test")

func TestBlockchain_Run(t *testing.T) {
	ctrl := gomock.NewController(t)

	f := ariadnemock.NewMockFetcher(ctrl)
	fs, is := storagemock.NewMockFileStorage(ctrl), storagemock.NewMockIndexStorage(ctrl)

	b := New(f, fs, is, time.Nanosecond, time.Nanosecond)

	is.EXPECT().GetHeight(gomock.Any()).Return(uint64(1), nil)

	f.EXPECT().FetchBlocks(gomock.Any(), uint64(1), gomock.Any(), gomock.Any()).Return(nil)

	require.NoError(t, b.Run(context.Background()))
}

func TestBlockchain_Run_Error(t *testing.T) {
	ctrl := gomock.NewController(t)

	f := ariadnemock.NewMockFetcher(ctrl)
	fs, is := storagemock.NewMockFileStorage(ctrl), storagemock.NewMockIndexStorage(ctrl)

	b := New(f, fs, is, time.Nanosecond, time.Nanosecond)

	is.EXPECT().GetHeight(gomock.Any()).Return(uint64(1), nil)

	f.EXPECT().FetchBlocks(gomock.Any(), uint64(1), gomock.Any(), gomock.Any()).Return(errTest)

	require.Equal(t, errTest, b.Run(context.Background()))
}

func TestBlockchain_processBlockFunc(t *testing.T) {
	timestamp := time.Now()
	owner, err := sdk.AccAddressFromBech32("decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz")
	require.NoError(t, err)

	owner2, err := sdk.AccAddressFromBech32("decentr1ltx6yymrs8eq4nmnhzfzxj6tspjuymh8mgd6gz")
	require.NoError(t, err)

	tt := []struct {
		name   string
		msg    sdk.Msg
		expect func(fs *storagemock.MockFileStorage, is *storagemock.MockIndexStorage)
	}{
		{
			name: "delete_account",
			msg: &operationstypes.MsgResetAccount{
				Owner:   owner,
				Address: owner2,
			},
			expect: func(fs *storagemock.MockFileStorage, is *storagemock.MockIndexStorage) {
				fs.EXPECT().DeleteData(gomock.Any(), owner2.String()).Return(nil)
				is.EXPECT().DeletePDV(gomock.Any(), owner2.String()).Return(nil)
				is.EXPECT().DeleteProfile(gomock.Any(), owner2.String()).Return(nil)
			},
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			fs, is := storagemock.NewMockFileStorage(ctrl), storagemock.NewMockIndexStorage(ctrl)

			is.EXPECT().InTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, f func(_ storage.IndexStorage) error) error {
				return f(is)
			})
			is.EXPECT().SetHeight(gomock.Any(), uint64(1)).Return(nil)
			tc.expect(fs, is)

			msg, err := ctypes.NewAnyWithValue(tc.msg)
			require.NoError(t, err)
			block := ariadne.Block{
				Height: 1,
				Time:   timestamp,
				Txs: []sdk.Tx{
					&tx.Tx{
						Body: &tx.TxBody{
							Messages: []*ctypes.Any{
								msg,
							},
						},
					},
				},
			}

			require.NoError(t, blockchain{fs: fs, is: is}.processBlockFunc(context.Background())(block))
			time.Sleep(100 * time.Millisecond) // wait for routine
		})
	}
}

func TestBlockchain_processBlockFunc_errors(t *testing.T) {
	is := storagemock.NewMockIndexStorage(gomock.NewController(t))

	is.EXPECT().InTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, f func(_ storage.IndexStorage) error) error {
		return context.Canceled
	})

	require.Error(t, blockchain{is: is}.processBlockFunc(context.Background())(ariadne.Block{Height: 1}))
}
