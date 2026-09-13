import '../models/user.dart';

/// 匹配事件类型枚举
enum MatchEventType {
  onMatched,
  onTimeout,
  onError,
  onPoolStatusChange,
}

/// 匹配事件数据类
class MatchEvent {
  final MatchEventType type;
  final User? matchedUser; // 匹配到的用户信息
  final String? roomName; // 通话房间名
  final String? rtcToken; // RTC 加入令牌
  final String? callId; // 通话 ID
  final String? errorMessage;
  final dynamic data;

  const MatchEvent({
    required this.type,
    this.matchedUser,
    this.roomName,
    this.rtcToken,
    this.callId,
    this.errorMessage,
    this.data,
  });

  factory MatchEvent.matched(
    User user,
    String roomName,
    String rtcToken,
    String callId,
  ) {
    return MatchEvent(
      type: MatchEventType.onMatched,
      matchedUser: user,
      roomName: roomName,
      rtcToken: rtcToken,
      callId: callId,
    );
  }

  factory MatchEvent.timeout() =>
      const MatchEvent(type: MatchEventType.onTimeout);

  factory MatchEvent.error(String message) =>
      MatchEvent(type: MatchEventType.onError, errorMessage: message);
}

/// 匹配服务抽象接口
/// 负责加入/离开匹配池，监听匹配结果，由另一个 Agent 实现
abstract class MatchService {
  /// 加入匹配池
  /// [matchType]: random / city / destination
  /// [city]: 同城匹配时的城市名
  /// [geohash]: 地理位置哈希
  /// [destination]: 目的地匹配时的目的地
  /// [tags]: 兴趣标签
  Future<void> joinPool({
    required String matchType,
    String? city,
    String? geohash,
    String? destination,
    List<String>? tags,
  });

  /// 离开匹配池
  Future<void> leavePool();

  /// 匹配事件流
  Stream<MatchEvent> get events;
}
