/// RTC 事件类型枚举
enum RtcEventType {
  onConnected,
  onDisconnected,
  onQualityChange,
  onLocalAudioTrackStats,
  onRemoteAudioTrackStats,
}

/// 通话质量等级
enum RtcQuality {
  good,    // 绿色
  fair,    // 黄色
  poor,    // 红色
  unknown,
}

/// RTC 事件数据类
class RtcEvent {
  final RtcEventType type;
  final RtcQuality? quality;
  final dynamic data;

  const RtcEvent({
    required this.type,
    this.quality,
    this.data,
  });

  /// 便捷构造：已连接
  factory RtcEvent.connected() =>
      const RtcEvent(type: RtcEventType.onConnected);

  /// 便捷构造：已断开
  factory RtcEvent.disconnected() =>
      const RtcEvent(type: RtcEventType.onDisconnected);

  /// 便捷构造：质量变化
  factory RtcEvent.qualityChange(RtcQuality quality) =>
      RtcEvent(type: RtcEventType.onQualityChange, quality: quality);
}

/// RTC 语音通话服务抽象接口
/// 基于 LiveKit 实现，由另一个 Agent 实现
abstract class RtcService {
  /// 加入房间
  Future<void> joinRoom(String token, String roomName);

  /// 离开房间
  Future<void> leaveRoom();

  /// 设置静音
  void setMuted(bool muted);

  /// 设置免提
  void setSpeakerOn(bool on);

  /// RTC 事件流
  Stream<RtcEvent> get events;
}
