package network

import (
	"testing"
	"time"
)

func TestLibP2PNetworkCreation(t *testing.T) {
	config := DefaultLibP2PConfig()
	
	p2p, err := NewLibP2PNetwork(config)
	if err != nil {
		t.Fatalf("Failed to create LibP2P network: %v", err)
	}
	
	if p2p == nil {
		t.Error("LibP2P network should not be nil")
	}
	
	if p2p.GetConfig() != config {
		t.Error("Config mismatch")
	}
	
	if p2p.IsRunning() {
		t.Error("Network should not be running initially")
	}
}

func TestLibP2PNetworkLifecycle(t *testing.T) {
	config := DefaultLibP2PConfig()
	
	p2p, err := NewLibP2PNetwork(config)
	if err != nil {
		t.Fatalf("Failed to create LibP2P network: %v", err)
	}
	
	// 测试启动
	if err := p2p.Start(); err != nil {
		t.Fatalf("Failed to start LibP2P network: %v", err)
	}
	
	if !p2p.IsRunning() {
		t.Error("Network should be running")
	}
	
	// 检查主机ID
	hostID := p2p.GetHostID()
	if hostID.String() == "" {
		t.Error("Host ID should not be empty")
	}
	
	// 检查监听地址
	addrs := p2p.GetListenAddrs()
	if len(addrs) == 0 {
		t.Error("Should have at least one listen address")
	}
	
	// 等待一秒钟让网络稳定
	time.Sleep(time.Second)
	
	// 测试停止
	if err := p2p.Stop(); err != nil {
		t.Fatalf("Failed to stop LibP2P network: %v", err)
	}
	
	if p2p.IsRunning() {
		t.Error("Network should not be running")
	}
}

func TestLibP2PNetworkPeerCount(t *testing.T) {
	config := DefaultLibP2PConfig()
	
	p2p, err := NewLibP2PNetwork(config)
	if err != nil {
		t.Fatalf("Failed to create LibP2P network: %v", err)
	}
	
	if err := p2p.Start(); err != nil {
		t.Fatalf("Failed to start LibP2P network: %v", err)
	}
	defer p2p.Stop()
	
	// 初始时应该没有连接的节点
	if count := p2p.GetPeerCount(); count != 0 {
		t.Errorf("Expected 0 peers, got %d", count)
	}
	
	peers := p2p.GetConnectedPeers()
	if len(peers) != 0 {
		t.Errorf("Expected 0 connected peers, got %d", len(peers))
	}
}