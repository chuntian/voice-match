import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import '../models/user.dart';
import '../models/preferences.dart';
import '../models/call_record.dart';
import '../models/blocked_user.dart';
import 'api_service.dart';

/// Exception thrown when the API returns a non-2xx status code.
@immutable
class ApiException implements Exception {
  final int statusCode;
  final String message;
  final int? code;

  const ApiException({
    required this.statusCode,
    required this.message,
    this.code,
  });

  @override
  String toString() =>
      'ApiException($statusCode, code=$code): $message';
}

/// Concrete implementation of [ApiService] using the `http` package.
///
/// All REST endpoints follow the pattern `https://{host}/api/v1/...`.
/// Authentication is via `Authorization: Bearer <token>` where the token
/// is persisted in [SharedPreferences].
class RealApiService implements ApiService {
  RealApiService({
    http.Client? client,
    this.baseUrl = 'https://api.voicematch.app',
  }) : _client = client ?? http.Client();

  final http.Client _client;
  final String baseUrl;

  static const String _tokenKey = 'auth_token';
  static const String _userIdKey = 'auth_user_id';

  // ---------------------------------------------------------------------------
  // Token management
  // ---------------------------------------------------------------------------

  Future<void> saveToken(String token, String userId) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_tokenKey, token);
    await prefs.setString(_userIdKey, userId);
  }

  Future<String?> getToken() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_tokenKey);
  }

  Future<String?> getUserId() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_userIdKey);
  }

  Future<void> clearToken() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_tokenKey);
    await prefs.remove(_userIdKey);
  }

  // ---------------------------------------------------------------------------
  // Internal helpers
  // ---------------------------------------------------------------------------

  Map<String, String> _headers(String? token) {
    final headers = <String, String>{
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
    if (token != null && token.isNotEmpty) {
      headers['Authorization'] = 'Bearer $token';
    }
    return headers;
  }

  /// Converts snake_case map keys to camelCase so the Dart model fromJson
  /// factories can consume server responses directly.
  Map<String, dynamic> _toCamelCase(Map<String, dynamic> json) {
    final result = <String, dynamic>{};
    json.forEach((key, value) {
      final camelKey = _snakeToCamel(key);
      if (value is Map<String, dynamic>) {
        result[camelKey] = _toCamelCase(value);
      } else if (value is List) {
        result[camelKey] = value
            .map((e) => e is Map<String, dynamic> ? _toCamelCase(e) : e)
            .toList();
      } else {
        result[camelKey] = value;
      }
    });
    return result;
  }

  String _snakeToCamel(String snake) {
    final parts = snake.split('_');
    if (parts.length == 1) return snake;
    final buffer = StringBuffer(parts.first);
    for (var i = 1; i < parts.length; i++) {
      final p = parts[i];
      buffer.write(p.isEmpty ? '' : p[0].toUpperCase() + p.substring(1));
    }
    return buffer.toString();
  }

  Future<Map<String, dynamic>> _sendJson({
    required String method,
    required String path,
    Map<String, dynamic>? body,
    String? token,
  }) async {
    final uri = Uri.parse('$baseUrl$path');
    final headers = _headers(token);
    http.Response resp;

    switch (method.toUpperCase()) {
      case 'POST':
        resp = await _client.post(uri, headers: headers, body: jsonEncode(body));
        break;
      case 'PUT':
        resp = await _client.put(uri, headers: headers, body: jsonEncode(body));
        break;
      case 'DELETE':
        resp = await _client.delete(uri, headers: headers, body: body != null ? jsonEncode(body) : null);
        break;
      case 'GET':
      default:
        resp = await _client.get(uri, headers: headers);
        break;
    }

    if (resp.statusCode < 200 || resp.statusCode >= 300) {
      String message = 'HTTP ${resp.statusCode}';
      int? code;
      try {
        final errBody = jsonDecode(resp.body) as Map<String, dynamic>;
        message = errBody['message'] as String? ?? message;
        code = (errBody['code'] as num?)?.toInt();
      } catch (_) {
        // ignore body parse errors
      }
      throw ApiException(
        statusCode: resp.statusCode,
        message: message,
        code: code,
      );
    }

    if (resp.body.isEmpty) return {};
    return jsonDecode(resp.body) as Map<String, dynamic>;
  }

  // ---------------------------------------------------------------------------
  // Auth
  // ---------------------------------------------------------------------------

  @override
  Future<User> loginWithApple(String identityToken) async {
    final resp = await _sendJson(
      method: 'POST',
      path: '/api/v1/auth/apple',
      body: {'identity_token': identityToken},
    );
    final token = resp['token'] as String?;
    final userJson = resp['user'] as Map<String, dynamic>?;
    if (token != null && userJson != null) {
      final user = User.fromJson(_toCamelCase(userJson));
      await saveToken(token, user.id);
      return user;
    }
    throw const ApiException(
      statusCode: 500,
      message: 'Invalid login response: missing token or user',
    );
  }

  @override
  Future<void> sendPhoneCode(String phone) async {
    await _sendJson(
      method: 'POST',
      path: '/api/v1/auth/phone',
      body: {'phone': phone},
    );
  }

  @override
  Future<User> loginWithPhone(String phone, String code) async {
    final resp = await _sendJson(
      method: 'POST',
      path: '/api/v1/auth/phone/verify',
      body: {'phone': phone, 'code': code},
    );
    final token = resp['token'] as String?;
    final userJson = resp['user'] as Map<String, dynamic>?;
    if (token != null && userJson != null) {
      final user = User.fromJson(_toCamelCase(userJson));
      await saveToken(token, user.id);
      return user;
    }
    throw const ApiException(
      statusCode: 500,
      message: 'Invalid login response: missing token or user',
    );
  }

  // ---------------------------------------------------------------------------
  // User profile
  // ---------------------------------------------------------------------------

  @override
  Future<User> getProfile() async {
    final token = await getToken();
    final resp = await _sendJson(
      method: 'GET',
      path: '/api/v1/users/me',
      token: token,
    );
    return User.fromJson(_toCamelCase(resp));
  }

  @override
  Future<User> updateProfile(Map<String, dynamic> data) async {
    final token = await getToken();
    final resp = await _sendJson(
      method: 'PUT',
      path: '/api/v1/users/me',
      body: data,
      token: token,
    );
    return User.fromJson(_toCamelCase(resp));
  }

  // ---------------------------------------------------------------------------
  // Preferences
  // ---------------------------------------------------------------------------

  @override
  Future<Preferences> getPreferences() async {
    final token = await getToken();
    final resp = await _sendJson(
      method: 'GET',
      path: '/api/v1/users/me/preferences',
      token: token,
    );
    return Preferences.fromJson(_toCamelCase(resp));
  }

  @override
  Future<void> updatePreferences(Preferences prefs) async {
    final token = await getToken();
    await _sendJson(
      method: 'PUT',
      path: '/api/v1/users/me/preferences',
      body: prefs.toJson(),
      token: token,
    );
  }

  // ---------------------------------------------------------------------------
  // Call records
  // ---------------------------------------------------------------------------

  @override
  Future<List<CallRecord>> getCallRecords({
    int page = 1,
    int pageSize = 20,
  }) async {
    final token = await getToken();
    final resp = await _sendJson(
      method: 'GET',
      path: '/api/v1/users/me/call-records?page=$page&page_size=$pageSize',
      token: token,
    );
    final list = resp['items'] as List<dynamic>? ?? resp['records'] as List<dynamic>? ?? [];
    return list
        .map((e) => CallRecord.fromJson(_toCamelCase(e as Map<String, dynamic>)))
        .toList();
  }

  // ---------------------------------------------------------------------------
  // Reports & blocklist
  // ---------------------------------------------------------------------------

  @override
  Future<void> submitReport(
      String reportedId, String reason, String content) async {
    final token = await getToken();
    await _sendJson(
      method: 'POST',
      path: '/api/v1/reports',
      body: {
        'reported_id': reportedId,
        'reason': reason,
        'content': content,
      },
      token: token,
    );
  }

  @override
  Future<void> blockUser(String blockedUserId) async {
    final token = await getToken();
    await _sendJson(
      method: 'POST',
      path: '/api/v1/users/me/blacklist',
      body: {'blocked_user_id': blockedUserId},
      token: token,
    );
  }

  @override
  Future<void> unblockUser(String blockedUserId) async {
    final token = await getToken();
    await _sendJson(
      method: 'DELETE',
      path: '/api/v1/users/me/blacklist/$blockedUserId',
      token: token,
    );
  }

  @override
  Future<List<BlockedUser>> getBlacklist() async {
    final token = await getToken();
    final resp = await _sendJson(
      method: 'GET',
      path: '/api/v1/users/me/blacklist',
      token: token,
    );
    final list = resp['items'] as List<dynamic>? ?? resp['blacklist'] as List<dynamic>? ?? [];
    return list
        .map((e) => BlockedUser.fromJson(_toCamelCase(e as Map<String, dynamic>)))
        .toList();
  }
}
