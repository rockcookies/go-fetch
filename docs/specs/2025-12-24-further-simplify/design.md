# go-fetch 进一步精简设计

## 目标

在第一阶段精简的基础上，继续删除非核心功能：SaveResponse、Retry、Digest Auth、Auth/Credentials。

## 删除范围

| 功能 | 文件 | 理由 |
|------|------|------|
| SaveResponse | `response.go`, `request.go` | 文件 I/O 不是 HTTP 客户端核心职责 |
| Retry | `retry.go`, `retry_test.go` 等 | 重试策略高度依赖业务场景，应由调用方实现 |
| Digest Auth | `digest.go`, `digest_test.go` 等 | Digest 认证使用率极低，现代 API 多用 Token/OAuth |
| Auth/Credentials | `client.go`, `request.go` 等 | 认证方式多样，直接用 SetHeader("Authorization", ...) 更灵活 |

## 预计删除代码量

约 950+ 行代码。

## 详细删除清单

### 1. 完整删除的文件

```
retry.go           # 重试策略实现
retry_test.go      # 重试测试
digest.go          # Digest 认证实现
digest_test.go     # Digest 认证测试
```

### 2. client.go 清理

**删除字段：**
- `credentials`
- `retryCount`
- `retryWaitTime`
- `retryStrategy`
- `retryConditions`

**删除方法：**
- `SetRetryCount()`
- `SetRetryWaitTime()`
- `SetRetryStrategy()`
- `AddRetryCondition()`
- `RetryCount()`
- `RetryWaitTime()`
- `RetryStrategy()`
- `SetCredentials()`
- `SetCredentialsBasicAuth()`
- `SetCredentialsDigestAuth()`
- `SetCredentialsAuthToken()`
- `Credentials()`
- `isDigestAuth()`
- `getDigestAuth()`

**删除结构体：**
- `Credentials`
- `RetryCondition`
- `RetryStrategyFunc`

### 3. request.go 清理

**删除字段：**
- `retryCount`
- `saveResponseFile`
- `isDigestAuth`

**删除方法：**
- `SetRetryCount()`
- `SetFileReader()`
- `SaveToFile()`
- `SetDigestAuth()`
- `SetBasicAuth()`
- `SetAuthToken()`
- `isDigestAuth()`
- `getDigestAuth()`
- `attemptDigestAuth()`

### 4. response.go 清理

**删除方法：**
- `SaveToFile()`

### 5. 其他文件

**middleware.go**
- 删除认证相关中间件逻辑

**resty.go**
- 删除 Retry 相关的便捷方法导出
- 删除 Auth/Credentials 相关的便捷方法导出

## 保留范围（精简后的核心 API）

### client.go 核心 API

```go
type Client struct {
    // 基础配置
    httpClient       *http.Client
    baseURL          string
    header           http.Header
    queryParams      url.Values
    formData         url.Values
    debug            bool
    allowGetMethodPayload bool
    outputDirectory  string  // 仅用于 curl 导出
    jsonEscapeHTML   bool
    disallowRedirect bool
    contentLength    bool
    closeConnection  bool
    cookies          []*http.Cookie
    log              Logger
    beforeRequest    []RequestMiddleware
    afterResponse    []ResponseMiddleware
}

// 配置方法
func New() *Client
func (c *Client) R() *Request
func (c *Client) SetBaseURL(url string) *Client
func (c *Client) SetHeader(header, value string) *Client
func (c *Client) SetHeaders(headers map[string]string) *Client
func (c *Client) SetQueryParam(param, value string) *Client
func (c *Client) SetQueryParams(params map[string]string) *Client
func (c *Client) SetCookie(cookie *http.Cookie) *Client
func (c *Client) SetCookies(cookies []*http.Cookie) *Client
func (c *Client) Debug(bool) *Client
func (c *Client) SetDisableWarn(bool) *Client
func (c *Client) SetLogger(logger Logger) *Client
func (c *Client) SetTLSClientConfig(config *tls.Config) *Client
func (c *Client) SetCloseConnection(bool) *Client
func (c *Client) OnBeforeRequest(m RequestMiddleware) *Client
func (c *Client) OnAfterResponse(m ResponseMiddleware) *Client
func (c *Client) Close()
```

### request.go 核心 API

```go
// HTTP 方法
func (r *Request) Get(url string) (*Response, error)
func (r *Request) Post(url string) (*Response, error)
func (r *Request) Put(url string) (*Response, error)
func (r *Request) Delete(url string) (*Response, error)
func (r *Request) Patch(url string) (*Response, error)
func (r *Request) Head(url string) (*Response, error)
func (r *Request) Options(url string) (*Response, error)

// 配置方法
func (r *Request) SetHeader(header, value string) *Request
func (r *Request) SetHeaders(headers map[string]string) *Request
func (r *Request) SetBody(body interface{}) *Request
func (r *Request) SetResult(res interface{}) *Request
func (r *Request) SetError(err interface{}) *Request
func (r *Request) SetFile(param, filePath string) *Request
func (r *Request) SetQueryParam(param, value string) *Request
func (r *Request) SetQueryParams(params map[string]string) *Request
func (r *Request) SetFormData(data map[string]string) *Request
```

## 执行步骤

1. 删除 4 个文件（retry.go, retry_test.go, digest.go, digest_test.go）
2. 清理 client.go 中的 retry、credentials、digest 相关代码
3. 清理 request.go 中的 SaveToFile、auth、retry 相关代码
4. 清理 response.go 中的 SaveToFile 方法
5. 清理 middleware.go 中的认证相关逻辑
6. 清理 resty.go 中的便捷方法导出
7. 编译检查：`go build ./...`
8. 测试检查：`go test ./...`

## 预期效果

- 删除约 950+ 行代码
- 删除约 20+ 个公开方法
- API 更加简洁专注
- 维护负担进一步降低

## 迁移指南

### SaveResponse

**之前：**
```go
resp, err := client.R().SetResult(result).SaveToFile("output.txt")
```

**之后：**
```go
resp, err := client.R().SetResult(result).Get(url)
if err == nil {
    os.WriteFile("output.txt", resp.Body(), 0644)
}
```

### Retry

**之前：**
```go
client.SetRetryCount(3).SetRetryWaitTime(time.Second)
```

**之后：**
```go
// 调用方实现自己的重试逻辑
for i := 0; i < 3; i++ {
    resp, err := client.R().Get(url)
    if err == nil {
        break
    }
    time.Sleep(time.Second)
}
```

### Basic Auth

**之前：**
```go
client.SetBasicAuth("user", "pass")
```

**之后：**
```go
client.SetHeader("Authorization", "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")))
```

### Digest Auth

**之前：**
```go
client.SetDigestAuth("user", "pass")
```

**之后：**
```go
// 需要用户自己实现 Digest 认证逻辑，或使用专门的认证库
```
