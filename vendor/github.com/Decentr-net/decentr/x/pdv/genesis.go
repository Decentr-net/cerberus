package pdv

import (
	"encoding/json"
	"fmt"
	"github.com/Decentr-net/decentr/x/pdv/types"
	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type GenesisState struct {
	Cerberuses []string `json:"cerberuses"`
}

// GetGenesisStateFromAppState returns community GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc *codec.Codec, appState map[string]json.RawMessage) GenesisState {
	var genesisState GenesisState
	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return genesisState
}

// DefaultGenesisState returns a default genesis state
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Cerberuses: types.DefaultCerberuses,
	}
}

// ValidateGenesis performs basic validation of PDV genesis data returning an
// error for any failed validation criteria.
func ValidateGenesis(data GenesisState) error {
	if len(data.Cerberuses) == 0 {
		return fmt.Errorf("at least one cerberus has to be specified")
	}
	return nil
}

// InitGenesis sets distribution information for genesis.
func InitGenesis(ctx sdk.Context, keeper Keeper, data GenesisState) {
	keeper.SetCerberuses(ctx, data.Cerberuses)
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper Keeper) GenesisState {
	return GenesisState{
		Cerberuses: keeper.GetCerberuses(ctx),
	}
}
