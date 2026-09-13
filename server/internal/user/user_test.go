package user

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ---------- mocks ----------

type fakeMySQL struct {
	users    map[string]*User // by id
	byApple  map[string]*User
	byPhone  map[string]*User
	prefs    map[string]*Preferences
	lastID   int
	createFn func(*User) error
}

func newFakeMySQL() *fakeMySQL {
	return &fakeMySQL{
		users:   map[string]*User{},
		byApple: map[string]*User{},
		byPhone: map[string]*User{},
		prefs:   map[string]*Preferences{},
	}
}

func (f *fakeMySQL) GetUserByID(_ context.Context, id string) (*User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}
func (f *fakeMySQL) GetUserByAppleID(_ context.Context, appleID string) (*User, error) {
	if u, ok := f.byApple[appleID]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}
func (f *fakeMySQL) GetUserByPhone(_ context.Context, phone string) (*User, error) {
	if u, ok := f.byPhone[phone]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}
func (f *fakeMySQL) CreateUser(_ context.Context, u *User) error {
	f.lastID++
	u.ID = "u-" + string(rune('0'+f.lastID))
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now
	f.users[u.ID] = u
	if u.AppleID != "" {
		f.byApple[u.AppleID] = u
	}
	if u.Phone != "" {
		f.byPhone[u.Phone] = u
	}
	return nil
}
func (f *fakeMySQL) UpdateProfile(_ context.Context, userID string, req UpdateProfileRequest) error {
	u, ok := f.users[userID]
	if !ok {
		return ErrNotFound
	}
	if req.Nickname != nil {
		u.Nickname = *req.Nickname
	}
	if req.Avatar != nil {
		u.Avatar = *req.Avatar
	}
	if req.Gender != nil {
		u.Gender = *req.Gender
	}
	if req.City != nil {
		u.City = *req.City
	}
	if req.Geohash != nil {
		u.Geohash = *req.Geohash
	}
	u.UpdatedAt = time.Now()
	return nil
}
func (f *fakeMySQL) GetPreferences(_ context.Context, userID string) (*Preferences, error) {
	if p, ok := f.prefs[userID]; ok {
		return p, nil
	}
	return &Preferences{UserID: userID}, nil // nolint: unused field handled below
}

// preferences carries UserID in test via extra field; we just ignore it here
func (f *fakeMySQL) UpsertPreferences(_ context.Context, userID string, p *Preferences) error {
	cp := *p
	cp.UpdatedAt = time.Now()
	f.prefs[userID] = &cp
	return nil
}

type fakeRedis struct {
	codes map[string]string
}

func newFakeRedis() *fakeRedis { return &fakeRedis{codes: map[string]string{}} }

func (r *fakeRedis) SetCode(_ context.Context, phone, code string, _ time.Duration) error {
	r.codes[phone] = code
	return nil
}
func (r *fakeRedis) GetCode(_ context.Context, phone string) (string, error) {
	return r.codes[phone], nil
}
func (r *fakeRedis) DelCode(_ context.Context, phone string) error {
	delete(r.codes, phone)
	return nil
}

// ---------- JWT tests ----------

func TestJWTGenerateAndValidate(t *testing.T) {
	svc := NewService(newFakeMySQL(), newFakeRedis(), Config{
		JWTSecret: "test-secret",
		JWTExpire: time.Hour,
	})
	tok, err := svc.GenerateToken("user-123")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	uid, err := svc.ValidateToken(tok)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if uid != "user-123" {
		t.Fatalf("uid = %s, want user-123", uid)
	}
}

func TestJWTRejectsWrongSecret(t *testing.T) {
	a := NewService(newFakeMySQL(), newFakeRedis(), Config{JWTSecret: "secret-a", JWTExpire: time.Hour})
	b := NewService(newFakeMySQL(), newFakeRedis(), Config{JWTSecret: "secret-b", JWTExpire: time.Hour})
	tok, _ := a.GenerateToken("u")
	if _, err := b.ValidateToken(tok); err == nil {
		t.Fatal("expected validation error with wrong secret")
	}
}

func TestJWTExpired(t *testing.T) {
	svc := NewService(newFakeMySQL(), newFakeRedis(), Config{
		JWTSecret: "s",
		JWTExpire: time.Hour,
	})
	// 手工签发一个 exp 已过去的 token
	now := time.Now()
	claims := tokenClaims{
		UserID: "u",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "voicematch",
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte("s"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ValidateToken(signed); err == nil {
		t.Fatal("expected expired error")
	}
}

// ---------- Apple token 解析测试（mock 公钥） ----------

func TestLoginWithApple_MockedKey(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	mysql := newFakeMySQL()
	rdb := newFakeRedis()
	svc := NewService(mysql, rdb, Config{
		JWTSecret:     "s",
		JWTExpire:     time.Hour,
		AppleClientID: "com.voicematch.app",
		DevMode:       true,
	})
	// 直接注入 mock 公钥到 JWKS 缓存
	svc.jwksMu.Lock()
	svc.jwksCache["mock-kid"] = &priv.PublicKey
	svc.jwksAt = time.Now()
	svc.jwksMu.Unlock()

	claims := appleClaims{
		Email: "t@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "apple-sub-123",
			Issuer:    "https://appleid.apple.com",
			Audience:  jwt.ClaimStrings{"com.voicematch.app"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = "mock-kid"
	signed, err := tok.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}

	u, err := svc.LoginWithApple(context.Background(), signed)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if u.AppleID != "apple-sub-123" {
		t.Fatalf("apple id = %s", u.AppleID)
	}
	// 再次登录应命中已存在用户，不重复创建
	u2, err := svc.LoginWithApple(context.Background(), signed)
	if err != nil {
		t.Fatal(err)
	}
	if u2.ID != u.ID {
		t.Fatalf("expected same user, got %s vs %s", u2.ID, u.ID)
	}
}

func TestLoginWithApple_RejectsBadAudience(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	svc := NewService(newFakeMySQL(), newFakeRedis(), Config{
		JWTSecret:     "s",
		JWTExpire:     time.Hour,
		AppleClientID: "com.voicematch.app",
	})
	svc.jwksMu.Lock()
	svc.jwksCache["k"] = &priv.PublicKey
	svc.jwksAt = time.Now()
	svc.jwksMu.Unlock()

	claims := appleClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "sub",
			Issuer:    "https://appleid.apple.com",
			Audience:  jwt.ClaimStrings{"com.wrong.app"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = "k"
	signed, _ := tok.SignedString(priv)
	if _, err := svc.LoginWithApple(context.Background(), signed); err == nil {
		t.Fatal("expected audience mismatch error")
	}
}

// ---------- 手机号登录测试 ----------

func TestPhoneLogin_DevMode(t *testing.T) {
	mysql := newFakeMySQL()
	rdb := newFakeRedis()
	svc := NewService(mysql, rdb, Config{JWTSecret: "s", DevMode: true})

	u, err := svc.LoginWithPhone(context.Background(), "13800000000", devSMSCode)
	if err != nil {
		t.Fatal(err)
	}
	if u.Phone != "13800000000" {
		t.Fatalf("phone = %s", u.Phone)
	}
}

func TestPhoneLogin_ProdModeValidCode(t *testing.T) {
	mysql := newFakeMySQL()
	rdb := newFakeRedis()
	svc := NewService(mysql, rdb, Config{JWTSecret: "s", DevMode: false})
	_ = rdb.SetCode(context.Background(), "13900000000", "123456", time.Minute)

	u, err := svc.LoginWithPhone(context.Background(), "13900000000", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if u.Phone != "13900000000" {
		t.Fatalf("phone = %s", u.Phone)
	}
}

func TestPhoneLogin_ProdModeWrongCode(t *testing.T) {
	mysql := newFakeMySQL()
	rdb := newFakeRedis()
	svc := NewService(mysql, rdb, Config{JWTSecret: "s", DevMode: false})
	_ = rdb.SetCode(context.Background(), "13900000000", "123456", time.Minute)

	if _, err := svc.LoginWithPhone(context.Background(), "13900000000", "000000"); err == nil {
		t.Fatal("expected wrong code error")
	}
}
