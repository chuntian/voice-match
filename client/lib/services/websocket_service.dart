/// WebSocket 服务抽象接口
/// 负责实时消息通信，由另一个 Agent 实现
abstract class WebSocketService {
  /// 建立连接
  Future<void> connect(String token);

  /// 断开连接
  void disconnect();

  /// 发送消息
  void send(Map<String, dynamic> message);

  /// 接收消息流
  Stream<Map<String, dynamic>> get messages;

  /// 是否已连接
  bool get isConnected;
}
