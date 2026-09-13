# VoiceMatch WebSocket 信令协议 — 客户端接入指南

本文档面向客户端开发者，介绍如何连接信令服务器、维护连接、处理重连。
完整消息字段说明见 `shared/protocol/README.md`。

## 1. 建立连接

### 1.1 WebSocket URL

```
wss://api.voicematch.app/ws
```

在开发环境使用 `ws://`。

### 1.2 握手流程

1. 客户端建立 TCP/WebSocket 连接。
2. 客户端发送 `hello` 消息：

```json
{
  "type": "hello",
  "payload": {
    "user_id": "usr_abc123",
    "token": "eyJhbGciOiJ...",
    "device_id": "ios_iphone15pro_xxx",
    "app_version": "1.0.0"
  }
}
```

3. 服务端校验 token 后返回 `hello_ack`：

```json
{
  "type": "hello_ack",
  "payload": {
    "status": "ok",
    "server_time": 1726233600
  }
}
```

4. 收到 `hello_ack.status == "ok"` 后视为连接就绪，可开始发送业务消息。

### 1.3 连接状态

```
connecting → connected (hello_ack 收到) → closed (手动/网络断开)
```

## 2. 心跳机制

| 参数 | 值 |
|---|---|
| 心跳间隔 | 15 秒 |
| 心跳超时 | 30 秒 |

### 客户端实现

1. 连接就绪后启动 `Timer.periodic(15s)`。
2. 每次定时发送 `heartbeat` 消息：

```json
{
  "type": "heartbeat",
  "payload": { "timestamp": 1726233600123 }
}
```

3. 同时启动一个 30s 的超时 Timer。
4. 收到 `heartbeat_ack` 后取消超时 Timer。
5. 如果 30s 内未收到 `heartbeat_ack`，主动关闭连接并触发重连。

```json
{
  "type": "heartbeat_ack",
  "payload": { "timestamp": 1726233600123, "server_time": 1726233600150 }
}
```

## 3. 重连机制

### 3.1 触发条件

- WebSocket `onDone` / `onError`
- 心跳超时（30s 无 ack）

### 3.2 退避策略

指数退避，每次失败后延迟翻倍：

| 尝试次数 | 延迟 |
|---|---|
| 1 | 1s |
| 2 | 2s |
| 3 | 4s |
| 4 | 8s |
| 5 | 16s |
| 6+ | 30s (封顶) |

最多重试 **10 次**，之后停止并通知用户网络异常。

### 3.3 重连后恢复

重连成功后需要：
1. 重新发送 `hello` 完成鉴权。
2. 如果之前在匹配池中，重新发送 `join_pool`。
3. 如果之前在通话中，重新发送 `call_accept` 或从服务端恢复 call 状态。

## 4. 消息类型速查

| type | 方向 | 客户端动作 |
|---|---|---|
| `hello` | C→S | 连接后首条消息，携带鉴权 |
| `hello_ack` | S→C | 确认连接就绪 |
| `heartbeat` | C→S | 15s 周期心跳 |
| `heartbeat_ack` | S→C | 心跳确认 |
| `join_pool` | C→S | 请求入池 |
| `join_pool_ack` | S→C | 入池确认 (status=ok/error) |
| `leave_pool` | C→S | 主动出池 |
| `match_found` | S→C | 匹配成功通知 |
| `call_invite` | S→C | 来电邀请（被叫方） |
| `call_accept` | C→S | 接听 |
| `call_reject` | C→S | 拒接 |
| `call_cancel` | C→S | 主叫取消 |
| `call_end` | C→S | 通话结束 |
| `ice_candidate` | C↔S | WebRTC ICE 透传 |
| `error` | S→C | 错误通知 (code, message) |

## 5. 客户端最佳实践

1. **消息分发**：收到消息后按 `type` 字段分发到对应处理器，不要逐个 if-else 硬编码。
2. **错误处理**：收到 `error` 消息后，根据 `code` 决定是否退出当前流程（如 `already_in_call` → 返回首页）。
3. **消息序列化**：使用 `voice_match_shared` 包中的 `SignalMessage` 子类构造消息，不要手动拼 JSON。
4. **断线兜底**：App 从后台恢复时主动检测 WebSocket 状态，如断开则立即重连。
