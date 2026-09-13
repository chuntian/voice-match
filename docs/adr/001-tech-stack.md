# ADR 001: 技术栈选型

## 背景

VoiceMatch 需要构建一个一对一实时语音匹配社交 App。核心需求：
- iOS + Android 双端
- 毫秒级信令延迟
- 高并发 WebSocket 连接
- 通话质量稳定（弱网适应）
- 快速迭代开发

## 决策

| 层级 | 技术 |
|---|---|
| 客户端 | Flutter (Dart) |
| 服务端 | Go (Gin + gorilla/websocket) |
| RTC | LiveKit (自托管 WebRTC SFU) |
| 数据存储 | MySQL 8 + Redis 7 |
| 反向代理 | Nginx (WSS + REST) |
| CI/CD | GitHub Actions + Docker |

## 理由

### Flutter 而非原生双端
- 团队规模有限，一套代码覆盖 iOS + Android，开发效率翻倍
- livekit_client 和 flutter_callkit_incoming 均有成熟插件，不阻塞核心功能
- UI 复杂度低（匹配等待、通话界面、个人中心），Flutter 完全够用
- 性能敏感的 RTC 和 CallKit 通过平台通道走原生实现，不损失体验

### Go 而非 Node.js / Java
- WebSocket 长连接场景，goroutine 比 Node.js 事件循环更透明可控
- 单机 5w+ 连接只需 ~300MB 内存，比 Node.js 低 3-5 倍
- 编译为静态二进制，Docker 镜像 < 30MB，部署快
- 团队有 Go 后端经验，学习成本低

### MySQL + Redis 而非纯 PostgreSQL
- MySQL 在国内云厂商（阿里云 RDS）生态最成熟，运维经验丰富
- Redis 用于匹配队列（List）和会话状态（Hash），性能要求高
- 不需要 PostgreSQL 的复杂查询能力

## 后果

### 正面
- 开发效率高：Flutter 一套代码 + Go 单二进制部署
- 资源占用低：2 台 8C16G 即可支撑初期 10w DAU
- 运维简单：Docker Compose 一键起所有服务

### 负面
- Flutter 在复杂动画和原生交互上仍有少量限制，需要 platform channel 补位
- Go 生态中 WebSocket 库选择少（主要 gorilla/websocket），锁定感强
- LiveKit 自托管需要额外维护一套 RTC 基础设施（TURN/STUN 服务器）
