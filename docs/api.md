# VoiceMatch REST API 文档

Base URL: `https://api.voicematch.app`

## 认证方式

所有需要认证的接口在 Header 中携带：

```
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json
```

未携带 Token 或 Token 过期时返回 `401 Unauthorized`。

---

## 1. 认证

### 1.1 Apple 登录

```
POST /api/v1/auth/apple
```

**请求体：**

```json
{
  "identity_token": "eyJhbGciOiJ...",
  "authorization_code": "optional",
  "full_name": "optional"
}
```

**响应 200：**

```json
{
  "token": "eyJhbGciOiJ...",
  "user": {
    "id": "usr_abc123",
    "apple_id": "001234.abcdef",
    "nickname": "Voice 用户",
    "avatar": "https://cdn.voicematch.app/avatars/abc.jpg",
    "gender": "",
    "city": "杭州",
    "created_at": "2026-09-01T10:00:00Z",
    "updated_at": "2026-09-01T10:00:00Z"
  }
}
```

### 1.2 发送手机验证码

```
POST /api/v1/auth/phone
```

**请求体：**

```json
{
  "phone": "+8613800138000"
}
```

**响应 200：** `{ "status": "ok" }`

**错误码：** `1001` — 发送频率限制

### 1.3 手机号验证码登录

```
POST /api/v1/auth/phone/verify
```

**请求体：**

```json
{
  "phone": "+8613800138000",
  "code": "123456"
}
```

**响应 200：** 同 Apple 登录，返回 `{token, user}`。

---

## 2. 用户资料

### 2.1 获取当前用户

```
GET /api/v1/users/me
Authorization: Bearer <token>
```

**响应 200：**

```json
{
  "id": "usr_abc123",
  "nickname": "小明",
  "avatar": "https://cdn...",
  "gender": "male",
  "city": "杭州",
  "geohash": "wtmkn0",
  "created_at": "2026-09-01T10:00:00Z",
  "updated_at": "2026-09-10T08:00:00Z"
}
```

### 2.2 更新用户资料

```
PUT /api/v1/users/me
```

**请求体：**

```json
{
  "nickname": "新昵称",
  "avatar": "https://cdn...",
  "gender": "female",
  "city": "上海",
  "geohash": "wtw3sz"
}
```

**响应 200：** 返回更新后的完整 User 对象。

---

## 3. 匹配偏好

### 3.1 获取偏好

```
GET /api/v1/users/me/preferences
```

**响应 200：**

```json
{
  "match_type": "random",
  "city": "",
  "destination": "",
  "tags": ["音乐", "旅行"],
  "gender_preference": "any",
  "age_range_min": 20,
  "age_range_max": 35
}
```

### 3.2 更新偏好

```
PUT /api/v1/users/me/preferences
```

**请求体：** 同 GET 响应结构。

**响应 200：** `{ "status": "ok" }`

---

## 4. 通话记录

### 4.1 获取通话记录

```
GET /api/v1/users/me/call-records?page=1&page_size=20
```

**响应 200：**

```json
{
  "total": 42,
  "page": 1,
  "page_size": 20,
  "items": [
    {
      "id": "call_001",
      "caller_id": "usr_abc",
      "callee_id": "usr_def",
      "start_time": "2026-09-12T20:00:00Z",
      "end_time": "2026-09-12T20:05:30Z",
      "duration": 330,
      "end_reason": "ended",
      "created_at": "2026-09-12T20:00:00Z",
      "peer": {
        "id": "usr_def",
        "nickname": "小红",
        "avatar": "https://..."
      }
    }
  ]
}
```

---

## 5. 举报与拉黑

### 5.1 举报用户

```
POST /api/v1/reports
```

**请求体：**

```json
{
  "reported_id": "usr_xyz",
  "reason": "abusive",
  "content": "对方在通话中辱骂"
}
```

**响应 200：** `{ "status": "ok" }`

### 5.2 拉黑用户

```
POST /api/v1/users/me/blacklist
```

**请求体：**

```json
{
  "blocked_user_id": "usr_xyz"
}
```

### 5.3 取消拉黑

```
DELETE /api/v1/users/me/blacklist/{blockedUserId}
```

### 5.4 获取黑名单

```
GET /api/v1/users/me/blacklist
```

**响应 200：**

```json
{
  "items": [
    {
      "user_id": "usr_abc",
      "blocked_user_id": "usr_xyz",
      "created_at": "2026-09-10T12:00:00Z",
      "user": {
        "id": "usr_xyz",
        "nickname": "不良用户",
        "avatar": "https://..."
      }
    }
  ]
}
```

---

## 6. 错误响应格式

所有非 2xx 响应统一格式：

```json
{
  "code": 3,
  "message": "user is already in a call",
  "details": ""
}
```

| HTTP 状态码 | 含义 |
|---|---|
| 400 | 请求参数错误 |
| 401 | Token 无效或过期 |
| 403 | 被拉黑/权限不足 |
| 404 | 资源不存在 |
| 429 | 频率限制 |
| 500 | 服务器内部错误 |

完整错误码定义见 `shared/errors/codes.go`。
