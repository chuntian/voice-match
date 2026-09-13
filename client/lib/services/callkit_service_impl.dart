import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_callkit_incoming/flutter_callkit_incoming.dart';

import 'callkit_service.dart';

/// Concrete implementation of [CallKitService].
///
/// Wraps the `flutter_callkit_incoming` plugin to show the native incoming
/// call UI (CallKit on iOS / ConnectionService on Android) and to surface
/// user actions (accept, decline, end, timeout) as [CallKitEvent]s.
class RealCallKitService extends ChangeNotifier implements CallKitService {
  RealCallKitService();

  final _eventsController = StreamController<CallKitEvent>.broadcast();
  StreamSubscription? _eventSub;

  static const _androidNotificationChannelId = 'voicematch_calls';
  static const _androidNotificationChannelName = 'VoiceMatch 来电';

  @override
  Stream<CallKitEvent> get events => _eventsController.stream;

  // ---------------------------------------------------------------------------
  // Setup
  // ---------------------------------------------------------------------------

  @override
  Future<void> setup() async {
    // Listen to CallKit events from the platform plugin
    _eventSub = FlutterCallkitIncoming.onEvent.listen(_onCallKitEvent);

    // Configure Android notification channel
    final params = CallKitParams(
      android: const AndroidParams(
        isCustomNotification: true,
        isShowLogo: true,
        ringtonePath: 'res://raw/ringtone_default',
        backgroundColor: '#1A1A2E',
        backgroundUrl: '',
        actionColor: '#4CAF50',
      ),
    );

    await FlutterCallkitIncoming.setup(params);
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
        ringtonePath: 'res://raw/ringtone_default',
        backgroundColor: '#1A1A2E',
        backgroundUrl: callerAvatar,
        actionColor: '#4CAF50',
        notificationChannelId: _androidNotificationChannelId,
        notificationChannelName: _androidNotificationChannelName,
        isShowCallkit: true,
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
        supportsConference: false,
        supportsCallswapping: false,
      ),
    );

    await FlutterCallkitIncoming.showIncomingCall(params);
    debugPrint('CallKit: showing incoming call from $callerName ($callId)');
  }

  /// Shows the outgoing call UI for the caller.
  Future<void> startOutgoingCall(
      String callId, String calleeName) async {
    final params: CallKitParams = CallKitParams(
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

  /// Sets the caller as unavailable (reject all).
  Future<void> activeCalls() async {
    final calls = await FlutterCallkitIncoming.activeCalls();
    debugPrint('CallKit active calls: $calls');
  }

  // ---------------------------------------------------------------------------
  // Event handling
  // ---------------------------------------------------------------------------

  void _onCallKitEvent(Event event) {
    final body = event.body;
    final callId = body['id']?.toString() ?? '';

    switch (event.event) {
      case Event.actionCallAccept:
        _eventsController.add(CallKitEvent.accept(callId));
        break;
      case Event.actionCallDecline:
        _eventsController.add(CallKitEvent.decline(callId));
        break;
      case Event.actionCallEnd:
        _eventsController.add(CallKitEvent.end(callId));
        break;
      case Event.actionCallTimeout:
        _eventsController.add(CallKitEvent(
          type: CallKitEventType.onTimeout,
          callId: callId,
        ));
        break;
      case Event.actionCallCallback:
      case Event.actionCallToggleMute:
      case Event.actionCallToggleSpeaker:
      case Event.actionCallToggleHold:
      case Event.actionDidUpdateCreate:
      case Event.actionCustom:
        // These are informational; not surfaced as business events
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
