// Package protocol defines all signaling message types used between
// the client and the voice-match signal server.
//
// Every message is wrapped in a Message envelope:
//
//	{"type": "hello", "payload": {...}}
//
// Payload is a JSON RawMessage that callers unmarshal into the
// strongly-typed struct for the given Type.
package protocol

import (
	"encoding/json"
	"fmt"
)

// ---- Message type constants ----

const (
	TypeHello         = "hello"
	TypeHelloAck      = "hello_ack"
	TypeHeartbeat     = "heartbeat"
	TypeHeartbeatAck  = "heartbeat_ack"
	TypeJoinPool      = "join_pool"
	TypeJoinPoolAck   = "join_pool_ack"
	TypeLeavePool     = "leave_pool"
	TypeLeavePoolAck  = "leave_pool_ack"
	TypeMatchFound    = "match_found"
	TypeCallInvite    = "call_invite"
	TypeCallAccept    = "call_accept"
	TypeCallReject    = "call_reject"
	TypeCallCancel    = "call_cancel"
	TypeCallEnd       = "call_end"
	TypeIceCandidate  = "ice_candidate"
	TypeError         = "error"
)

// Match-type constants used in JoinPool.MatchType.
const (
	MatchTypeRandom      = "random"
	MatchTypeCity        = "city"
	MatchTypeDestination = "destination"
)

// Call state constants.
const (
	CallStateIdle      = "idle"
	CallStateRinging   = "ringing"
	CallStateConnected = "connected"
	CallStateEnded     = "ended"
)

// ---- Envelope ----

// Message is the on-the-wire envelope for every signaling frame.
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Marshal serializes a typed payload struct into a Message envelope.
// It returns an error if the payload cannot be marshaled to JSON.
func Marshal(msgType string, payload interface{}) ([]byte, error) {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload for %s: %w", msgType, err)
		}
		raw = b
	}
	env := Message{Type: msgType, Payload: raw}
	return json.Marshal(env)
}

// Unmarshal parses a raw JSON frame into the envelope, leaving the
// Payload as RawMessage for the caller to decode by Type.
func Unmarshal(data []byte) (*Message, error) {
	var env Message
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	if env.Type == "" {
		return nil, fmt.Errorf("message type is empty")
	}
	return &env, nil
}

// UnmarshalPayload decodes the envelope Payload into v.
func UnmarshalPayload(env *Message, v interface{}) error {
	if len(env.Payload) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Payload, v); err != nil {
		return fmt.Errorf("unmarshal payload for %s: %w", env.Type, err)
	}
	return nil
}

// ---- Payload structs ----

// Hello is sent by the client immediately after the WebSocket upgrade.
type Hello struct {
	UserID     string `json:"user_id"`
	Token      string `json:"token"`
	DeviceID   string `json:"device_id"`
	AppVersion string `json:"app_version"`
}

// HelloAck is sent by the server after successful authentication.
type HelloAck struct {
	Status     string `json:"status"`
	ServerTime int64  `json:"server_time"`
}

// Heartbeat keeps the connection alive.
type Heartbeat struct {
	Timestamp int64 `json:"timestamp"`
}

// HeartbeatAck is the server response to Heartbeat.
type HeartbeatAck struct {
	Timestamp  int64 `json:"timestamp"`
	ServerTime int64 `json:"server_time"`
}

// JoinPool requests entry into a matching pool.
type JoinPool struct {
	MatchType        string   `json:"match_type"`
	City             string   `json:"city,omitempty"`
	Geohash          string   `json:"geohash,omitempty"`
	Destination      string   `json:"destination,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	GenderPreference string   `json:"gender_preference,omitempty"`
}

// JoinPoolAck confirms pool entry.
type JoinPoolAck struct {
	Status   string `json:"status"`
	PoolSize int    `json:"pool_size"`
	WaitTime int    `json:"wait_time"`
}

// LeavePool asks the server to remove the user from the pool.
type LeavePool struct{}

// LeavePoolAck confirms pool exit.
type LeavePoolAck struct {
	Status string `json:"status"`
}

// MatchFound notifies both sides that a match was made.
type MatchFound struct {
	CallID           string `json:"call_id"`
	MatchedUserID    string `json:"matched_user_id"`
	MatchedNickname  string `json:"matched_nickname"`
	MatchedAvatar    string `json:"matched_avatar"`
	MatchType        string `json:"match_type"`
}

// CallInvite is sent to the callee when the caller initiates a call.
type CallInvite struct {
	CallID         string `json:"call_id"`
	CallerID       string `json:"caller_id"`
	CallerNickname string `json:"caller_nickname"`
	CallerAvatar   string `json:"caller_avatar"`
	RoomName       string `json:"room_name"`
	RtcToken       string `json:"rtc_token"`
	ExpiresAt      int64  `json:"expires_at"`
}

// CallAccept is sent by the callee to accept the call.
type CallAccept struct {
	CallID string `json:"call_id"`
}

// CallReject is sent by the callee to reject the call.
type CallReject struct {
	CallID string `json:"call_id"`
	Reason string `json:"reason,omitempty"`
}

// CallCancel is sent by the caller to cancel a ringing call.
type CallCancel struct {
	CallID string `json:"call_id"`
	Reason string `json:"reason,omitempty"`
}

// CallEnd terminates an active call.
type CallEnd struct {
	CallID   string `json:"call_id"`
	Reason   string `json:"reason,omitempty"`
	Duration int    `json:"duration,omitempty"`
}

// IceCandidate carries a WebRTC ICE candidate between peers via the server.
type IceCandidate struct {
	CallID         string `json:"call_id"`
	TargetUserID   string `json:"target_user_id"`
	Candidate      string `json:"candidate"`
	SdpMid         string `json:"sdp_mid,omitempty"`
	SdpMLineIndex  int    `json:"sdp_m_line_index"`
}

// ErrorMsg is sent by the server when something goes wrong.
type ErrorMsg struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
