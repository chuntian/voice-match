package report

import (
	"context"
	"testing"
	"time"
)

type fakeMySQL struct {
	reports      map[string]*Report
	blacklist    map[string]map[string]time.Time // user -> blocked -> at
	nextReportID int
}

func newFakeMySQL() *fakeMySQL {
	return &fakeMySQL{
		reports:   map[string]*Report{},
		blacklist: map[string]map[string]time.Time{},
	}
}

func (f *fakeMySQL) CreateReport(_ context.Context, r *Report) error {
	f.nextReportID++
	r.ID = "r-" + itoa(f.nextReportID)
	if r.Status == "" {
		r.Status = StatusPending
	}
	r.CreatedAt = time.Now()
	r.UpdatedAt = r.CreatedAt
	f.reports[r.ID] = r
	return nil
}
func (f *fakeMySQL) UpdateReportStatus(_ context.Context, reportID string, status ReportStatus) error {
	r, ok := f.reports[reportID]
	if !ok {
		return ErrNotFound
	}
	r.Status = status
	r.UpdatedAt = time.Now()
	return nil
}
func (f *fakeMySQL) ListReports(_ context.Context, status string, page, pageSize int) ([]*Report, int64, error) {
	var out []*Report
	for _, r := range f.reports {
		if status == "" || string(r.Status) == status {
			out = append(out, r)
		}
	}
	return out, int64(len(out)), nil
}
func (f *fakeMySQL) AddBlacklist(_ context.Context, userID, blockedUserID string) error {
	m, ok := f.blacklist[userID]
	if !ok {
		m = map[string]time.Time{}
		f.blacklist[userID] = m
	}
	m[blockedUserID] = time.Now()
	return nil
}
func (f *fakeMySQL) RemoveBlacklist(_ context.Context, userID, blockedUserID string) error {
	if m, ok := f.blacklist[userID]; ok {
		delete(m, blockedUserID)
	}
	return nil
}
func (f *fakeMySQL) ListBlacklist(_ context.Context, userID string) ([]BlockedUser, error) {
	m := f.blacklist[userID]
	out := make([]BlockedUser, 0, len(m))
	for bid, at := range m {
		out = append(out, BlockedUser{UserID: bid, BlockedAt: at})
	}
	return out, nil
}

type fakeRedis struct {
	sets map[string]map[string]struct{}
}

func newFakeRedis() *fakeRedis { return &fakeRedis{sets: map[string]map[string]struct{}{}} }

func (r *fakeRedis) AddToBlacklistSet(_ context.Context, userID, blocked string) error {
	m, ok := r.sets[userID]
	if !ok {
		m = map[string]struct{}{}
		r.sets[userID] = m
	}
	m[blocked] = struct{}{}
	return nil
}
func (r *fakeRedis) RemoveFromBlacklistSet(_ context.Context, userID, blocked string) error {
	if m, ok := r.sets[userID]; ok {
		delete(m, blocked)
	}
	return nil
}
func (r *fakeRedis) GetBlacklistSet(_ context.Context, userID string) ([]string, error) {
	m := r.sets[userID]
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// ---------- tests ----------

func TestSubmitReport(t *testing.T) {
	svc := NewService(newFakeMySQL(), newFakeRedis())
	r, err := svc.SubmitReport(context.Background(), "u1", "u2", "spam", "bad words")
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != StatusPending {
		t.Fatalf("status = %s", r.Status)
	}
	if r.ID == "" {
		t.Fatal("id not assigned")
	}
}

func TestSubmitReport_Self(t *testing.T) {
	svc := NewService(newFakeMySQL(), newFakeRedis())
	if _, err := svc.SubmitReport(context.Background(), "u1", "u1", "x", ""); err == nil {
		t.Fatal("expected self-report error")
	}
}

func TestProcessReport(t *testing.T) {
	mysql := newFakeMySQL()
	svc := NewService(mysql, newFakeRedis())
	r, _ := svc.SubmitReport(context.Background(), "u1", "u2", "spam", "")
	if err := svc.ProcessReport(context.Background(), r.ID, string(StatusResolved)); err != nil {
		t.Fatal(err)
	}
	if mysql.reports[r.ID].Status != StatusResolved {
		t.Fatal("status not updated")
	}
	if err := svc.ProcessReport(context.Background(), r.ID, "weird"); err == nil {
		t.Fatal("expected invalid status error")
	}
}

func TestBlockAndUnblock(t *testing.T) {
	mysql := newFakeMySQL()
	rdb := newFakeRedis()
	svc := NewService(mysql, rdb)

	if err := svc.BlockUser(context.Background(), "u1", "u2"); err != nil {
		t.Fatal(err)
	}
	list, err := svc.GetBlacklist(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].UserID != "u2" {
		t.Fatalf("list = %+v", list)
	}
	// redis 也应写入
	set, _ := rdb.GetBlacklistSet(context.Background(), "u1")
	if len(set) != 1 || set[0] != "u2" {
		t.Fatalf("redis set = %v", set)
	}

	if err := svc.UnblockUser(context.Background(), "u1", "u2"); err != nil {
		t.Fatal(err)
	}
	list, _ = svc.GetBlacklist(context.Background(), "u1")
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %v", list)
	}
}

func TestBlock_Self(t *testing.T) {
	svc := NewService(newFakeMySQL(), newFakeRedis())
	if err := svc.BlockUser(context.Background(), "u1", "u1"); err == nil {
		t.Fatal("expected self-block error")
	}
}

func TestListReportsFilter(t *testing.T) {
	mysql := newFakeMySQL()
	svc := NewService(mysql, newFakeRedis())
	r1, _ := svc.SubmitReport(context.Background(), "u1", "u2", "a", "")
	_ = svc.ProcessReport(context.Background(), r1.ID, string(StatusResolved))
	_, _ = svc.SubmitReport(context.Background(), "u1", "u3", "b", "")

	res, err := svc.ListReports(context.Background(), string(StatusPending), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 {
		t.Fatalf("total = %d, want 1", res.Total)
	}
	if res.Items[0].Status != StatusPending {
		t.Fatalf("item status = %s", res.Items[0].Status)
	}
}
