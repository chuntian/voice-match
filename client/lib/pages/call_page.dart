import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../services/rtc_service.dart';
import '../services/callkit_service.dart';
import '../theme/app_theme.dart';
import '../widgets/common_avatar.dart';
import '../router/app_router.dart';

class CallPage extends StatefulWidget {
  final String callId;
  final String roomName;
  final String rtcToken;
  final String peerId;
  final String peerName;
  final String? peerAvatar;
  final bool isCaller;

  const CallPage({
    super.key,
    required this.callId,
    required this.roomName,
    required this.rtcToken,
    required this.peerId,
    required this.peerName,
    this.peerAvatar,
    this.isCaller = true,
  });

  @override
  State<CallPage> createState() => _CallPageState();
}

class _CallPageState extends State<CallPage> {
  Timer? _callTimer;
  int _secondsElapsed = 0;
  bool _isMuted = false;
  bool _isSpeakerOn = false;
  bool _isConnected = false;
  RtcQuality _quality = RtcQuality.unknown;
  StreamSubscription<RtcEvent>? _rtcSubscription;
  StreamSubscription<CallKitEvent>? _callKitSubscription;

  @override
  void initState() {
    super.initState();
    _setupCall();
  }

  @override
  void dispose() {
    _callTimer?.cancel();
    _rtcSubscription?.cancel();
    _callKitSubscription?.cancel();
    super.dispose();
  }

  Future<void> _setupCall() async {
    final rtcService = Provider.of<RtcService>(context, listen: false);
    final callKitService =
        Provider.of<CallKitService>(context, listen: false);

    // 监听 RTC 事件
    _rtcSubscription = rtcService.events.listen(_onRtcEvent);

    // 监听 CallKit 事件
    _callKitSubscription = callKitService.events.listen(_onCallKitEvent);

    if (widget.isCaller) {
      // 主叫：直接加入房间
      try {
        await rtcService.joinRoom(widget.rtcToken, widget.roomName);
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('加入通话失败: $e')),
          );
        }
      }
    } else {
      // 被叫：等待用户接听后再加入
      // CallKit 会触发 onAccept 事件
    }
  }

  void _onRtcEvent(RtcEvent event) {
    switch (event.type) {
      case RtcEventType.onConnected:
        setState(() => _isConnected = true);
        _startCallTimer();
        break;

      case RtcEventType.onDisconnected:
        _endCall();
        break;

      case RtcEventType.onQualityChange:
        if (event.quality != null) {
          setState(() => _quality = event.quality!);
        }
        break;

      default:
        break;
    }
  }

  void _onCallKitEvent(CallKitEvent event) {
    switch (event.type) {
      case CallKitEventType.onAccept:
        // 被叫接听，加入房间
        _joinRoomAfterAccept();
        break;
      case CallKitEventType.onDecline:
      case CallKitEventType.onEnd:
        _endCall();
        break;
      default:
        break;
    }
  }

  Future<void> _joinRoomAfterAccept() async {
    final rtcService = Provider.of<RtcService>(context, listen: false);
    try {
      await rtcService.joinRoom(widget.rtcToken, widget.roomName);
    } catch (_) {}
  }

  void _startCallTimer() {
    _callTimer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (mounted) {
        setState(() => _secondsElapsed++);
      }
    });
  }

  void _toggleMute() {
    setState(() => _isMuted = !_isMuted);
    final rtcService = Provider.of<RtcService>(context, listen: false);
    rtcService.setMuted(_isMuted);
  }

  void _toggleSpeaker() {
    setState(() => _isSpeakerOn = !_isSpeakerOn);
    final rtcService = Provider.of<RtcService>(context, listen: false);
    rtcService.setSpeakerOn(_isSpeakerOn);
  }

  Future<void> _endCall() async {
    _callTimer?.cancel();

    final rtcService = Provider.of<RtcService>(context, listen: false);
    final callKitService =
        Provider.of<CallKitService>(context, listen: false);

    try {
      await rtcService.leaveRoom();
    } catch (_) {}

    try {
      await callKitService.endCall(widget.callId);
    } catch (_) {}

    if (mounted) {
      Navigator.pushReplacementNamed(
        context,
        AppRoutes.end,
        arguments: EndPageArgs(
          durationSeconds: _secondsElapsed,
          peerName: widget.peerName,
          peerAvatar: widget.peerAvatar,
        ),
      );
    }
  }

  String get _formattedDuration {
    final m = (_secondsElapsed ~/ 60).toString().padLeft(2, '0');
    final s = (_secondsElapsed % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  /// 网络质量指示器
  Widget _buildQualityIndicator() {
    Color color;
    String text;

    switch (_quality) {
      case RtcQuality.good:
        color = AppColors.accentGreen;
        text = '网络良好';
        break;
      case RtcQuality.fair:
        color = AppColors.accentOrange;
        text = '网络一般';
        break;
      case RtcQuality.poor:
        color = AppColors.accentRed;
        text = '网络较差';
        break;
      case RtcQuality.unknown:
        color = Colors.white38;
        text = '连接中...';
        break;
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(
              color: color,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 6),
          Text(
            text,
            style: TextStyle(
              fontSize: 12,
              color: color,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.darkBg,
      body: SafeArea(
        child: Column(
          children: [
            const SizedBox(height: 40),

            // 网络质量
            _buildQualityIndicator(),
            const SizedBox(height: 24),

            // 对方头像
            CommonAvatar.xlarge(
              imageUrl: widget.peerAvatar,
              placeholderText:
                  widget.peerName.isNotEmpty ? widget.peerName.substring(0, 1) : '?',
            ),
            const SizedBox(height: 20),

            // 昵称
            Text(
              widget.peerName,
              style: const TextStyle(
                fontSize: 24,
                fontWeight: FontWeight.bold,
                color: Colors.white,
              ),
            ),
            const SizedBox(height: 8),

            // 通话状态
            Text(
              _isConnected ? _formattedDuration : '正在呼叫...',
              style: const TextStyle(
                fontSize: 16,
                color: Colors.white60,
                fontFeatures: [FontFeature.tabularFigures()],
              ),
            ),

            const Spacer(),

            // 底部操作按钮
            Padding(
              padding: const EdgeInsets.only(bottom: 50),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                children: [
                  // 静音
                  _buildCircleButton(
                    icon: _isMuted ? Icons.mic_off : Icons.mic,
                    color: _isMuted ? Colors.white : AppColors.darkCard,
                    iconColor: _isMuted ? Colors.black : Colors.white,
                    onTap: _toggleMute,
                  ),

                  // 挂断
                  _buildCircleButton(
                    icon: Icons.call_end,
                    color: AppColors.accentRed,
                    iconColor: Colors.white,
                    size: 72,
                    onTap: _endCall,
                  ),

                  // 免提
                  _buildCircleButton(
                    icon: _isSpeakerOn
                        ? Icons.volume_up
                        : Icons.volume_up_outlined,
                    color: _isSpeakerOn ? Colors.white : AppColors.darkCard,
                    iconColor: _isSpeakerOn ? Colors.black : Colors.white,
                    onTap: _toggleSpeaker,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildCircleButton({
    required IconData icon,
    required Color color,
    required Color iconColor,
    required VoidCallback onTap,
    double size = 60,
  }) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        width: size,
        height: size,
        decoration: BoxDecoration(
          color: color,
          shape: BoxShape.circle,
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.2),
              blurRadius: 12,
              offset: const Offset(0, 4),
            ),
          ],
        ),
        child: Icon(
          icon,
          color: iconColor,
          size: size * 0.4,
        ),
      ),
    );
  }
}
