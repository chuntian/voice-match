# VoiceMatch

> 一键匹配，实时语音连麦。陌生人社交从未如此简单。

VoiceMatch 是一款一对一实时语音匹配社交应用：用户进入匹配池后，系统自动为其匹配一位聊友，双方通过加密语音通道即时对话。

## 功能亮点

- **智能匹配**：随机/同城/目的地三种匹配模式，支持兴趣标签和性别偏好
- **实时通话**：基于 WebRTC 的低延迟语音通话，支持降噪、回声消除
- **系统来电体验**：iOS CallKit / Android ConnectionService，锁屏也能接听
- **隐私安全**：通话内容端到端加密，支持一键举报和拉黑

## 技术栈

| 层 | 技术 |
|---|---|
| 客户端 | Flutter 3.22 / Dart 3.0 |
| 服务端 | Go 1.22 / Gin / gorilla/websocket |
| RTC | LiveKit (自托管 WebRTC SFU) |
| 数据库 | MySQL 8.0 / Redis 7 |
| 反向代理 | Nginx (WSS + REST) |
| CI/CD | GitHub Actions + Docker |

## 架构概览

```
┌──────────┐     WSS      ┌────────┐     WebSocket    ┌──────────────┐
│ Flutter  │◄──────────►│ Nginx  │◄──────────────►│ Signal (Go)  │
│ Client   │             └────────┘                  └──────┬───────┘
└────┬─────┘              ┌────────┐                       │
     │                     │ Nginx  │◄──── REST ────────────┐
     │  wss://rtc          └────────┘                       ▼
     ▼                     ┌────────┐                  ┌──────────────┐
┌──────────┐               │ LiveKit│                 │  User (Go)   │
│  RTC     │               │  RTC   │                 │  Match (Go)  │
└──────────┘               └────────┘                  └──────┬───────┘
                                                               │
                                              ┌────────────────┼────────────────┐
                                              ▼                ▼                ▼
                                         ┌─────────┐    ┌─────────┐    ┌──────────┐
                                         │  Redis  │    │  MySQL  │    │  APNs/FCM│
                                         └─────────┘    └─────────┘    └──────────┘
```

## 快速开始

### 前置要求

| 工具 | 版本 |
|---|---|
| Flutter SDK | >= 3.22.0 |
| Go | >= 1.22 |
| Docker | >= 24.0 |
| Docker Compose | >= 2.20 |
| Node.js (LiveKit CLI) | >= 18 |

### 1. 启动基础设施 + 服务端

```bash
cd deploy
docker-compose up -d
```

这会启动：
- Redis 7 (端口 6379)
- MySQL 8 (端口 3306)
- Nginx (端口 80/443)
- LiveKit Server (端口 7880)

### 2. 启动服务端

```bash
cd server
go run ./cmd/signal &
go run ./cmd/match &
```

### 3. 启动客户端

```bash
cd client
flutter pub get
flutter run
```

> iOS 需要先安装 Pods：
> ```bash
> cd client/ios && pod install && cd ..
> ```

## 目录结构

```
voice-match/
├── client/                 # Flutter 客户端
│   ├── lib/
│   │   ├── models/         # 数据模型 (User, Preferences, CallRecord)
│   │   ├── services/       # 服务层 (WebSocket, RTC, API, Match, CallKit)
│   │   ├── pages/          # 页面 (Login, Home, MatchWaiting, Call, Profile)
│   │   ├── widgets/        # 通用组件
│   │   ├── router/         # 路由
│   │   └── main.dart       # 入口
│   ├── ios/                # iOS 原生配置 (CallKit, PushKit)
│   ├── android/            # Android 原生配置 (ConnectionService)
│   └── pubspec.yaml
├── server/                 # Go 服务端
│   ├── cmd/
│   │   ├── signal/         # 信令服务入口
│   │   └── match/          # 匹配服务入口
│   ├── internal/
│   │   ├── signal/         # WebSocket 处理
│   │   ├── match/          # 匹配引擎
│   │   ├── user/           # 用户/认证/REST API
│   │   ├── report/         # 举报
│   │   └── store/          # Redis/MySQL 存储
│   └── pkg/push/           # APNs / FCM 推送
├── shared/                 # 共享协议定义
│   ├── protocol/           # 消息模型 (Dart + Go)
│   └── errors/             # 错误码 (Dart + Go)
├── deploy/                 # Docker Compose 部署
│   ├── docker-compose.yml
│   ├── nginx/              # Nginx 配置
│   └── redis/              # Redis 配置
├── docs/                   # 项目文档
│   ├── architecture.md
│   ├── api.md
│   ├── protocol.md
│   └── adr/                # 架构决策记录
└── .github/workflows/      # CI/CD
    ├── client.yml
    ├── server.yml
    └── deploy.yml
```

## 开发指南

### 代码规范

- **Dart**：遵循 [Effective Dart](https://dart.dev/guides/language/effective-dart)，使用 `flutter_lints`
- **Go**：遵循 [Effective Go](https://go.dev/doc/effective_go)，`gofmt` + `go vet`

### 提交规范

```
<type>(<scope>): <subject>

type: feat | fix | docs | style | refactor | test | chore
```

### 测试

```bash
# 客户端
cd client && flutter test

# 服务端
cd server && go test ./... -v -race
```

## 部署指南

推送 `v*` tag 即可触发自动部署：

```bash
git tag v1.0.0
git push origin v1.0.0
```

CI 会自动构建 Docker 镜像并 SSH 到生产服务器执行 `docker-compose up -d`。

## License

MIT License
