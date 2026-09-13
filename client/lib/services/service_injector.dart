import 'dart:async';

import 'package:flutter/foundation.dart';

import '../models/user.dart';
import '../models/preferences.dart';
import '../models/call_record.dart';
import '../models/blocked_user.dart';
import 'api_service.dart';
import 'websocket_service.dart';
import 'rtc_service.dart';
import 'callkit_service.dart';
import 'match_service.dart';
import 'api_service_impl.dart';
import 'websocket_service_impl.dart';
import 'rtc_service_impl.dart';
import 'callkit_service_impl.dart';
import 'match_service_impl.dart';

/// 服务注入器
///
/// 本文件提供各服务的默认实现。另一个 Agent 完成真实服务层后，
/// 可通过 [setupServices] 注入真实实例，替换这里的默认实现。
/// 页面层始终通过 Provider 依赖抽象接口，不直接依赖具体实现。

class ServiceInjector {
  ServiceInjector._();

  static final ApiService apiService = _MockApiService();
  static final WebSocketService webSocketService = _MockWebSocketService();
  static final RtcService rtcService = _MockRtcService();
  static final CallKitService callKitService = _MockCallKitService();
  static final MatchService matchService = _MockMatchService();

  /// 另一个 Agent 实现真实服务后，调用此方法注入
  static void setupServices({
    ApiService? api,
    WebSocketService? ws,
    RtcService? rtc,
    CallKitService? callKit,
    MatchService? match,
  }) {
    if (api != null) _apiService = api;
    if (ws != null) _wsService = ws;
    if (rtc != null) _rtcService = rtc;
    if (callKit != null) _callKitService = callKit;
    if (match != null) _matchService = match;
  }

  /// 在生产环境调用，注入所有真实服务实现。
  ///
  /// [apiBaseUrl] 默认为 `https://api.voicematch.app`。
  /// [wsHost] 默认为 `api.voicematch.app`（通过 wss:// 连接）。
  static void setupRealServices({
    String apiBaseUrl = 'https://api.voicematch.app',
    String wsHost = 'api.voicematch.app',
  }) {
    final api = RealApiService(baseUrl: apiBaseUrl);
    final ws = RealWebSocketService(host: wsHost);
    final rtc = RealRtcService();
    final callKit = RealCallKitService();
    final match = RealMatchService(webSocketService: ws);

    setupServices(
      api: api,
      ws: ws,
      rtc: rtc,
      callKit: callKit,
      match: match,
    );
  }

  static ApiService get currentApiService => _apiService;
  static WebSocketService get currentWsService => _wsService;
  static RtcService get currentRtcService => _rtcService;
  static CallKitService get currentCallKitService => _callKitService;
  static MatchService get currentMatchService => _matchService;

  static ApiService _apiService = apiService;
  static WebSocketService _wsService = webSocketService;
  static RtcService _rtcService = rtcService;
  static CallKitService _callKitService = callKitService;
  static MatchService _matchService = matchService;
}

// ==================== Mock 实现 ====================
// 以下为可运行的默认实现，返回模拟数据。
// 另一个 Agent 实现真实服务后，通过 ServiceInjector.setupServices 替换。

class _MockApiService implements ApiService {
  User? _currentUser;
  Preferences? _prefs;

  @override
  Future<User> loginWithApple(String identityToken) async {
    await Future.delayed(const Duration(milliseconds: 500));
    _currentUser = User(
      id: 'mock_user_1',
      appleId: identityToken,
      nickname: 'Voice 用户',
      gender: 'unknown',
      city: '杭州',
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
    );
    return _currentUser!;
  }

  @override
  Future<User> loginWithPhone(String phone, String code) async {
    await Future.delayed(const Duration(milliseconds: 500));
    _currentUser = User(
      id: 'mock_user_$phone',
      phone: phone,
      nickname: '用户${phone.substring(phone.length - 4)}',
      gender: 'unknown',
      city: '杭州',
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
    );
    return _currentUser!;
  }

  @override
  Future<void> sendPhoneCode(String phone) async {
    await Future.delayed(const Duration(milliseconds: 300));
    debugPrint('验证码已发送至 $phone');
  }

  @override
  Future<User> getProfile() async {
    await Future.delayed(const Duration(milliseconds: 300));
    if (_currentUser != null) return _currentUser!;
    return User(
      id: 'mock_user_1',
      nickname: 'Voice 用户',
      city: '杭州',
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
    );
  }

  @override
  Future<User> updateProfile(Map<String, dynamic> data) async {
    await Future.delayed(const Duration(milliseconds: 300));
    final existing = await getProfile();
    _currentUser = existing.copyWith(
      nickname: data['nickname'] as String?,
      city: data['city'] as String?,
      gender: data['gender'] as String?,
    );
    return _currentUser!;
  }

  @override
  Future<Preferences> getPreferences() async {
    await Future.delayed(const Duration(milliseconds: 200));
    return _prefs ?? const Preferences();
  }

  @override
  Future<void> updatePreferences(Preferences prefs) async {
    await Future.delayed(const Duration(milliseconds: 200));
    _prefs = prefs;
  }

  @override
  Future<List<CallRecord>> getCallRecords(
      {int page = 1, int pageSize = 20}) async {
    await Future.delayed(const Duration(milliseconds: 300));
    return [];
  }

  @override
  Future<void> submitReport(
      String reportedId, String reason, String content) async {
    await Future.delayed(const Duration(milliseconds: 300));
    debugPrint('已举报 $reportedId: $reason - $content');
  }

  @override
  Future<void> blockUser(String blockedUserId) async {
    await Future.delayed(const Duration(milliseconds: 200));
    debugPrint('已拉黑 $blockedUserId');
  }

  @override
  Future<void> unblockUser(String blockedUserId) async {
    await Future.delayed(const Duration(milliseconds: 200));
    debugPrint('已取消拉黑 $blockedUserId');
  }

  @override
  Future<List<BlockedUser>> getBlacklist() async {
    await Future.delayed(const Duration(milliseconds: 200));
    return [];
  }
}

class _MockWebSocketService implements WebSocketService {
  final _controller = StreamController<Map<String, dynamic>>.broadcast();
  bool _connected = false;

  @override
  Future<void> connect(String token) async {
    await Future.delayed(const Duration(milliseconds: 300));
    _connected = true;
    debugPrint('WebSocket 已连接');
  }

  @override
  void disconnect() {
    _connected = false;
    debugPrint('WebSocket 已断开');
  }

  @override
  void send(Map<String, dynamic> message) {
    debugPrint('WebSocket 发送: $message');
  }

  @override
  Stream<Map<String, dynamic>> get messages => _controller.stream;

  @override
  bool get isConnected => _connected;
}

class _MockRtcService implements RtcService {
  final _controller = StreamController<RtcEvent>.broadcast();

  @override
  Future<void> joinRoom(String token, String roomName) async {
    await Future.delayed(const Duration(milliseconds: 800));
    _controller.add(RtcEvent.connected());
  }

  @override
  Future<void> leaveRoom() async {
    _controller.add(RtcEvent.disconnected());
  }

  @override
  void setMuted(bool muted) {
    debugPrint('静音: $muted');
  }

  @override
  void setSpeakerOn(bool on) {
    debugPrint('免提: $on');
  }

  @override
  Stream<RtcEvent> get events => _controller.stream;
}

class _MockCallKitService implements CallKitService {
  final _controller = StreamController<CallKitEvent>.broadcast();

  @override
  Future<void> setup() async {
    await Future.delayed(const Duration(milliseconds: 200));
    debugPrint('CallKit 已初始化');
  }

  @override
  Future<void> showIncomingCall(
      String callId, String callerName, String callerAvatar) async {
    debugPrint('显示来电: $callerName');
  }

  @override
  Future<void> endCall(String callId) async {
    debugPrint('结束通话: $callId');
  }

  @override
  Stream<CallKitEvent> get events => _controller.stream;
}

class _MockMatchService implements MatchService {
  final _controller = StreamController<MatchEvent>.broadcast();
  bool _inPool = false;

  @override
  Future<void> joinPool({
    required String matchType,
    String? city,
    String? geohash,
    String? destination,
    List<String>? tags,
  }) async {
    _inPool = true;
    await Future.delayed(const Duration(seconds: 3));
    if (_inPool) {
      // 模拟匹配成功
      final mockUser = User(
        id: 'peer_${DateTime.now().millisecondsSinceEpoch}',
        nickname: '神秘听众',
        city: city ?? '杭州',
        createdAt: DateTime.now(),
        updatedAt: DateTime.now(),
      );
      _controller.add(MatchEvent.matched(
        mockUser,
        'room_${DateTime.now().millisecondsSinceEpoch}',
        'mock_rtc_token',
        'call_${DateTime.now().millisecondsSinceEpoch}',
      ));
    }
  }

  @override
  Future<void> leavePool() async {
    _inPool = false;
    debugPrint('已离开匹配池');
  }

  @override
  Stream<MatchEvent> get events => _controller.stream;
}
