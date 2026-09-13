package signal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	apperrors "github.com/voicematch/voice-match/shared/errors"
	"github.com/voicematch/voice-match/shared/protocol"
)

// UserValidator abstracts the user service so the signal package does
// not import it directly (avoids circular dependency).
type UserValidator interface {
	// ValidateToken verifies the token and returns the user ID if valid.
	ValidateToken(userID, token string) (valid bool, err error)
	// GetUserProfile returns public profile info for a user.
	GetUserProfile(userID string) (nickname, avatar string, err error)
}

// MatchService abstracts the match pool operations.
type MatchService interface {
	JoinPool(userID, matchType, city, geohash, destination string, tags []string, genderPreference string) (poolSize, waitTime int, err error)
	LeavePool(userID, matchType string) error
}

// RTCService abstracts RTC token / room name generation.
type RTCService interface {
	CreateRoom(callID, callerID, calleeID string) (roomName, token string, expiresAt int64, err error)
}

// Handler holds dependencies for the WebSocket message handler.
type Handler struct {
	hub      *Hub
	calls    *CallManager
	users    UserValidator
	match    MatchService
	rtc      RTCService

	// pendingHellos stores hello payloads keyed by temporary connection
	// until authentication completes. We use a map keyed by conn pointer.
	pendingMu    sync.Mutex
	pendingConns map[*websocket.Conn]*pendingHello
}

type pendingHello struct {
	userID   string
	deviceID string
}

// NewHandler creates a new signal handler.
func NewHandler(
	hub *Hub,
	calls *CallManager,
	users UserValidator,
	match MatchService,
	rtc RTCService,
) *Handler {
	return &Handler{
		hub:          hub,
		calls:        calls,
		users:        users,
		match:        match,
		rtc:          rtc,
		pendingConns: make(map[*websocket.Conn]*pendingHello),
	}
}

// upgrader upgrades HTTP connections to WebSocket.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// In production this should check the Origin header.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeHTTP upgrades an HTTP request to WebSocket and handles the
// connection lifecycle. This is the main entry point for /ws.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[handler] upgrade failed: %v", err)
		return
	}

	// The first message must be a hello. We do a special read here.
	_, data, err := conn.ReadMessage()
	if err != nil {
		_ = conn.Close()
		log.Printf("[handler] initial read failed: %v", err)
		return
	}

	env, err := protocol.Unmarshal(data)
	if err != nil {
		h.sendError(conn, apperrors.ErrInvalidParams, "invalid message envelope", err.Error())
		_ = conn.Close()
		return
	}

	if env.Type != protocol.TypeHello {
		h.sendError(conn, apperrors.ErrInvalidParams, "first message must be hello", "")
		_ = conn.Close()
		return
	}

	var hello protocol.Hello
	if err := protocol.UnmarshalPayload(env, &hello); err != nil {
		h.sendError(conn, apperrors.ErrInvalidParams, "invalid hello payload", err.Error())
		_ = conn.Close()
		return
	}

	// Validate token via user service.
	valid, err := h.users.ValidateToken(hello.UserID, hello.Token)
	if err != nil || !valid {
		h.sendError(conn, apperrors.ErrInvalidToken, "invalid token", "")
		_ = conn.Close()
		return
	}

	// Authentication succeeded.
	client := NewClient(h.hub, conn, hello.UserID, hello.DeviceID)
	h.hub.Register(client)
	client.MarkAuthenticated()

	// Send hello_ack.
	ack := protocol.HelloAck{Status: "ok", ServerTime: time.Now().Unix()}
	if err := h.sendEnvelope(conn, protocol.TypeHelloAck, ack); err != nil {
		_ = conn.Close()
		return
	}

	log.Printf("[handler] user=%s connected", hello.UserID)

	// Set up message routing for this client.
	h.hub.SetMessageHandler(func(userID string, raw []byte) {
		h.dispatch(userID, raw)
	})

	// Start pumps.
	go client.writePump()
	client.readPump()
}

// dispatch routes an inbound message to the appropriate handler.
func (h *Handler) dispatch(userID string, raw []byte) {
	env, err := protocol.Unmarshal(raw)
	if err != nil {
		log.Printf("[handler] unmarshal error user=%s: %v", userID, err)
		h.sendErrorToUser(userID, apperrors.ErrInvalidParams, "invalid envelope", err.Error())
		return
	}

	switch env.Type {
	case protocol.TypeHeartbeat:
		h.handleHeartbeat(userID, env)
	case protocol.TypeJoinPool:
		h.handleJoinPool(userID, env)
	case protocol.TypeLeavePool:
		h.handleLeavePool(userID, env)
	case protocol.TypeCallAccept:
		h.handleCallAccept(userID, env)
	case protocol.TypeCallReject:
		h.handleCallReject(userID, env)
	case protocol.TypeCallCancel:
		h.handleCallCancel(userID, env)
	case protocol.TypeCallEnd:
		h.handleCallEnd(userID, env)
	case protocol.TypeIceCandidate:
		h.handleIceCandidate(userID, env)
	default:
		h.sendErrorToUser(userID, apperrors.ErrInvalidParams,
			fmt.Sprintf("unknown message type: %s", env.Type), "")
	}
}

// ---- Message handlers ----

func (h *Handler) handleHeartbeat(userID string, env *protocol.Message) {
	var hb protocol.Heartbeat
	_ = protocol.UnmarshalPayload(env, &hb)

	// Update last heartbeat time on the client.
	h.hub.mu.RLock()
	client := h.hub.clients[userID]
	h.hub.mu.RUnlock()
	if client != nil {
		client.UpdateHeartbeat(hb.Timestamp)
	}

	ack := protocol.HeartbeatAck{
		Timestamp:  hb.Timestamp,
		ServerTime: time.Now().UnixMilli(),
	}
	_ = h.sendEnvelopeToUser(userID, protocol.TypeHeartbeatAck, ack)
}

func (h *Handler) handleJoinPool(userID string, env *protocol.Message) {
	var jp protocol.JoinPool
	if err := protocol.UnmarshalPayload(env, &jp); err != nil {
		h.sendErrorToUser(userID, apperrors.ErrInvalidParams, "invalid join_pool payload", err.Error())
		return
	}

	poolSize, waitTime, err := h.match.JoinPool(userID, jp.MatchType, jp.City, jp.Geohash,
		jp.Destination, jp.Tags, jp.GenderPreference)
	if err != nil {
		h.sendErrorToUser(userID, apperrors.ErrInternal, "join pool failed", err.Error())
		return
	}

	ack := protocol.JoinPoolAck{
		Status:   "ok",
		PoolSize: poolSize,
		WaitTime: waitTime,
	}
	_ = h.sendEnvelopeToUser(userID, protocol.TypeJoinPoolAck, ack)
}

func (h *Handler) handleLeavePool(userID string, env *protocol.Message) {
	// LeavePool has no payload; match type is inferred from the pool
	// the user is currently in. We attempt to leave all pools.
	err := h.match.LeavePool(userID, "")
	if err != nil {
		h.sendErrorToUser(userID, apperrors.ErrNotInPool, "leave pool failed", err.Error())
		return
	}
	ack := protocol.LeavePoolAck{Status: "ok"}
	_ = h.sendEnvelopeToUser(userID, protocol.TypeLeavePoolAck, ack)
}

func (h *Handler) handleCallAccept(userID string, env *protocol.Message) {
	var ca protocol.CallAccept
	if err := protocol.UnmarshalPayload(env, &ca); err != nil {
		h.sendErrorToUser(userID, apperrors.ErrInvalidParams, "invalid call_accept", err.Error())
		return
	}

	sess, err := h.calls.Accept(ca.CallID, userID)
	if err != nil {
		h.sendErrorToUser(userID, apperrors.ErrCallRejected, "accept failed", err.Error())
		return
	}

	// Notify the caller that the call was accepted.
	_ = h.sendEnvelopeToUser(sess.CallerID, protocol.TypeCallAccept, ca)
}

func (h *Handler) handleCallReject(userID string, env *protocol.Message) {
	var cr protocol.CallReject
	if err := protocol.UnmarshalPayload(env, &cr); err != nil {
		h.sendErrorToUser(userID, apperrors.ErrInvalidParams, "invalid call_reject", err.Error())
		return
	}

	sess, err := h.calls.Reject(cr.CallID, userID, cr.Reason)
	if err != nil {
		h.sendErrorToUser(userID, apperrors.ErrCallNotFound, "reject failed", err.Error())
		return
	}

	// Notify the caller.
	_ = h.sendEnvelopeToUser(sess.CallerID, protocol.TypeCallReject, cr)

	// Clean up after notification.
	go func() {
		time.Sleep(500 * time.Millisecond)
		h.calls.Remove(cr.CallID)
	}()
}

func (h *Handler) handleCallCancel(userID string, env *protocol.Message) {
	var cc protocol.CallCancel
	if err := protocol.UnmarshalPayload(env, &cc); err != nil {
		h.sendErrorToUser(userID, apperrors.ErrInvalidParams, "invalid call_cancel", err.Error())
		return
	}

	sess, ok := h.calls.Get(cc.CallID)
	if !ok {
		h.sendErrorToUser(userID, apperrors.ErrCallNotFound, "call not found", "")
		return
	}

	if err := h.calls.Cancel(cc.CallID, cc.Reason); err != nil {
		h.sendErrorToUser(userID, apperrors.ErrInternal, "cancel failed", err.Error())
		return
	}

	// Notify the callee.
	_ = h.sendEnvelopeToUser(sess.CalleeID, protocol.TypeCallCancel, cc)

	go func() {
		time.Sleep(500 * time.Millisecond)
		h.calls.Remove(cc.CallID)
	}()
}

func (h *Handler) handleCallEnd(userID string, env *protocol.Message) {
	var ce protocol.CallEnd
	if err := protocol.UnmarshalPayload(env, &ce); err != nil {
		h.sendErrorToUser(userID, apperrors.ErrInvalidParams, "invalid call_end", err.Error())
		return
	}

	sess, duration, err := h.calls.End(ce.CallID, userID, ce.Reason)
	if err != nil {
		h.sendErrorToUser(userID, apperrors.ErrCallNotFound, "end failed", err.Error())
		return
	}

	ce.Duration = duration

	// Notify the other party.
	otherID := sess.CallerID
	if userID == sess.CallerID {
		otherID = sess.CalleeID
	}
	_ = h.sendEnvelopeToUser(otherID, protocol.TypeCallEnd, ce)

	go func() {
		time.Sleep(time.Second)
		h.calls.Remove(ce.CallID)
	}()
}

func (h *Handler) handleIceCandidate(userID string, env *protocol.Message) {
	var ice protocol.IceCandidate
	if err := protocol.UnmarshalPayload(env, &ice); err != nil {
		h.sendErrorToUser(userID, apperrors.ErrInvalidParams, "invalid ice_candidate", err.Error())
		return
	}

	// Transparent relay to the target user.
	_ = h.sendEnvelopeToUser(ice.TargetUserID, protocol.TypeIceCandidate, ice)
}

// ---- Helpers ----

func (h *Handler) sendEnvelope(conn *websocket.Conn, msgType string, payload interface{}) error {
	data, err := protocol.Marshal(msgType, payload)
	if err != nil {
		return err
	}
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteMessage(websocket.TextMessage, data)
}

func (h *Handler) sendEnvelopeToUser(userID string, msgType string, payload interface{}) error {
	data, err := protocol.Marshal(msgType, payload)
	if err != nil {
		return err
	}
	if !h.hub.SendToUser(userID, data) {
		log.Printf("[handler] user %s offline, cannot deliver %s", userID, msgType)
	}
	return nil
}

func (h *Handler) sendError(conn *websocket.Conn, code apperrors.ErrorCode, message, details string) {
	errMsg := protocol.ErrorMsg{
		Code:    code.Int(),
		Message: message,
		Details: details,
	}
	data, _ := protocol.Marshal(protocol.TypeError, errMsg)
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	_ = conn.WriteMessage(websocket.TextMessage, data)
}

func (h *Handler) sendErrorToUser(userID string, code apperrors.ErrorCode, message, details string) {
	errMsg := protocol.ErrorMsg{
		Code:    code.Int(),
		Message: message,
		Details: details,
	}
	data, _ := protocol.Marshal(protocol.TypeError, errMsg)
	h.hub.SendToUser(userID, data)
}

// Marshal is a small wrapper for convenience.
func marshalJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
