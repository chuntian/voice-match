/// Unified error codes shared between server and client.
///
/// Mirrors `shared/errors/codes.go` on the Go side.
library;

/// Error code constants.
class ErrorCode {
  static const int ok = 0;
  static const int invalidToken = 1;
  static const int userNotFound = 2;
  static const int alreadyInCall = 3;
  static const int notInPool = 4;
  static const int poolTimeout = 5;
  static const int callNotFound = 6;
  static const int callRejected = 7;
  static const int network = 8;
  static const int internal = 9;
  static const int rateLimited = 10;
  static const int blacklisted = 11;
  static const int invalidParams = 12;
}

/// Human-readable messages for each error code.
const Map<int, String> errorMessages = {
  ErrorCode.ok: 'ok',
  ErrorCode.invalidToken: 'invalid or expired token',
  ErrorCode.userNotFound: 'user not found',
  ErrorCode.alreadyInCall: 'user is already in a call',
  ErrorCode.notInPool: 'user is not in any matching pool',
  ErrorCode.poolTimeout: 'matching pool wait timeout',
  ErrorCode.callNotFound: 'call not found',
  ErrorCode.callRejected: 'call was rejected',
  ErrorCode.network: 'network error',
  ErrorCode.internal: 'internal server error',
  ErrorCode.rateLimited: 'too many requests',
  ErrorCode.blacklisted: 'user is blacklisted by peer',
  ErrorCode.invalidParams: 'invalid request parameters',
};

/// Returns the human-readable message for [code].
String errorMessage(int code) =>
    errorMessages[code] ?? 'unknown error';
