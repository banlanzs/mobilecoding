.PHONY: build clean dev build-web build-go

# 默认目标：构建前端和后端
build: build-web build-go

# 构建前端
build-web:
	cd web && npm run build

# 构建后端
build-go:
	go build -o dist/mobilecoding.exe ./cmd/server

# 清理构建产物
clean:
	rm -rf dist/
	rm -rf web/dist/
	rm -rf cmd/server/web/

# 开发模式（仅前端）
dev:
	cd web && npm run dev

# 安装依赖
install:
	npm install
	cd web && npm install

# 运行测试
test:
	go test ./...

# 格式化代码
fmt:
	go fmt ./...

# 检查代码
lint:
	cd web && npm run lint