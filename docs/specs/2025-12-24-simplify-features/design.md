# go-fetch 精简重构设计

## 目标

将 go-fetch 从"功能齐全"的 HTTP 客户端精简为"专注核心"的 HTTP 客户端，删除非核心的高级功能，减少维护负担。

## 删除范围

| 模块 | 文件 | 理由 |
|------|------|------|
| 证书监视 | `cert_watcher.go`, `cert_watcher_test.go` | TLS 证书动态更新属于边缘场景，标准库已足够 |
| 熔断器 | `circuit_breaker.go` | 应由调用方实现，非 HTTP 客户端核心职责 |
| 负载均衡 | `load_balancer.go`, `load_balancer_test.go` | 属于服务网格/代理层，客户端库不应承担 |
| SSE | `sse.go`, `sse_test.go` | Server-Sent Events 是专用协议，可独立库实现 |

## 保留范围

```
go-fetch/
├── client.go              # 核心客户端
├── request.go             # 请求构建器
├── response.go            # 响应封装
├── retry.go               # 重试策略
├── redirect.go            # 重定向处理
├── digest.go              # Digest 认证
├── multipart.go           # Multipart 表单
├── stream.go              # 流式响应
├── middleware.go          # 中间件框架
├── curl.go                # cURL 导出
├── debug.go               # 调试模式
├── trace.go               # 请求追踪
├── util.go                # 工具函数
├── transport_dial.go      # 传输层
└── resty.go               # 公共 API
```

## 依赖清理

### client.go

删除内容：
- 字段：`loadBalancer`, `circuitBreaker`
- 方法：`SetLoadBalancer()`, `LoadBalancer()`, `SetCircuitBreaker()`
- 方法：`SetRootCertificatesWatcher()`, `SetClientRootCertificatesWatcher()`, `initCertWatcher()`
- 类型：`CertWatcherOptions`
- Close 中的 loadBalancer 清理逻辑

### middleware.go

删除 LoadBalancer URL 获取逻辑（第 117-127 行）

### request.go

删除 `sendLoadBalancerFeedback()` 方法及其调用

### client_test.go

删除 `TestClientCircuitBreaker` 及 CertWatcher 相关测试

## 执行步骤

1. 删除 8 个文件（4 个实现 + 4 个测试）
2. 修改 3 个文件（client.go, middleware.go, request.go）
3. 编译检查：`go build ./...`
4. 测试检查：`go test ./...`

## 预期效果

- 删除约 1200+ 行代码
- 公开 API 更加简洁
- 维护负担降低
