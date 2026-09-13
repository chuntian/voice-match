package user

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ---------- 存储接口（由内部 store 包实现，这里由消费方定义） ----------

// MySQLStore 用户相关的 MySQL 持久化接口
type MySQLStore interface {
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByAppleID(ctx context.Context, appleID string) (*User, error)
	GetUserByPhone(ctx context.Context, phone string) (*User, error)
	CreateUser(ctx context.Context, u *User) error
	UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) error
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p *Preferences) error
}

// RedisStore 用户相关的 Redis 接口（短信验证码、设备态等）
type RedisStore interface {
	SetCode(ctx context.Context, phone, code string, ttl time.Duration) error
	GetCode(ctx context.Context, phone string) (string, error)
	DelCode(ctx context.Context, phone string) error
}

// ErrNotFound 记录不存在
var ErrNotFound = errors.New("user: record not found")

// ---------- 配置 ----------

// Config 服务依赖配置
type Config struct {
	JWTSecret     string
	JWTExpire     time.Duration
	AppleClientID string // Apple Bundle ID / Service ID（aud 校验）
	DevMode       bool   // 开发模式下短信验证码固定 888888
}

// Service 用户服务
type Service struct {
	mysql MySQLStore
	rdb   RedisStore
	cfg   Config

	// Apple JWKS 缓存
	jwksMu    sync.RWMutex
	jwksCache map[string]*rsa.PublicKey // kid -> key
	jwksAt    time.Time
	jwksTTL   time.Duration

	httpClient *http.Client
}

// NewService 构造用户服务
func NewService(mysql MySQLStore, rdb RedisStore, cfg Config) *Service {
	if cfg.JWTExpire <= 0 {
		cfg.JWTExpire = 7 * 24 * time.Hour
	}
	return &Service{
		mysql:      mysql,
		rdb:        rdb,
		cfg:        cfg,
		jwksCache:  make(map[string]*rsa.PublicKey),
		jwksTTL:    24 * time.Hour,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ---------- Apple JWKS ----------

const appleJWKSURL = "https://appleid.apple.com/auth/keys"

// appleJWKS 结构
type appleJWKS struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Use string `json:"use"`
		Alg string `json:"alg"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

// fetchAppleKeys 拉取并刷新 JWKS 缓存
func (s *Service) fetchAppleKeys(ctx context.Context) error {
	s.jwksMu.RLock()
	if time.Since(s.jwksAt) < s.jwksTTL && len(s.jwksCache) > 0 {
		s.jwksMu.RUnlock()
		return nil
	}
	s.jwksMu.RUnlock()

	s.jwksMu.Lock()
	defer s.jwksMu.Unlock()
	// double check
	if time.Since(s.jwksAt) < s.jwksTTL && len(s.jwksCache) > 0 {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleJWKSURL, nil)
	if err != nil {
		return err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apple jwks status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var jwks appleJWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return err
	}
	next := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		nb, err := decodeBase64URL(k.N)
		if err != nil {
			continue
		}
		eb, err := decodeBase64URL(k.E)
		if err != nil {
			continue
		}
		// exponent 一般是 65537，4 字节
		exp := 0
		for _, b := range eb {
			exp = exp<<8 | int(b)
		}
		pub := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nb),
			E: exp,
		}
		next[k.Kid] = pub
	}
	if len(next) == 0 {
		return errors.New("apple jwks empty keys")
	}
	s.jwksCache = next
	s.jwksAt = time.Now()
	return nil
}

// decodeBase64URL 无填充 base64url 解码
func decodeBase64URL(s string) ([]byte, error) {
	return jwt.NewParser().DecodeSegment(s)
}

// appleClaims Apple identityToken 的声明
type appleClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// keyFuncForApple 构造 jwt keyfunc
func (s *Service) keyFuncForApple(ctx context.Context) jwt.Keyfunc {
	return func(tok *jwt.Token) (interface{}, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", tok.Header["alg"])
		}
		kid, _ := tok.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing kid")
		}
		s.jwksMu.RLock()
		pub, ok := s.jwksCache[kid]
		s.jwksMu.RUnlock()
		if ok {
			return pub, nil
		}
		// 缓存未命中，刷新
		if err := s.fetchAppleKeys(ctx); err != nil {
			return nil, err
		}
		s.jwksMu.RLock()
		pub, ok = s.jwksCache[kid]
		s.jwksMu.RUnlock()
		if !ok {
			return nil, fmt.Errorf("kid %s not found", kid)
		}
		return pub, nil
	}
}

// LoginWithApple 使用 Apple identityToken 登录，查找或创建用户
func (s *Service) LoginWithApple(ctx context.Context, identityToken string) (*User, error) {
	if identityToken == "" {
		return nil, errors.New("empty identity token")
	}
	if err := s.fetchAppleKeys(ctx); err != nil {
		return nil, fmt.Errorf("load apple keys: %w", err)
	}

	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer("https://appleid.apple.com"),
		jwt.WithExpirationRequired(),
	}
	if s.cfg.AppleClientID != "" {
		opts = append(opts, jwt.WithAudience(s.cfg.AppleClientID))
	}

	claims := &appleClaims{}
	_, err := jwt.ParseWithClaims(identityToken, claims, s.keyFuncForApple(ctx), opts...)
	if err != nil {
		return nil, fmt.Errorf("invalid apple token: %w", err)
	}
	if claims.Subject == "" {
		return nil, errors.New("apple token missing sub")
	}

	u, err := s.mysql.GetUserByAppleID(ctx, claims.Subject)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	// 创建新用户
	u = &User{
		AppleID:  claims.Subject,
		Nickname: "Apple用户",
		Gender:   GenderUnknown,
	}
	if err := s.mysql.CreateUser(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// ---------- 手机号登录 ----------

const devSMSCode = "888888"

// SendPhoneCode 发送短信验证码（开发模式直接固定 888888）
func (s *Service) SendPhoneCode(ctx context.Context, phone string) error {
	if phone == "" {
		return errors.New("empty phone")
	}
	code := devSMSCode
	if !s.cfg.DevMode {
		// 生产模式：调用真实短信服务（此处为接口占位，实际应注入 SMS 提供方）
		smsCode, err := s.produceProdCode(phone)
		if err != nil {
			return err
		}
		code = smsCode
	}
	return s.rdb.SetCode(ctx, phone, code, 5*time.Minute)
}

// produceProdCode 生产环境生成并下发验证码占位
func (s *Service) produceProdCode(phone string) (string, error) {
	// TODO(prod): 接入阿里云/腾讯云短信 SDK 并返回生成的 6 位码
	// 当前实现：生成本地随机 6 位码，由上层通过日志/通道下发。
	code, err := generateNumericCode(6)
	if err != nil {
		return "", err
	}
	return code, nil
}

// LoginWithPhone 校验验证码并登录
func (s *Service) LoginWithPhone(ctx context.Context, phone, code string) (*User, error) {
	if phone == "" || code == "" {
		return nil, errors.New("phone and code required")
	}
	if s.cfg.DevMode && code == devSMSCode {
		// 开发模式短路
	} else {
		stored, err := s.rdb.GetCode(ctx, phone)
		if err != nil {
			return nil, fmt.Errorf("verify code: %w", err)
		}
		if stored == "" || stored != code {
			return nil, errors.New("invalid or expired code")
		}
	}
	_ = s.rdb.DelCode(ctx, phone)

	u, err := s.mysql.GetUserByPhone(ctx, phone)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	u = &User{
		Phone:    phone,
		Nickname: "用户" + phone[len(phone)-4:],
		Gender:   GenderUnknown,
	}
	if err := s.mysql.CreateUser(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// ---------- 资料 / 偏好 ----------

// GetProfile 获取用户资料
func (s *Service) GetProfile(ctx context.Context, userID string) (*User, error) {
	return s.mysql.GetUserByID(ctx, userID)
}

// UpdateProfile 更新用户资料
func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*User, error) {
	if err := s.mysql.UpdateProfile(ctx, userID, req); err != nil {
		return nil, err
	}
	return s.mysql.GetUserByID(ctx, userID)
}

// GetPreferences 获取匹配偏好
func (s *Service) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	return s.mysql.GetPreferences(ctx, userID)
}

// UpdatePreferences 更新匹配偏好
func (s *Service) UpdatePreferences(ctx context.Context, userID string, p *Preferences) (*Preferences, error) {
	if p == nil {
		return nil, errors.New("nil preferences")
	}
	if err := s.mysql.UpsertPreferences(ctx, userID, p); err != nil {
		return nil, err
	}
	return s.mysql.GetPreferences(ctx, userID)
}

// ---------- JWT ----------

// tokenClaims 会话 token 声明
type tokenClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

// GenerateToken 签发 HMAC-SHA256 会话 token
func (s *Service) GenerateToken(userID string) (string, error) {
	if userID == "" {
		return "", errors.New("empty user id")
	}
	now := time.Now()
	claims := tokenClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTExpire)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "voicematch",
			Subject:   userID,
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(s.cfg.JWTSecret))
}

// ValidateToken 校验 token 并返回 userID
func (s *Service) ValidateToken(token string) (string, error) {
	claims := &tokenClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", err
	}
	if !parsed.Valid {
		return "", errors.New("invalid token")
	}
	if claims.UserID == "" {
		return "", errors.New("token missing uid")
	}
	return claims.UserID, nil
}
