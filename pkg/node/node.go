package node

import (
	"fmt"

	"github.com/govm-net/shardmatrix/pkg/config"
)

// Node represents a blockchain node
type Node struct {
	config *config.Config
	// TODO: Add other components like storage, consensus, network
}

// New creates a new blockchain node
func New(cfg *config.Config) (*Node, error) {
	return &Node{
		config: cfg,
	}, nil
}

// Start starts the node
func (n *Node) Start() error {
	// TODO: Initialize and start all components
	fmt.Printf("Starting ShardMatrix node on port %d\n", n.config.Network.Port)
	return nil
}

// Stop stops the node
func (n *Node) Stop() error {
	// TODO: Gracefully shutdown all components
	fmt.Println("Stopping ShardMatrix node")
	return nil
}
