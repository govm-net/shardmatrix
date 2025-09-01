package main

import (
	"fmt"

	"github.com/govm-net/shardmatrix/pkg/types"
)

func main() {
	fmt.Println("Generating test validator addresses for multi-node network:")
	fmt.Println("========================================================")

	for i := 1; i <= 3; i++ {
		addr := types.GenerateAddress()
		fmt.Printf("Validator %d: %s\n", i, addr.String())
	}
}
