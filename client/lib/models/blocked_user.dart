import 'user.dart';

class BlockedUser {
  final String userId;
  final String blockedUserId;
  final DateTime createdAt;
  final User? user; // 被拉黑用户信息

  const BlockedUser({
    required this.userId,
    required this.blockedUserId,
    required this.createdAt,
    this.user,
  });

  factory BlockedUser.fromJson(Map<String, dynamic> json) {
    return BlockedUser(
      userId: json['userId'] as String,
      blockedUserId: json['blockedUserId'] as String,
      createdAt: json['createdAt'] != null
          ? DateTime.parse(json['createdAt'] as String)
          : DateTime.now(),
      user: json['user'] != null
          ? User.fromJson(json['user'] as Map<String, dynamic>)
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'userId': userId,
      'blockedUserId': blockedUserId,
      'createdAt': createdAt.toIso8601String(),
      'user': user?.toJson(),
    };
  }
}
