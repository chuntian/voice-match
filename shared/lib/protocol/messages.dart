/// Shared signaling protocol message definitions (Dart side).
///
/// Every message on the wire looks like:
/// ```json
/// {"type": "hello", "payload": {...}}
/// ```
///
/// All classes implement `SignalMessage` and have manual `fromJson` /
/// `toJson` methods (no build_runner needed).
library;

/// Base envelope.
class SignalMessage {
  final String type;
  final Map<String, dynamic>? payload;

  const SignalMessage({required this.type, this.payload});

  /// Factory that dispatches to the correct concrete subclass based on [type].
  factory SignalMessage.fromJson(Map<String, dynamic> json) {
    final type = json['type'] as String? ?? '';
    final payload = json['payload'] as Map<String, dynamic>?;

    switch (type) {
      case 'hello':
        return Hello.fromJson(payload ?? {});
      case 'hello_ack':
        return HelloAck.fromJson(payload ?? {});
      case 'heartbeat':
        return Heartbeat.fromJson(payload ?? {});
      case 'heartbeat_ack':
        return HeartbeatAck.fromJson(payload ?? {});
      case 'join_pool':
        return JoinPool.fromJson(payload ?? {});
      case 'join_pool_ack':
        return JoinPoolAck.fromJson(payload ?? {});
      case 'leave_pool':
        return LeavePool.fromJson(payload ?? {});
      case 'leave_pool_ack':
        return LeavePoolAck.fromJson(payload ?? {});
      case 'match_found':
        return MatchFound.fromJson(payload ?? {});
      case 'call_invite':
        return CallInvite.fromJson(payload ?? {});
      case 'call_accept':
        return CallAccept.fromJson(payload ?? {});
      case 'call_reject':
        return CallReject.fromJson(payload ?? {});
      case 'call_cancel':
        return CallCancel.fromJson(payload ?? {});
      case 'call_end':
        return CallEnd.fromJson(payload ?? {});
      case 'ice_candidate':
        return IceCandidate.fromJson(payload ?? {});
      case 'error':
        return ErrorMsg.fromJson(payload ?? {});
      default:
        return SignalMessage(type: type, payload: payload);
    }
  }

  Map<String, dynamic> toJson() {
    return {
      'type': type,
      if (payload != null) 'payload': payload,
    };
  }
}

// ---------------------------------------------------------------------------
// Message type constants
// ---------------------------------------------------------------------------

class MessageType {
  static const String hello = 'hello';
  static const String helloAck = 'hello_ack';
  static const String heartbeat = 'heartbeat';
  static const String heartbeatAck = 'heartbeat_ack';
  static const String joinPool = 'join_pool';
  static const String joinPoolAck = 'join_pool_ack';
  static const String leavePool = 'leave_pool';
  static const String leavePoolAck = 'leave_pool_ack';
  static const String matchFound = 'match_found';
  static const String callInvite = 'call_invite';
  static const String callAccept = 'call_accept';
  static const String callReject = 'call_reject';
  static const String callCancel = 'call_cancel';
  static const String callEnd = 'call_end';
  static const String iceCandidate = 'ice_candidate';
  static const String error = 'error';
}

// ---------------------------------------------------------------------------
// Concrete payload classes
// ---------------------------------------------------------------------------

class Hello extends SignalMessage {
  final String userId;
  final String token;
  final String deviceId;
  final String appVersion;

  const Hello({
    required this.userId,
    required this.token,
    required this.deviceId,
    required this.appVersion,
  }) : super(type: MessageType.hello);

  factory Hello.fromJson(Map<String, dynamic> json) {
    return Hello(
      userId: json['user_id'] as String? ?? '',
      token: json['token'] as String? ?? '',
      deviceId: json['device_id'] as String? ?? '',
      appVersion: json['app_version'] as String? ?? '',
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'user_id': userId,
          'token': token,
          'device_id': deviceId,
          'app_version': appVersion,
        },
      };
}

class HelloAck extends SignalMessage {
  final String status;
  final int serverTime;

  const HelloAck({required this.status, required this.serverTime})
      : super(type: MessageType.helloAck);

  factory HelloAck.fromJson(Map<String, dynamic> json) {
    return HelloAck(
      status: json['status'] as String? ?? '',
      serverTime: (json['server_time'] as num? ?? 0).toInt(),
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'status': status,
          'server_time': serverTime,
        },
      };
}

class Heartbeat extends SignalMessage {
  final int timestamp;

  const Heartbeat({required this.timestamp})
      : super(type: MessageType.heartbeat);

  factory Heartbeat.fromJson(Map<String, dynamic> json) {
    return Heartbeat(
      timestamp: (json['timestamp'] as num? ?? 0).toInt(),
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {'timestamp': timestamp},
      };
}

class HeartbeatAck extends SignalMessage {
  final int timestamp;
  final int serverTime;

  const HeartbeatAck({required this.timestamp, required this.serverTime})
      : super(type: MessageType.heartbeatAck);

  factory HeartbeatAck.fromJson(Map<String, dynamic> json) {
    return HeartbeatAck(
      timestamp: (json['timestamp'] as num? ?? 0).toInt(),
      serverTime: (json['server_time'] as num? ?? 0).toInt(),
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'timestamp': timestamp,
          'server_time': serverTime,
        },
      };
}

class JoinPool extends SignalMessage {
  final String matchType;
  final String city;
  final String geohash;
  final String destination;
  final List<String> tags;
  final String genderPreference;

  const JoinPool({
    required this.matchType,
    this.city = '',
    this.geohash = '',
    this.destination = '',
    this.tags = const [],
    this.genderPreference = '',
  }) : super(type: MessageType.joinPool);

  factory JoinPool.fromJson(Map<String, dynamic> json) {
    return JoinPool(
      matchType: json['match_type'] as String? ?? '',
      city: json['city'] as String? ?? '',
      geohash: json['geohash'] as String? ?? '',
      destination: json['destination'] as String? ?? '',
      tags: (json['tags'] as List? ?? []).cast<String>(),
      genderPreference: json['gender_preference'] as String? ?? '',
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'match_type': matchType,
          if (city.isNotEmpty) 'city': city,
          if (geohash.isNotEmpty) 'geohash': geohash,
          if (destination.isNotEmpty) 'destination': destination,
          if (tags.isNotEmpty) 'tags': tags,
          if (genderPreference.isNotEmpty) 'gender_preference': genderPreference,
        },
      };
}

class JoinPoolAck extends SignalMessage {
  final String status;
  final int poolSize;
  final int waitTime;

  const JoinPoolAck({
    required this.status,
    required this.poolSize,
    required this.waitTime,
  }) : super(type: MessageType.joinPoolAck);

  factory JoinPoolAck.fromJson(Map<String, dynamic> json) {
    return JoinPoolAck(
      status: json['status'] as String? ?? '',
      poolSize: (json['pool_size'] as num? ?? 0).toInt(),
      waitTime: (json['wait_time'] as num? ?? 0).toInt(),
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'status': status,
          'pool_size': poolSize,
          'wait_time': waitTime,
        },
      };
}

class LeavePool extends SignalMessage {
  const LeavePool() : super(type: MessageType.leavePool);

  factory LeavePool.fromJson(Map<String, dynamic> json) => const LeavePool();

  @override
  Map<String, dynamic> toJson() => {'type': type};
}

class LeavePoolAck extends SignalMessage {
  final String status;

  const LeavePoolAck({required this.status})
      : super(type: MessageType.leavePoolAck);

  factory LeavePoolAck.fromJson(Map<String, dynamic> json) {
    return LeavePoolAck(status: json['status'] as String? ?? '');
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {'status': status},
      };
}

class MatchFound extends SignalMessage {
  final String callId;
  final String matchedUserId;
  final String matchedNickname;
  final String matchedAvatar;
  final String matchType;

  const MatchFound({
    required this.callId,
    required this.matchedUserId,
    required this.matchedNickname,
    required this.matchedAvatar,
    required this.matchType,
  }) : super(type: MessageType.matchFound);

  factory MatchFound.fromJson(Map<String, dynamic> json) {
    return MatchFound(
      callId: json['call_id'] as String? ?? '',
      matchedUserId: json['matched_user_id'] as String? ?? '',
      matchedNickname: json['matched_nickname'] as String? ?? '',
      matchedAvatar: json['matched_avatar'] as String? ?? '',
      matchType: json['match_type'] as String? ?? '',
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'call_id': callId,
          'matched_user_id': matchedUserId,
          'matched_nickname': matchedNickname,
          'matched_avatar': matchedAvatar,
          'match_type': matchType,
        },
      };
}

class CallInvite extends SignalMessage {
  final String callId;
  final String callerId;
  final String callerNickname;
  final String callerAvatar;
  final String roomName;
  final String rtcToken;
  final int expiresAt;

  const CallInvite({
    required this.callId,
    required this.callerId,
    required this.callerNickname,
    required this.callerAvatar,
    required this.roomName,
    required this.rtcToken,
    required this.expiresAt,
  }) : super(type: MessageType.callInvite);

  factory CallInvite.fromJson(Map<String, dynamic> json) {
    return CallInvite(
      callId: json['call_id'] as String? ?? '',
      callerId: json['caller_id'] as String? ?? '',
      callerNickname: json['caller_nickname'] as String? ?? '',
      callerAvatar: json['caller_avatar'] as String? ?? '',
      roomName: json['room_name'] as String? ?? '',
      rtcToken: json['rtc_token'] as String? ?? '',
      expiresAt: (json['expires_at'] as num? ?? 0).toInt(),
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'call_id': callId,
          'caller_id': callerId,
          'caller_nickname': callerNickname,
          'caller_avatar': callerAvatar,
          'room_name': roomName,
          'rtc_token': rtcToken,
          'expires_at': expiresAt,
        },
      };
}

class CallAccept extends SignalMessage {
  final String callId;

  const CallAccept({required this.callId})
      : super(type: MessageType.callAccept);

  factory CallAccept.fromJson(Map<String, dynamic> json) {
    return CallAccept(callId: json['call_id'] as String? ?? '');
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {'call_id': callId},
      };
}

class CallReject extends SignalMessage {
  final String callId;
  final String reason;

  const CallReject({required this.callId, this.reason = ''})
      : super(type: MessageType.callReject);

  factory CallReject.fromJson(Map<String, dynamic> json) {
    return CallReject(
      callId: json['call_id'] as String? ?? '',
      reason: json['reason'] as String? ?? '',
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'call_id': callId,
          if (reason.isNotEmpty) 'reason': reason,
        },
      };
}

class CallCancel extends SignalMessage {
  final String callId;
  final String reason;

  const CallCancel({required this.callId, this.reason = ''})
      : super(type: MessageType.callCancel);

  factory CallCancel.fromJson(Map<String, dynamic> json) {
    return CallCancel(
      callId: json['call_id'] as String? ?? '',
      reason: json['reason'] as String? ?? '',
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'call_id': callId,
          if (reason.isNotEmpty) 'reason': reason,
        },
      };
}

class CallEnd extends SignalMessage {
  final String callId;
  final String reason;
  final int duration;

  const CallEnd({required this.callId, this.reason = '', this.duration = 0})
      : super(type: MessageType.callEnd);

  factory CallEnd.fromJson(Map<String, dynamic> json) {
    return CallEnd(
      callId: json['call_id'] as String? ?? '',
      reason: json['reason'] as String? ?? '',
      duration: (json['duration'] as num? ?? 0).toInt(),
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'call_id': callId,
          if (reason.isNotEmpty) 'reason': reason,
          if (duration > 0) 'duration': duration,
        },
      };
}

class IceCandidate extends SignalMessage {
  final String callId;
  final String targetUserId;
  final String candidate;
  final String sdpMid;
  final int sdpMLineIndex;

  const IceCandidate({
    required this.callId,
    required this.targetUserId,
    required this.candidate,
    this.sdpMid = '',
    this.sdpMLineIndex = 0,
  }) : super(type: MessageType.iceCandidate);

  factory IceCandidate.fromJson(Map<String, dynamic> json) {
    return IceCandidate(
      callId: json['call_id'] as String? ?? '',
      targetUserId: json['target_user_id'] as String? ?? '',
      candidate: json['candidate'] as String? ?? '',
      sdpMid: json['sdp_mid'] as String? ?? '',
      sdpMLineIndex: (json['sdp_m_line_index'] as num? ?? 0).toInt(),
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'call_id': callId,
          'target_user_id': targetUserId,
          'candidate': candidate,
          if (sdpMid.isNotEmpty) 'sdp_mid': sdpMid,
          'sdp_m_line_index': sdpMLineIndex,
        },
      };
}

class ErrorMsg extends SignalMessage {
  final int code;
  final String message;
  final String details;

  const ErrorMsg({
    required this.code,
    required this.message,
    this.details = '',
  }) : super(type: MessageType.error);

  factory ErrorMsg.fromJson(Map<String, dynamic> json) {
    return ErrorMsg(
      code: (json['code'] as num? ?? 0).toInt(),
      message: json['message'] as String? ?? '',
      details: json['details'] as String? ?? '',
    );
  }

  @override
  Map<String, dynamic> toJson() => {
        'type': type,
        'payload': {
          'code': code,
          'message': message,
          if (details.isNotEmpty) 'details': details,
        },
      };
}
