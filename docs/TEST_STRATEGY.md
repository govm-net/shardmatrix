# ShardMatrix 测试策略

## 测试策略概述

本文档定义了 ShardMatrix 第一阶段的全面测试策略，确保交付的系统具有高质量、高可靠性。测试策略遵循"测试左移"和"质量内建"的原则，在开发过程中持续保证代码质量。

## 测试目标

### 质量目标
- **功能正确性**: 所有功能按照规格说明正确实现
- **系统稳定性**: 系统能够长时间稳定运行
- **性能达标**: 关键性能指标满足要求
- **安全可靠**: 关键安全机制有效保护

### 量化指标
- 单元测试覆盖率 ≥ 80%
- 集成测试覆盖率 ≥ 70%
- 关键路径测试覆盖率 = 100%
- 缺陷逃逸率 < 5%

## 测试分层策略

### 测试金字塔

```
       /\\
      /E2E\\     端到端测试 (10%)
     /______\\
    /        \\
   /Integration\\  集成测试 (30%)
  /______________\\
 /                \\
/   Unit Tests     \\  单元测试 (60%)
/____________________\\
```

## 单元测试策略

### 测试范围
**包含内容**:
- 所有公开方法和函数
- 核心业务逻辑
- 边界条件和异常情况
- 数据结构的序列化/反序列化

**不包含内容**:
- 第三方库的功能
- 简单的getter/setter方法
- 纯配置代码

### 测试标准

#### 代码覆盖率要求
| 模块 | 最低覆盖率 | 目标覆盖率 |
|------|------------|------------|
| pkg/types/ | 90% | 95% |
| pkg/crypto/ | 95% | 98% |
| pkg/consensus/ | 85% | 90% |
| pkg/storage/ | 85% | 90% |
| pkg/network/ | 80% | 85% |
| pkg/mempool/ | 85% | 90% |
| pkg/node/ | 75% | 80% |
| pkg/api/ | 80% | 85% |

#### 测试实现规范
```go
// 测试文件命名: *_test.go
// 测试函数命名: TestFunctionName
// 基准测试命名: BenchmarkFunctionName
// 示例测试命名: ExampleFunctionName

func TestHashCreation(t *testing.T) {
    tests := []struct {
        name     string
        input    []byte
        expected Hash
        wantErr  bool
    }{
        {
            name:     "valid hash creation",
            input:    []byte("test data"),
            expected: Hash{...},
            wantErr:  false,
        },
        {
            name:     "empty input",
            input:    []byte{},
            expected: Hash{},
            wantErr:  false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := NewHashFromBytes(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewHashFromBytes() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("NewHashFromBytes() = %v, want %v", result, tt.expected)
            }
        })
    }
}\n```

### 关键模块测试要求

#### pkg/types/ 测试
```go
// 必须测试的功能
- Hash 类型的所有方法
- Address 类型的所有方法
- Block 和 Transaction 的序列化/反序列化
- 数据验证逻辑
- 边界条件处理

// 性能测试要求
- Hash 计算性能基准
- 序列化性能基准
- 内存分配测试
```

#### pkg/crypto/ 测试
```go
// 安全测试要求
- 签名和验证的正确性
- 密钥生成的随机性
- 加密算法的标准兼容性
- 边界攻击防护

// 性能测试要求
- 签名生成性能
- 签名验证性能
- 哈希计算性能
```

#### pkg/consensus/ 测试
```go
// 共识逻辑测试
- 验证者选择算法
- 投票权重计算
- 区块生产逻辑
- 区块验证逻辑
- 惩罚机制

// 边界条件测试
- 验证者数量边界
- 投票权重边界
- 时间窗口边界
```

## 集成测试策略

### 测试层次

#### 模块间集成测试
**存储 + 数据类型集成**:
```go
func TestBlockStorageIntegration(t *testing.T) {
    // 测试区块的完整存储和检索流程
    store := NewLevelDBStore(t.TempDir())
    block := createTestBlock()
    
    // 存储区块
    err := store.PutBlock(block)
    assert.NoError(t, err)
    
    // 检索区块
    retrieved, err := store.GetBlock(block.Hash())
    assert.NoError(t, err)
    assert.Equal(t, block, retrieved)
}\n```

**网络 + 共识集成**:
```go
func TestNetworkConsensusIntegration(t *testing.T) {
    // 测试网络层和共识层的协作
    nodes := setupTestNetwork(3)
    defer teardownNetwork(nodes)
    
    // 发送交易并验证共识
    tx := createTestTransaction()
    nodes[0].SubmitTransaction(tx)
    
    // 等待共识完成
    waitForConsensus(nodes, 5*time.Second)
    
    // 验证所有节点状态一致
    for i := 1; i < len(nodes); i++ {
        assert.Equal(t, nodes[0].GetLatestBlock(), nodes[i].GetLatestBlock())
    }
}\n```

#### 子系统集成测试
**交易处理子系统**:
- 交易提交 → 验证 → 内存池 → 区块打包 → 共识 → 状态更新

**区块同步子系统**:
- 新节点加入 → 状态同步 → 区块追赶 → 参与共识

### 集成测试环境

#### 测试网络配置
```yaml
# 测试网络配置
network:
  nodes: 3
  consensus_threshold: 2
  block_interval: 1s  # 测试时使用更短间隔
  network_timeout: 5s
  
validators:
  - address: "test_validator_1"
    stake: 10000
  - address: "test_validator_2"
    stake: 10000
  - address: "test_validator_3"
    stake: 10000
```

#### 测试数据管理
```go
// 测试数据工厂
type TestDataFactory struct {
    keyPairs map[string]*KeyPair
    accounts map[string]*Account
}

func (f *TestDataFactory) CreateTestBlock(height uint64) *Block {
    // 创建标准测试区块
}

func (f *TestDataFactory) CreateTestTransaction(from, to string, amount uint64) *Transaction {
    // 创建标准测试交易
}
```

## 端到端测试策略

### 测试场景

#### 基础功能场景
1. **节点启动和网络形成**
   - 启动3个节点
   - 验证网络连接建立
   - 验证初始状态同步

2. **交易处理完整流程**
   - 创建和签名交易
   - 提交到网络
   - 验证交易被打包
   - 确认状态更新

3. **共识机制验证**
   - 验证者轮换
   - 区块生产和确认
   - 分叉处理

#### 异常场景测试
1. **网络分区场景**
   - 模拟网络分区
   - 验证分区恢复后状态一致性

2. **节点故障场景**
   - 节点崩溃和重启
   - 验证状态恢复

3. **恶意行为场景**
   - 双重签名检测
   - 无效交易拒绝

### E2E测试框架

#### 测试环境搭建
```go
type E2ETestSuite struct {
    nodes     []*Node
    network   *TestNetwork
    factory   *TestDataFactory
    cleanup   []func()
}

func (suite *E2ETestSuite) SetupSuite() {
    // 创建测试网络
    suite.network = NewTestNetwork(3)
    suite.nodes = suite.network.StartNodes()
    suite.factory = NewTestDataFactory()
}

func (suite *E2ETestSuite) TearDownSuite() {
    // 清理测试环境
    for _, cleanup := range suite.cleanup {
        cleanup()
    }
}
```

#### 测试用例示例
```go
func (suite *E2ETestSuite) TestCompleteTransactionFlow() {
    // 1. 准备测试数据
    sender := suite.factory.CreateTestAccount("sender", 1000000)
    receiver := suite.factory.CreateTestAccount("receiver", 0)
    tx := suite.factory.CreateTestTransaction(sender, receiver, 100)
    
    // 2. 提交交易
    err := suite.nodes[0].SubmitTransaction(tx)
    suite.Require().NoError(err)
    
    // 3. 等待交易被打包
    suite.Eventually(func() bool {
        block := suite.nodes[0].GetLatestBlock()
        return containsTransaction(block, tx.Hash())
    }, 10*time.Second, 100*time.Millisecond)
    
    // 4. 验证状态更新
    senderBalance := suite.nodes[0].GetBalance(sender.Address)
    receiverBalance := suite.nodes[0].GetBalance(receiver.Address)
    
    suite.Equal(uint64(999900), senderBalance) // 1000000 - 100
    suite.Equal(uint64(100), receiverBalance)
    
    // 5. 验证所有节点状态一致
    for i := 1; i < len(suite.nodes); i++ {
        suite.Equal(senderBalance, suite.nodes[i].GetBalance(sender.Address))
        suite.Equal(receiverBalance, suite.nodes[i].GetBalance(receiver.Address))
    }
}
```

## 性能测试策略

### 性能测试目标

#### 关键性能指标
| 指标 | 目标值 | 测试方法 |
|------|--------|----------|
| 区块生产时间 | 3秒 ± 100ms | 连续监控100个区块 |
| 交易处理延迟 | < 1秒 | 端到端时延测量 |
| 交易吞吐量 | > 100 tx/sec | 压力测试 |
| 内存使用 | < 500MB | 长时间运行监控 |
| 磁盘I/O | < 10MB/sec | 存储性能测试 |

#### 微基准测试
```go
func BenchmarkHashCalculation(b *testing.B) {
    data := make([]byte, 1024)
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _ = sha256.Sum256(data)
    }
}

func BenchmarkSignatureVerification(b *testing.B) {
    // 准备测试数据
    keyPair := GenerateKeyPair()
    data := []byte("test message")
    signature := keyPair.Sign(data)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        keyPair.Verify(data, signature)
    }
}
```

### 压力测试

#### 负载生成器
```go
type LoadGenerator struct {
    nodes       []*Node
    accounts    []*Account
    txRate      int // 每秒交易数
    duration    time.Duration
}

func (lg *LoadGenerator) GenerateLoad() {
    ticker := time.NewTicker(time.Second / time.Duration(lg.txRate))
    defer ticker.Stop()
    
    timeout := time.After(lg.duration)
    
    for {
        select {
        case <-ticker.C:
            tx := lg.createRandomTransaction()
            node := lg.selectRandomNode()
            go node.SubmitTransaction(tx)
            
        case <-timeout:
            return
        }
    }
}
```

### 长期稳定性测试

#### 24小时运行测试
```go
func TestLongTermStability(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping long-term stability test in short mode")
    }
    
    // 启动测试网络
    nodes := setupTestNetwork(3)
    defer teardownNetwork(nodes)
    
    // 启动负载生成器
    loadGen := NewLoadGenerator(nodes, 10) // 10 tx/sec
    go loadGen.Run(24 * time.Hour)
    
    // 监控系统状态
    monitor := NewSystemMonitor(nodes)
    stats := monitor.Run(24 * time.Hour)
    
    // 验证稳定性指标
    assert.True(t, stats.MaxMemoryUsage < 500*1024*1024) // < 500MB
    assert.True(t, stats.AvgBlockTime < 3.1*time.Second)  // < 3.1s
    assert.Equal(t, 0, stats.CrashCount) // 零崩溃
}
```

## 安全测试策略

### 密码学安全测试

#### 签名安全测试
```go
func TestSignatureSecurity(t *testing.T) {
    keyPair := GenerateKeyPair()
    message := []byte("test message")
    
    // 测试签名验证
    signature := keyPair.Sign(message)
    assert.True(t, keyPair.Verify(message, signature))
    
    // 测试错误消息拒绝
    wrongMessage := []byte("wrong message")
    assert.False(t, keyPair.Verify(wrongMessage, signature))
    
    // 测试错误签名拒绝
    wrongSignature := make([]byte, len(signature))
    assert.False(t, keyPair.Verify(message, wrongSignature))
}
```

#### 哈希碰撞测试
```go
func TestHashCollisionResistance(t *testing.T) {
    seen := make(map[string]bool)
    
    // 生成大量随机数据并计算哈希
    for i := 0; i < 1000000; i++ {
        data := generateRandomData(64)
        hash := sha256.Sum256(data)
        hashStr := hex.EncodeToString(hash[:])
        
        if seen[hashStr] {
            t.Fatalf("Hash collision detected at iteration %d", i)
        }
        seen[hashStr] = true
    }
}
```

### 网络安全测试

#### 恶意消息测试
```go
func TestMaliciousMessageHandling(t *testing.T) {
    node := setupSingleNode()
    defer node.Stop()
    
    // 测试各种恶意消息
    testCases := []struct {
        name    string
        message []byte
        expect  string
    }{
        {"oversized_message", make([]byte, 10*1024*1024), "message too large"},
        {"invalid_format", []byte("invalid"), "parse error"},
        {"malformed_signature", createMalformedSignature(), "signature invalid"},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            err := node.HandleMessage(tc.message)
            assert.Error(t, err)
            assert.Contains(t, err.Error(), tc.expect)
        })
    }
}
```

## 测试工具和基础设施

### 测试工具链

#### 单元测试工具
```bash
# Go 内置测试
go test ./...

# 测试覆盖率
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 竞态条件检测
go test -race ./...

# 基准测试
go test -bench=. ./...
```

#### 集成测试工具
```go
// 测试工具包
type TestUtils struct {
    tempDirs []string
    ports    []int
}

func (tu *TestUtils) CreateTempDir() string {
    dir, _ := ioutil.TempDir("", "shardmatrix_test_")
    tu.tempDirs = append(tu.tempDirs, dir)
    return dir
}

func (tu *TestUtils) GetFreePort() int {
    // 获取可用端口
}

func (tu *TestUtils) Cleanup() {
    // 清理所有临时资源
}
```

### 持续集成

#### GitHub Actions 配置
```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    
    - name: Set up Go
      uses: actions/setup-go@v2
      with:
        go-version: 1.21
    
    - name: Run tests
      run: |
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -func=coverage.out
    
    - name: Run benchmarks
      run: go test -bench=. ./...
    
    - name: Upload coverage
      uses: codecov/codecov-action@v1
      with:
        file: ./coverage.out
```

### 质量门禁

#### 自动化质量检查
```bash
#!/bin/bash
# scripts/quality-check.sh

set -e

echo "Running quality checks..."

# 代码格式检查
gofmt -d .
if [ $? -ne 0 ]; then
    echo "Code formatting issues found"
    exit 1
fi

# 静态分析
golangci-lint run
if [ $? -ne 0 ]; then
    echo "Linting issues found"
    exit 1
fi

# 单元测试
go test -race ./...
if [ $? -ne 0 ]; then
    echo "Unit tests failed"
    exit 1
fi

# 覆盖率检查
go test -coverprofile=coverage.out ./...
coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if [ $(echo "$coverage < 80" | bc) -eq 1 ]; then
    echo "Coverage $coverage% is below 80%"
    exit 1
fi

echo "All quality checks passed!"
```

## 测试执行计划

### 开发阶段测试
- **每次提交**: 单元测试 + 静态分析
- **每日构建**: 完整测试套件
- **每周**: 性能基准测试
- **里程碑**: 完整验收测试

### 测试排期

| 周次 | 测试重点 | 测试类型 |
|------|----------|----------|
| 1-2 | 数据结构和存储 | 单元测试 |
| 3-4 | 共识和网络 | 单元测试 + 集成测试 |
| 5-6 | 节点和API | 集成测试 + E2E测试 |
| 7-8 | 系统完整性 | 性能测试 + 安全测试 |

### 测试报告

#### 每日测试报告
```
测试执行摘要 - 2024-01-15
================================
单元测试: ✅ 1,245 passed, 0 failed
集成测试: ✅ 87 passed, 0 failed
覆盖率: 85.3% (目标: 80%)
性能基准: ✅ 所有基准测试通过
静态分析: ✅ 无问题发现

新增测试: 15
修复缺陷: 3
性能改进: 2
```

---

**重要提醒**: 测试是质量保证的基石，所有开发人员都必须严格遵循测试策略，确保代码质量和系统稳定性。测试代码同样需要进行代码审查和维护。