/// CallKit 事件类型枚举
enum CallKitEventType {
  onAccept,
  onDecline,
  onEnd,
  onTimeout,
}

/// CallKit 事件数据类
class CallKitEvent {
  final CallKitEventType type;
  final String? callId;
  final dynamic data;

  const CallKitEvent({
    required this.type,
    this.callId,
    this.data,
  });

  factory CallKitEvent.accept(String callId) =>
      CallKitEvent(type: CallKitEventType.onAccept, callId: callId);

  factory CallKitEvent.decline(String callId) =>
      CallKitEvent(type: CallKitEventType.onDecline, callId: callId);

  factory CallKitEvent.end(String callId) =>
      CallKitEvent(type: CallKitEventType.onEnd, callId: callId);
}

/// CallKit 系统级通话服务抽象接口
/// 用于系统来电界面、接听/挂断事件，由另一个 Agent 实现
abstract class CallKitService {
  /// 初始化 CallKit
  Future<void> setup();

  /// 显示来电界面
  Future<void> showIncomingCall(
      String callId, String callerName, String callerAvatar);

  /// 结束通话（系统级）
  Future<void> endCall(String callId);

  /// CallKit 事件流
  Stream<CallKitEvent> get events;
}
