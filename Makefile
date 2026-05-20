.PHONY: help dev build tidy docker-up docker-down docker-logs test

help:
	@echo "PayGate-Omni 开发命令"
	@echo "  make tidy         初始化/更新 Go 依赖（生成 go.sum）"
	@echo "  make docker-up    启动全部容器"
	@echo "  make docker-down  停止容器（保留数据）"
	@echo "  make docker-clean 彻底清除（含数据卷！）"
	@echo "  make docker-logs  实时日志"
	@echo "  make build        本地编译"
	@echo "  make test         运行单元测试"

tidy:
	cd backend && go mod tidy

build:
	cd backend && CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/paygate-server ./cmd/server

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-clean:
	docker compose down -v --remove-orphans

docker-logs:
	docker compose logs -f --tail=100

test:
	cd backend && go test -race -coverprofile=coverage.out ./...
