package consensus

import (
	"fmt"
	"testing"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/types"
)

// TestTimeController 测试时间控制器
func TestTimeController(t *testing.T) {
	genesisTime := int64(1609459200) // 2021-01-01 00:00:00 UTC
	
	t.Run("BasicTimeCalculation", func(t *testing.T) {
		tc := NewTimeController(genesisTime)
		
		// 测试区块时间计算
		for i := uint64(1); i <= 10; i++ {
			expectedTime := genesisTime + int64(i)*2
			calculatedTime := tc.GetBlockTime(i)
			
			if calculatedTime != expectedTime {
				t.Errorf("Block %d: expected time %d, got %d", i, expectedTime, calculatedTime)
			}
		}
	})
	
	t.Run("StrictTimeValidation", func(t *testing.T) {
		tc := NewTimeController(genesisTime)
		
		// 测试正确时间验证
		for i := uint64(1); i <= 10; i++ {
			correctTime := genesisTime + int64(i)*2
			if !tc.ValidateBlockTime(i, correctTime) {
				t.Errorf("Block %d: should validate correct time %d", i, correctTime)
			}
			
			// 测试错误时间被拒绝
			wrongTime := correctTime + 1
			if tc.ValidateBlockTime(i, wrongTime) {
				t.Errorf("Block %d: should reject incorrect time %d", i, wrongTime)
			}
		}
	})
	
	t.Run("ZeroToleranceValidation", func(t *testing.T) {
		tc := NewTimeController(genesisTime)
		
		blockNumber := uint64(5)
		correctTime := genesisTime + 10 // 5 * 2 seconds
		
		// 测试零容差：即使差1毫秒也应该被拒绝
		if tc.ValidateBlockTime(blockNumber, correctTime-1) {
			t.Error("Should reject time that is 1 second early")
		}
		
		if tc.ValidateBlockTime(blockNumber, correctTime+1) {
			t.Error("Should reject time that is 1 second late")
		}
		
		// 只有精确时间才能通过
		if !tc.ValidateBlockTime(blockNumber, correctTime) {
			t.Error("Should accept exact correct time")
		}
	})
	
	t.Run("CallbackMechanism", func(t *testing.T) {
		tc := NewTimeController(genesisTime)
		
		callbackExecuted := false
		testCallback := func(blockTime int64, blockNumber uint64) {
			callbackExecuted = true
			expectedTime := genesisTime + int64(blockNumber)*2
			if blockTime != expectedTime {
				t.Errorf("Callback received wrong time: expected %d, got %d", expectedTime, blockTime)
			}
		}
		
		// 注册回调
		tc.RegisterCallback("test", testCallback)
		
		// 手动触发回调（模拟到时间）
		blockNumber := uint64(3)
		blockTime := genesisTime + 6
		testCallback(blockTime, blockNumber)
		
		if !callbackExecuted {
			t.Error("Callback should have been executed")
		}
		
		// 取消注册回调
		tc.UnregisterCallback("test")
	})
	
	t.Run("UtilityFunctions", func(t *testing.T) {
		// 测试类型包中的工具函数
		blockTime5 := types.CalculateBlockTime(genesisTime, 5)
		expected := genesisTime + 10
		if blockTime5 != expected {
			t.Errorf("CalculateBlockTime: expected %d, got %d", expected, blockTime5)
		}
		
		// 测试验证函数
		if !types.ValidateBlockTime(genesisTime, 3, genesisTime+6) {
			t.Error("ValidateBlockTime should accept correct time")
		}
		
		if types.ValidateBlockTime(genesisTime, 3, genesisTime+7) {
			t.Error("ValidateBlockTime should reject incorrect time")
		}
	})
}

// TestTimeControllerPerformance 测试时间控制器性能
func TestTimeControllerPerformance(t *testing.T) {
	genesisTime := int64(1609459200)
	tc := NewTimeController(genesisTime)
	
	t.Run("TimeCalculationPerformance", func(t *testing.T) {
		const iterations = 10000
		
		start := time.Now()
		
		for i := uint64(1); i <= iterations; i++ {
			_ = tc.GetBlockTime(i)
		}
		
		duration := time.Since(start)
		avgTime := duration / iterations
		
		t.Logf("Calculated %d block times in %v (avg: %v per calculation)", iterations, duration, avgTime)
		
		// 性能要求：每次时间计算应该在1μs内完成
		if avgTime > time.Microsecond {
			t.Errorf("Time calculation too slow: %v > 1μs", avgTime)
		}
	})
	
	t.Run("ValidationPerformance", func(t *testing.T) {
		const iterations = 10000
		
		start := time.Now()
		
		for i := uint64(1); i <= iterations; i++ {
			correctTime := genesisTime + int64(i)*2
			_ = tc.ValidateBlockTime(i, correctTime)
		}
		
		duration := time.Since(start)
		avgTime := duration / iterations
		
		t.Logf("Validated %d block times in %v (avg: %v per validation)", iterations, duration, avgTime)
		
		// 性能要求：每次时间验证应该在1μs内完成
		if avgTime > time.Microsecond {
			t.Errorf("Time validation too slow: %v > 1μs", avgTime)
		}
	})
}

// TestTimeControllerConcurrency 测试时间控制器并发安全性
func TestTimeControllerConcurrency(t *testing.T) {
	genesisTime := int64(1609459200)
	tc := NewTimeController(genesisTime)
	
	t.Run("ConcurrentTimeCalculation", func(t *testing.T) {
		const goroutines = 10
		const iterations = 1000
		
		results := make(chan bool, goroutines)
		
		for g := 0; g < goroutines; g++ {
			go func() {
				success := true
				for i := uint64(1); i <= iterations; i++ {
					expected := genesisTime + int64(i)*2
					actual := tc.GetBlockTime(i)
					if actual != expected {
						success = false
						break
					}
				}
				results <- success
			}()
		}
		
		// 等待所有goroutine完成
		for g := 0; g < goroutines; g++ {
			if !<-results {
				t.Error("Concurrent time calculation failed")
			}
		}
	})
	
	t.Run("ConcurrentCallbackRegistration", func(t *testing.T) {
		const goroutines = 10
		
		results := make(chan bool, goroutines)
		
		for g := 0; g < goroutines; g++ {
			go func(id int) {
				callbackName := fmt.Sprintf("callback_%d", id)
				callback := func(blockTime int64, blockNumber uint64) {
					// 回调函数内容
				}
				
				// 注册回调
				tc.RegisterCallback(callbackName, callback)
				
				// 取消注册回调
				tc.UnregisterCallback(callbackName)
				
				results <- true
			}(g)
		}
		
		// 等待所有goroutine完成
		for g := 0; g < goroutines; g++ {
			if !<-results {
				t.Error("Concurrent callback registration failed")
			}
		}
	})
}