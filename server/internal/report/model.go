package report

import "time"

// ReportStatus 举报状态
type ReportStatus string

const (
	StatusPending  ReportStatus = "pending"
	StatusResolved ReportStatus = "resolved"
	StatusRejected ReportStatus = "rejected"
)

// Report 举报记录
type Report struct {
	ID         string       `json:"id"`
	ReporterID string       `json:"reporter_id"`
	ReportedID string       `json:"reported_id"`
	Reason     string       `json:"reason"`
	Content    string       `json:"content"`
	Status     ReportStatus `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

// BlockedUser 黑名单用户（带基础展示信息）
type BlockedUser struct {
	UserID    string    `json:"user_id"`
	Nickname  string    `json:"nickname,omitempty"`
	Avatar    string    `json:"avatar,omitempty"`
	BlockedAt time.Time `json:"blocked_at"`
}

// SubmitReportRequest 提交举报请求
type SubmitReportRequest struct {
	ReportedID string `json:"reported_id"`
	Reason     string `json:"reason"`
	Content    string `json:"content"`
}

// UpdateStatusRequest 更新举报状态请求
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// BlockRequest 拉黑请求
type BlockRequest struct {
	BlockedUserID string `json:"blocked_user_id"`
}

// ListResult 分页结果
type ListResult struct {
	Items    []*Report `json:"items"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}
