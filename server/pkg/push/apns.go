package push

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// apnsDefaultHost APNs provider API（PushKit/VoIP）端点
const apnsDefaultHost = "https://api.pushkit.apple.com"

// APNsPush iOS APNs 推送（HTTP/2 + .p8 JWT）
type APNsPush struct {
	keyID   string
	teamID  string
	topic   string
	sandbox bool
	host    string
	privKey *ecdsa.PrivateKey
	client  *http.Client

	mu        sync.Mutex
	cachedJWT string
	jwtAt     time.Time
}

// APNsConfig 构造参数
type APNsConfig struct {
	KeyID        string
	TeamID       string
	PrivateKey   *ecdsa.PrivateKey
	Topic        string
	Sandbox      bool
	HostOverride string // 可选，覆盖默认 host
}

// NewAPNsPush 从配置构造 APNs 客户端
func NewAPNsPush(cfg APNsConfig) (*APNsPush, error) {
	if cfg.PrivateKey == nil {
		return nil, errors.New("apns private key required")
	}
	if cfg.KeyID == "" || cfg.TeamID == "" || cfg.Topic == "" {
		return nil, errors.New("apns key_id/team_id/topic required")
	}
	host := cfg.HostOverride
	if host == "" {
		host = apnsDefaultHost
	}
	return &APNsPush{
		keyID:   cfg.KeyID,
		teamID:  cfg.TeamID,
		topic:   cfg.Topic,
		sandbox: cfg.Sandbox,
		host:    host,
		privKey: cfg.PrivateKey,
		client:  &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// LoadP8Key 从磁盘加载 Apple .p8 私钥（PKCS#8 PEM, EC P-256）
func LoadP8Key(path string) (*ecdsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("invalid p8 pem")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("p8 is not EC key")
	}
	return ecKey, nil
}

// providerJWT 生成 APNs provider token（ES256），缓存 50 分钟
func (a *APNsPush) providerJWT() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cachedJWT != "" && time.Since(a.jwtAt) < 50*time.Minute {
		return a.cachedJWT, nil
	}
	claims := jwt.RegisteredClaims{
		Issuer:   a.teamID,
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = a.keyID
	signed, err := tok.SignedString(a.privKey)
	if err != nil {
		return "", err
	}
	a.cachedJWT = signed
	a.jwtAt = time.Now()
	return signed, nil
}

// apnsPayload APNs JSON payload（静音/VoIP，content-available）
type apnsPayload struct {
	APS        apnsAPS `json:"aps"`
	CallID     string  `json:"call_id,omitempty"`
	CallerName string  `json:"caller_name,omitempty"`
	Event      string  `json:"event,omitempty"`
}

type apnsAPS struct {
	ContentAvailable int    `json:"content-available"`
	Category         string `json:"category,omitempty"`
}

func (a *APNsPush) send(deviceToken string, p apnsPayload, pushType string) error {
	jwtTok, err := a.providerJWT()
	if err != nil {
		return err
	}
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	url := a.host + "/3/device/" + deviceToken
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "bearer "+jwtTok)
	req.Header.Set("apns-topic", a.topic)
	req.Header.Set("apns-push-type", pushType)
	req.Header.Set("apns-priority", "10")
	req.Header.Set("content-type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apns status %d", resp.StatusCode)
	}
	return nil
}

// SendCallInvite 发送 VoIP 呼叫邀请（静音唤醒 CallKit）
func (a *APNsPush) SendCallInvite(deviceToken, title, body, callID, callerName string) error {
	_ = title
	_ = body
	return a.send(deviceToken, apnsPayload{
		APS:        apnsAPS{ContentAvailable: 1, Category: "INCOMING_CALL"},
		CallID:     callID,
		CallerName: callerName,
		Event:      "call_invite",
	}, "voip")
}

// SendCancel 发送呼叫取消
func (a *APNsPush) SendCancel(deviceToken, callID string) error {
	return a.send(deviceToken, apnsPayload{
		APS:    apnsAPS{ContentAvailable: 1},
		CallID: callID,
		Event:  "call_cancel",
	}, "voip")
}
