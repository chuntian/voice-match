import 'package:flutter/material.dart';
import 'package:flutter/cupertino.dart';

/// App 主题常量
class AppColors {
  AppColors._();

  // 主色调 - 深蓝渐变
  static const Color primaryStart = Color(0xFF1A1A2E);
  static const Color primaryEnd = Color(0xFF16213E);
  static const Color primaryBlue = Color(0xFF4A6CF7);
  static const Color primaryBlueLight = Color(0xFF6B8AFF);

  // 强调色
  static const Color accentBlue = Color(0xFF007AFF); // iOS 系统蓝
  static const Color accentGreen = Color(0xFF34C759);
  static const Color accentRed = Color(0xFFFF3B30);
  static const Color accentOrange = Color(0xFFFF9500);

  // 深色背景（通话页等）
  static const Color darkBg = Color(0xFF0D0D1A);
  static const Color darkSurface = Color(0xFF1C1C2E);
  static const Color darkCard = Color(0xFF2A2A3E);

  // 文字
  static const Color textPrimary = Color(0xFF1C1C1E);
  static const Color textSecondary = Color(0xFF8E8E93);
  static const Color textLight = Color(0xFFFFFFFF);
  static const Color textOnDark = Color(0xFFEBEBF5);

  // 分割线
  static const Color divider = Color(0xFFE5E5EA);

  // 背景
  static const Color bgLight = Color(0xFFF2F2F7);
  static const Color bgWhite = Color(0xFFFFFFFF);
}

/// App 主题数据
class AppTheme {
  AppTheme._();

  /// 亮色主题
  static ThemeData get lightTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.light,
      primaryColor: AppColors.primaryBlue,
      scaffoldBackgroundColor: AppColors.bgLight,
      colorScheme: const ColorScheme.light(
        primary: AppColors.primaryBlue,
        secondary: AppColors.accentBlue,
        surface: AppColors.bgWhite,
        onPrimary: Colors.white,
        onSecondary: Colors.white,
        onSurface: AppColors.textPrimary,
        error: AppColors.accentRed,
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: AppColors.bgWhite,
        foregroundColor: AppColors.textPrimary,
        elevation: 0,
        centerTitle: true,
        titleTextStyle: TextStyle(
          fontSize: 17,
          fontWeight: FontWeight.w600,
          color: AppColors.textPrimary,
        ),
      ),
      cardTheme: CardThemeData(
        color: AppColors.bgWhite,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: Colors.grey.shade50,
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 14,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: AppColors.divider),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: AppColors.divider),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: AppColors.primaryBlue, width: 1.5),
        ),
        hintStyle: TextStyle(
          color: Colors.grey.shade400,
          fontSize: 15,
        ),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: AppColors.primaryBlue,
          foregroundColor: Colors.white,
          minimumSize: const Size(double.infinity, 50),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(25),
          ),
          textStyle: const TextStyle(
            fontSize: 16,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: AppColors.primaryBlue,
          textStyle: const TextStyle(
            fontSize: 15,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      dividerTheme: const DividerThemeData(
        color: AppColors.divider,
        thickness: 0.5,
        space: 0.5,
      ),
    );
  }

  /// 深色主题（通话页等使用）
  static ThemeData get darkTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      primaryColor: AppColors.primaryBlue,
      scaffoldBackgroundColor: AppColors.darkBg,
      colorScheme: const ColorScheme.dark(
        primary: AppColors.primaryBlue,
        secondary: AppColors.accentBlue,
        surface: AppColors.darkSurface,
        onPrimary: Colors.white,
        onSecondary: Colors.white,
        onSurface: AppColors.textOnDark,
        error: AppColors.accentRed,
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: Colors.transparent,
        foregroundColor: AppColors.textOnDark,
        elevation: 0,
        centerTitle: true,
      ),
    );
  }

  /// 文字样式集合
  static const TextStyles text = TextStyles._();
}

class TextStyles {
  const TextStyles._();

  static const TextStyle largeTitle = TextStyle(
    fontSize: 34,
    fontWeight: FontWeight.bold,
    color: AppColors.textPrimary,
    letterSpacing: -0.5,
  );

  static const TextStyle title = TextStyle(
    fontSize: 22,
    fontWeight: FontWeight.w600,
    color: AppColors.textPrimary,
  );

  static const TextStyle headline = TextStyle(
    fontSize: 17,
    fontWeight: FontWeight.w600,
    color: AppColors.textPrimary,
  );

  static const TextStyle body = TextStyle(
    fontSize: 15,
    fontWeight: FontWeight.normal,
    color: AppColors.textPrimary,
  );

  static const TextStyle caption = TextStyle(
    fontSize: 13,
    fontWeight: FontWeight.normal,
    color: AppColors.textSecondary,
  );

  static const TextStyle footnote = TextStyle(
    fontSize: 12,
    fontWeight: FontWeight.normal,
    color: AppColors.textSecondary,
  );

  // 深色背景上的文字
  static const TextStyle largeTitleOnDark = TextStyle(
    fontSize: 34,
    fontWeight: FontWeight.bold,
    color: AppColors.textOnDark,
    letterSpacing: -0.5,
  );

  static const TextStyle titleOnDark = TextStyle(
    fontSize: 22,
    fontWeight: FontWeight.w600,
    color: AppColors.textOnDark,
  );

  static const TextStyle bodyOnDark = TextStyle(
    fontSize: 15,
    color: AppColors.textOnDark,
  );

  static const TextStyle captionOnDark = TextStyle(
    fontSize: 13,
    color: Colors.white60,
  );
}
