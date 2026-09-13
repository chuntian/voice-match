import 'user.dart';

class CallRecord {
  final String id;
  final String callerId;
  final String calleeId;
  final DateTime startTime;
  final DateTime? endTime;
  final int? duration; // seconds
  final String? endReason;
  final DateTime createdAt;
  final User? peer; // 对方用户信息（可选，用于列表展示）

  const CallRecord({
    required this.id,
    required this.callerId,
    required this.calleeId,
    required this.startTime,
    this.endTime,
    this.duration,
    this.endReason,
    required this.createdAt,
    this.peer,
  });

  factory CallRecord.fromJson(Map<String, dynamic> json) {
    return CallRecord(
      id: json['id'] as String,
      callerId: json['callerId'] as String,
      calleeId: json['calleeId'] as String,
      startTime: DateTime.parse(json['startTime'] as String),
      endTime: json['endTime'] != null
          ? DateTime.parse(json['endTime'] as String)
          : null,
      duration: json['duration'] as int?,
      endReason: json['endReason'] as String?,
      createdAt: json['createdAt'] != null
          ? DateTime.parse(json['createdAt'] as String)
          : DateTime.now(),
      peer: json['peer'] != null
          ? User.fromJson(json['peer'] as Map<String, dynamic>)
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'callerId': callerId,
      'calleeId': calleeId,
      'startTime': startTime.toIso8601String(),
      'endTime': endTime?.toIso8601String(),
      'duration': duration,
      'endReason': endReason,
      'createdAt': createdAt.toIso8601String(),
      'peer': peer?.toJson(),
    };
  }

  /// 格式化通话时长为 mm:ss
  String get formattedDuration {
    if (duration == null) return '--:--';
    final minutes = duration! ~/ 60;
    final seconds = duration! % 60;
    return '${minutes.toString().padLeft(2, '0')}:${seconds.toString().padLeft(2, '0')}';
  }

  /// 格式化开始时间
  String get formattedStartTime {
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final recordDate = DateTime(startTime.year, startTime.month, startTime.day);
    final diffDays = today.difference(recordDate).inDays;

    final hh = startTime.hour.toString().padLeft(2, '0');
    final mm = startTime.minute.toString().padLeft(2, '0');

    if (diffDays == 0) {
      return '今天 $hh:$mm';
    } else if (diffDays == 1) {
      return '昨天 $hh:$mm';
    } else {
      return '${startTime.month}月${startTime.day}日 $hh:$mm';
    }
  }
}
