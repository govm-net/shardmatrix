# 加密模块设计文档

## 1. 概述

加密模块提供了多种签名算法的实现，包括：
- Secp256k1 (ECDSA)
- Schnorr
- Ed25519

每种算法都实现了统一的接口，支持：
- 密钥生成
- 签名和验签
- 密钥导入导出
- 从签名恢复公钥（部分算法支持）

## 2. 接口设计

### 2.1 算法类型

```go
type Algorithm byte

const (
    Secp256k1 Algorithm = iota + 1
    Schnorr
    Ed25519
)
```

### 2.2 签名器接口

```go
type Signer interface {
    Type() Algorithm
    Sign(msg []byte) ([]byte, error)
    Verify(msg, sig []byte) bool
    PublicKey() crypto.PublicKey
    PrivateKey() crypto.PrivateKey
    GenerateKey(rand io.Reader) error
    ImportPrivateKey(privKey []byte) error
}
```

### 2.3 验证器接口

```go
type Verifier interface {
    Type() Algorithm
    Verify(msg, sig []byte) bool
    PublicKey() crypto.PublicKey
    ImportPublicKey(pubKey []byte) error
    RecoverPublicKey(msg, sig []byte) (crypto.PublicKey, error)
}
```

## 3. 算法特性

### 3.1 Secp256k1 (ECDSA)

- 基于 secp256k1 椭圆曲线
- 支持从签名恢复公钥
- 签名格式：压缩格式（65字节）
- 公钥格式：压缩格式（33字节）
- 私钥格式：32字节

### 3.2 Schnorr

- 基于 secp256k1 椭圆曲线
- 支持从签名恢复公钥
- 签名格式：64字节
- 公钥格式：压缩格式（33字节）
- 私钥格式：32字节

### 3.3 Ed25519

- 基于 Ed25519 曲线
- 不支持从签名恢复公钥
- 签名格式：64字节
- 公钥格式：32字节
- 私钥格式：64字节

## 4. 使用示例

### 4.1 创建签名器

```go
signer, err := crypto.NewSigner(types.Secp256k1)
if err != nil {
    // 处理错误
}
```

### 4.2 生成密钥对

```go
if err := signer.GenerateKey(rand.Reader); err != nil {
    // 处理错误
}
```

### 4.3 签名

```go
msg := []byte("hello world")
sig, err := signer.Sign(msg)
if err != nil {
    // 处理错误
}
```

### 4.4 验签

```go
valid := signer.Verify(msg, sig)
if !valid {
    // 签名无效
}
```

### 4.5 导入密钥

```go
// 导入私钥
if err := signer.ImportPrivateKey(privKey); err != nil {
    // 处理错误
}

// 导入公钥到验证器
verifier, err := crypto.NewVerifier(types.Secp256k1)
if err != nil {
    // 处理错误
}
if err := verifier.ImportPublicKey(pubKey); err != nil {
    // 处理错误
}
```

### 4.6 从签名恢复公钥

```go
pubKey, err := verifier.RecoverPublicKey(msg, sig)
if err != nil {
    // 处理错误
}
```

## 5. 注意事项

1. 签名器不支持导入公钥，验证器不支持导入私钥
2. 只有 Secp256k1 和 Schnorr 支持从签名恢复公钥
3. 不同算法的密钥格式不同，导入时需要注意
4. 签名和验签时使用相同的哈希算法
5. 密钥生成使用安全的随机数生成器
6. 签名器只能导入私钥，验证器只能导入公钥

## 6. 安全建议

1. 使用安全的随机数生成器生成密钥
2. 妥善保管私钥，不要泄露
3. 定期更换密钥对
4. 使用最新的算法实现
5. 验证签名时检查公钥的有效性 