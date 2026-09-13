# LiveKit 云服务配置指南（voice-match）

> 适用范围：服务端 Go（`server/`）、Flutter 客户端（`client/`）。
> 本文只讲**配置接入**，不涉及业务代码重构；代码侧待补的事项在文末「附录 A」列出。

---

## 1. 什么是 LiveKit，本项目为什么需要它

LiveKit 是开源的 WebRTC SFU（选择性转发服务），负责两个人之间**实时语音媒体流**的转发。在本项目里，它和业务后端是两个完全独立的角色：

| 组件 | 职责 | 技术 |
|---|---|---|
| 业务后端（`server/` Go） | 登录、匹配、呼叫信令（振铃/接听/挂断）、下发 RTC Token | HTTP + WebSocket |
| **LiveKit（本次配置对象）** | 真正的语音媒体流传输（麦克风采集 → SFU 转发 → 对方播放） | WebRTC |
| Flutter 客户端（`client/`） | 通过 `livekit_client` SDK 连接 LiveKit 房间，收发音频 | Dart |

**为什么需要它**：一对一语音通话的媒体面不能走业务 WebSocket 通道（带宽、延迟、协议都不合适）。项目协议层已经为此预留了接口——`shared/protocol/messages.go` 中的 `CallInvite` 消息已经定义了 `room_name` / `rtc_token` / `expires_at` 三个字段（见 `shared/protocol/messages.go:162-170`），信令面只负责把这三个值交给客户端，客户端拿着 token 自己去连 LiveKit。

> 备注：`docs/adr/002-rtc-livekit.md` 当初的决策是**自托管** LiveKit。本次改用 **LiveKit Cloud**（`https://cloud.livekit.io`）是自托管的托管替代——API 完全一致，区别只是 URL 和 Key 来自云控制台而非自建服务器。

---

## 2. 当前代码现状（接入前必须知道的三件事）

1. **服务端 RTC Token 目前是假的。** `server/cmd/signal/main.go` 中的 `stubRTCService.CreateRoom()` 直接拼接出 `rtc_token_<callID>` 这样的假字符串返回（`server/cmd/signal/main.go:100-107`），且 `go.mod` 里**还没有引入 LiveKit 服务端 SDK**。真实接入时需要用 LiveKit 官方 Go SDK 签发真 JWT。
2. **客户端 LiveKit 地址是写死的占位符。** `client/lib/services/rtc_service_impl.dart` 第 181 行硬编码返回 `wss://rtc.voicematch.app`，这是占位域名，必须替换成 LiveKit Cloud 分配的真实地址。
3. **`server/config.yaml` 中目前没有任何 LiveKit 配置段。** 需要按下文第 4 节新增。

---

## 3. 注册 LiveKit Cloud 并获取凭证

### 3.1 注册与创建 Project

1. 浏览器打开 <https://cloud.livekit.io>（当前已打开，尚未完成注册）。
2. 注册账号（支持 GitHub / Google / 邮箱）。
3. 登录后点击 **New Project**，创建一个新项目，例如命名 `voice-match`。
4. 选择 Region（机房区域）：面向国内用户建议选 **Singapore / Tokyo** 等离用户最近的区域；开发测试任意区域均可。

### 3.2 获取三样必需信息

在项目控制台 **Project → Settings / Keys** 页面，记录以下三项：

| 名称 | 控制台中的位置 | 格式示例 | 用途 |
|---|---|---|---|
| **Project URL** | Project 概览页，形如 `https://xxxx.livekit.cloud` | `wss://dev-example123.livekit.cloud` | 客户端连接地址（注意：客户端 SDK 用 `wss://` 前缀，把控制台显示的 `https://` 换成 `wss://`） |
| **API Key** | Keys 页，形如 `API******` / `dev******` | `devXXXXXXXX` | 服务端签发 Token 用 |
| **API Secret** | Keys 页，创建项目时生成，**只完整展示一次**，请立即保存 | 一串长 base64 字符串 | 服务端签发 Token 的签名密钥，**等同于密码** |

> ⚠️ API Secret 离开创建页面后通常无法再次完整查看，只能 Rotate（轮换）。丢失后需要重新生成并同步更新服务端配置。

---

## 4. 服务端配置：`server/config.yaml` 新增 `livekit:` 段

**当前 `server/config.yaml` 已有的段**：`server` / `redis` / `mysql` / `jwt` / `apple` / `fcm` / `apns` / `match` / `log` / `app`。**没有** `livekit` 段，需要在文件末尾追加如下内容（密钥位置一律用占位符，不要填真实值提交）：

```yaml
# ---------- LiveKit（RTC 媒体面） ----------
livekit:
  url: "wss://YOUR_PROJECT.livekit.cloud"   # LiveKit Cloud 项目地址，wss:// 开头
  api_key: "YOUR_API_KEY"                   # 控制台 Keys 页的 API Key
  api_secret: "YOUR_API_SECRET"             # 控制台 Keys 页的 API Secret（敏感！）
  token_ttl_seconds: 3600                   # 每次通话签发的 RTC Token 有效期（秒），建议 ≥ 单场通话最长时长 + 60s 缓冲
```

**字段说明：**

| 字段 | 说明 |
|---|---|
| `livekit.url` | 客户端 `room.connect(url, token)` 的第一个参数。本地开发用 LiveKit Cloud 分配的 `wss://...livekit.cloud`；若日后自托管，则填自建 SFU 的 `wss://` 地址 |
| `livekit.api_key` | LiveKit 项目的 API Key，用于服务端 SDK 构造 `AccessToken` |
| `livekit.api_secret` | 用 HS256 对 RTC Token 签名。**任何情况下不得写入代码仓库** |
| `livekit.token_ttl_seconds` | RTC Token 有效期。当前 stub 是 120 秒，生产建议 3600（1 小时），覆盖单次通话即可，过期后客户端重进会重新信令申请 |

**环境变量覆盖**（项目已有约定：`VM_` 前缀，见 `server/README.md` 与 `deploy/docker-compose.yml`）：

```bash
VM_LIVEKIT_URL="wss://YOUR_PROJECT.livekit.cloud"
VM_LIVEKIT_API_KEY="YOUR_API_KEY"
VM_LIVEKIT_API_SECRET="YOUR_API_SECRET"
VM_LIVEKIT_TOKEN_TTL_SECONDS="3600"
./signal
```

Docker Compose 对应在 `deploy/docker-compose.yml` 的 `signal` 服务 `environment:` 段追加同样三行。

> **注意**：目前 `server/cmd/signal/main.go` 里的 `Config` 结构体只绑定了 `server/redis/mysql/jwt` 四个段，新增 `livekit:` 段后代码侧还需要：
> 1. 在 `Config` 中增加 `LiveKit LiveKitConfig \`yaml:"livekit"\``；
> 2. 引入 SDK：`go get github.com/livekit/server-sdk-go/v2`；
> 3. 用它替换 `stubRTCService`：用 `livekit.NewAccessToken(apiKey, apiSecret)`，设置 `Identity`（= 用户 ID）、`Name`、`Grant{RoomJoin: true, Room: roomName}`，`ToJWT()` 产出真 token 返回给信令层。
>
> 以上属于代码改动，不在本文档配置范围内，仅作提醒。

---

## 5. 客户端（Flutter）配置

客户端**不持有** API Key / Secret——这是安全红线。客户端只需要两样东西：**LiveKit 服务器地址** 和 **服务端下发的 RTC Token**（走 `CallInvite.rtc_token` 信令消息）。

需要改的位置：

### 5.1 替换硬编码 URL

文件：`client/lib/services/rtc_service_impl.dart`

当前第 175-182 行 `_liveKitUrlFromToken()` 写死了占位值：

```dart
return 'wss://rtc.voicematch.app';   // ← 占位符，必须替换
```

改为由构造函数注入真实地址，例如：

```dart
class RealRtcService extends ChangeNotifier implements RtcService {
  RealRtcService({required String liveKitUrl});
  final String liveKitUrl;
  ...
  String _liveKitUrlFromToken(String token) => liveKitUrl;  // 返回 wss://YOUR_PROJECT.livekit.cloud
}
```

### 5.2 在注入处传入地址

文件：`client/lib/services/service_injector.dart`

`setupRealServices()` 目前只接收 `apiBaseUrl` 和 `wsHost` 两个参数（第 54-71 行），需要增加一个 `liveKitUrl` 参数并传给 `RealRtcService`：

```dart
static void setupRealServices({
  String apiBaseUrl = 'https://api.voicematch.app',
  String wsHost = 'api.voicematch.app',
  String liveKitUrl = 'wss://YOUR_PROJECT.livekit.cloud',  // ← 新增
}) {
  ...
  final rtc = RealRtcService(liveKitUrl: liveKitUrl);
  ...
}
```

### 5.3 客户端 Android 网络权限（确认项）

`client/android/app/src/main/AndroidManifest.xml` 需确认已声明 `INTERNET` 权限；iOS 侧 `Info.plist` 因 LiveKit 走 `wss://`（加密），无需额外 ATS 例外。`livekit_client: ^1.5.4` 已在 `client/pubspec.yaml:18` 声明，无需再加依赖。

---

## 6. 本地开发 vs 生产环境差异

| 项 | 本地开发 | 生产环境 |
|---|---|---|
| LiveKit URL | 直接用 LiveKit Cloud 分配的 `wss://xxx.livekit.cloud`（开发环境即可用，无需自建） | 同上，或换成正式项目/自托管 SFU 的 `wss://rtc.your-domain.com` |
| API Key / Secret | 用同一个 Project 的 dev 凭证；建议在 Cloud 控制台建**独立 Project**（如 `voice-match-dev`）与生产隔离 | 生产 Project 的 Key/Secret，通过部署环境变量注入 |
| 服务端配置来源 | 本地 `server/config.yaml` 填 dev 占位，或直接 `VM_LIVEKIT_*` 环境变量 | 容器环境变量 / 部署平台密钥管理，**不进镜像、不进 git** |
| `server/config.yaml` | 可保留一份 dev 占位副本（见第 7 节安全建议） | 仓库中的 config.yaml 永远只留占位符 |
| 通话测试 | 用两台真机（或两个登录态）拨号，观察两端是否进入 `RtcEventType.onConnected` | 同左，另需在 LiveKit Cloud 控制台 → Monitor 面板确认房间真实存在、媒体在转发 |

---

## 7. 安全注意事项（重要）

1. **API Secret 绝对不能提交 Git。** 当前 `.gitignore` 已忽略 `.env` / `.env.local` / `secrets/` / `*.pem` / `*.p8` / `*.key`，但 **`server/config.yaml` 本身没有被忽略**——已用 `git check-ignore` 验证：`config.yaml NOT ignored`，且该文件目前已在 git 跟踪中。该文件里 `jwt.secret`、`fcm.server_key`、`apns.*` 未来填真值后都会泄密。
   **建议二选一（本文不替你执行，仅建议）：**
   - 方案 A：把 `server/config.yaml` 从 git 移除跟踪（`git rm --cached server/config.yaml`）并在 `.gitignore` 增加 `/server/config.yaml`，仓库里只保留一份 `server/config.example.yaml` 模板（含占位符）。
   - 方案 B：仓库继续跟踪 `server/config.yaml` 但**只允许放占位符**，真值一律通过 `VM_LIVEKIT_API_SECRET` 等环境变量注入（项目现有 `VM_` 前缀机制已支持），并在文件头注释里写明。
2. RTC Token 是**短时效、按房间签发**的，客户端即使泄露也只能进特定房间且很快过期；但 API Secret 泄露等于把整个项目的媒体面交出去，一旦怀疑泄露立即在控制台 **Rotate Keys**。
3. 不要把 LiveKit Cloud 的 API 调用能力暴露给客户端——签发 Token 永远只在服务端做。
4. 定期在控制台 Keys 页轮换 Key/Secret（建议每 3-6 个月一次），轮换时先加新 Key、灰度切换、再删旧 Key。

---

## 8. 接入完成自检清单

- [ ] LiveKit Cloud 已注册，Project 已创建，拿到 URL / API Key / API Secret
- [ ] `server/config.yaml` 已追加 `livekit:` 段（占位符），或已通过 `VM_LIVEKIT_*` 环境变量注入真值
- [ ] 服务端 `Config` 结构体已增加 `livekit` 段绑定，`stubRTCService` 已替换为真实 LiveKit SDK 实现
- [ ] `client/lib/services/rtc_service_impl.dart` 的 `wss://rtc.voicematch.app` 硬编码已移除
- [ ] `client/lib/services/service_injector.dart` 已新增 `liveKitUrl` 注入参数
- [ ] 两台设备互拨，通话可建立，LiveKit Cloud 控制台 Monitor 中可见对应房间与参与者
- [ ] 真实 Secret 未出现在任何 git 提交记录中（`git log -p -- server/config.yaml` 检查一遍）

---

## 附录 A：与本文档配套的代码待办（非配置项）

| 位置 | 待办 |
|---|---|
| `go.mod` | `go get github.com/livekit/server-sdk-go/v2` |
| `server/cmd/signal/main.go` | `Config` 增加 `LiveKitConfig`；用真实现替换 `stubRTCService`（第 100-107 行） |
| `server/internal/signal/handler.go:43-45` | `RTCService.CreateRoom` 接口签名已就绪，无需改动 |
| `client/lib/services/rtc_service_impl.dart:181` | 删除硬编码 `wss://rtc.voicematch.app` |
| `client/lib/services/service_injector.dart:54` | `setupRealServices` 增加 `liveKitUrl` 参数 |
| `.gitignore` | 视第 7 节建议，决定是否忽略 `server/config.yaml` |
