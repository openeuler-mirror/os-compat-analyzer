# Makefile for os-checker

.PHONY: all frontend build clean

# 默认目标：构建前端并编译后端
all: frontend build

# 前端构建：构建 Vue 项目并将输出复制到 cmd/templates
frontend:
	cd web && npm run build
	cp web/dist/index.html cmd/templates/report.html

# 后端编译：使用 embed 嵌入前端模板
build:
	go build -ldflags="-s -w" -o os-checker .

# 开发模式：运行前端开发服务器
dev:
	cd web && npm run dev

# 清理构建产物
clean:
	rm -f os-checker
	cd web && rm -rf dist

# 完整清理：清理所有构建产物
distclean: clean
	cd web && rm -rf node_modules
