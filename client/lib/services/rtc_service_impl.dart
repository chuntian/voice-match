import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:livekit_client/livekit_client.dart';

import 'rtc_service.dart';

/// Concrete implementation of [RtcService] backed by LiveKit.
///
/// Wraps a LiveKit [Room] instance and emits [RtcEvent]s on the
/// [events] broadcast stream.
class RealRtcService extends ChangeNotifier implements RtcService {
  RealRtcService();

  Room? _room;
  StreamSubscription<RoomEvent>? _roomEventSub;
  StreamSubscription<RoomSignalEvent>? _signalEventSub;

  final _eventsController = StreamController<RtcEvent>.broadcast();

  @override
  Stream<RtcEvent> get events => _eventsController.stream;

  /// Whether currently connected to a room.
  bool get isConnected => _room?.connectionState == ConnectionState.connected;

  // ---------------------------------------------------------------------------
  // Room lifecycle
  // ---------------------------------------------------------------------------

  @override
  Future<void> joinRoom(String token, String roomName) async {
    // Clean up any previous room
    await leaveRoom();

    final room = Room();
    _room = room;

    // Set up event listeners
    _roomEventSub = room.events.listen(_onRoomEvent);

    final connectOptions = const ConnectOptions(
      autoSubscribe: true,
    );

    final roomOptions = RoomOptions(
      adaptiveStream: true,
      dynacast: true,
      defaultAudioPublishOptions: const AudioPublishOptions(
        destination: TrackPublishOptionsDestination.all,
      ),
      defaultCameraPublishOptions: const CameraPublishOptions(),
    );

    try {
      final url = _liveKitUrlFromToken(token);
      await room.connect(url, token,
          connectOptions: connectOptions, roomOptions: roomOptions);

      // Publish local audio track
      await room.localParticipant?.setMicrophoneEnabled(true);

      _eventsController.add(RtcEvent.connected());
      notifyListeners();
    } catch (e) {
      _eventsController.add(RtcEvent.disconnected());
      rethrow;
    }
  }

  @override
  Future<void> leaveRoom() async {
    await _roomEventSub?.cancel();
    _roomEventSub = null;
    await _signalEventSub?.cancel();
    _signalEventSub = null;

    final room = _room;
    _room = null;

    if (room != null) {
      try {
        await room.disconnect();
      } catch (e) {
        debugPrint('Error leaving room: $e');
      }
      await room.dispose();
    }

    _eventsController.add(RtcEvent.disconnected());
    notifyListeners();
  }

  // ---------------------------------------------------------------------------
  // Audio controls
  // ---------------------------------------------------------------------------

  @override
  void setMuted(bool muted) {
    final participant = _room?.localParticipant;
    if (participant != null) {
      participant.setMicrophoneEnabled(!muted);
    }
    notifyListeners();
  }

  @override
  void setSpeakerOn(bool on) {
    Hardware.setSpeakerphoneOn(on);
    notifyListeners();
  }

  // ---------------------------------------------------------------------------
  // Room event handling
  // ---------------------------------------------------------------------------

  void _onRoomEvent(RoomEvent event) {
    if (event is RoomDisconnectedEvent) {
      _eventsController.add(RtcEvent.disconnected());
      notifyListeners();
    } else if (event is RoomConnectedEvent) {
      _eventsController.add(RtcEvent.connected());
      notifyListeners();
    } else if (event is ParticipantConnectedEvent) {
      final participant = event.participant;
      debugPrint('Remote participant joined: ${participant.identity}');
    } else if (event is ParticipantDisconnectedEvent) {
      final participant = event.participant;
      debugPrint('Remote participant left: ${participant.identity}');
    } else if (event is LocalTrackPublishedEvent) {
      final publication = event.publication;
      if (publication.kind == TrackType.audio) {
        _monitorAudioQuality(publication);
      }
    } else if (event is RemoteTrackPublishedEvent) {
      final publication = event.publication;
      if (publication.kind == TrackType.audio) {
        _monitorAudioQuality(publication);
      }
    }
  }

  void _monitorAudioQuality(TrackPublication publication) {
    if (publication is LocalTrackPublication) {
      // Monitor local track stats for quality changes
      // LiveKit provides quality via TrackPublication.quality
      final quality = _mapLiveKitQuality(publication.quality);
      _eventsController.add(RtcEvent.qualityChange(quality));
    } else if (publication is RemoteTrackPublication) {
      final quality = _mapLiveKitQuality(publication.quality);
      _eventsController.add(RtcEvent.qualityChange(quality));
    }
  }

  RtcQuality _mapLiveKitQuality(RetrySchedule schedule) {
    // Map LiveKit connection quality to our RtcQuality enum
    switch (schedule) {
      case RetrySchedule.fast:
        return RtcQuality.good;
      case RetrySchedule.normal:
        return RtcQuality.fair;
      case RetrySchedule.slow:
        return RtcQuality.poor;
      case RetrySchedule.unknown:
        return RtcQuality.unknown;
    }
  }

  /// Extracts the LiveKit WebSocket URL from a JWT token's video URL claim,
  /// or falls back to a default URL derived from the token's domain.
  String _liveKitUrlFromToken(String token) {
    // In production, the server should return the LiveKit URL alongside
    // the token. Here we use a placeholder that the app config will override.
    // The LiveKit client uses the `wss://` URL embedded in the token or
    // passed as the first argument to connect().
    // Default: use the same host as the API.
    return 'wss://rtc.voicematch.app';
  }

  @override
  void dispose() {
    leaveRoom();
    _eventsController.close();
    super.dispose();
  }
}
