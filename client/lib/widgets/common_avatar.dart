import 'package:flutter/material.dart';
import 'package:cached_network_image/cached_network_image.dart';

/// 圆形头像组件，支持网络图片和首字母占位
class CommonAvatar extends StatelessWidget {
  final String? imageUrl;
  final String? placeholderText; // 占位文字（通常为首字符）
  final double size;
  final Color? backgroundColor;
  final Color? textColor;
  final BoxBorder? border;

  const CommonAvatar({
    super.key,
    this.imageUrl,
    this.placeholderText,
    this.size = 48,
    this.backgroundColor,
    this.textColor,
    this.border,
  });

  /// 从小尺寸快速构造
  const CommonAvatar.small({
    super.key,
    this.imageUrl,
    this.placeholderText,
    this.backgroundColor,
    this.textColor,
    this.border,
  }) : size = 32;

  /// 中等尺寸
  const CommonAvatar.medium({
    super.key,
    this.imageUrl,
    this.placeholderText,
    this.backgroundColor,
    this.textColor,
    this.border,
  }) : size = 48;

  /// 大尺寸
  const CommonAvatar.large({
    super.key,
    this.imageUrl,
    this.placeholderText,
    this.backgroundColor,
    this.textColor,
    this.border,
  }) : size = 80;

  /// 超大尺寸（通话页用）
  const CommonAvatar.xlarge({
    super.key,
    this.imageUrl,
    this.placeholderText,
    this.backgroundColor,
    this.textColor,
    this.border,
  }) : size = 120;

  @override
  Widget build(BuildContext context) {
    final bgColor = backgroundColor ?? const Color(0xFF4A6CF7);
    final fgColor = textColor ?? Colors.white;

    Widget content;
    if (imageUrl != null && imageUrl!.isNotEmpty) {
      content = CachedNetworkImage(
        imageUrl: imageUrl!,
        fit: BoxFit.cover,
        placeholder: (context, url) => Container(
          color: bgColor,
          alignment: Alignment.center,
          child: Text(
            placeholderText ?? '?',
            style: TextStyle(
              color: fgColor,
              fontSize: size * 0.4,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
        errorWidget: (context, url, error) => Container(
          color: bgColor,
          alignment: Alignment.center,
          child: Text(
            placeholderText ?? '?',
            style: TextStyle(
              color: fgColor,
              fontSize: size * 0.4,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      );
    } else {
      content = Container(
        color: bgColor,
        alignment: Alignment.center,
        child: Text(
          placeholderText ?? '?',
          style: TextStyle(
            color: fgColor,
            fontSize: size * 0.4,
            fontWeight: FontWeight.w600,
          ),
        ),
      );
    }

    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        border: border,
      ),
      child: ClipOval(child: content),
    );
  }
}
