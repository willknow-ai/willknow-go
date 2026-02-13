# Example Application

这是一个演示如何使用 Willknow 的示例程序。

## 快速开始

### 方式 1: 使用 Docker（推荐）

```bash
# 1. 进入项目根目录
cd /path/to/willknow

# 2. 构建 Docker 镜像
docker build -f examples/Dockerfile -t willknow-demo .

# 3. 运行容器（需要设置 Claude API Key）
docker run -p 8080:8080 -p 8888:8888 \
  -e CLAUDE_API_KEY=your-claude-api-key \
  willknow-demo
```

### 方式 2: 本地运行（用于开发）

```bash
# 1. 设置 API Key
export CLAUDE_API_KEY=your-claude-api-key

# 2. 创建日志目录
sudo mkdir -p /var/log
sudo chmod 777 /var/log

# 3. 运行程序
go run main.go
```

## 使用说明

### 访问应用

- **主应用**: http://localhost:8080
- **AI 助手**: http://localhost:8888

### 测试 AI 诊断功能

1. 打开浏览器访问 http://localhost:8080
2. 点击 "GET /api/error" 按钮触发一个错误
3. 复制返回的 `RequestID`（例如: `abc12345`）
4. 打开 AI 助手页面 http://localhost:8888
5. 输入："RequestID abc12345 出错了，帮我分析"
6. AI 会自动：
   - 查询日志文件找到相关错误
   - 读取源代码分析问题
   - 给出诊断结果和修复建议

### 示例对话

**用户**: RequestID abc12345 出错了，帮我分析

**AI**: 让我查看相关日志...
[AI 使用 read_logs 工具查询日志]

我发现了错误：
- 文件: main.go:145
- 错误: Database connection timeout
- 原因: 代码中没有设置数据库连接超时时间

建议修改为:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
db.QueryContext(ctx, sql)
```

## 其他测试场景

### 查看源代码
```
用户: 显示 main.go 文件的 handleError 函数
```

### 搜索代码
```
用户: 搜索所有日志相关的代码
```

### 查询日志
```
用户: 查询最近的错误日志
```

## 注意事项

1. **API Key**: 必须设置有效的 Claude API Key
2. **日志文件**: 确保 `/var/log/app.log` 可写
3. **端口**: 确保 8080 和 8888 端口未被占用
4. **Docker**: 建议使用 Docker 运行以获得完整体验

## 故障排除

### 问题: "CLAUDE_API_KEY not set"
**解决**: 设置环境变量 `export CLAUDE_API_KEY=your-key` 或在 Docker 运行时通过 `-e` 传递

### 问题: "Permission denied: /var/log/app.log"
**解决**:
- Docker: 自动处理，无需操作
- 本地: `sudo chmod 777 /var/log` 或使用当前目录 `./logs/app.log`

### 问题: "AI Assistant failed"
**解决**: 检查 API Key 是否正确，检查网络连接是否正常

## 自定义

你可以修改 `main.go` 来：
- 更改日志格式
- 添加更多端点
- 自定义错误场景
- 集成到你自己的应用

查看主 README 了解更多集成选项。
