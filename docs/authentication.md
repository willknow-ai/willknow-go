# Willknow Authentication

本文档描述 Willknow 的认证机制设计与使用方式。

## 概览

Willknow 支持三种认证模式，按优先级顺序：

| 优先级 | 模式 | 触发条件 | 适用场景 |
|--------|------|----------|----------|
| 1 | **宿主系统认证** | `GetUser` 设为自定义函数 | 宿主系统已有认证，集成 SSO |
| 2 | **完全开放** | `GetUser` 设为 `aiassistant.NoAuth` | 开发/内网环境，明确声明无需认证 |
| 3 | **密码保护** | `GetUser` 为 nil（默认） | 无现有认证系统时的安全保底 |

---

## 模式一：宿主系统认证（推荐用于生产）

当宿主系统已有认证系统时，通过 `GetUser` 回调函数将用户信息传递给 Willknow。Willknow 会在每个请求（包括 WebSocket 握手）时调用此函数。

### 使用方式

```go
assistant, _ := aiassistant.New(aiassistant.Config{
    // ...其他配置...
    Auth: aiassistant.AuthConfig{
        GetUser: func(r *http.Request) (*aiassistant.User, error) {
            // 从请求中提取并验证用户身份
            // 可以读取 Cookie、JWT Header、Session 等
            token := r.Header.Get("Authorization")
            user, err := myAuthSystem.ValidateToken(token)
            if err != nil {
                return nil, err // 返回 error → Willknow 拒绝请求（401）
            }
            return &aiassistant.User{
                ID:    user.ID,
                Name:  user.Name,
                Email: user.Email,
            }, nil
        },
    },
})
```

### 认证失败行为

- `GetUser` 返回 error → HTTP 请求返回 `401 JSON`，WebSocket 握手被拒绝
- 用户需要先到宿主系统完成登录，无登录页面跳转

### 常见集成场景

**从 JWT Header 获取用户：**
```go
GetUser: func(r *http.Request) (*aiassistant.User, error) {
    authHeader := r.Header.Get("Authorization")
    if !strings.HasPrefix(authHeader, "Bearer ") {
        return nil, errors.New("missing token")
    }
    claims, err := jwt.Parse(strings.TrimPrefix(authHeader, "Bearer "), jwtSecret)
    if err != nil {
        return nil, err
    }
    return &aiassistant.User{ID: claims.UserID, Name: claims.Name}, nil
},
```

**从 Session Cookie 获取用户：**
```go
GetUser: func(r *http.Request) (*aiassistant.User, error) {
    cookie, err := r.Cookie("session_id")
    if err != nil {
        return nil, errors.New("no session")
    }
    user, err := sessionStore.Get(cookie.Value)
    if err != nil {
        return nil, err
    }
    return &aiassistant.User{ID: user.ID, Name: user.Name}, nil
},
```

---

## 模式二：完全开放（明确声明）

使用预定义的 `NoAuth` 哨兵值，明确表示不需要认证。这与"忘记设置认证"（密码保护模式）形成区分——使用 `NoAuth` 是一种主动的、有意识的选择。

### 使用方式

```go
assistant, _ := aiassistant.New(aiassistant.Config{
    Auth: aiassistant.AuthConfig{
        GetUser: aiassistant.NoAuth,
    },
})
```

### 启动时提示

```
[AI Assistant] WARNING: Authentication is disabled. Anyone can access the assistant.
```

> **注意**：仅建议在开发环境或严格访问控制的内网环境中使用。

---

## 模式三：密码保护（默认行为）

当 `Auth.GetUser` 为 nil 时（即完全不配置 `Auth`），Willknow 自动启用密码保护。

### 自动生成密码

```go
// 最简配置，不设置 Auth
assistant, _ := aiassistant.New(aiassistant.Config{
    Provider: "anthropic",
    APIKey:   os.Getenv("AI_API_KEY"),
})
```

启动时控制台输出：

```
╔══════════════════════════════════════════════╗
║         AI Assistant - Access Password         ║
╠══════════════════════════════════════════════╣
║  Password: a3f7c2b1                            ║
║  URL:      http://localhost:8888               ║
╚══════════════════════════════════════════════╝
```

> **注意**：密码每次重启时重新生成。

### 手动指定密码

```go
assistant, _ := aiassistant.New(aiassistant.Config{
    Auth: aiassistant.AuthConfig{
        Password: "my-team-password",
    },
})
```

### 认证流程

```
用户访问 http://localhost:8888
  ↓
未认证？→ 重定向到 /auth/login（显示密码输入框）
  ↓
POST /auth/login（提交密码）
  ↓
密码正确？→ 设置 willknow_session Cookie → 重定向到 /
  ↓
WebSocket 连接：验证 Cookie → 允许连接
```

---

## 用户信息结构

`GetUser` 函数返回的 `User` 对象：

```go
type User struct {
    ID    string // 必填，用于审计日志
    Name  string // 显示名称
    Email string // 可选
}
```

---

## 审计日志

认证成功后，用户信息会记录到 `./sessions/` 目录下的 JSONL 日志文件中：

```jsonl
{"timestamp":"2026-02-14T08:00:00Z","session_id":"abc123","type":"session_start","data":{"user_id":"user42","user_name":"张三","remote_addr":"192.168.1.10:54321"}}
{"timestamp":"2026-02-14T08:00:05Z","session_id":"abc123","type":"user_message","data":{"content":"帮我分析一下这个错误"}}
```

这样可以追踪每个会话是由哪个用户发起的，满足审计需求。

---

## 配置选项速查

```go
type AuthConfig struct {
    // GetUser：用户认证函数
    //   - 自定义函数 → 宿主系统认证模式
    //   - aiassistant.NoAuth → 完全开放模式
    //   - nil（默认） → 密码保护模式
    GetUser GetUserFunc

    // Password：仅在密码保护模式下生效
    //   - 留空 → 自动生成随机密码（打印到控制台）
    //   - 设置 → 使用指定密码
    Password string
}
```

---

## 模式决策树

```
配置了 Auth.GetUser？
├── Yes → GetUser == aiassistant.NoAuth？
│         ├── Yes → 完全开放模式
│         └── No  → 宿主系统认证模式
└── No  → 密码保护模式
           └── Auth.Password 已设置？
               ├── Yes → 使用指定密码
               └── No  → 自动生成随机密码
```
