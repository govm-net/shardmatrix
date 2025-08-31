package network

import (
	"fmt"
	"sync"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

// BroadcastManager 广播管理器
type BroadcastManager struct {
	p2p          *LibP2PNetwork
	messageCache map[string]bool // 消息去重缓存
	cacheMutex   sync.RWMutex
	cacheTimeout time.Duration
	maxCacheSize int

	// 消息处理器
	blockHandler BlockMessageHandler
	txHandler    TransactionMessageHandler
}

// NewBroadcastManager 创建广播管理器
func NewBroadcastManager(p2p *LibP2PNetwork) *BroadcastManager {
	bm := &BroadcastManager{
		p2p:          p2p,
		messageCache: make(map[string]bool),
		cacheTimeout: time.Minute * 10, // 10分钟缓存
		maxCacheSize: 10000,
	}

	// 启动缓存清理
	go bm.cleanupCache()

	return bm
}

// SetBlockHandler 设置区块处理器
func (bm *BroadcastManager) SetBlockHandler(handler BlockMessageHandler) {
	bm.blockHandler = handler
}

// SetTransactionHandler 设置交易处理器
func (bm *BroadcastManager) SetTransactionHandler(handler TransactionMessageHandler) {
	bm.txHandler = handler
}

// BroadcastBlock 广播新区块
func (bm *BroadcastManager) BroadcastBlock(block *types.Block) error {
	// 添加到缓存
	messageID := fmt.Sprintf("block_%s", block.Hash().String())
	bm.addToCache(messageID)

	// 使用libp2p广播
	return bm.p2p.BroadcastBlock(block)
}

// BroadcastTransaction 广播新交易
func (bm *BroadcastManager) BroadcastTransaction(tx *types.Transaction) error {
	// 添加到缓存
	messageID := fmt.Sprintf("tx_%s", tx.Hash().String())
	bm.addToCache(messageID)

	// 使用libp2p广播
	return bm.p2p.BroadcastTransaction(tx)
}

// RequestBlock 请求区块
func (bm *BroadcastManager) RequestBlock(peerID peer.ID, blockHash types.Hash) error {
	// TODO: 实现libp2p区块请求
	return fmt.Errorf("libp2p block request not implemented yet")
}

// RequestTransaction 请求交易
func (bm *BroadcastManager) RequestTransaction(peerID peer.ID, txHash types.Hash) error {
	// TODO: 实现libp2p交易请求
	return fmt.Errorf("libp2p transaction request not implemented yet")
}

// addToCache 添加到缓存
func (bm *BroadcastManager) addToCache(messageID string) {
	bm.cacheMutex.Lock()
	defer bm.cacheMutex.Unlock()

	// 检查缓存大小
	if len(bm.messageCache) >= bm.maxCacheSize {
		// 清理一半缓存
		count := 0
		for id := range bm.messageCache {
			delete(bm.messageCache, id)
			count++
			if count >= bm.maxCacheSize/2 {
				break
			}
		}
	}

	bm.messageCache[messageID] = true
}

// isInCache 检查是否在缓存中
func (bm *BroadcastManager) isInCache(messageID string) bool {
	bm.cacheMutex.RLock()
	defer bm.cacheMutex.RUnlock()

	return bm.messageCache[messageID]
}

// cleanupCache 清理缓存
func (bm *BroadcastManager) cleanupCache() {
	ticker := time.NewTicker(bm.cacheTimeout)
	defer ticker.Stop()

	for range ticker.C {
		bm.cacheMutex.Lock()
		bm.messageCache = make(map[string]bool)
		bm.cacheMutex.Unlock()
	}
}

// 消息处理器接口

// BlockMessageHandler 区块消息处理器接口
type BlockMessageHandler interface {
	HandleNewBlock(block *types.Block, fromPeer peer.ID) error
	HandleReceivedBlock(block *types.Block, fromPeer peer.ID) error
	GetBlock(blockHash types.Hash) (*types.Block, error)
}

// TransactionMessageHandler 交易消息处理器接口
type TransactionMessageHandler interface {
	HandleNewTransaction(tx *types.Transaction, fromPeer peer.ID) error
	HandleReceivedTransaction(tx *types.Transaction, fromPeer peer.ID) error
	GetTransaction(txHash types.Hash) (*types.Transaction, error)
}
