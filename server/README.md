# Voice Match 服务端

基于 Go 1.22 + 标准库 `net/http` 的轻量后端，负责用户认证、匹配、举报/黑名单、呼叫信令与推送。

## 架构概述

```
┌────────────┐   HTTP/WS    ┌──────────────────────────────────────┐
│ Flutter    │ ───────────▶ │  nginx (443/80)                      │
│ Client     │ ◀─────────── │   ├─ /api/  → signal:8080 (REST)    │
└────────────┘              │   └─ /ws/   → signal:8080 (WebSocket)│
                            └──────────────┬───────────────────────┘
                                          │
                          ┌───────────────┴───────────────┐
                          ▼                               ▼
                    ┌───────────┐                  ┌──────────────┐
                    │  signal   │ ─── Redis ───▶  │  redis:6379  │
                    │ (Go, 8080)│ ─── MySQL ────▶  │  mysql:3306  │
                    └─────┬─────┘                  └──────────────┘
                          │
              ┌───────────┼────────────┐
              ▼           ▼            ▼
          FCM API     APNs/VoIP    匹配池/信令
```

## 模块依赖

| 包 | 职责 | 依赖 |
|---|---|---|
| `internal/user` | 登录、JWT、资料、偏好 | MySQL, Redis, Apple JWKS |
| `internal/report` | 举报、黑名单 | MySQL, Redis |
| `internal/signal` | WebSocket 信令、呼叫状态 | Redis, push |
| `internal/match` | 随机匹配排队 | Redis |
| `internal/store` | MySQL/Redis 实现 | database/sql, go-redis |
| `pkg/push` | FCM / APNs 推送封装 | net/http |

- 所有 `internal/*` 服务层依赖**消费方定义的接口**（MySQLStore / RedisStore），由 `internal/store` 注入具体实现，便于单测 mock。
- HTTP 层只使用标准库 `net/http`（Go 1.22 路由模式），不引入 Gin/Echo。

## 目录结构

```
server/
├── cmd/
│   ├── signal/        # 主入口：HTTP + WebSocket 服务
│   └── match/         # 匹配 worker（可选独立进程）
├── internal/
│   ├── user/          # 用户服务
│   ├── report/        # 举报/黑名单
│   ├── signal/        # WebSocket 信令
│   ├── match/         # 匹配逻辑
│   └── store/         # MySQL/Redis 实现
├── pkg/
│   └── push/          # FCM / APNs 推送封装
├── config.yaml        # 默认配置
├── Dockerfile
└── go.mod
```

## 配置说明

见 [`config.yaml`](./config.yaml)。关键字段：

| 段 | 字段 | 说明 |
|---|---|---|
| `server.port` | 8080 | HTTP 监听端口 |
| `mysql.dsn` | — | MySQL 连接串 |
| `redis.addr` | redis:6379 | Redis 地址 |
| `jwt.secret` | — | HMAC-SHA256 密钥 |
| `jwt.expire_hours` | 168 | token 有效期（7 天） |
| `apple.client_id` | — | Apple identityToken 的 aud |
| `fcm.server_key` | — | FCM Legacy Server Key |
| `apns.*` | — | VoIP .p8 凭证 |
| `app.dev_mode` | false | true 时短信验证码固定 `888888` |

### 环境变量覆盖

配置项支持 `VM_` 前缀环境变量覆盖，例：

```
VM_JWT_SECRET=xxx VM_MYSQL_DSN=... VM_APP_DEV_MODE=false ./signal
```

## 本地运行（docker-compose）

```bash
# 在项目根目录
make run-dev        # 启动 mysql + redis + signal
make init-db        # 执行建表
make test-server    # 跑单元测试
```

或手动：

```bash
cd deploy
docker-compose up -d
docker-compose ps
curl http://localhost:8080/healthz
```

## API 端点

### 认证
| Method | Path | 说明 |
|---|---|---|
| POST | `/api/v1/auth/apple` | Apple 登录，body: `{identity_token}` |
| POST | `/api/v1/auth/phone` | 发送短信验证码 |
| POST | `/api/v1/auth/phone/verify` | 校验验证码并登录 |

### 用户
| Method | Path | 说明 |
|---|---|---|
| GET | `/api/v1/users/me` | 当前用户资料（需 token） |
| PUT | `/api/v1/users/me` | 更新资料 |
| GET | `/api/v1/users/me/preferences` | 获取匹配偏好 |
| PUT | `/api/v1/users/me/preferences` | 更新偏好 |

### 举报 / 黑名单
| Method | Path | 说明 |
|---|---|---|
| POST | `/api/v1/reports` | 提交举报 |
| GET | `/api/v1/reports` | 举报列表（管理员） |
| PUT | `/api/v1/reports/{id}/status` | 更新举报状态 |
| POST | `/api/v1/users/me/blacklist` | 拉黑 |
| DELETE | `/api/v1/users/me/blacklist/{blockedUserID}` | 取消拉黑 |
| GET | `/api/v1/users/me/blacklist` | 黑名单列表 |

### 呼叫
| Method | Path | 说明 |
|---|---|---|
| GET | `/ws/` | WebSocket 信令（token 通过 query 或 header） |
| GET | `/healthz` | 健康检查 |

除 `/api/v1/auth/*` 与 `/healthz` 外，均需 `Authorization: Bearer <token>`。

## 开发说明

```bash
cd server
go test ./...          # 跑全部单测
gofmt -l . && go vet ./...
go run ./cmd/signal    # 本地直跑（需本地有 mysql/redis）
```

单元测试使用内存 mock（fakeMySQL / fakeRedis），不依赖真实中间件。
