# VoiceMatch Client (Flutter)

Flutter 客户端，支持 iOS 13+ 和 Android 6.0+ (API 23+)。

## 环境要求

| 工具 | 版本 |
|---|---|
| Flutter SDK | >= 3.22.0 (stable) |
| Dart SDK | >= 3.4.0 |
| Xcode | >= 15.0 (iOS 构建) |
| Android Studio | >= Hedgehog (Android 构建) |
| CocoaPods | >= 1.14 |

## 运行步骤

```bash
# 安装依赖
cd client
flutter pub get

# 运行（连接开发服务器）
flutter run --dart-define=API_BASE_URL=http://localhost:8080
```

## 环境变量

通过 `--dart-define` 注入：

| 变量名 | 默认值 | 说明 |
|---|---|---|
| `API_BASE_URL` | `https://api.voicematch.app` | REST API 地址 |
| `WS_HOST` | `api.voicematch.app` | WebSocket 主机 |
| `LIVEKIT_URL` | `wss://rtc.voicematch.app` | LiveKit RTC 地址 |

示例：

```bash
flutter run --dart-define=API_BASE_URL=http://10.0.2.2:8080 --dart-define=WS_HOST=10.0.2.2
```

## iOS 特殊配置

### PushKit (VoIP Push)

1. 在 [Apple Developer Console](https://developer.apple.com) 注册 App ID，启用 **Push Notifications** 和 **Voice over IP** capability。
2. 导出 VoIP Services Certificate (`.p12`)，配置到服务端 APNs 推送模块。
3. `Info.plist` 已配置 `UIBackgroundModes` 包含 `voip`。
4. `AppDelegate.swift` 已实现 `PKPushRegistryDelegate`，收到 VoIP 推送时自动通过 `flutter_callkit_incoming` 显示来电界面。

### CallKit

- `flutter_callkit_incoming` 插件自动注册 CallKit 扩展。
- 确保 `Runner/Info.plist` 中 `NSMicrophoneUsageDescription` 已配置。
- 铃声文件需放置在 `Runner/Resources/ringtone.caf`。

## Android 特殊配置

### Telecom ConnectionService

`AndroidManifest.xml` 中已注册：
- `BIND_TELECOM_CONNECTION_SERVICE` 权限
- `FlutterCallkitIncomingConnectionService` service
- `FlutterCallkitIncomingReceiver` broadcast receiver

### 通知渠道

- 渠道 ID：`voicematch_calls`
- 渠道名称：`VoiceMatch 来电`
- 优先级：`HIGH`（弹出全屏通知）

### 最低 SDK 版本

- `minSdk 23` (Android 6.0 Marshmallow)
- `targetSdk 34` (Android 14)

## 服务层架构

```
lib/services/
├── api_service.dart           # 抽象接口
├── api_service_impl.dart      # RealApiService (HTTP + SharedPreferences)
├── websocket_service.dart     # 抽象接口
├── websocket_service_impl.dart # RealWebSocketService (wss + heartbeat + reconnect)
├── rtc_service.dart           # 抽象接口 + RtcEvent
├── rtc_service_impl.dart      # RealRtcService (LiveKit)
├── callkit_service.dart       # 抽象接口 + CallKitEvent
├── callkit_service_impl.dart  # RealCallKitService (flutter_callkit_incoming)
├── match_service.dart         # 抽象接口 + MatchEvent
├── match_service_impl.dart    # RealMatchService (join/leave pool + 60s timeout)
└── service_injector.dart      # 依赖注入 (Mock + Real)
```

### 使用真实服务

在 `main.dart` 启动时调用：

```dart
void main() {
  ServiceInjector.setupRealServices(
    apiBaseUrl: const String.fromEnvironment('API_BASE_URL'),
    wsHost: const String.fromEnvironment('WS_HOST'),
  );
  runApp(const MyApp());
}
```

开发调试时不调用 `setupRealServices()`，自动使用 Mock 实现返回模拟数据。
