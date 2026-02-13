# 快速开始指南

## 5 分钟体验 Willknow

### 前置要求

- Docker 已安装
- AI API Key
  - Anthropic Claude: [获取地址](https://console.anthropic.com/)
  - DeepSeek: [获取地址](https://platform.deepseek.com/)

### 步骤

**1. 克隆项目**
```bash
cd /path/to/your/workspace
# 如果你还没有克隆，这里假设你已经在项目目录中
cd willknow
```

**2. 设置 API Key**
```bash
# 使用 Anthropic Claude
export AI_API_KEY=sk-ant-xxxxx  # 替换为你的 API Key
export AI_PROVIDER=anthropic

# 或使用 DeepSeek
export AI_API_KEY=sk-xxxxx  # 替换为你的 API Key
export AI_PROVIDER=deepseek
```

**3. 构建并运行**
```bash
# 从项目根目录构建
docker build -f examples/Dockerfile -t demo .

# 运行容器
docker run -p 8080:8080 -p 8888:8888 \
  -e AI_API_KEY=$AI_API_KEY \
  -e AI_PROVIDER=$AI_PROVIDER \
  demo
```

**4. 测试功能**

打开两个浏览器标签：
- Tab 1: http://localhost:8080 （主应用）
- Tab 2: http://localhost:8888 （AI 助手）

在 Tab 1 中点击 "GET /api/error" 触发错误，复制 RequestID

在 Tab 2 的 AI 助手中输入：
```
RequestID abc12345 出错了，帮我分析
```

观看 AI 自动分析代码和日志！🎉

---

## 集成到你的项目

**1. 安装库**
```bash
go get github.com/willknow-ai/willknow-go
```

**2. 添加代码**
```go
import aiassistant "github.com/willknow-ai/willknow-go"

func main() {
    go func() {
        assistant, _ := aiassistant.New(aiassistant.Config{
            SourcePath: "/app/source",
            Port:       8888,
            Provider:   "anthropic", // 或 "deepseek"
            APIKey:     os.Getenv("AI_API_KEY"),
        })
        assistant.Start()
    }()

    // 你的代码...
}
```

**3. 修改 Dockerfile**
```dockerfile
# 在最终阶段添加这一行
COPY --from=builder /src /app/source
```

完成！详细文档请查看 [README.md](README.md)
