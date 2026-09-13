// Package user 提供用户服务：登录、资料、偏好与 JWT 会话管理。
package user

import (
	"time"
)

// Gender 性别枚举
type Gender string

const (
	GenderUnknown Gender = ""
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderOther   Gender = "other"
)

// MatchType 匹配类型：语音/文字等
type MatchType string

const (
	MatchTypeVoice MatchType = "voice"
	MatchTypeText  MatchType = "text"
)

// User 用户主表模型
type User struct {
	ID        string    `json:"id"`
	AppleID   string    `json:"apple_id,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Gender    Gender    `json:"gender"`
	City      string    `json:"city"`
	Geohash   string    `json:"geohash"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Preferences 用户匹配偏好
type Preferences struct {
	UserID           string    `json:"user_id,omitempty"`
	MatchType        MatchType `json:"match_type"`
	City             string    `json:"city"`
	Destination      string    `json:"destination"`
	Tags             []string  `json:"tags"`
	GenderPreference Gender    `json:"gender_preference"`
	AgeRangeMin      int       `json:"age_range_min"`
	AgeRangeMax      int       `json:"age_range_max"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

// UpdateProfileRequest 更新资料请求 DTO
type UpdateProfileRequest struct {
	Nickname *string `json:"nickname"`
	Avatar   *string `json:"avatar"`
	Gender   *Gender `json:"gender"`
	City     *string `json:"city"`
	Geohash  *string `json:"geohash"`
}

// AppleLoginRequest Apple 登录请求
type AppleLoginRequest struct {
	IdentityToken string `json:"identity_token"`
	// AuthorizationCode 可选，预留
	AuthorizationCode string `json:"authorization_code,omitempty"`
	// FullName 可选，Apple 仅首次登录下发
	FullName string `json:"full_name,omitempty"`
}

// SendCodeRequest 发送短信验证码请求
type SendCodeRequest struct {
	Phone string `json:"phone"`
}

// VerifyCodeRequest 校验验证码并登录请求
type VerifyCodeRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// LoginResponse 登录返回
type LoginResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}
