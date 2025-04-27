package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/govm-net/shardmatrix/testutil/keeper"
	"github.com/govm-net/shardmatrix/x/shardmatrix/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.ShardmatrixKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
