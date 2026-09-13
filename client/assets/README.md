# VoiceMatch 资源文件目录

## 图片资源放置规范

将以下图片资源放置在此目录中：

### 命名规范
- 全部使用小写字母 + 下划线
- 格式：`功能_名称_尺寸.png`

### 推荐资源
| 文件名 | 用途 | 尺寸建议 |
|--------|------|----------|
| `logo_app.png` | App 主 Logo | 512x512 |
| `logo_login.png` | 登录页 Logo | 192x192 |
| `icon_audio_wave.png` | 声波动画图标 | 120x120 |
| `avatar_default.png` | 默认头像占位 | 200x200 |
| `bg_login.png` | 登录页背景 | 1170x2532 |

### 使用方式
在 Dart 代码中通过 `Assets.xxx` 或直接路径引用：
```dart
Image.asset('assets/logo_app.png')
```

### 注意事项
- 所有图片必须经过压缩优化
- 不包含带透明通道的背景图片请使用 JPG 格式
- iOS/Android 各平台的启动图标请通过对应平台工具生成
