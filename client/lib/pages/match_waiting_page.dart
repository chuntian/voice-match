import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../services/match_service.dart';
import '../theme/app_theme.dart';
import '../widgets/common_avatar.dart';
import '../router/app_router.dart';

class MatchWaitingPage extends StatefulWidget {
  const MatchWaitingPage({super.key});

  @override
  State<MatchWaitingPage> createState() => _MatchWaitingPageState();
}

class _MatchWaitingPageState extends State<MatchWaitingPage>
    with TickerProviderStateMixin {
  Timer? _countdownTimer;
  Timer? _matchTimeoutTimer;
  int _secondsElapsed = 0;
  StreamSubscription<MatchEvent>? _matchSubscription;

  late AnimationController _rotateController;
  late AnimationController _pulseController;
  late Animation<double> _rotateAnimation;
  late Animation<double> _pulseAnimation;

  bool _matched = false;
  String? _matchedName;
  String? _matchedAvatar;

  @override
  void initState() {
    super.initState();

    _rotateController = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 3),
    )..repeat();

    _pulseController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1500),
    )..repeat(reverse: true);

    _rotateAnimation = Tween<double>(begin: 0, end: 1).animate(
      CurvedAnimation(parent: _rotateController, curve: Curves.linear),
    );

    _pulseAnimation = Tween<double>(begin: 0.8, end: 1.0).animate(
      CurvedAnimation(parent: _pulseController, curve: Curves.easeInOut),
    );

    // 开始计时
    _countdownTimer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (mounted) {
        setState(() => _secondsElapsed++);
      }
    });

    // 60秒超时
    _matchTimeoutTimer = Timer(const Duration(seconds: 60), () {
      if (mounted && !_matched) {
        _onTimeout();
      }
    });

    // 监听匹配事件
    final matchService = Provider.of<MatchService>(context, listen: false);
    _matchSubscription = matchService.events.listen(_onMatchEvent);
  }

  @override
  void dispose() {
    _countdownTimer?.cancel();
    _matchTimeoutTimer?.cancel();
    _matchSubscription?.cancel();
    _rotateController.dispose();
    _pulseController.dispose();
    super.dispose();
  }

  void _onMatchEvent(MatchEvent event) {
    switch (event.type) {
      case MatchEventType.onMatched:
        setState(() {
          _matched = true;
          _matchedName = event.matchedUser?.nickname ?? '未知用户';
          _matchedAvatar = event.matchedUser?.avatar;
        });
        _countdownTimer?.cancel();
        _matchTimeoutTimer?.cancel();

        // 延迟跳转，让用户看到匹配结果
        Future.delayed(const Duration(milliseconds: 1500), () {
          if (mounted) {
            Navigator.pushReplacementNamed(
              context,
              AppRoutes.call,
              arguments: CallPageArgs(
                callId: event.callId ?? '',
                roomName: event.roomName ?? '',
                rtcToken: event.rtcToken ?? '',
                peerId: event.matchedUser?.id ?? '',
                peerName: event.matchedUser?.nickname ?? '未知用户',
                peerAvatar: event.matchedUser?.avatar,
                isCaller: true,
              ),
            );
          }
        });
        break;

      case MatchEventType.onTimeout:
        _onTimeout();
        break;

      case MatchEventType.onError:
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(event.errorMessage ?? '匹配出错')),
          );
          Navigator.pop(context);
        }
        break;

      default:
        break;
    }
  }

  void _onTimeout() {
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('匹配超时，请重试')),
    );
    Navigator.pop(context);
  }

  Future<void> _cancelMatching() async {
    try {
      final matchService =
          Provider.of<MatchService>(context, listen: false);
      await matchService.leavePool();
    } catch (_) {
      // 忽略错误，直接返回
    }
    if (mounted) {
      Navigator.pop(context);
    }
  }

  String get _formattedTime {
    final m = (_secondsElapsed ~/ 60).toString().padLeft(2, '0');
    final s = (_secondsElapsed % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.darkBg,
      body: SafeArea(
        child: Column(
          children: [
            const Spacer(),

            // 旋转动画 + 脉冲
            AnimatedBuilder(
              animation: Listenable.merge([_rotateAnimation, _pulseAnimation]),
              builder: (context, child) {
                return Transform.scale(
                  scale: _pulseAnimation.value,
                  child: SizedBox(
                    width: 160,
                    height: 160,
                    child: Stack(
                      alignment: Alignment.center,
                      children: [
                        // 旋转圆环
                        Transform.rotate(
                          angle: _rotateAnimation.value * 6.283,
                          child: Container(
                            width: 160,
                            height: 160,
                            decoration: BoxDecoration(
                              shape: BoxShape.circle,
                              border: Border.all(
                                color: AppColors.primaryBlue.withValues(alpha: 0.3),
                                width: 2,
                              ),
                            ),
                          ),
                        ),
                        // 中心图标
                        Container(
                          width: 100,
                          height: 100,
                          decoration: BoxDecoration(
                            shape: BoxShape.circle,
                            gradient: const LinearGradient(
                              colors: [
                                AppColors.primaryBlue,
                                AppColors.primaryBlueLight,
                              ],
                            ),
                            boxShadow: [
                              BoxShadow(
                                color: AppColors.primaryBlue.withValues(alpha: 0.4),
                                blurRadius: 20,
                              ),
                            ],
                          ),
                          child: Icon(
                            _matched ? Icons.person : Icons.graphic_eq,
                            size: 44,
                            color: Colors.white,
                          ),
                        ),
                      ],
                    ),
                  ),
                );
              },
            ),
            const SizedBox(height: 48),

            // 状态文字
            Text(
              _matched ? '匹配成功！' : '正在为你匹配...',
              style: const TextStyle(
                fontSize: 24,
                fontWeight: FontWeight.bold,
                color: Colors.white,
              ),
            ),
            const SizedBox(height: 12),

            // 计时器
            Text(
              _formattedTime,
              style: const TextStyle(
                fontSize: 16,
                color: Colors.white60,
                fontFeatures: [FontFeature.tabularFigures()],
              ),
            ),
            const SizedBox(height: 40),

            // 匹配对方信息预览
            if (_matched)
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  CommonAvatar(
                    imageUrl: _matchedAvatar,
                    placeholderText: _matchedName?.substring(0, 1) ?? '?',
                    size: 56,
                  ),
                  const SizedBox(width: 16),
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        _matchedName ?? '',
                        style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w600,
                          color: Colors.white,
                        ),
                      ),
                      const SizedBox(height: 4),
                      const Text(
                        '正在接通通话...',
                        style: TextStyle(
                          fontSize: 13,
                          color: Colors.white60,
                        ),
                      ),
                    ],
                  ),
                ],
              )
            else
              const Text(
                '请耐心等待，我们正在为你寻找合适的通话对象',
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 14,
                  color: Colors.white54,
                ),
              ),

            const Spacer(),

            // 取消按钮
            Padding(
              padding: const EdgeInsets.only(bottom: 40),
              child: SizedBox(
                width: 160,
                height: 50,
                child: OutlinedButton(
                  onPressed: _matched ? null : _cancelMatching,
                  style: OutlinedButton.styleFrom(
                    foregroundColor: Colors.white,
                    side: const BorderSide(color: Colors.white24),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(25),
                    ),
                  ),
                  child: const Text(
                    '取消匹配',
                    style: TextStyle(
                      fontSize: 16,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
