package shardmatrix_test

import (
	"testing"

	keepertest "github.com/govm-net/shardmatrix/testutil/keeper"
	"github.com/govm-net/shardmatrix/testutil/nullify"
	shardmatrix "github.com/govm-net/shardmatrix/x/shardmatrix/module"
	"github.com/govm-net/shardmatrix/x/shardmatrix/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ShardmatrixKeeper(t)
	shardmatrix.InitGenesis(ctx, k, genesisState)
	got := shardmatrix.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
