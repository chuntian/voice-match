import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_callkit_incoming/flutter_callkit_incoming.dart';
import 'package:flutter_callkit_incoming/entities/entities.dart';

import 'callkit_service.dart';

/// Concrete implementation of [CallKitService].
///
/// Wraps the `flutter_callkit_incoming` plugin (v2.5.8) to show the native
/// incoming call UI (CallKit on iOS / ConnectionService on Android) and to
/// surface user actions (accept, decline, end, timeout) as [CallKitEvent]s.
///
/// Note: `flutter_callkit_incoming` v2.5.x only `import`s its `entities/`
/// library from the entry point (it does not re-export it), so callers must
/// import `entities/entities.dart` explicitly to access [CallKitParams],
/// [AndroidParams], [IOSParams], [Event] and [CallEvent].
class RealCallKitService extends ChangeNotifier implements CallKitService {
  RealCallKitService();

  final _eventsController = StreamController<CallKitEvent>.broadcast();
  StreamSubscription<CallEvent?>? _eventSub;

  static const _androidIncomingChannelName = 'voicematch_calls';
  static const _androidMissedChannelName = 'voicematch_missed';

  @override
  Stream<CallKitEvent> get events => _eventsController.stream;

  // ---------------------------------------------------------------------------
  // Setup
  // ---------------------------------------------------------------------------

  @override
  Future<void> setup() async {
    // Since v2.5.x there is no explicit `setup()` on the plugin side: the
    // Android notification channel / ringtone configuration is declared
    // per-call via [AndroidParams] passed to `showCallkitIncoming`. We only
    // need to register the event listener here.
    _eventSub = FlutterCallkitIncoming.onEvent.listen(_onCallKitEvent);
    debugPrint('CallKit setup complete');
  }

  // ---------------------------------------------------------------------------
  // Incoming / outgoing call UI
  // ---------------------------------------------------------------------------

  @override
  Future<void> showIncomingCall(
      String callId, String callerName, String callerAvatar) async {
    final params = CallKitParams(
      id: callId,
      nameCaller: callerName,
      avatar: callerAvatar,
      appName: 'VoiceMatch',
      handle: '',
      type: 0, // Audio call
      textAccept: '接听',
      textDecline: '挂断',
      duration: 30000, // Ringing timeout 30 s
      extra: <String, dynamic>{'call_id': callId},
      headers: <String, dynamic>{},
      android: AndroidParams(
        isCustomNotification: true,
        isShowLogo: true,
        ringtonePath: 'ringtone_default',
        backgroundColor: '#1A1A2E',
        backgroundUrl: callerAvatar,
        actionColor: '#4CAF50',
        incomingCallNotificationChannelName: _androidIncomingChannelName,
        missedCallNotificationChannelName: _androidMissedChannelName,
        isImportant: true,
      ),
      ios: const IOSParams(
        iconName: 'AppIcon',
        handleType: 'generic',
        supportsVideo: false,
        maximumCallGroups: 1,
        maximumCallsPerCallGroup: 1,
        audioSessionMode: 'voiceChat',
        audioSessionActive: true,
        audioSessionPreferredSampleRate: 44100.0,
        audioSessionPreferredIOBufferDuration: 0.005,
        supportsDTMF: false,
        supportsHolding: false,
        supportsGrouping: false,
        supportsUngrouping: false,
      ),
    );

    await FlutterCallkitIncoming.showCallkitIncoming(params);
    debugPrint('CallKit: showing incoming call from $callerName ($callId)');
  }

  /// Shows the outgoing call UI for the caller.
  Future<void> startOutgoingCall(
      String callId, String calleeName) async {
    final params = CallKitParams(
      id: callId,
      nameCaller: calleeName,
      appName: 'VoiceMatch',
      handle: '',
      type: 0,
      textAccept: '接听',
      textDecline: '挂断',
      extra: <String, dynamic>{'call_id': callId},
      ios: const IOSParams(
        handleType: 'generic',
        supportsVideo: false,
      ),
    );

    await FlutterCallkitIncoming.startCall(params);
    debugPrint('CallKit: starting outgoing call to $calleeName ($callId)');
  }

  @override
  Future<void> endCall(String callId) async {
    await FlutterCallkitIncoming.endCall(callId);
    debugPrint('CallKit: ending call $callId');
  }

  /// Ends all active calls.
  Future<void> endAllCalls() async {
    await FlutterCallkitIncoming.endAllCalls();
  }

  /// Lists the currently active calls.
  Future<void> activeCalls() async {
    final calls = await FlutterCallkitIncoming.activeCalls();
    debugPrint('CallKit active calls: $calls');
  }

  // ---------------------------------------------------------------------------
  // Event handling
  // ---------------------------------------------------------------------------

  void _onCallKitEvent(CallEvent? event) {
    if (event == null) return;
    final body = event.body;
    final callId =
        (body is Map) ? (body['id']?.toString() ?? '') : '';

    switch (event.event) {
      case Event.actionCallAccept:
        _eventsController.add(CallKitEvent.accept(callId));
        break;
      case Event.actionCallDecline:
        _eventsController.add(CallKitEvent.decline(callId));
        break;
      case Event.actionCallEnded:
        _eventsController.add(CallKitEvent.end(callId));
        break;
      case Event.actionCallTimeout:
        _eventsController.add(CallKitEvent(
          type: CallKitEventType.onTimeout,
          callId: callId,
        ));
        break;
      default:
        // actionCallIncoming / actionCallStart / actionCallConnected /
        // actionCallCallback / actionCallToggle* / actionCallCustom
        // are informational and not surfaced as business events.
        break;
    }
  }

  @override
  void dispose() {
    _eventSub?.cancel();
    _eventsController.close();
    super.dispose();
  }
}
