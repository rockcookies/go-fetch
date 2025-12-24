# further-simplify Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use evo-executing-plans to implement this plan task-by-task.

**Goal:** 删除 SaveResponse、Retry、Digest Auth、Auth/Credentials 功能，精简 go-fetch HTTP 客户端库

**Architecture:** 直接删除相关代码，不修改保留功能的实现

**Tech Stack:** Go 1.21+, 标准库优先

---

## Task 1: 删除 retry.go 和 retry_test.go

**Files:**
- Delete: `retry.go`
- Delete: `retry_test.go`

**Step 1: 删除 retry.go 文件**

Run: `rm retry.go`
Expected: 文件已删除

**Step 2: 删除 retry_test.go 文件**

Run: `rm retry_test.go`
Expected: 文件已删除

---

## Task 2: 删除 digest.go 和 digest_test.go

**Files:**
- Delete: `digest.go`
- Delete: `digest_test.go`

**Step 1: 删除 digest.go 文件**

Run: `rm digest.go`
Expected: 文件已删除

**Step 2: 删除 digest_test.go 文件**

Run: `rm digest_test.go`
Expected: 文件已删除

---

## Task 3: 清理 client.go 中的 credentials 相关字段

**Files:**
- Modify: `client.go:174`

**Step 1: 删除 credentials 字段**

在 `Client` 结构体中删除第 174 行：
```go
credentials              *credentials
```

---

## Task 4: 清理 client.go 中的 retry 相关字段

**Files:**
- Modify: `client.go:184-191`

**Step 1: 删除 retry 相关字段**

在 `Client` 结构体中删除以下字段：
```go
retryCount               int
retryWaitTime            time.Duration
retryMaxWaitTime         time.Duration
retryConditions          []RetryConditionFunc
retryHooks               []RetryHookFunc
retryStrategy            RetryStrategyFunc
isRetryDefaultConditions bool
allowNonIdempotentRetry  bool
```

---

## Task 5: 清理 client.go 中的 SetBasicAuth 方法

**Files:**
- Modify: `client.go:470-487`

**Step 1: 删除 SetBasicAuth 方法**

删除整个方法：
```go
// SetBasicAuth method sets the basic authentication header in the HTTP request...
func (c *Client) SetBasicAuth(username, password string) *Client {
    c.lock.Lock()
    defer c.lock.Unlock()
    c.credentials = &credentials{Username: username, Password: password}
    return c
}
```

---

## Task 6: 清理 client.go 中的 AuthToken 相关方法

**Files:**
- Modify: `client.go:489-533`

**Step 1: 删除 AuthToken 方法**

删除以下方法：
```go
func (c *Client) AuthToken() string { ... }
func (c *Client) SetAuthToken(token string) *Client { ... }
func (c *Client) AuthScheme() string { ... }
func (c *Client) SetAuthScheme(scheme string) *Client { ... }
func (c *Client) SetHeaderAuthorizationKey(k string) *Client { ... }
```

---

## Task 7: 清理 client.go 中的 SetDigestAuth 方法

**Files:**
- Modify: `client.go:568-592`

**Step 1: 删除 SetDigestAuth 方法**

删除整个方法：
```go
// SetDigestAuth method sets the Digest Auth transport...
func (c *Client) SetDigestAuth(username, password string) *Client { ... }
```

---

## Task 8: 清理 client.go 中的 retry 相关方法

**Files:**
- Modify: `client.go:1202-1384`

**Step 1: 删除 retry getter/setter 方法**

删除以下方法：
```go
func (c *Client) RetryCount() int { ... }
func (c *Client) SetRetryCount(count int) *Client { ... }
func (c *Client) RetryWaitTime() time.Duration { ... }
func (c *Client) SetRetryWaitTime(waitTime time.Duration) *Client { ... }
func (c *Client) RetryMaxWaitTime() time.Duration { ... }
func (c *Client) SetRetryMaxWaitTime(maxWaitTime time.Duration) *Client { ... }
func (c *Client) RetryStrategy() RetryStrategyFunc { ... }
func (c *Client) SetRetryStrategy(rs RetryStrategyFunc) *Client { ... }
func (c *Client) EnableRetryDefaultConditions() *Client { ... }
func (c *Client) DisableRetryDefaultConditions() *Client { ... }
func (c *Client) IsRetryDefaultConditions() bool { ... }
func (c *Client) SetRetryDefaultConditions(b bool) *Client { ... }
func (c *Client) AllowNonIdempotentRetry() bool { ... }
func (c *Client) SetAllowNonIdempotentRetry(b bool) *Client { ... }
func (c *Client) RetryConditions() []RetryConditionFunc { ... }
func (c *Client) AddRetryConditions(conditions ...RetryConditionFunc) *Client { ... }
func (c *Client) RetryHooks() []RetryHookFunc { ... }
func (c *Client) AddRetryHooks(hooks ...RetryHookFunc) *Client { ... }
```

---

## Task 9: 清理 client.go 中的 save response 相关方法

**Files:**
- Modify: `client.go:1663-1687`

**Step 1: 删除 SaveResponse 相关方法**

删除以下方法：
```go
func (c *Client) IsSaveResponse() bool { ... }
func (c *Client) SetSaveResponse(save bool) *Client { ... }
```

---

## Task 10: 清理 client.go 中的 R() 方法 - 移除 credentials 和 retry 初始化

**Files:**
- Modify: `client.go:595-644`

**Step 1: 删除 credentials 字段赋值**

在 `R()` 方法中删除：
```go
credentials:         c.credentials,
```

**Step 2: 删除 retry 相关字段赋值**

在 `R()` 方法中删除：
```go
RetryCount:                 c.retryCount,
RetryWaitTime:              c.retryWaitTime,
RetryMaxWaitTime:           c.retryMaxWaitTime,
RetryStrategy:              c.retryStrategy,
IsRetryDefaultConditions:   c.isRetryDefaultConditions,
AllowNonIdempotentRetry:    c.allowNonIdempotentRetry,
```

**Step 3: 删除 retryConditions 和 retryHooks 赋值**

在 `R()` 方法中删除：
```go
retryConditions:     slices.Clone(c.retryConditions),
retryHooks:          slices.Clone(c.retryHooks),
```

**Step 4: 删除 IsSaveResponse 赋值**

在 `R()` 方法中删除：
```go
IsSaveResponse:             c.isSaveResponse,
```

---

## Task 11: 清理 client.go 中的 Clone 方法 - 移除 credentials 克隆

**Files:**
- Modify: `client.go:2083-2085`

**Step 1: 删除 credentials 克隆逻辑**

删除以下代码：
```go
if c.credentials != nil {
    cc.credentials = c.credentials.Clone()
}
```

---

## Task 12: 清理 client.go 结构体定义 - 删除保存响应相关字段

**Files:**
- Modify: `client.go:202`

**Step 1: 删除 isSaveResponse 字段**

在 `Client` 结构体中删除：
```go
isSaveResponse           bool
```

---

## Task 13: 清理 resty.go 中的 retry 和 auth 初始化

**Files:**
- Modify: `resty.go:172-174`

**Step 1: 删除 createClient 中的 retry 初始化**

删除：
```go
retryWaitTime:            defaultWaitTime,
retryMaxWaitTime:         defaultMaxWaitTime,
isRetryDefaultConditions: true,
```

---

## Task 14: 清理 resty.go 中的响应中间件 - 移除 SaveToFileResponseMiddleware

**Files:**
- Modify: `resty.go:206-209`

**Step 1: 删除 SaveToFileResponseMiddleware**

将：
```go
c.SetResponseMiddlewares(
    AutoParseResponseMiddleware,
    SaveToFileResponseMiddleware,
)
```

改为：
```go
c.SetResponseMiddlewares(
    AutoParseResponseMiddleware,
)
```

---

## Task 15: 清理 request.go 中的认证相关字段

**Files:**
- Modify: `request.go:36-37, 63, 80`

**Step 1: 删除 AuthToken 和 AuthScheme 字段**

删除：
```go
AuthToken                  string
AuthScheme                 string
```

**Step 2: 删除 HeaderAuthorizationKey 字段**

删除：
```go
HeaderAuthorizationKey     string
```

**Step 3: 删除 credentials 字段**

删除：
```go
credentials         *credentials
```

---

## Task 16: 清理 request.go 中的 retry 相关字段

**Files:**
- Modify: `request.go:64-72`

**Step 1: 删除 retry 相关字段**

删除：
```go
RetryCount                 int
RetryWaitTime              time.Duration
RetryMaxWaitTime           time.Duration
RetryStrategy              RetryStrategyFunc
IsRetryDefaultConditions   bool
AllowNonIdempotentRetry    bool

// RetryTraceID provides GUID for retry count > 0
RetryTraceID string

// Attempt provides insights into no. of attempts
// Resty made.
//
//	first attempt + retry count = total attempts
Attempt int
```

---

## Task 17: 清理 request.go 中的保存响应相关字段

**Files:**
- Modify: `request.go:51, 61`

**Step 1: 删除 OutputFileName 和 IsSaveResponse 字段**

删除：
```go
OutputFileName             string
```

删除：
```go
IsSaveResponse             bool
```

---

## Task 18: 清理 request.go 中的 SetBasicAuth 方法

**Files:**
- Modify: `request.go:601-615`

**Step 1: 删除 SetBasicAuth 方法**

删除整个方法：
```go
// SetBasicAuth method sets the basic authentication header...
func (r *Request) SetBasicAuth(username, password string) *Request { ... }
```

---

## Task 19: 清理 request.go 中的 SetAuthToken 相关方法

**Files:**
- Modify: `request.go:617-663`

**Step 1: 删除认证相关方法**

删除以下方法：
```go
func (r *Request) SetAuthToken(authToken string) *Request { ... }
func (r *Request) SetAuthScheme(scheme string) *Request { ... }
func (r *Request) SetHeaderAuthorizationKey(k string) *Request { ... }
```

---

## Task 20: 清理 request.go 中的 SetOutputFileName 和 SetSaveResponse 方法

**Files:**
- Modify: `request.go:665-700`

**Step 1: 删除 SetOutputFileName 方法**

删除：
```go
func (r *Request) SetOutputFileName(file string) *Request { ... }
```

**Step 2: 删除 SetSaveResponse 方法**

删除：
```go
func (r *Request) SetSaveResponse(save bool) *Request { ... }
```

---

## Task 21: 清理 request.go 中的 retry 相关方法

**Files:**
- Modify: `request.go:967-1085`

**Step 1: 删除 retry 条件和 hooks 方法**

删除：
```go
func (r *Request) AddRetryConditions(conditions ...RetryConditionFunc) *Request { ... }
func (r *Request) SetRetryConditions(conditions ...RetryConditionFunc) *Request { ... }
func (r *Request) AddRetryHooks(hooks ...RetryHookFunc) *Request { ... }
func (r *Request) SetRetryHooks(hooks ...RetryHookFunc) *Request { ... }
```

**Step 2: 删除 retry 配置方法**

删除：
```go
func (r *Request) SetRetryCount(count int) *Request { ... }
func (r *Request) SetRetryWaitTime(waitTime time.Duration) *Request { ... }
func (r *Request) SetRetryMaxWaitTime(maxWaitTime time.Duration) *Request { ... }
func (r *Request) SetRetryStrategy(rs RetryStrategyFunc) *Request { ... }
func (r *Request) EnableRetryDefaultConditions() *Request { ... }
func (r *Request) DisableRetryDefaultConditions() *Request { ... }
func (r *Request) SetRetryDefaultConditions(b bool) *Request { ... }
func (r *Request) SetAllowNonIdempotentRetry(b bool) *Request { ... }
```

---

## Task 22: 清理 request.go 中的 Execute 方法 - 移除 retry 逻辑

**Files:**
- Modify: `request.go:1376-1508`

**Step 1: 简化 Execute 方法**

将整个 `Execute` 方法简化为：
```go
func (r *Request) Execute(method, url string) (res *Response, err error) {
    defer func() {
        if rec := recover(); rec != nil {
            if err, ok := rec.(error); ok {
                r.client.onPanicHooks(r, err)
            } else {
                r.client.onPanicHooks(r, fmt.Errorf("panic %v", rec))
            }
            panic(rec)
        }
    }()

    r.Method = method
    r.URL = url
    res, err = r.client.execute(r)

    if r.isMultiPart {
        for _, mf := range r.multipartFields {
            mf.close()
        }
    }

    r.IsDone = true
    r.client.onErrorHooks(r, res, err)

    backToBufPool(r.bodyBuf)
    return
}
```

---

## Task 23: 清理 request.go 中的 Clone 方法 - 移除 credentials 和 retry 克隆

**Files:**
- Modify: `request.go:1520-1579`

**Step 1: 删除 credentials 克隆**

删除：
```go
// clone basic auth
if r.credentials != nil {
    rr.credentials = r.credentials.Clone()
}
```

---

## Task 24: 清理 request.go 中的 isIdempotent 和 resetFileReaders 方法

**Files:**
- Modify: `request.go:1726-1749`

**Step 1: 删除 isIdempotent 方法**

删除整个方法及相关代码（idempotentMethods 变量也可删除）

**Step 2: 删除 resetFileReaders 方法**

删除整个方法（因为不再有 retry）

---

## Task 25: 清理 response.go 中的 IsSaveResponse 相关逻辑

**Files:**
- Modify: `response.go:188-190`

**Step 1: 删除 fmtBodyString 中的 IsSaveResponse 检查**

删除以下代码块：
```go
if r.Request.IsSaveResponse {
    return "***** RESPONSE WRITTEN INTO FILE *****"
}
```

---

## Task 26: 清理 middleware.go 中的 addCredentials 函数

**Files:**
- Modify: `middleware.go:266-287`

**Step 1: 删除 addCredentials 函数**

删除整个函数：
```go
func addCredentials(c *Client, r *Request) error { ... }
```

**Step 2: 在 PrepareRequestMiddleware 中移除 addCredentials 调用**

删除：
```go
addCredentials(c, r)
```

---

## Task 27: 清理 middleware.go 中的 SaveToFileResponseMiddleware

**Files:**
- Modify: `middleware.go:533-584`

**Step 1: 删除 SaveToFileResponseMiddleware 函数**

删除整个函数：
```go
// SaveToFileResponseMiddleware method used to write HTTP response body into file...
func SaveToFileResponseMiddleware(c *Client, res *Response) error { ... }
```

---

## Task 28: 清理 middleware.go 中的 handleRequestBody 中的 retry 重置逻辑

**Files:**
- Modify: `middleware.go:431-437`

**Step 1: 删除 io.ReadSeeker 的 retry 重置逻辑**

删除以下代码：
```go
// do seek start for retry attempt if io.ReadSeeker
// interface supported
if r.Attempt > 1 {
    if rs, ok := r.Body.(io.ReadSeeker); ok {
        _, _ = rs.Seek(0, io.SeekStart)
    }
}
```

---

## Task 29: 验证编译

**Files:**
- All

**Step 1: 编译检查**

Run: `go build ./...`
Expected: 编译成功，无错误

---

## Task 30: 运行测试

**Files:**
- All

**Step 1: 运行测试**

Run: `go test ./...`
Expected: 测试通过（需要修复失败的测试）

**Step 2: 修复失败的测试**

根据测试失败情况，删除或修改依赖于已删除功能的测试

---

## 验证清单

完成以上任务后，验证：

1. [ ] 代码编译通过
2. [ ] 测试通过
3. [ ] 以下功能已被删除：
   - [ ] SaveResponse (response.go 中的 SaveToFile 相关)
   - [ ] Retry (所有重试相关代码)
   - [ ] Digest Auth (digest.go 文件及相关调用)
   - [ ] Auth/Credentials (Basic Auth、Token Auth 相关)

## 迁移指南

### Basic Auth

**之前：**
```go
client.SetBasicAuth("user", "pass")
```

**之后：**
```go
client.SetHeader("Authorization", "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")))
```

### Auth Token

**之前：**
```go
client.SetAuthToken("token")
```

**之后：**
```go
client.SetHeader("Authorization", "Bearer " + token)
```

### SaveResponse

**之前：**
```go
client.R().SetOutputFileName("output.txt").Get(url)
```

**之后：**
```go
resp, err := client.R().Get(url)
if err == nil {
    os.WriteFile("output.txt", resp.Body(), 0644)
}
```

### Retry

**之前：**
```go
client.SetRetryCount(3)
```

**之后：**
```go
for i := 0; i < 3; i++ {
    resp, err := client.R().Get(url)
    if err == nil {
        break
    }
    time.Sleep(time.Second)
}
```
