import Flutter
import UIKit
import PushKit
import CallKit
import flutter_callkit_incoming

@main
@objc class AppDelegate: FlutterAppDelegate, PKPushRegistryDelegate {

    var pushRegistry: PKPushRegistry?

    override func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
    ) -> Bool {
        // Register with PushKit for VoIP push notifications
        setupPushKit()

        // Register Flutter plugins
        GeneratedPluginRegistrant.register(with: self)

        return super.application(application, didFinishLaunchingWithOptions: launchOptions)
    }

    // MARK: - PushKit

    private func setupPushKit() {
        pushRegistry = PKPushRegistry(queue: DispatchQueue.main)
        pushRegistry?.delegate = self
        pushRegistry?.desiredPushTypes = Set([PKPushType.voIP])
    }

    func pushRegistry(_ registry: PKPushRegistry, didUpdate credentials: PKPushCredentials, for type: PKPushType) {
        let token = credentials.token.map { String(format: "%02x", $0) }.joined()
        NSLog("PushKit VoIP token: \(token)")
        // Send token to server so the backend can deliver VoIP pushes
        // via APNs to wake the app and display the incoming call UI.
        // The Dart side listens for this via the method channel.
        if let controller = window?.rootViewController as? FlutterViewController {
            let channel = FlutterMethodChannel(
                name: "voicematch/pushkit",
                binaryMessenger: controller.binaryMessenger
            )
            channel.invokeMethod("onVoipToken", arguments: ["token": token])
        }
    }

    func pushRegistry(_ registry: PKPushRegistry, didReceiveIncomingPushWith payload: PKPushPayload, for type: PKPushType, completion: @escaping () -> Void) {
        NSLog("PushKit didReceiveIncomingPushWith: \(payload.dictionaryPayload)")

        // Extract call info from the push payload
        let aps = payload.dictionaryPayload as? [String: Any]
        let data = aps?["data"] as? [String: Any] ?? [:]

        let callId = data["call_id"] as? String ?? UUID().uuidString
        let callerName = data["caller_name"] as? String ?? "未知来电"
        let callerAvatar = data["caller_avatar"] as? String ?? ""

        // Display the incoming call via flutter_callkit_incoming
        let callkitParams = [
            "id": callId,
            "nameCaller": callerName,
            "avatar": callerAvatar,
            "handle": "",
            "type": 0,
            "textAccept": "接听",
            "textDecline": "挂断",
            "duration": 30000,
            "ios": [
                "handleType": "generic",
                "supportsVideo": false
            ]
        ] as [String: Any]

        FlutterCallkitIncoming.showCallkitIncoming(callkitParams)

        completion()
    }

    func pushRegistry(_ registry: PKPushRegistry, didInvalidatePushTokenFor type: PKPushType) {
        NSLog("PushKit invalidated token for type: \(type.rawValue)")
    }

    // MARK: - App lifecycle

    override func applicationDidEnterBackground(_ application: UIApplication) {
        // Ensure the audio session remains active during background calls
        super.applicationDidEnterBackground(application)
    }

    override func applicationWillTerminate(_ application: UIApplication) {
        // Clean up CallKit when the app is terminated
        FlutterCallkitIncoming.endAllCalls()
        super.applicationWillTerminate(application)
    }
}
