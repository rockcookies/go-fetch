# go-fetch 精简重构实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use evo-executing-plans to implement this plan task-by-task.

**Goal:** 删除 cert_watcher、circuit_breaker、load_balancer、sse 四个模块，精简 go-fetch HTTP 客户端库

**Architecture:** 完全删除 4 个模块文件和其测试文件，清理 client.go、middleware.go、request.go 中的相关代码引用

**Tech Stack:** Go 1.21+, 标准库 net/http

---

## Task 1: 删除 SSE 模块文件

**Files:**
- Delete: `sse.go`
- Delete: `sse_test.go`

**Step 1: 删除 sse.go**

Run: `rm sse.go`

**Step 2: 删除 sse_test.go**

Run: `rm sse_test.go`

---

## Task 2: 删除 LoadBalancer 模块文件

**Files:**
- Delete: `load_balancer.go`
- Delete: `load_balancer_test.go`

**Step 1: 删除 load_balancer.go**

Run: `rm load_balancer.go`

**Step 2: 删除 load_balancer_test.go**

Run: `rm load_balancer_test.go`

---

## Task 3: 删除 CircuitBreaker 模块文件

**Files:**
- Delete: `circuit_breaker.go`

**Step 1: 删除 circuit_breaker.go**

Run: `rm circuit_breaker.go`

---

## Task 4: 删除 CertWatcher 模块文件

**Files:**
- Delete: `cert_watcher.go` (如果存在独立文件)
- Delete: `cert_watcher_test.go`

**Step 1: 删除 cert_watcher.go**

Run: `rm cert_watcher.go`

**Step 2: 删除 cert_watcher_test.go**

Run: `rm cert_watcher_test.go`

---

## Task 5: 清理 client.go 中的字段定义

**Files:**
- Modify: `client.go:217,230`

**Step 1: 删除 loadBalancer 字段**

删除 `client.go` 第 217 行：
```go
loadBalancer             LoadBalancer
```

**Step 2: 删除 circuitBreaker 字段**

删除 `client.go` 第 230 行：
```go
circuitBreaker           *CircuitBreaker
```

**Step 3: 删除 certWatcherStopChan 字段**

删除 `client.go` 中：
```go
certWatcherStopChan      chan bool
```

**Step 4: 删除 CertWatcherOptions 类型定义**

删除 `client.go` 第 233-238 行：
```go
// CertWatcherOptions allows configuring a watcher that reloads dynamically TLS certs.
type CertWatcherOptions struct {
	// PoolInterval is the frequency at which resty will check if the PEM file needs to be reloaded.
	// Default is 24 hours.
	PoolInterval time.Duration
}
```

**Step 5: 删除 defaultWatcherPoolingInterval 常量**

删除 `client.go` 第 53 行：
```go
defaultWatcherPoolingInterval = 24 * time.Hour
```

**Step 6: 编译检查**

Run: `go build ./...`
Expected: 编译失败，因为还有方法引用这些字段

---

## Task 6: 清理 client.go 中的 LoadBalancer 方法

**Files:**
- Modify: `client.go:266-279`

**Step 1: 删除 LoadBalancer() 方法**

删除 `client.go` 第 266-272 行：
```go
// LoadBalancer method returns the request load balancer instance from the client
// instance. Otherwise returns nil.
func (c *Client) LoadBalancer() LoadBalancer {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.loadBalancer
}
```

**Step 2: 删除 SetLoadBalancer() 方法**

删除 `client.go` 第 274-279 行：
```go
// SetLoadBalancer method is used to set the new request load balancer into the client.
func (c *Client) SetLoadBalancer(b LoadBalancer) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.loadBalancer = b
	return c
}
```

**Step 3: 编译检查**

Run: `go build ./...`
Expected: 编译失败，middleware.go 中仍有引用

---

## Task 7: 清理 client.go 中的 CircuitBreaker 方法

**Files:**
- Modify: `client.go:968-978`

**Step 1: 删除 SetCircuitBreaker() 方法**

删除 `client.go` 第 968-978 行：
```go
// SetCircuitBreaker method sets the Circuit Breaker instance into the client.
// It is used to prevent the client from sending requests that are likely to fail.
// For Example: To use the default Circuit Breaker:
//
//	client.SetCircuitBreaker(NewCircuitBreaker())
func (c *Client) SetCircuitBreaker(b *CircuitBreaker) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.circuitBreaker = b
	return c
}
```

**Step 8: 编译检查**

Run: `go build ./...`
Expected: 编译失败，execute() 方法中有引用

---

## Task 8: 清理 client.go 中的 CertWatcher 方法

**Files:**
- Modify: `client.go:1610-1625,1667-1682,1720-1772`

**Step 1: 删除 SetRootCertificatesWatcher() 方法**

删除 `client.go` 第 1610-1625 行：
```go
// SetRootCertificatesWatcher method enables dynamic reloading of one or more root certificate files.
// It is designed for scenarios involving long-running Resty clients where certificates may be renewed.
//
//	client.SetRootCertificatesWatcher(
//		&resty.CertWatcherOptions{
//			PoolInterval: 24 * time.Hour,
//		},
//		"root-ca.pem",
//	)
func (c *Client) SetRootCertificatesWatcher(options *CertWatcherOptions, pemFilePaths ...string) *Client {
	c.SetRootCertificates(pemFilePaths...)
	for _, fp := range pemFilePaths {
		c.initCertWatcher(fp, "root", options)
	}
	return c
}
```

**Step 2: 删除 SetClientRootCertificatesWatcher() 方法**

删除 `client.go` 第 1667-1682 行：
```go
// SetClientRootCertificatesWatcher method enables dynamic reloading of one or more client root certificate files.
// It is designed for scenarios involving long-running Resty clients where certificates may be renewed.
//
//	client.SetClientRootCertificatesWatcher(
//		&resty.CertWatcherOptions{
//			PoolInterval: 24 * time.Hour,
//		},
//		"client-root-ca.pem",
//	)
func (c *Client) SetClientRootCertificatesWatcher(options *CertWatcherOptions, pemFilePaths ...string) *Client {
	c.SetClientRootCertificates(pemFilePaths...)
	for _, fp := range pemFilePaths {
		c.initCertWatcher(fp, "client-root", options)
	}
	return c
}
```

**Step 3: 删除 initCertWatcher() 方法**

删除 `client.go` 第 1720-1772 行：
```go
func (c *Client) initCertWatcher(pemFilePath, scope string, options *CertWatcherOptions) {
	tickerDuration := defaultWatcherPoolingInterval
	if options != nil && options.PoolInterval > 0 {
		tickerDuration = options.PoolInterval
	}

	go func() {
		ticker := time.NewTicker(tickerDuration)
		st, err := os.Stat(pemFilePath)
		if err != nil {
			c.Logger().Errorf("%v", err)
			return
		}

		modTime := st.ModTime().UTC()

		for {
			select {
			case <-c.certWatcherStopChan:
				ticker.Stop()
				return
			case <-ticker.C:

				c.debugf("Checking if cert %s has changed...", pemFilePath)

				st, err = os.Stat(pemFilePath)
				if err != nil {
					c.Logger().Errorf("%v", err)
					continue
				}
				newModTime := st.ModTime().UTC()

				if modTime.Equal(newModTime) {
					c.debugf("Cert %s hasn't changed.", pemFilePath)
					continue
				}

				modTime = newModTime

				c.debugf("Reloading cert %s ...", pemFilePath)

				switch scope {
				case "root":
					c.SetRootCertificates(pemFilePath)
				case "client-root":
					c.SetClientRootCertificates(pemFilePath)
				}

				c.debugf("Cert %s reloaded.", pemFilePath)
			}
		}
	}()
}
```

**Step 4: 编译检查**

Run: `go build ./...`
Expected: 编译失败，Close() 方法中有引用

---

## Task 9: 清理 client.go 中的 Close() 方法

**Files:**
- Modify: `client.go:2238-2249`

**Step 1: 删除 Close() 中的 LoadBalancer 和 certWatcherStopChan 清理**

将 `client.go` 第 2238-2249 行的 Close() 方法简化为：
```go
// Close method performs cleanup and closure activities on the client instance
func (c *Client) Close() error {
	// Execute close hooks first
	c.onCloseHooks()

	return nil
}
```

**Step 2: 编译检查**

Run: `go build ./...`
Expected: 编译失败，execute() 方法中有 circuitBreaker 引用

---

## Task 10: 清理 client.go 中的 execute() 方法

**Files:**
- Modify: `client.go:2260-2272`

**Step 1: 删除 execute() 中的 circuitBreaker 检查**

删除 `client.go` 第 2263-2267 行的 circuitBreaker 检查逻辑：
```go
if c.circuitBreaker != nil {
	if err := c.circuitBreaker.allow(); err != nil {
		return nil, err
	}
}
```

修改后的 execute() 方法开头应该是：
```go
// Executes method executes the given `Request` object and returns
// response or error.
func (c *Client) execute(req *Request) (*Response, error) {
	if err := c.executeRequestMiddlewares(req); err != nil {
		return nil, err
	}
	// ... 其余代码保持不变
```

**Step 2: 编译检查**

Run: `go build ./...`
Expected: 仍然失败，middleware.go 中有 LoadBalancer 引用

---

## Task 11: 清理 middleware.go 中的 LoadBalancer 引用

**Files:**
- Modify: `middleware.go:117-131`

**Step 1: 删除 LoadBalancer URL 获取逻辑**

将 `middleware.go` 第 117-131 行简化为：
```go
// If [Request.URL] is a relative path, then the following
// gets evaluated in the order
//	1. [Client.BaseURL] is used to obtain the base URL
//	2. Otherwise [Request.URL] is used as-is
if !reqURL.IsAbs() {
	r.URL = reqURL.String()
	if len(r.URL) > 0 && r.URL[0] != '/' {
		r.URL = "/" + r.URL
	}

	reqURL, err = url.Parse(r.baseURL + r.URL)
	if err != nil {
		return &invalidRequestError{Err: err}
	}
}
```

即删除整个 LoadBalancer 检查块：
```go
if r.client.LoadBalancer() != nil {
	r.baseURL, err = r.client.LoadBalancer().Next()
	if err != nil {
		return &invalidRequestError{Err: err}
	}
}
```

**Step 2: 编译检查**

Run: `go build ./...`
Expected: 仍然失败，request.go 中有 sendLoadBalancerFeedback 引用

---

## Task 12: 清理 request.go 中的 sendLoadBalancerFeedback

**Files:**
- Modify: `request.go:1508,1729-1756`

**Step 1: 删除 sendLoadBalancerFeedback() 调用**

删除 `request.go` 第 1508 行：
```go
r.sendLoadBalancerFeedback(res, err)
```

**Step 2: 删除 sendLoadBalancerFeedback() 方法定义**

删除 `request.go` 第 1729-1756 行：
```go
func (r *Request) sendLoadBalancerFeedback(res *Response, err error) {
	if r.client.LoadBalancer() == nil {
		return
	}

	success := true

	// load balancer feedback mainly focuses on connection
	// failures and status code >= 500
	// so that we can prevent sending the request to
	// that server which may fail
	if err != nil {
		var noe *net.OpError
		if errors.As(err, &noe) {
			success = !errors.Is(noe.Err, syscall.ECONNREFUSED) || noe.Timeout()
		}
	}
	if success && res != nil &&
		(res.StatusCode() >= 500 && res.StatusCode() != http.StatusNotImplemented) {
		success = false
	}

	r.client.LoadBalancer().Feedback(&RequestFeedback{
		BaseURL: r.baseURL,
		Success: success,
		Attempt: r.Attempt,
	})
}
```

**Step 3: 编译检查**

Run: `go build ./...`
Expected: 编译成功

---

## Task 13: 清理 client_test.go 中的测试

**Files:**
- Modify: `client_test.go:395-418,1458-1515`

**Step 1: 删除 TestClientSetClientRootCertificateWatcher 测试**

删除 `client_test.go` 第 395-418 行：
```go
func TestClientSetClientRootCertificateWatcher(t *testing.T) {
	t.Run("Cert exists", func(t *testing.T) {
		client := dcnl()
		client.SetClientRootCertificatesWatcher(
			&CertWatcherOptions{PoolInterval: time.Second * 1},
			filepath.Join(getTestDataPath(), "sample-root.pem"),
		)

		transport, err := client.HTTPTransport()

		assertNil(t, err)
		assertNotNil(t, transport.TLSClientConfig.ClientCAs)
	})

	t.Run("Cert does not exist", func(t *testing.T) {
		client := dcnl()
		client.SetClientRootCertificatesWatcher(nil, filepath.Join(getTestDataPath(), "not-exists-sample-root.pem"))

		transport, err := client.HTTPTransport()

		assertNil(t, err)
		assertNil(t, transport.TLSClientConfig)
	})
}
```

**Step 2: 删除 CircuitBreaker 类型断言**

删除 `client_test.go` 第 1458 行：
```go
var _ CircuitBreakerPolicy = CircuitBreaker5xxPolicy
```

**Step 3: 删除 TestClientCircuitBreaker 测试**

删除 `client_test.go` 第 1460-1515 行左右的整个测试函数：
```go
func TestClientCircuitBreaker(t *testing.T) {
	// ... 整个测试函数
}
```

**Step 4: 运行测试检查**

Run: `go test ./...`
Expected: 所有测试通过（除了已删除的测试）

---

## Task 14: 最终验证

**Files:**
- All

**Step 1: 编译检查**

Run: `go build ./...`
Expected: 编译成功，无错误

**Step 2: 运行所有测试**

Run: `go test -v ./...`
Expected: 所有测试通过

**Step 3: 检查是否有残留引用**

Run: `grep -r "LoadBalancer\|CircuitBreaker\|CertWatcher" --include="*.go" .`
Expected: 只在 git 历史或注释中存在，无代码引用

**Step 4: 统计删除的代码行数**

Run: `git diff --stat HEAD`
Expected: 显示删除了约 1200+ 行

---

## 完成标准

- [ ] `go build ./...` 编译成功
- [ ] `go test ./...` 测试全部通过
- [ ] 代码中无 LoadBalancer、CircuitBreaker、CertWatcher 引用
- [ ] 公开 API 简洁，只保留核心 HTTP 功能
