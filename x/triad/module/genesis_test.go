package triad_test

import (
	"testing"

	keepertest "triad/testutil/keeper"
	"triad/testutil/nullify"
	triad "triad/x/triad/module"
	"triad/x/triad/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.TriadKeeper(t)
	triad.InitGenesis(ctx, k, genesisState)
	got := triad.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
