package network

import (
	"testing"
	"time"
)

// 简化的测试，适配libp2p网络
func TestNewNetworkManager(t *testing.T) {
	config := DefaultLibP2PConfig()
	
	nm, err := NewNetworkManager(config)
	if err != nil {
		t.Fatalf("Failed to create network manager: %v", err)
	}
	
	if nm == nil {
		t.Error("Network manager should not be nil")
	}
	
	if !nm.GetConfig().Equals(config) {
		t.Error("Config mismatch")
	}
}

func TestNetworkManagerLifecycle(t *testing.T) {
	config := DefaultLibP2PConfig()
	
	nm, err := NewNetworkManager(config)
	if err != nil {
		t.Fatalf("Failed to create network manager: %v", err)
	}
	
	// 测试启动
	if err := nm.Start(); err != nil {
		t.Fatalf("Failed to start network manager: %v", err)
	}
	
	if !nm.IsRunning() {
		t.Error("Network manager should be running")
	}
	
	// 等待一秒钟让网络稳定
	time.Sleep(time.Second)
	
	// 测试停止
	if err := nm.Stop(); err != nil {
		t.Fatalf("Failed to stop network manager: %v", err)
	}
	
	if nm.IsRunning() {
		t.Error("Network manager should not be running")
	}
}

func TestNetworkManagerPeerCount(t *testing.T) {
	config := DefaultLibP2PConfig()
	
	nm, err := NewNetworkManager(config)
	if err != nil {
		t.Fatalf("Failed to create network manager: %v", err)
	}
	
	if err := nm.Start(); err != nil {
		t.Fatalf("Failed to start network manager: %v", err)
	}
	defer nm.Stop()
	
	// 初始时应该没有连接的节点
	if count := nm.GetPeerCount(); count != 0 {
		t.Errorf("Expected 0 peers, got %d", count)
	}
	
	if count := nm.GetActivePeerCount(); count != 0 {
		t.Errorf("Expected 0 active peers, got %d", count)
	}
}

// 扩展 LibP2PConfig 以支持 Equals 方法
func (c *LibP2PConfig) Equals(other *LibP2PConfig) bool {
	if c == nil || other == nil {
		return c == other
	}
	
	return c.NodeID == other.NodeID &&
		c.ChainID == other.ChainID &&
		c.Version == other.Version &&
		c.ConnTimeout == other.ConnTimeout &&
		c.MaxPeers == other.MaxPeers
}