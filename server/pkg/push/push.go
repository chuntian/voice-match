// Package push 封装 FCM / APNs 推送，用于呼叫邀请与取消的下行通知。
package push

import (
	"context"
	"fmt"
)

// PushService 推送服务抽象
type PushService interface {
	// SendCallInvite 发送语音呼叫邀请
	SendCallInvite(deviceToken, title, body, callID, callerName string) error
	// SendCancel 发送呼叫取消
	SendCancel(deviceToken, callID string) error
}

// DeviceStore 设备注册存储（Redis hash）
type DeviceStore interface {
	// RegisterDevice 写入 user:devices:{user_id} field=deviceToken value=platform
	RegisterDevice(ctx context.Context, userID, deviceToken, platform string) error
	// GetDevice 查询某用户某设备 token 对应的平台
	GetDevice(ctx context.Context, userID, deviceToken string) (string, error)
}

// Platform 设备平台
type Platform string

const (
	PlatformAndroid Platform = "android"
	PlatformIOS     Platform = "ios"
)

// MultiPush 根据设备平台路由到 FCM 或 APNs
type MultiPush struct {
	fcm PushService
	apn PushService
}

// NewMultiPush 构造多通道推送
func NewMultiPush(fcm, apn PushService) *MultiPush {
	return &MultiPush{fcm: fcm, apn: apn}
}

// SendCallInvite 按平台分发呼叫邀请
func (m *MultiPush) SendCallInvite(deviceToken, title, body, callID, callerName string, platform Platform) error {
	switch platform {
	case PlatformAndroid:
		if m.fcm == nil {
			return fmt.Errorf("fcm not configured")
		}
		return m.fcm.SendCallInvite(deviceToken, title, body, callID, callerName)
	case PlatformIOS:
		if m.apn == nil {
			return fmt.Errorf("apns not configured")
		}
		return m.apn.SendCallInvite(deviceToken, title, body, callID, callerName)
	default:
		return fmt.Errorf("unknown platform %q", platform)
	}
}

// SendCancel 按平台分发取消
func (m *MultiPush) SendCancel(deviceToken, callID string, platform Platform) error {
	switch platform {
	case PlatformAndroid:
		if m.fcm == nil {
			return fmt.Errorf("fcm not configured")
		}
		return m.fcm.SendCancel(deviceToken, callID)
	case PlatformIOS:
		if m.apn == nil {
			return fmt.Errorf("apns not configured")
		}
		return m.apn.SendCancel(deviceToken, callID)
	default:
		return fmt.Errorf("unknown platform %q", platform)
	}
}

// RegisterDevice 注册设备 token 到 Redis
func (m *MultiPush) RegisterDevice(ctx context.Context, store DeviceStore, userID, deviceToken, platform string) error {
	if userID == "" || deviceToken == "" || platform == "" {
		return fmt.Errorf("invalid device register args")
	}
	return store.RegisterDevice(ctx, userID, deviceToken, platform)
}
