import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:web_socket_channel/status.dart' as ws_status;

import 'websocket_service.dart';
import 'package:voice_match_shared/protocol/messages.dart' as shared;

/// Concrete implementation of [WebSocketService].
///
/// Manages the WebSocket connection lifecycle:
/// - Connect → send `hello` → wait for `hello_ack`
/// - Heartbeat every 15 s; if no `heartbeat_ack` within 30 s, reconnect
/// - Exponential backoff reconnection (1 s → 2 s → 4 s → 8 s → … max 30 s,
///   up to 10 attempts)
/// - Broadcasts every decoded inbound message as a `Map<String, dynamic>`
///   via the [messages] stream.
class RealWebSocketService extends ChangeNotifier
    implements WebSocketService {
  RealWebSocketService({
    required this.host,
    this.isSecure = true,
  });

  /// WebSocket host, e.g. `api.voicematch.app`.
  final String host;

  /// Whether to use `wss://` (true) or `ws://` (false).
  final bool isSecure;

  WebSocketChannel? _channel;
  StreamSubscription? _incomingSub;

  final _messagesController =
      StreamController<Map<String, dynamic>>.broadcast();

  Timer? _heartbeatTimer;
  Timer? _heartbeatTimeoutTimer;
  Timer? _reconnectTimer;

  bool _manualDisconnect = false;
  int _reconnectAttempts = 0;
  bool _isConnected = false;

  String? _token;
  String? _userId;
  final String _deviceId = '';
  final String _appVersion = '1.0.0';

  static const Duration _heartbeatInterval = Duration(seconds: 15);
  static const Duration _heartbeatTimeout = Duration(seconds: 30);
  static const int _maxReconnectAttempts = 10;
  static const int _maxBackoffSeconds = 30;

  // ---------------------------------------------------------------------------
  // Public getters
  // ---------------------------------------------------------------------------

  @override
  Stream<Map<String, dynamic>> get messages => _messagesController.stream;

  @override
  bool get isConnected => _isConnected;

  // ---------------------------------------------------------------------------
  // Connection management
  // ---------------------------------------------------------------------------

  @override
  Future<void> connect(String token) async {
    _token = token;
    _manualDisconnect = false;
    await _openConnection();
  }

  Future<void> _openConnection() async {
    if (_manualDisconnect) return;

    final scheme = isSecure ? 'wss' : 'ws';
    final uri = Uri.parse('$scheme://$host/ws');

    try {
      _channel = WebSocketChannel.connect(uri);

      // Wait for the socket to be ready
      await _channel!.ready;
      _isConnected = false;

      // Listen to incoming messages
      _incomingSub = _channel!.stream.listen(
        _onMessage,
        onError: _onError,
        onDone: _onDone,
      );

      // Send hello
      final hello = shared.Hello(
        userId: _userId ?? '',
        token: _token ?? '',
        deviceId: _deviceId,
        appVersion: _appVersion,
      );
      _channel!.sink.add(jsonEncode(hello.toJson()));
    } catch (e) {
      debugPrint('WebSocket connection error: $e');
      _scheduleReconnect();
    }
  }

  void _onMessage(dynamic raw) {
    Map<String, dynamic> decoded;
    try {
      decoded = jsonDecode(raw as String) as Map<String, dynamic>;
    } catch (e) {
      debugPrint('WebSocket decode error: $e');
      return;
    }

    final type = decoded['type'] as String? ?? '';

    // Handle hello_ack
    if (type == shared.MessageType.helloAck) {
      _isConnected = true;
      _reconnectAttempts = 0;
      _startHeartbeat();
      notifyListeners();
      debugPrint('WebSocket connected (hello_ack received)');
    }

    // Reset heartbeat timeout on heartbeat_ack
    if (type == shared.MessageType.heartbeatAck) {
      _heartbeatTimeoutTimer?.cancel();
    }

    // Broadcast to subscribers
    _messagesController.add(decoded);
  }

  void _onError(Object error) {
    debugPrint('WebSocket stream error: $error');
    _handleDisconnect();
  }

  void _onDone() {
    debugPrint('WebSocket stream closed');
    _handleDisconnect();
  }

  void _handleDisconnect() {
    _isConnected = false;
    _stopHeartbeat();
    notifyListeners();
    if (!_manualDisconnect) {
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    if (_manualDisconnect) return;
    if (_reconnectAttempts >= _maxReconnectAttempts) {
      debugPrint('Max reconnect attempts reached ($_maxReconnectAttempts)');
      return;
    }

    final delaySeconds =
        (1 << _reconnectAttempts).clamp(1, _maxBackoffSeconds);
    _reconnectAttempts++;

    debugPrint(
        'Reconnecting in ${delaySeconds}s (attempt $_reconnectAttempts/$_maxReconnectAttempts)');

    _reconnectTimer = Timer(Duration(seconds: delaySeconds), () {
      _openConnection();
    });
  }

  // ---------------------------------------------------------------------------
  // Heartbeat
  // ---------------------------------------------------------------------------

  void _startHeartbeat() {
    _stopHeartbeat();
    _heartbeatTimer = Timer.periodic(_heartbeatInterval, (_) {
      final heartbeat = shared.Heartbeat(
        timestamp: DateTime.now().millisecondsSinceEpoch,
      );
      send(heartbeat.toJson());

      // Arm timeout: if no ack within 30 s, reconnect
      _heartbeatTimeoutTimer?.cancel();
      _heartbeatTimeoutTimer = Timer(_heartbeatTimeout, () {
        debugPrint('Heartbeat timeout — closing and reconnecting');
        _channel?.sink.close(ws_status.goingAway);
        _handleDisconnect();
      });
    });
  }

  void _stopHeartbeat() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = null;
    _heartbeatTimeoutTimer?.cancel();
    _heartbeatTimeoutTimer = null;
  }

  // ---------------------------------------------------------------------------
  // Sending
  // ---------------------------------------------------------------------------

  @override
  void send(Map<String, dynamic> message) {
    if (_channel == null || !_isConnected) {
      debugPrint('WebSocket send skipped: not connected');
      return;
    }
    try {
      _channel!.sink.add(jsonEncode(message));
    } catch (e) {
      debugPrint('WebSocket send error: $e');
    }
  }

  // ---------------------------------------------------------------------------
  // Disconnect
  // ---------------------------------------------------------------------------

  @override
  void disconnect() {
    _manualDisconnect = true;
    _reconnectTimer?.cancel();
    _stopHeartbeat();
    _incomingSub?.cancel();
    _channel?.sink.close(ws_status.normalClosure);
    _channel = null;
    _isConnected = false;
    notifyListeners();
    debugPrint('WebSocket disconnected manually');
  }

  @override
  void dispose() {
    disconnect();
    _messagesController.close();
    super.dispose();
  }
}
