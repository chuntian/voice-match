import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../services/api_service.dart';
import '../services/match_service.dart';
import '../models/preferences.dart';
import '../theme/app_theme.dart';
import '../widgets/common_button.dart';
import '../widgets/loading_overlay.dart';
import '../router/app_router.dart';

class MatchSettingsPage extends StatefulWidget {
  const MatchSettingsPage({super.key});

  @override
  State<MatchSettingsPage> createState() => _MatchSettingsPageState();
}

class _MatchSettingsPageState extends State<MatchSettingsPage> {
  String _matchType = 'random'; // random, city, destination
  final List<String> _selectedTags = [];
  String _genderPreference = 'all'; // all, male, female
  final _cityController = TextEditingController();
  final _destinationController = TextEditingController();
  bool _isLoading = false;

  static const List<String> presetTags = [
    '聊天', '音乐', '游戏', '旅行', '美食',
    '电影', '读书', '运动', '工作', '情感',
  ];

  static const Map<String, String> matchTypeLabels = {
    'random': '不限',
    'city': '同城',
    'destination': '目的地',
  };

  static const Map<String, String> genderLabels = {
    'all': '不限',
    'male': '男',
    'female': '女',
  };

  @override
  void initState() {
    super.initState();
    _loadPreferences();
  }

  @override
  void dispose() {
    _cityController.dispose();
    _destinationController.dispose();
    super.dispose();
  }

  Future<void> _loadPreferences() async {
    setState(() => _isLoading = true);
    try {
      final apiService = Provider.of<ApiService>(context, listen: false);
      final prefs = await apiService.getPreferences();
      if (mounted) {
        setState(() {
          _matchType = prefs.matchType;
          _genderPreference = prefs.genderPreference ?? 'all';
          _selectedTags.addAll(prefs.tags);
          if (prefs.city != null) _cityController.text = prefs.city!;
          if (prefs.destination != null) {
            _destinationController.text = prefs.destination!;
          }
        });
      }
    } catch (_) {
      // 使用默认设置
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  void _toggleTag(String tag) {
    setState(() {
      if (_selectedTags.contains(tag)) {
        _selectedTags.remove(tag);
      } else {
        if (_selectedTags.length < 5) {
          _selectedTags.add(tag);
        } else {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('最多选择5个兴趣标签')),
          );
        }
      }
    });
  }

  Future<void> _startMatching() async {
    // 校验
    if (_matchType == 'city' && _cityController.text.trim().isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请输入同城匹配的城市')),
      );
      return;
    }
    if (_matchType == 'destination' &&
        _destinationController.text.trim().isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请输入目的地')),
      );
      return;
    }

    setState(() => _isLoading = true);
    try {
      // 保存偏好
      final apiService = Provider.of<ApiService>(context, listen: false);
      final prefs = Preferences(
        matchType: _matchType,
        city: _matchType == 'city' ? _cityController.text.trim() : null,
        destination: _matchType == 'destination'
            ? _destinationController.text.trim()
            : null,
        tags: _selectedTags,
        genderPreference: _genderPreference,
      );
      await apiService.updatePreferences(prefs);

      // 加入匹配池
      final matchService =
          Provider.of<MatchService>(context, listen: false);
      await matchService.joinPool(
        matchType: _matchType,
        city: _matchType == 'city' ? _cityController.text.trim() : null,
        destination:
            _matchType == 'destination' ? _destinationController.text.trim() : null,
        tags: _selectedTags,
      );

      if (mounted) {
        Navigator.pushReplacementNamed(context, AppRoutes.matchWaiting);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('匹配失败: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('匹配设置'),
      ),
      body: LoadingOverlay(
        isLoading: _isLoading,
        message: '正在加入匹配池...',
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // 匹配范围
              const Text(
                '匹配范围',
                style: TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w600,
                  color: AppColors.textPrimary,
                ),
              ),
              const SizedBox(height: 12),
              _buildMatchTypeSelector(),
              const SizedBox(height: 24),

              // 目的地输入
              if (_matchType == 'destination') ...[
                const Text(
                  '目的地',
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w600,
                    color: AppColors.textPrimary,
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _destinationController,
                  decoration: const InputDecoration(
                    hintText: '想去哪里？输入城市或地点',
                    prefixIcon: Icon(Icons.place_outlined),
                  ),
                ),
                const SizedBox(height: 24),
              ],

              // 同城城市输入
              if (_matchType == 'city') ...[
                const Text(
                  '所在城市',
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w600,
                    color: AppColors.textPrimary,
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _cityController,
                  decoration: const InputDecoration(
                    hintText: '输入你的城市',
                    prefixIcon: Icon(Icons.location_city_outlined),
                  ),
                ),
                const SizedBox(height: 24),
              ],

              // 性别偏好
              const Text(
                '性别偏好',
                style: TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w600,
                  color: AppColors.textPrimary,
                ),
              ),
              const SizedBox(height: 12),
              _buildGenderSelector(),
              const SizedBox(height: 24),

              // 兴趣标签
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  const Text(
                    '兴趣标签',
                    style: TextStyle(
                      fontSize: 16,
                      fontWeight: FontWeight.w600,
                      color: AppColors.textPrimary,
                    ),
                  ),
                  Text(
                    '已选 ${_selectedTags.length}/5',
                    style: const TextStyle(
                      fontSize: 13,
                      color: AppColors.textSecondary,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              _buildTagSelector(),
            ],
          ),
        ),
      ),
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: PrimaryButton(
            text: '开始匹配',
            icon: Icons.play_arrow,
            onPressed: _startMatching,
          ),
        ),
      ),
    );
  }

  Widget _buildMatchTypeSelector() {
    return Container(
      decoration: BoxDecoration(
        color: Colors.grey.shade100,
        borderRadius: BorderRadius.circular(12),
      ),
      padding: const EdgeInsets.all(4),
      child: Row(
        children: matchTypeLabels.entries.map((entry) {
          final isSelected = _matchType == entry.key;
          return Expanded(
            child: GestureDetector(
              onTap: () => setState(() => _matchType = entry.key),
              child: Container(
                padding: const EdgeInsets.symmetric(vertical: 10),
                decoration: BoxDecoration(
                  color: isSelected ? Colors.white : Colors.transparent,
                  borderRadius: BorderRadius.circular(8),
                  boxShadow: isSelected
                      ? [
                          BoxShadow(
                            color: Colors.black.withValues(alpha: 0.05),
                            blurRadius: 4,
                          ),
                        ]
                      : null,
                ),
                child: Text(
                  entry.value,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight:
                        isSelected ? FontWeight.w600 : FontWeight.normal,
                    color: isSelected
                        ? AppColors.primaryBlue
                        : AppColors.textSecondary,
                  ),
                ),
              ),
            ),
          );
        }).toList(),
      ),
    );
  }

  Widget _buildGenderSelector() {
    return Container(
      decoration: BoxDecoration(
        color: Colors.grey.shade100,
        borderRadius: BorderRadius.circular(12),
      ),
      padding: const EdgeInsets.all(4),
      child: Row(
        children: genderLabels.entries.map((entry) {
          final isSelected = _genderPreference == entry.key;
          return Expanded(
            child: GestureDetector(
              onTap: () => setState(() => _genderPreference = entry.key),
              child: Container(
                padding: const EdgeInsets.symmetric(vertical: 10),
                decoration: BoxDecoration(
                  color: isSelected ? Colors.white : Colors.transparent,
                  borderRadius: BorderRadius.circular(8),
                  boxShadow: isSelected
                      ? [
                          BoxShadow(
                            color: Colors.black.withValues(alpha: 0.05),
                            blurRadius: 4,
                          ),
                        ]
                      : null,
                ),
                child: Text(
                  entry.value,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight:
                        isSelected ? FontWeight.w600 : FontWeight.normal,
                    color: isSelected
                        ? AppColors.primaryBlue
                        : AppColors.textSecondary,
                  ),
                ),
              ),
            ),
          );
        }).toList(),
      ),
    );
  }

  Widget _buildTagSelector() {
    return Wrap(
      spacing: 10,
      runSpacing: 10,
      children: presetTags.map((tag) {
        final isSelected = _selectedTags.contains(tag);
        return GestureDetector(
          onTap: () => _toggleTag(tag),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            decoration: BoxDecoration(
              color: isSelected ? AppColors.primaryBlue : Colors.white,
              borderRadius: BorderRadius.circular(20),
              border: Border.all(
                color: isSelected
                    ? AppColors.primaryBlue
                    : Colors.grey.shade300,
                width: 1,
              ),
            ),
            child: Text(
              tag,
              style: TextStyle(
                fontSize: 14,
                fontWeight: isSelected ? FontWeight.w600 : FontWeight.normal,
                color: isSelected ? Colors.white : AppColors.textPrimary,
              ),
            ),
          ),
        );
      }).toList(),
    );
  }
}

