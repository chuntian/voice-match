import 'package:flutter/material.dart';

import '../theme/app_theme.dart';
import '../widgets/common_button.dart';
import '../widgets/common_avatar.dart';
import '../router/app_router.dart';

class EndPage extends StatefulWidget {
  final int durationSeconds;
  final String peerName;
  final String? peerAvatar;

  const EndPage({
    super.key,
    required this.durationSeconds,
    required this.peerName,
    this.peerAvatar,
  });

  @override
  State<EndPage> createState() => _EndPageState();
}

class _EndPageState extends State<EndPage> {
  int _rating = 0;
  bool _hasSentFriendRequest = false;

  String get _formattedDuration {
    final m = (widget.durationSeconds ~/ 60).toString().padLeft(2, '0');
    final s = (widget.durationSeconds % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  void _sendFriendRequest() {
    setState(() => _hasSentFriendRequest = true);
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('好友请求已发送'),
        duration: Duration(seconds: 2),
      ),
    );
  }

  void _rateCall(int stars) {
    setState(() => _rating = stars);
  }

  void _rematch() {
    Navigator.pushReplacementNamed(context, AppRoutes.matchSettings);
  }

  void _goHome() {
    Navigator.popUntil(context, ModalRoute.withName(AppRoutes.home));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.darkBg,
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            children: [
              const Spacer(),

              // 通话结束图标
              Container(
                width: 80,
                height: 80,
                decoration: BoxDecoration(
                  color: AppColors.accentRed.withValues(alpha: 0.15),
                  shape: BoxShape.circle,
                ),
                child: const Icon(
                  Icons.call_end,
                  color: AppColors.accentRed,
                  size: 36,
                ),
              ),
              const SizedBox(height: 24),

              const Text(
                '通话已结束',
                style: TextStyle(
                  fontSize: 22,
                  fontWeight: FontWeight.w600,
                  color: Colors.white,
                ),
              ),
              const SizedBox(height: 16),

              // 通话时长
              Text(
                _formattedDuration,
                style: const TextStyle(
                  fontSize: 48,
                  fontWeight: FontWeight.bold,
                  color: Colors.white,
                  letterSpacing: -1,
                  fontFeatures: [FontFeature.tabularFigures()],
                ),
              ),
              const SizedBox(height: 8),

              // 对方信息
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  CommonAvatar.small(
                    imageUrl: widget.peerAvatar,
                    placeholderText: widget.peerName.isNotEmpty
                        ? widget.peerName.substring(0, 1)
                        : '?',
                  ),
                  const SizedBox(width: 8),
                  Text(
                    '与 ${widget.peerName} 的通话',
                    style: const TextStyle(
                      fontSize: 14,
                      color: Colors.white60,
                    ),
                  ),
                ],
              ),

              const SizedBox(height: 32),

              // 通话质量评分
              const Text(
                '本次通话体验如何？',
                style: TextStyle(
                  fontSize: 14,
                  color: Colors.white60,
                ),
              ),
              const SizedBox(height: 12),
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: List.generate(5, (index) {
                  final starIndex = index + 1;
                  return IconButton(
                    onPressed: () => _rateCall(starIndex),
                    icon: Icon(
                      starIndex <= _rating ? Icons.star : Icons.star_border,
                      color: starIndex <= _rating
                          ? AppColors.accentOrange
                          : Colors.white24,
                      size: 32,
                    ),
                  );
                }),
              ),

              const SizedBox(height: 24),

              // 添加好友按钮
              GestureDetector(
                onTap: _hasSentFriendRequest ? null : _sendFriendRequest,
                child: Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
                  decoration: BoxDecoration(
                    color: _hasSentFriendRequest
                        ? Colors.white12
                        : AppColors.primaryBlue.withValues(alpha: 0.2),
                    borderRadius: BorderRadius.circular(20),
                    border: Border.all(
                      color: _hasSentFriendRequest
                          ? Colors.white24
                          : AppColors.primaryBlue.withValues(alpha: 0.5),
                    ),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        _hasSentFriendRequest
                            ? Icons.check_circle
                            : Icons.person_add_outlined,
                        color: _hasSentFriendRequest
                            ? Colors.white38
                            : AppColors.primaryBlueLight,
                        size: 18,
                      ),
                      const SizedBox(width: 8),
                      Text(
                        _hasSentFriendRequest ? '已发送请求' : '添加好友',
                        style: TextStyle(
                          fontSize: 14,
                          color: _hasSentFriendRequest
                              ? Colors.white38
                              : AppColors.primaryBlueLight,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ],
                  ),
                ),
              ),

              const Spacer(),

              // 按钮组
              PrimaryButton(
                text: '再匹配一个',
                icon: Icons.refresh,
                onPressed: _rematch,
              ),
              const SizedBox(height: 12),
              SecondaryButton(
                text: '返回首页',
                onPressed: _goHome,
              ),
              const SizedBox(height: 20),
            ],
          ),
        ),
      ),
    );
  }
}
