package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/govm-net/shardmatrix/testutil/keeper"
	"github.com/govm-net/shardmatrix/x/shardmatrix/types"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := keepertest.ShardmatrixKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
