package keeper

import (
	"github.com/govm-net/shardmatrix/x/shardmatrix/types"
)

var _ types.QueryServer = Keeper{}
