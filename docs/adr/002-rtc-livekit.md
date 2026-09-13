# ADR 002: RTC 层选择 LiveKit 而非 Agora

## 背景

一对一语音通话需要 WebRTC 媒体通道。可选方案：
1. **Agora / 声网**：商用 RTC PaaS，SDK 成熟，国内 CDN 覆盖好
2. **TRTC (腾讯云)**：类似 Agora，腾讯生态
3. **LiveKit**：开源 WebRTC SFU，可自托管
4. **自建 WebRTC + STUN/TURN**：完全自研，工作量大

## 决策

选择 **LiveKit**（自托管开源自建 SFU）。

## 理由

### 成本控制
- Agora/TRTC 按分钟计费，初期免费额度有限（通常 10000 分钟/月）
- 语音通话约 24kbps 带宽，1v1 通话每方约 1 分钟 = 2 分钟计费
- 若 DAU 5w，每人每天通话 10 分钟 → 50w 分钟/天 → 1500w 分钟/月
- Agora 费率约 ¥0.0015/分钟 → 月成本 ¥22,500，年成本 ¥27w
- LiveKit 自托管：1 台 4C8G 服务器约 ¥500/月，完全覆盖

### 数据隐私
- 语音数据完全在自有服务器流转，不经过第三方 PaaS
- 用户对隐私敏感的社交产品更有信心

### 灵活性
- 可自定义编解码策略、拥塞控制参数
- 可集成 AI 降噪（如 rnnoise）在服务端
- 无厂商锁定，可随时切换

### 成熟度
- LiveKit 已被数千个生产应用使用，GitHub 8k+ stars
- Flutter SDK 维护活跃，API 设计良好
- 内置 iOS CallKit / Android ConnectionService 集成示例

## 后果

### 正面
- 长期成本趋近于零（仅服务器费用）
- 数据自主可控，可做端到端加密扩展
- 无厂商锁定

### 负面
- 需要自行部署和维护 LiveKit 服务器（含 TURN 中继）
- 弱网优化不如 Agora 成熟（Agora 有十余年经验）
- 初期需要投入 1-2 周调优 LiveKit 部署和 NAT 穿透策略
- 无 SLA 保障，需要自行监控告警
