import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../services/api_service.dart';
import '../models/user.dart';
import '../models/call_record.dart';
import '../models/blocked_user.dart';
import '../theme/app_theme.dart';
import '../widgets/common_avatar.dart';
import '../widgets/common_button.dart';
import '../widgets/empty_state.dart';
import '../widgets/loading_overlay.dart';
import '../router/app_router.dart';

class ProfilePage extends StatefulWidget {
  const ProfilePage({super.key});

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  User? _user;
  bool _isLoading = false;
  List<CallRecord> _callRecords = [];
  List<BlockedUser> _blacklist = [];

  @override
  void initState() {
    super.initState();
    _loadUserData();
  }

  Future<void> _loadUserData() async {
    setState(() => _isLoading = true);
    try {
      final apiService = Provider.of<ApiService>(context, listen: false);
      final user = await apiService.getProfile();
      final records = await apiService.getCallRecords();
      final blacklist = await apiService.getBlacklist();
      if (mounted) {
        setState(() {
          _user = user;
          _callRecords = records;
          _blacklist = blacklist;
        });
      }
    } catch (_) {
      // 使用缓存数据或默认值
      _loadFromPrefs();
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  void _loadFromPrefs() async {
    final prefs = await SharedPreferences.getInstance();
    final nickname = prefs.getString('nickname') ?? 'VoiceMatch 用户';
    final avatar = prefs.getString('avatar');
    if (mounted) {
      setState(() {
        _user = User(
          id: prefs.getString('user_id') ?? '',
          nickname: nickname,
          avatar: avatar,
          createdAt: DateTime.now(),
          updatedAt: DateTime.now(),
        );
      });
    }
  }

  Future<void> _logout() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('退出登录'),
        content: const Text('确定要退出登录吗？'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('退出'),
          ),
        ],
      ),
    );

    if (confirmed == true) {
      final prefs = await SharedPreferences.getInstance();
      await prefs.clear();
      if (mounted) {
        Navigator.pushNamedAndRemoveUntil(
          context,
          AppRoutes.login,
          (route) => false,
        );
      }
    }
  }

  void _openEditProfileSheet() {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (ctx) => _EditProfileSheet(
        user: _user,
        onSaved: (updatedUser) {
          setState(() => _user = updatedUser);
        },
      ),
    );
  }

  void _openCallRecordsList() {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => _CallRecordsListPage(records: _callRecords),
      ),
    );
  }

  void _openBlacklistList() {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => _BlacklistListPage(
          blacklist: _blacklist,
          onUnblock: (blockedUserId) async {
            try {
              final apiService =
                  Provider.of<ApiService>(context, listen: false);
              await apiService.unblockUser(blockedUserId);
              setState(() {
                _blacklist
                    .removeWhere((b) => b.blockedUserId == blockedUserId);
              });
              if (mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('已取消拉黑')),
                );
              }
            } catch (e) {
              if (mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('操作失败: $e')),
                );
              }
            }
          },
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('我的'),
      ),
      body: LoadingOverlay(
        isLoading: _isLoading,
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(20),
          child: Column(
            children: [
              // 用户信息卡片
              _buildUserCard(),
              const SizedBox(height: 24),

              // 功能列表
              _buildMenuSection(),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildUserCard() {
    final nickname = _user?.nickname ?? 'VoiceMatch 用户';
    final avatar = _user?.avatar;
    final city = _user?.city;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.04),
            blurRadius: 10,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Row(
        children: [
          CommonAvatar.large(
            imageUrl: avatar,
            placeholderText: nickname.isNotEmpty ? nickname.substring(0, 1) : '?',
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  nickname,
                  style: const TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                    color: AppColors.textPrimary,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  city ?? '未设置城市',
                  style: const TextStyle(
                    fontSize: 14,
                    color: AppColors.textSecondary,
                  ),
                ),
              ],
            ),
          ),
          IconButton(
            onPressed: _openEditProfileSheet,
            icon: const Icon(Icons.edit_outlined,
                color: AppColors.textSecondary),
          ),
        ],
      ),
    );
  }

  Widget _buildMenuSection() {
    return Container(
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        children: [
          _buildMenuItem(
            icon: Icons.person_outline,
            title: '个人资料',
            onTap: _openEditProfileSheet,
          ),
          _buildMenuItem(
            icon: Icons.tune_outlined,
            title: '匹配偏好',
            onTap: () {
              Navigator.pushNamed(context, AppRoutes.matchSettings);
            },
          ),
          _buildMenuItem(
            icon: Icons.history,
            title: '通话记录',
            trailing: '${_callRecords.length} 条',
            onTap: _openCallRecordsList,
          ),
          _buildMenuItem(
            icon: Icons.block,
            title: '黑名单管理',
            trailing: '${_blacklist.length} 人',
            onTap: _openBlacklistList,
          ),
          _buildMenuItem(
            icon: Icons.info_outline,
            title: '关于我们',
            onTap: () {
              showAboutDialog(
                context: context,
                applicationName: 'VoiceMatch',
                applicationVersion: '1.0.0',
                applicationLegalese: '© 2024 VoiceMatch',
              );
            },
          ),
          const Divider(height: 1),
          _buildMenuItem(
            icon: Icons.logout,
            title: '退出登录',
            titleColor: Colors.red,
            onTap: _logout,
          ),
        ],
      ),
    );
  }

  Widget _buildMenuItem({
    required IconData icon,
    required String title,
    String? trailing,
    Color? titleColor,
    required VoidCallback onTap,
  }) {
    return ListTile(
      leading: Icon(icon, color: titleColor ?? AppColors.textSecondary),
      title: Text(
        title,
        style: TextStyle(
          fontSize: 16,
          color: titleColor ?? AppColors.textPrimary,
          fontWeight: FontWeight.w500,
        ),
      ),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (trailing != null)
            Text(
              trailing,
              style: const TextStyle(
                fontSize: 13,
                color: AppColors.textSecondary,
              ),
            ),
          const SizedBox(width: 4),
          const Icon(Icons.chevron_right, color: AppColors.textSecondary),
        ],
      ),
      onTap: onTap,
    );
  }
}

/// 通话记录列表页
class _CallRecordsListPage extends StatelessWidget {
  final List<CallRecord> records;

  const _CallRecordsListPage({required this.records});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('通话记录')),
      body: records.isEmpty
          ? const EmptyState(
              icon: Icons.history,
              title: '暂无通话记录',
              subtitle: '开始匹配后，通话记录将显示在这里',
            )
          : ListView.separated(
              padding: const EdgeInsets.all(16),
              itemCount: records.length,
              separatorBuilder: (_, __) => const SizedBox(height: 12),
              itemBuilder: (context, index) {
                final record = records[index];
                final peer = record.peer;
                final peerName = peer?.nickname ?? '未知用户';
                final peerAvatar = peer?.avatar;

                return Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Row(
                    children: [
                      CommonAvatar.medium(
                        imageUrl: peerAvatar,
                        placeholderText:
                            peerName.isNotEmpty ? peerName.substring(0, 1) : '?',
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              peerName,
                              style: const TextStyle(
                                fontSize: 16,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                            const SizedBox(height: 4),
                            Text(
                              record.formattedStartTime,
                              style: const TextStyle(
                                fontSize: 13,
                                color: AppColors.textSecondary,
                              ),
                            ),
                          ],
                        ),
                      ),
                      Column(
                        crossAxisAlignment: CrossAxisAlignment.end,
                        children: [
                          const Icon(Icons.call,
                              size: 18, color: AppColors.primaryBlue),
                          const SizedBox(height: 4),
                          Text(
                            record.formattedDuration,
                            style: const TextStyle(
                              fontSize: 13,
                              color: AppColors.textSecondary,
                              fontFeatures: [FontFeature.tabularFigures()],
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                );
              },
            ),
    );
  }
}

/// 黑名单列表页
class _BlacklistListPage extends StatelessWidget {
  final List<BlockedUser> blacklist;
  final Function(String blockedUserId) onUnblock;

  const _BlacklistListPage({
    required this.blacklist,
    required this.onUnblock,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('黑名单管理')),
      body: blacklist.isEmpty
          ? const EmptyState(
              icon: Icons.block,
              title: '黑名单为空',
              subtitle: '你没有拉黑任何用户',
            )
          : ListView.separated(
              padding: const EdgeInsets.all(16),
              itemCount: blacklist.length,
              separatorBuilder: (_, __) => const SizedBox(height: 12),
              itemBuilder: (context, index) {
                final blocked = blacklist[index];
                final user = blocked.user;
                final userName = user?.nickname ?? '未知用户';
                final userAvatar = user?.avatar;

                return Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Row(
                    children: [
                      CommonAvatar.medium(
                        imageUrl: userAvatar,
                        placeholderText:
                            userName.isNotEmpty ? userName.substring(0, 1) : '?',
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              userName,
                              style: const TextStyle(
                                fontSize: 16,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                            const SizedBox(height: 4),
                            Text(
                              '拉黑于 ${blocked.createdAt.month}月${blocked.createdAt.day}日',
                              style: const TextStyle(
                                fontSize: 13,
                                color: AppColors.textSecondary,
                              ),
                            ),
                          ],
                        ),
                      ),
                      TextButton(
                        onPressed: () => onUnblock(blocked.blockedUserId),
                        child: const Text(
                          '取消拉黑',
                          style: TextStyle(color: AppColors.primaryBlue),
                        ),
                      ),
                    ],
                  ),
                );
              },
            ),
    );
  }
}

/// 编辑资料 Bottom Sheet
class _EditProfileSheet extends StatefulWidget {
  final User? user;
  final Function(User) onSaved;

  const _EditProfileSheet({this.user, required this.onSaved});

  @override
  State<_EditProfileSheet> createState() => _EditProfileSheetState();
}

class _EditProfileSheetState extends State<_EditProfileSheet> {
  late final TextEditingController _nicknameController;
  late final TextEditingController _cityController;
  String? _gender;
  bool _isSaving = false;

  @override
  void initState() {
    super.initState();
    _nicknameController = TextEditingController(text: widget.user?.nickname ?? '');
    _cityController = TextEditingController(text: widget.user?.city ?? '');
    _gender = widget.user?.gender;
  }

  @override
  void dispose() {
    _nicknameController.dispose();
    _cityController.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    final nickname = _nicknameController.text.trim();
    if (nickname.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('昵称不能为空')),
      );
      return;
    }

    setState(() => _isSaving = true);
    try {
      final apiService = Provider.of<ApiService>(context, listen: false);
      final updated = await apiService.updateProfile({
        'nickname': nickname,
        'gender': _gender,
        'city': _cityController.text.trim(),
      });

      // 保存到本地
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString('nickname', updated.nickname);

      widget.onSaved(updated);
      if (mounted) {
        Navigator.pop(context);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('资料已更新')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('保存失败: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _isSaving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final bottomInset = MediaQuery.of(context).viewInsets.bottom;

    return Container(
      padding: EdgeInsets.only(bottom: bottomInset),
      decoration: const BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Center(
              child: Container(
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                  color: Colors.grey.shade300,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
            ),
            const SizedBox(height: 24),
            const Text(
              '编辑资料',
              style: TextStyle(
                fontSize: 20,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 24),

            // 昵称
            const Text('昵称', style: TextStyle(fontSize: 14, color: AppColors.textSecondary)),
            const SizedBox(height: 8),
            TextField(
              controller: _nicknameController,
              decoration: const InputDecoration(hintText: '输入你的昵称'),
            ),
            const SizedBox(height: 16),

            // 性别
            const Text('性别', style: TextStyle(fontSize: 14, color: AppColors.textSecondary)),
            const SizedBox(height: 8),
            Row(
              children: [
                _buildGenderOption('male', '男'),
                const SizedBox(width: 12),
                _buildGenderOption('female', '女'),
                const SizedBox(width: 12),
                _buildGenderOption(null, '保密'),
              ],
            ),
            const SizedBox(height: 16),

            // 城市
            const Text('城市', style: TextStyle(fontSize: 14, color: AppColors.textSecondary)),
            const SizedBox(height: 8),
            TextField(
              controller: _cityController,
              decoration: const InputDecoration(hintText: '输入所在城市'),
            ),
            const SizedBox(height: 24),

            PrimaryButton(
              text: '保存',
              loading: _isSaving,
              onPressed: _save,
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }

  Widget _buildGenderOption(String? value, String label) {
    final isSelected = _gender == value;
    return Expanded(
      child: GestureDetector(
        onTap: () => setState(() => _gender = value),
        child: Container(
          padding: const EdgeInsets.symmetric(vertical: 12),
          decoration: BoxDecoration(
            color: isSelected ? AppColors.primaryBlue : Colors.grey.shade50,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: isSelected ? AppColors.primaryBlue : Colors.grey.shade300,
            ),
          ),
          child: Text(
            label,
            textAlign: TextAlign.center,
            style: TextStyle(
              fontSize: 14,
              fontWeight: isSelected ? FontWeight.w600 : FontWeight.normal,
              color: isSelected ? Colors.white : AppColors.textPrimary,
            ),
          ),
        ),
      ),
    );
  }
}
