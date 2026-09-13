# Voice Match 项目根 Makefile

COMPOSE := docker compose
COMPOSE_DEV := $(COMPOSE) -f deploy/docker-compose.yml
SQL_FILE ?= deploy/init.sql

.PHONY: all
all: build-server test-server

## 构建服务端镜像
build-server:
	docker build -t voicematch-server -f server/Dockerfile .

## 本地启动（docker-compose）
run-dev:
	cd deploy && $(COMPOSE) up -d

## 停止本地环境
stop-dev:
	cd deploy && $(COMPOSE) down

## 生产环境启动（需要生产 env）
run-prod:
	cd deploy && $(COMPOSE) -f docker-compose.yml -f docker-compose.prod.yml up -d

## 服务端测试
test-server:
	go test ./server/... ./shared/...

## 客户端测试（Flutter）
test-client:
	cd client && flutter test

## 代码风格与静态检查
lint-server:
	gofmt -l server/ shared/ && go vet ./server/... ./shared/...

## 初始化数据库（通过 docker exec 执行建表）
init-db:
	@if [ -f "$(SQL_FILE)" ]; then \
		docker exec -i vm-mysql mysql -uroot -prootpass voicematch < $(SQL_FILE); \
	else \
		echo "未找到 $(SQL_FILE)，跳过"; \
	fi

## 清理构建产物
clean:
	rm -f server/signal
	docker rmi voicematch-server 2>/dev/null || true

## help
help:
	@echo "Targets:"
	@echo "  build-server   构建服务端 Docker 镜像"
	@echo "  run-dev        启动本地 docker-compose"
	@echo "  stop-dev       停止本地环境"
	@echo "  test-server    运行 Go 单元测试"
	@echo "  test-client    运行 Flutter 测试"
	@echo "  lint-server    gofmt + go vet"
	@echo "  init-db        执行建表脚本"
	@echo "  clean          清理构建产物"
