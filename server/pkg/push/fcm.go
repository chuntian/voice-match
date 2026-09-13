package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const fcmEndpoint = "https://fcm.googleapis.com/fcm/send"

// FCMPush Android FCM 推送（Legacy HTTP API）
type FCMPush struct {
	serverKey string
	endpoint  string
	client    *http.Client
}

// NewFCMPush 构造 FCM 推送客户端
func NewFCMPush(serverKey string) *FCMPush {
	return &FCMPush{
		serverKey: serverKey,
		endpoint:  fcmEndpoint,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// fcmPayload FCM 请求体
type fcmPayload struct {
	To           string            `json:"to"`
	Notification fcmNotification   `json:"notification,omitempty"`
	Data         map[string]string `json:"data"`
	Priority     string            `json:"priority"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (f *FCMPush) post(p fcmPayload) error {
	if f.serverKey == "" {
		return fmt.Errorf("fcm server key not configured")
	}
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, f.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "key="+f.serverKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("fcm respond status %d", resp.StatusCode)
	}
	return nil
}

// SendCallInvite 发送呼叫邀请（notification + data）
func (f *FCMPush) SendCallInvite(deviceToken, title, body, callID, callerName string) error {
	return f.post(fcmPayload{
		To:           deviceToken,
		Notification: fcmNotification{Title: title, Body: body},
		Data: map[string]string{
			"event":       "call_invite",
			"call_id":     callID,
			"caller_name": callerName,
		},
		Priority: "high",
	})
}

// SendCancel 发送取消（data-only，高优先级唤醒）
func (f *FCMPush) SendCancel(deviceToken, callID string) error {
	return f.post(fcmPayload{
		To: deviceToken,
		Data: map[string]string{
			"event":   "call_cancel",
			"call_id": callID,
		},
		Priority: "high",
	})
}
