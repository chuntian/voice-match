import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'services/api_service.dart';
import 'services/websocket_service.dart';
import 'services/rtc_service.dart';
import 'services/callkit_service.dart';
import 'services/match_service.dart';
import 'services/service_injector.dart';
import 'theme/app_theme.dart';
import 'router/app_router.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // 初始化本地存储，检查登录状态
  final prefs = await SharedPreferences.getInstance();
  final hasToken = prefs.getString('user_id') != null;

  // 初始化 CallKit
  await ServiceInjector.currentCallKitService.setup();

  runApp(VoiceMatchApp(
    initialRoute: hasToken ? AppRoutes.home : AppRoutes.login,
  ));
}

class VoiceMatchApp extends StatelessWidget {
  final String initialRoute;

  const VoiceMatchApp({
    super.key,
    required this.initialRoute,
  });

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        Provider<ApiService>.value(value: ServiceInjector.currentApiService),
        Provider<WebSocketService>.value(
            value: ServiceInjector.currentWsService),
        Provider<RtcService>.value(value: ServiceInjector.currentRtcService),
        Provider<CallKitService>.value(
            value: ServiceInjector.currentCallKitService),
        Provider<MatchService>.value(
            value: ServiceInjector.currentMatchService),
      ],
      child: MaterialApp(
        title: 'VoiceMatch',
        debugShowCheckedModeBanner: false,
        theme: AppTheme.lightTheme,
        darkTheme: AppTheme.darkTheme,
        themeMode: ThemeMode.system,
        initialRoute: initialRoute,
        onGenerateRoute: AppRouter.onGenerateRoute,
      ),
    );
  }
}
