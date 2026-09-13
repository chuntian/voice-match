# Voice Match 信令协议文档

## 1. 概述

客户端与服务端通过 WebSocket 进行双向 JSON 消息通信。每条消息使用统一的 Envelope 包装：

```json
{
  "type": "<message_type>",
  "payload": { ... }
}
```

- `type`：消息类型字符串，见下方常量列表。
- `payload`：对应消息类型的具体内容；无 payload 的消息（如 `leave_pool`）可省略。

传输层：`wss://host/ws?token=xxx`，子协议为空。

---

## 2. 消息类型常量

| type | 方向 | 说明 |
|---|---|---|
| `hello` | C→S | 连接建立后第一条消息，携带鉴权信息 |
| `hello_ack` | S→C | 鉴权结果与服务器时间 |
| `heartbeat` | C→S | 客户端心跳，每 10s 发送 |
| `heartbeat_ack` | S→C | 心跳响应 |
| `join_pool` | C→S | 请求进入匹配池 |
| `join_pool_ack` | S→C | 入池确认 |
| `leave_pool` | C→S | 主动离开匹配池 |
| `leave_pool_ack` | S→C | 出池确认 |
| `match_found` | S→C | 匹配成功通知双方 |
| `call_invite` | S→C | 服务端转发给被叫的呼叫邀请 |
| `call_accept` | C→S | 被叫接听 |
| `call_reject` | C→S | 被叫拒接 |
| `call_cancel` | C→S | 主叫取消 |
| `call_end` | C→S | 通话结束（任一方） |
| `ice_candidate` | C↔S | WebRTC ICE 候选透传 |
| `error` | S→C | 错误通知 |

---

## 3. 消息流程图

### 3.1 连接与鉴权

```
Client                          Server
  |                               |
  |---------- WebSocket upgrade -->|
  |                               |
  |---- hello {user_id, token} -->|
  |                               |  (验证 token)
  |<-- hello_ack {status, time} --|
  |                               |
  |<==== 连接就绪 ====>            |
```

### 3.2 匹配流程

```
Client A                         Server                         Client B
  |                               |                               |
  |---- join_pool {random} ----->|                               |
  |<-- join_pool_ack {ok} -------|                               |
  |                               |                               |
  |                               |<-- join_pool {random} -------|
  |                               |-- join_pool_ack {ok} ------->|
  |                               |                               |
  |                               |  (Matcher 每 500ms LPOP x2)  |
  |<-- match_found {call_id} ----|                               |
  |                               |-- match_found {call_id} --->|
  |                               |                               |
  |<-- call_invite {rtc_token} ---|                               |
  |                               |<-- call_accept {call_id} ----|
  |--- call_accept {call_id} --->|                               |
  |                               |                               |
  |  (WebRTC P2P media flows)     |                               |
  |                               |                               |
```

### 3.3 呼叫状态流转

```
                 ┌─────────┐
                 │  idle  │
                 └────┬────┘
                      │ caller: call_invite → match_found
                      ▼
                 ┌─────────┐
            ┌────│ ringing│────┐
            │    └────┬────┘    │
   reject / cancel    │ accept   │ ringing timeout (30s)
            │         ▼          │
            │    ┌─────────┐     │
            │    │connected│     │
            │    └────┬────┘     │
            │         │ end       │
            ▼         ▼          ▼
                 ┌─────────┐
                 │  ended  │
                 └─────────┘
```

---

## 4. 各消息字段说明

### 4.1 Hello（C→S）

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| user_id | string | ✓ | 用户唯一 ID |
| token | string | ✓ | 鉴权 JWT/Token |
| device_id | string | ✓ | 设备标识（多端登录区分） |
| app_version | string | ✓ | 客户端版本号 |

### 4.2 HelloAck（S→C）

| 字段 | 类型 | 说明 |
|---|---|---|
| status | string | `"ok"` 或错误状态 |
| server_time | int64 | 服务器 Unix 时间戳（秒） |

### 4.3 Heartbeat / HeartbeatAck

| 字段 | 类型 | 说明 |
|---|---|---|
| timestamp | int64 | 客户端发送时的 Unix 毫秒时间戳 |
| server_time | int64 | （仅 ack）服务器时间戳 |

### 4.4 JoinPool（C→S）

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| match_type | string | ✓ | `"random"` / `"city"` / `"destination"` |
| city | string | 条件 | 城市 ID（city/destination 模式必填） |
| geohash | string | 条件 | 地理位置 geohash（city 模式必填） |
| destination | string | 条件 | 目的地标签（destination 模式必填） |
| tags | []string | ✗ | 用户标签列表 |
| gender_preference | string | ✗ | 性别偏好：`"male"` / `"female"` / `"any"` |

### 4.5 JoinPoolAck（S→C）

| 字段 | 类型 | 说明 |
|---|---|---|
| status | string | `"ok"` / 错误 |
| pool_size | int | 当前池人数 |
| wait_time | int | 预计等待时间（秒） |

### 4.6 MatchFound（S→C）

| 字段 | 类型 | 说明 |
|---|---|---|
| call_id | string | 本次通话唯一 ID |
| matched_user_id | string | 对方用户 ID |
| matched_nickname | string | 对方昵称 |
| matched_avatar | string | 对方头像 URL |
| match_type | string | 匹配类型 |

### 4.7 CallInvite（S→C，仅发给被叫）

| 字段 | 类型 | 说明 |
|---|---|---|
| call_id | string | 通话 ID |
| caller_id | string | 主叫 ID |
| caller_nickname | string | 主叫昵称 |
| caller_avatar | string | 主叫头像 |
| room_name | string | RTC 房间名 |
| rtc_token | string | RTC 鉴权 token |
| expires_at | int64 | token 过期时间戳 |

### 4.8 CallAccept / CallReject / CallCancel / CallEnd

| 字段 | 类型 | 说明 |
|---|---|---|
| call_id | string | 通话 ID |
| reason | string | （可选）原因码 |
| duration | int | （仅 call_end）通话时长（秒） |

### 4.9 IceCandidate（C↔S）

| 字段 | 类型 | 说明 |
|---|---|---|
| call_id | string | 通话 ID |
| target_user_id | string | 接收方用户 ID |
| candidate | string | ICE candidate SDP 字符串 |
| sdp_mid | string | SDP mid |
| sdp_m_line_index | int | SDP m-line index |

### 4.10 Error（S→C）

| 字段 | 类型 | 说明 |
|---|---|---|
| code | int | 错误码（见 shared/errors） |
| message | string | 人类可读错误描述 |
| details | string | （可选）详细信息 |

---

## 5. 超时规则

| 场景 | 超时值 | 超时行为 |
|---|---|---|
| 心跳间隔 | 10s | 客户端每 10s 发送 heartbeat |
| 心跳超时 | 30s | 服务端 30s 未收到 heartbeat，断开 WebSocket |
| 呼叫振铃 | 30s | ringing 状态 30s 未接听，自动 cancel 并通知双方 |
| 匹配池等待 | 60s | 用户入池 60s 未匹配到，自动出池并通知 |
| RTC token | 默认 60s | CallInvite 中 expires_at 指定过期时间 |

---

## 6. 状态流转详细说明

```
连接状态：
  connecting → connected(hello后) → closed

通话状态机（CallSession.State）：
  idle → ringing   (call_invite 发出，被叫未响应)
  ringing → connected  (call_accept)
  ringing → ended      (call_reject / call_cancel / 30s 超时)
  connected → ended    (call_end，任一方发起)
  ended → (终态，不可再转换)

非法转换（如 connected → ringing）会返回错误并发送 error 消息。
```
