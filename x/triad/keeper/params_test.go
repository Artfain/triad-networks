package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "triad/testutil/keeper"
	"triad/x/triad/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.TriadKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
