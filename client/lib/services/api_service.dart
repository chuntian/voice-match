import '../models/user.dart';
import '../models/preferences.dart';
import '../models/call_record.dart';
import '../models/blocked_user.dart';

/// API 服务抽象接口
/// 由另一个 Agent 实现具体的 HTTP 请求逻辑
abstract class ApiService {
  /// Apple 登录
  Future<User> loginWithApple(String identityToken);

  /// 手机号验证码登录
  Future<User> loginWithPhone(String phone, String code);

  /// 发送手机验证码
  Future<void> sendPhoneCode(String phone);

  /// 获取当前用户资料
  Future<User> getProfile();

  /// 更新用户资料
  Future<User> updateProfile(Map<String, dynamic> data);

  /// 获取匹配偏好
  Future<Preferences> getPreferences();

  /// 更新匹配偏好
  Future<void> updatePreferences(Preferences prefs);

  /// 获取通话记录
  Future<List<CallRecord>> getCallRecords({int page = 1, int pageSize = 20});

  /// 举报用户
  Future<void> submitReport(String reportedId, String reason, String content);

  /// 拉黑用户
  Future<void> blockUser(String blockedUserId);

  /// 取消拉黑
  Future<void> unblockUser(String blockedUserId);

  /// 获取黑名单列表
  Future<List<BlockedUser>> getBlacklist();
}
