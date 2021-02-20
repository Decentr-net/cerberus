package types

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params"
)

const (
	// DefaultParamspace for params keeper
	DefaultParamspace = ModuleName
)

var (
	DefaultCerberuses = []string{}
)

// ParamCerberusKey is store's key for ParamCerberus
var ParamCerberusesKey = []byte("ParamCerberuses")

// ParamKeyTable type declaration for parameters
func ParamKeyTable() params.KeyTable {
	return params.NewKeyTable(
		params.NewParamSetPair(ParamCerberusesKey, &DefaultCerberuses, validateCerberuses),
	)
}

func validateCerberuses(i interface{}) error {
	moderators, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	for _, moderator := range moderators {
		if _, err := sdk.AccAddressFromBech32(moderator); err != nil {
			return fmt.Errorf("%s is an invalid cerberus address, err=%w", moderator, err)
		}
	}
	return nil
}
