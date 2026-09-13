# ADR 003: 信令服务无状态设计

## 背景

信令服务器（Signal Service）维护 WebSocket 长连接，需要处理：
- 连接鉴权
- 心跳维持
- 消息路由（转发给特定用户）
- 呼叫信令（call_invite / accept / reject）

如果信令服务保存用户连接状态（如 `map[userID]*WebSocket`），则服务实例是有状态的。多实例部署时需要共享状态。

## 决策

**信令服务实例本身无状态，所有连接状态存 Redis。**

### 具体设计

1. **连接注册表**：用户 `usr_abc` 连接到 Signal 实例 A 时，在 Redis 写入：
   ```
   SET session:usr_abc instance=A, conn_id=xxx EX 35
   ```
   TTL 35s（略大于心跳超时 30s）。

2. **消息路由**：需要给某用户发消息时，先查 Redis 获取该用户所在的实例，再通过 Redis Pub/Sub 将消息投递到对应实例。

3. **实例订阅**：每个 Signal 实例启动时订阅自己的 channel `signal:instance:A`，收到消息后直接通过本地 WebSocket 发送给对应连接。

4. **匹配服务独立**：Matcher 不维护 WebSocket 连接，只操作 Redis List（匹配队列）。匹配成功后通过 Pub/Sub 通知两个用户所在的 Signal 实例。

## 理由

### 水平扩展简单
- 新增 Signal 实例只需注册到负载均衡，无需迁移状态
- 任意实例宕机，客户端只需重连到新实例，Redis 中的旧 session 自然过期

### 故障恢复快
- 实例宕机后 35s 内 Redis session 过期，其他实例自动接管
- 不需要主从切换逻辑

### 调试方便
- 直接在 Redis 中查看任意用户的当前连接实例，排查路由问题
- Pub/Sub 消息可通过 `redis-cli SUBSCRIBE` 实时观察

## 后果

### 正面
- 实例可随意扩缩容，无状态迁移
- 单实例故障不影响其他用户
- 开发和测试简单（本地起一个实例即可）

### 负面
- 每条消息多一次 Redis Pub/Sub 跳转，增加 ~1-2ms 延迟
- 需要设计 Redis 连接池的优雅关闭逻辑
- 心跳超时设为 30s + Redis TTL 35s，意味着实例宕机后客户端最长 35s 才感知
