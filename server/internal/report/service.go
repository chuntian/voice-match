package report

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// MySQLStore 举报与黑名单的 MySQL 接口
type MySQLStore interface {
	CreateReport(ctx context.Context, r *Report) error
	UpdateReportStatus(ctx context.Context, reportID string, status ReportStatus) error
	ListReports(ctx context.Context, status string, page, pageSize int) ([]*Report, int64, error)
	AddBlacklist(ctx context.Context, userID, blockedUserID string) error
	RemoveBlacklist(ctx context.Context, userID, blockedUserID string) error
	ListBlacklist(ctx context.Context, userID string) ([]BlockedUser, error)
}

// RedisStore 黑名单缓存接口
type RedisStore interface {
	AddToBlacklistSet(ctx context.Context, userID, blockedUserID string) error
	RemoveFromBlacklistSet(ctx context.Context, userID, blockedUserID string) error
	GetBlacklistSet(ctx context.Context, userID string) ([]string, error)
}

// ErrNotFound 记录不存在
var ErrNotFound = errors.New("report: record not found")

// Service 举报/黑名单服务
type Service struct {
	mysql MySQLStore
	rdb   RedisStore
}

// NewService 构造服务
func NewService(mysql MySQLStore, rdb RedisStore) *Service {
	return &Service{mysql: mysql, rdb: rdb}
}

// SubmitReport 提交举报，状态置为 pending
func (s *Service) SubmitReport(ctx context.Context, reporterID, reportedID, reason, content string) (*Report, error) {
	if reporterID == "" || reportedID == "" {
		return nil, errors.New("reporter and reported required")
	}
	if reporterID == reportedID {
		return nil, errors.New("cannot report yourself")
	}
	if reason == "" {
		return nil, errors.New("reason required")
	}
	r := &Report{
		ReporterID: reporterID,
		ReportedID: reportedID,
		Reason:     reason,
		Content:    content,
		Status:     StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.mysql.CreateReport(ctx, r); err != nil {
		return nil, fmt.Errorf("create report: %w", err)
	}
	return r, nil
}

// ProcessReport 处理举报：resolved / rejected
func (s *Service) ProcessReport(ctx context.Context, reportID string, status string) error {
	switch ReportStatus(status) {
	case StatusResolved, StatusRejected, StatusPending:
	default:
		return fmt.Errorf("invalid status %q", status)
	}
	if reportID == "" {
		return errors.New("report id required")
	}
	if err := s.mysql.UpdateReportStatus(ctx, reportID, ReportStatus(status)); err != nil {
		return fmt.Errorf("update report status: %w", err)
	}
	return nil
}

// ListReports 管理员分页查询举报
func (s *Service) ListReports(ctx context.Context, status string, page, pageSize int) (*ListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	items, total, err := s.mysql.ListReports(ctx, status, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// BlockUser 拉黑：写 MySQL + Redis set
func (s *Service) BlockUser(ctx context.Context, userID, blockedUserID string) error {
	if userID == "" || blockedUserID == "" {
		return errors.New("user id and blocked user id required")
	}
	if userID == blockedUserID {
		return errors.New("cannot block yourself")
	}
	if err := s.mysql.AddBlacklist(ctx, userID, blockedUserID); err != nil {
		return fmt.Errorf("mysql block: %w", err)
	}
	if err := s.rdb.AddToBlacklistSet(ctx, userID, blockedUserID); err != nil {
		// Redis 失败不回滚 MySQL（黑名单以 MySQL 为准，缓存异步重建）
		_ = err
	}
	return nil
}

// UnblockUser 取消拉黑
func (s *Service) UnblockUser(ctx context.Context, userID, blockedUserID string) error {
	if userID == "" || blockedUserID == "" {
		return errors.New("user id and blocked user id required")
	}
	if err := s.mysql.RemoveBlacklist(ctx, userID, blockedUserID); err != nil {
		return fmt.Errorf("mysql unblock: %w", err)
	}
	if err := s.rdb.RemoveFromBlacklistSet(ctx, userID, blockedUserID); err != nil {
		_ = err
	}
	return nil
}

// GetBlacklist 获取黑名单列表（MySQL 为准，保证一致性）
func (s *Service) GetBlacklist(ctx context.Context, userID string) ([]BlockedUser, error) {
	if userID == "" {
		return nil, errors.New("user id required")
	}
	return s.mysql.ListBlacklist(ctx, userID)
}
