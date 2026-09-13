class User {
  final String id;
  final String? appleId;
  final String? phone;
  final String nickname;
  final String? avatar;
  final String? gender;
  final String? city;
  final String? geohash;
  final DateTime createdAt;
  final DateTime updatedAt;

  const User({
    required this.id,
    this.appleId,
    this.phone,
    required this.nickname,
    this.avatar,
    this.gender,
    this.city,
    this.geohash,
    required this.createdAt,
    required this.updatedAt,
  });

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      id: json['id'] as String,
      appleId: json['appleId'] as String?,
      phone: json['phone'] as String?,
      nickname: json['nickname'] as String? ?? '未知用户',
      avatar: json['avatar'] as String?,
      gender: json['gender'] as String?,
      city: json['city'] as String?,
      geohash: json['geohash'] as String?,
      createdAt: json['createdAt'] != null
          ? DateTime.parse(json['createdAt'] as String)
          : DateTime.now(),
      updatedAt: json['updatedAt'] != null
          ? DateTime.parse(json['updatedAt'] as String)
          : DateTime.now(),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'appleId': appleId,
      'phone': phone,
      'nickname': nickname,
      'avatar': avatar,
      'gender': gender,
      'city': city,
      'geohash': geohash,
      'createdAt': createdAt.toIso8601String(),
      'updatedAt': updatedAt.toIso8601String(),
    };
  }

  User copyWith({
    String? id,
    String? appleId,
    String? phone,
    String? nickname,
    String? avatar,
    String? gender,
    String? city,
    String? geohash,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return User(
      id: id ?? this.id,
      appleId: appleId ?? this.appleId,
      phone: phone ?? this.phone,
      nickname: nickname ?? this.nickname,
      avatar: avatar ?? this.avatar,
      gender: gender ?? this.gender,
      city: city ?? this.city,
      geohash: geohash ?? this.geohash,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  /// 获取昵称首字符作为占位头像
  String get initialChar {
    if (nickname.isEmpty) return '?';
    return nickname.substring(0, 1).toUpperCase();
  }
}
