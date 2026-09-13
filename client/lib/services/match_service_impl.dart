import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';

import '../models/user.dart';
import 'match_service.dart';
import 'websocket_service.dart';
import 'package:voice_match_shared/protocol/messages.dart' as shared;

/// Concrete implementation of [MatchService].
///
/// Sends `join_pool` / `leave_pool` signaling messages through the
/// injected [WebSocketService] and listens for `match_found` / `error`
/// messages on the WebSocket stream.
///
/// A 60-second timeout is enforced: if no match is found, the service
/// automatically sends `leave_pool` and emits [MatchEvent.timeout].
class RealMatchService extends ChangeNotifier implements MatchService {
  RealMatchService({
    required WebSocketService webSocketService,
  }) : _ws = webSocketService;

  final WebSocketService _ws;
  StreamSubscription? _wsSub;

  final _eventsController = StreamController<MatchEvent>.broadcast();

  Timer? _timeoutTimer;
  bool _isInPool = false;

  /// Whether the user is currently waiting in a matching pool.
  bool get isInPool => _isInPool;

  @override
  Stream<MatchEvent> get events => _eventsController.stream;

  // ---------------------------------------------------------------------------
  // Pool lifecycle
  // ---------------------------------------------------------------------------

  @override
  Future<void> joinPool({
    required String matchType,
    String? city,
    String? geohash,
    String? destination,
    List<String>? tags,
  }) async {
    if (_isInPool) {
      debugPrint('joinPool: already in pool, leaving first');
      await leavePool();
    }

    // Subscribe to WebSocket messages if not already
    _wsSub ??= _ws.messages.listen(_onWsMessage);

    final joinMsg = shared.JoinPool(
      matchType: matchType,
      city: city ?? '',
      geohash: geohash ?? '',
      destination: destination ?? '',
      tags: tags ?? const [],
    );

    _ws.send(joinMsg.toJson());
    _isInPool = true;
    notifyListeners();

    // Start 60-second timeout
    _timeoutTimer?.cancel();
    _timeoutTimer = Timer(const Duration(seconds: 60), () {
      if (_isInPool) {
        debugPrint('Match pool timeout (60 s) — leaving pool');
        leavePool();
        _eventsController.add(MatchEvent.timeout());
      }
    });
  }

  @override
  Future<void> leavePool() async {
    if (!_isInPool) return;

    _timeoutTimer?.cancel();
    _timeoutTimer = null;

    final leaveMsg = const shared.LeavePool();
    _ws.send(leaveMsg.toJson());

    _isInPool = false;
    notifyListeners();
    debugPrint('MatchService: left pool');
  }

  // ---------------------------------------------------------------------------
  // WebSocket message handling
  // ---------------------------------------------------------------------------

  void _onWsMessage(Map<String, dynamic> raw) {
    final type = raw['type'] as String? ?? '';

    if (type == shared.MessageType.matchFound) {
      _handleMatchFound(raw);
    } else if (type == shared.MessageType.error) {
      _handleError(raw);
    } else if (type == shared.MessageType.joinPoolAck) {
      final payload = raw['payload'] as Map<String, dynamic>? ?? {};
      final status = payload['status'] as String? ?? '';
      if (status != 'ok') {
        _isInPool = false;
        _timeoutTimer?.cancel();
        _eventsController.add(MatchEvent.error(
            payload['message']?.toString() ?? 'Failed to join pool'));
        notifyListeners();
      }
    }
  }

  void _handleMatchFound(Map<String, dynamic> raw) {
    final payload = raw['payload'] as Map<String, dynamic>? ?? {};

    final callId = payload['call_id'] as String? ?? '';
    final matchedUserId = payload['matched_user_id'] as String? ?? '';
    final matchedNickname = payload['matched_nickname'] as String? ?? '神秘听众';
    final matchedAvatar = payload['matched_avatar'] as String? ?? '';

    // Cancel timeout — we found a match
    _timeoutTimer?.cancel();
    _timeoutTimer = null;
    _isInPool = false;
    notifyListeners();

    // Construct a lightweight User object from match data
    final matchedUser = User(
      id: matchedUserId,
      nickname: matchedNickname,
      avatar: matchedAvatar,
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
    );

    // The server will send a subsequent call_invite with room_name and
    // rtc_token. For now, we emit matched with empty room/token, and the
    // call_invite handler (in the call flow coordinator) will fill them in.
    _eventsController.add(MatchEvent.matched(
      matchedUser,
      '', // roomName — filled by call_invite handler
      '', // rtcToken — filled by call_invite handler
      callId,
    ));

    debugPrint('Match found: $matchedNickname (callId=$callId)');
  }

  void _handleError(Map<String, dynamic> raw) {
    final payload = raw['payload'] as Map<String, dynamic>? ?? {};
    final code = (payload['code'] as num?)?.toInt() ?? 0;
    final message = payload['message'] as String? ?? 'Unknown error';

    // If we're in a pool and get an error, leave the pool
    if (_isInPool) {
      _isInPool = false;
      _timeoutTimer?.cancel();
      notifyListeners();
    }

    _eventsController.add(MatchEvent.error('[$code] $message'));
  }

  @override
  void dispose() {
    _timeoutTimer?.cancel();
    _wsSub?.cancel();
    if (_isInPool) {
      _ws.send(const shared.LeavePool().toJson());
    }
    _eventsController.close();
    super.dispose();
  }
}
