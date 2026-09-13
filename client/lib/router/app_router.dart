import 'package:flutter/material.dart';

import '../pages/login_page.dart';
import '../pages/home_page.dart';
import '../pages/match_settings_page.dart';
import '../pages/match_waiting_page.dart';
import '../pages/call_page.dart';
import '../pages/end_page.dart';
import '../pages/profile_page.dart';

/// 路由常量
class AppRoutes {
  AppRoutes._();

  static const String login = '/login';
  static const String home = '/home';
  static const String matchSettings = '/match-settings';
  static const String matchWaiting = '/match-waiting';
  static const String call = '/call';
  static const String end = '/end';
  static const String profile = '/profile';
}

/// 路由参数模型
class CallPageArgs {
  final String callId;
  final String roomName;
  final String rtcToken;
  final String peerId;
  final String peerName;
  final String? peerAvatar;
  final bool isCaller; // true = 我方发起, false = 被叫

  const CallPageArgs({
    required this.callId,
    required this.roomName,
    required this.rtcToken,
    required this.peerId,
    required this.peerName,
    this.peerAvatar,
    this.isCaller = true,
  });
}

class EndPageArgs {
  final int durationSeconds;
  final String peerName;
  final String? peerAvatar;

  const EndPageArgs({
    required this.durationSeconds,
    required this.peerName,
    this.peerAvatar,
  });
}

/// 路由生成
class AppRouter {
  AppRouter._();

  static Route<dynamic> onGenerateRoute(RouteSettings settings) {
    switch (settings.name) {
      case AppRoutes.login:
        return MaterialPageRoute(
          settings: settings,
          builder: (_) => const LoginPage(),
        );

      case AppRoutes.home:
        return MaterialPageRoute(
          settings: settings,
          builder: (_) => const HomePage(),
        );

      case AppRoutes.matchSettings:
        return MaterialPageRoute(
          settings: settings,
          builder: (_) => const MatchSettingsPage(),
        );

      case AppRoutes.matchWaiting:
        return MaterialPageRoute(
          settings: settings,
          builder: (_) => const MatchWaitingPage(),
        );

      case AppRoutes.call:
        final args = settings.arguments as CallPageArgs?;
        return MaterialPageRoute(
          settings: settings,
          builder: (_) => CallPage(
            callId: args?.callId ?? '',
            roomName: args?.roomName ?? '',
            rtcToken: args?.rtcToken ?? '',
            peerId: args?.peerId ?? '',
            peerName: args?.peerName ?? '未知用户',
            peerAvatar: args?.peerAvatar,
            isCaller: args?.isCaller ?? true,
          ),
        );

      case AppRoutes.end:
        final args = settings.arguments as EndPageArgs?;
        return MaterialPageRoute(
          settings: settings,
          builder: (_) => EndPage(
            durationSeconds: args?.durationSeconds ?? 0,
            peerName: args?.peerName ?? '未知用户',
            peerAvatar: args?.peerAvatar,
          ),
        );

      case AppRoutes.profile:
        return MaterialPageRoute(
          settings: settings,
          builder: (_) => const ProfilePage(),
        );

      default:
        return MaterialPageRoute(
          builder: (_) => const Scaffold(
            body: Center(child: Text('页面不存在')),
          ),
        );
    }
  }
}
