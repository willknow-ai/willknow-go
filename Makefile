.PHONY: help build run docker-build docker-run clean test

help:
	@echo "Willknow - 可用命令:"
	@echo "  make build        - 构建示例程序"
	@echo "  make run          - 本地运行示例程序"
	@echo "  make docker-build - 构建 Docker 镜像"
	@echo "  make docker-run   - 运行 Docker 容器"
	@echo "  make test         - 运行测试"
	@echo "  make clean        - 清理构建文件"

build:
	@echo "构建示例程序..."
	cd examples && go build -o ../bin/willknow-demo main.go
	@echo "✅ 构建完成: bin/willknow-demo"

run:
	@if [ -z "$$CLAUDE_API_KEY" ]; then \
		echo "❌ 错误: 请设置 CLAUDE_API_KEY 环境变量"; \
		echo "   export CLAUDE_API_KEY=your-api-key"; \
		exit 1; \
	fi
	@echo "启动示例程序..."
	@mkdir -p /var/log || sudo mkdir -p /var/log
	cd examples && go run main.go

docker-build:
	@echo "构建 Docker 镜像..."
	docker build -f examples/Dockerfile -t willknow-demo .
	@echo "✅ Docker 镜像构建完成: willknow-demo"

docker-run:
	@if [ -z "$$CLAUDE_API_KEY" ]; then \
		echo "❌ 错误: 请设置 CLAUDE_API_KEY 环境变量"; \
		echo "   export CLAUDE_API_KEY=your-api-key"; \
		exit 1; \
	fi
	@echo "启动 Docker 容器..."
	docker run --rm -p 8080:8080 -p 8888:8888 \
		-e CLAUDE_API_KEY=$$CLAUDE_API_KEY \
		willknow-demo

test:
	@echo "运行测试..."
	go test ./...

clean:
	@echo "清理构建文件..."
	rm -rf bin/
	rm -rf /tmp/test-build
	@echo "✅ 清理完成"
