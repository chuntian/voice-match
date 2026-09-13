package store

import (
	"database/sql"
	"fmt"
	"time"
)

// ---- Schema ----

const (
	schemaUsers = `
CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR(64)  NOT NULL PRIMARY KEY,
    apple_id      VARCHAR(128) DEFAULT NULL,
    phone         VARCHAR(32)  DEFAULT NULL,
    nickname      VARCHAR(128) NOT NULL DEFAULT '',
    avatar        VARCHAR(512) NOT NULL DEFAULT '',
    gender        TINYINT      NOT NULL DEFAULT 0,
    city          VARCHAR(64)  NOT NULL DEFAULT '',
    bio           VARCHAR(512) NOT NULL DEFAULT '',
    status        TINYINT      NOT NULL DEFAULT 1,
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_apple_id (apple_id),
    UNIQUE KEY uk_phone (phone)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	schemaCallRecords = `
CREATE TABLE IF NOT EXISTS call_records (
    id            VARCHAR(64)  NOT NULL PRIMARY KEY,
    caller_id     VARCHAR(64)  NOT NULL,
    callee_id     VARCHAR(64)  NOT NULL,
    call_type     VARCHAR(32)  NOT NULL DEFAULT 'random',
    status        VARCHAR(32)  NOT NULL DEFAULT 'completed',
    duration      INT          NOT NULL DEFAULT 0,
    started_at    DATETIME     NOT NULL,
    ended_at      DATETIME     DEFAULT NULL,
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_caller (caller_id),
    INDEX idx_callee (callee_id),
    INDEX idx_started (started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	schemaReports = `
CREATE TABLE IF NOT EXISTS reports (
    id            VARCHAR(64)  NOT NULL PRIMARY KEY,
    reporter_id   VARCHAR(64)  NOT NULL,
    reported_id   VARCHAR(64)  NOT NULL,
    reason        VARCHAR(256) NOT NULL DEFAULT '',
    detail        TEXT,
    status        VARCHAR(32)  NOT NULL DEFAULT 'pending',
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_reporter (reporter_id),
    INDEX idx_reported (reported_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	schemaBlacklist = `
CREATE TABLE IF NOT EXISTS blacklist (
    id            BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id       VARCHAR(64)  NOT NULL,
    blocked_id    VARCHAR(64)  NOT NULL,
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_blocked (user_id, blocked_id),
    INDEX idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
)

// ---- Row structs ----

// User represents a row in the users table.
type User struct {
	ID        string
	AppleID   string
	Phone     string
	Nickname  string
	Avatar    string
	Gender    int
	City      string
	Bio       string
	Status    int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CallRecord represents a row in call_records.
type CallRecord struct {
	ID        string
	CallerID  string
	CalleeID  string
	CallType  string
	Status    string
	Duration  int
	StartedAt time.Time
	EndedAt   sql.NullTime
	CreatedAt time.Time
}

// Report represents a row in reports.
type Report struct {
	ID         string
	ReporterID string
	ReportedID string
	Reason     string
	Detail     string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ---- MySQLStore ----

// MySQLStore wraps a *sql.DB for persistent data.
type MySQLStore struct {
	db *sql.DB
}

// NewMySQLStore opens a MySQL connection pool.
// dsn format: "user:pass@tcp(host:port)/dbname?parseTime=true"
func NewMySQLStore(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return &MySQLStore{db: db}, nil
}

// Close closes the underlying database connection.
func (m *MySQLStore) Close() error {
	return m.db.Close()
}

// DB returns the underlying *sql.DB for advanced queries.
func (m *MySQLStore) DB() *sql.DB {
	return m.db
}

// AutoMigrate creates all tables if they do not exist.
func (m *MySQLStore) AutoMigrate() error {
	stmts := []string{schemaUsers, schemaCallRecords, schemaReports, schemaBlacklist}
	for _, s := range stmts {
		if _, err := m.db.Exec(s); err != nil {
			return fmt.Errorf("auto migrate: %w", err)
		}
	}
	return nil
}

// ---- User operations ----

// CreateUser inserts a new user.
func (m *MySQLStore) CreateUser(u *User) error {
	_, err := m.db.Exec(`
		INSERT INTO users (id, apple_id, phone, nickname, avatar, gender, city, bio, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, nullIfEmpty(u.AppleID), nullIfEmpty(u.Phone),
		u.Nickname, u.Avatar, u.Gender, u.City, u.Bio, u.Status,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetUserByID fetches a user by ID. Returns nil, nil if not found.
func (m *MySQLStore) GetUserByID(id string) (*User, error) {
	row := m.db.QueryRow(`
		SELECT id, apple_id, phone, nickname, avatar, gender, city, bio, status, created_at, updated_at
		FROM users WHERE id = ?`, id)
	return scanUser(row)
}

// GetUserByAppleID fetches a user by Apple sign-in ID.
func (m *MySQLStore) GetUserByAppleID(appleID string) (*User, error) {
	row := m.db.QueryRow(`
		SELECT id, apple_id, phone, nickname, avatar, gender, city, bio, status, created_at, updated_at
		FROM users WHERE apple_id = ?`, appleID)
	return scanUser(row)
}

// GetUserByPhone fetches a user by phone number.
func (m *MySQLStore) GetUserByPhone(phone string) (*User, error) {
	row := m.db.QueryRow(`
		SELECT id, apple_id, phone, nickname, avatar, gender, city, bio, status, created_at, updated_at
		FROM users WHERE phone = ?`, phone)
	return scanUser(row)
}

// UpdateUser updates mutable fields on a user.
func (m *MySQLStore) UpdateUser(u *User) error {
	_, err := m.db.Exec(`
		UPDATE users SET nickname = ?, avatar = ?, gender = ?, city = ?, bio = ?, status = ?
		WHERE id = ?`,
		u.Nickname, u.Avatar, u.Gender, u.City, u.Bio, u.Status, u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// scanUser scans a *sql.Row into a User.
func scanUser(row *sql.Row) (*User, error) {
	var u User
	var appleID, phone sql.NullString
	err := row.Scan(&u.ID, &appleID, &phone, &u.Nickname, &u.Avatar,
		&u.Gender, &u.City, &u.Bio, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.AppleID = appleID.String
	u.Phone = phone.String
	return &u, nil
}

// ---- Call record operations ----

// CreateCallRecord inserts a new call record.
func (m *MySQLStore) CreateCallRecord(r *CallRecord) error {
	_, err := m.db.Exec(`
		INSERT INTO call_records (id, caller_id, callee_id, call_type, status, duration, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.CallerID, r.CalleeID, r.CallType, r.Status, r.Duration, r.StartedAt,
	)
	if err != nil {
		return fmt.Errorf("create call record: %w", err)
	}
	return nil
}

// UpdateCallRecord updates the status and duration of a call record.
func (m *MySQLStore) UpdateCallRecord(id, status string, duration int, endedAt time.Time) error {
	_, err := m.db.Exec(`
		UPDATE call_records SET status = ?, duration = ?, ended_at = ?
		WHERE id = ?`,
		status, duration, endedAt, id,
	)
	if err != nil {
		return fmt.Errorf("update call record: %w", err)
	}
	return nil
}

// GetCallRecordsByUser returns recent call records for a user (either caller or callee).
func (m *MySQLStore) GetCallRecordsByUser(userID string, limit int) ([]*CallRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := m.db.Query(`
		SELECT id, caller_id, callee_id, call_type, status, duration, started_at, ended_at, created_at
		FROM call_records
		WHERE caller_id = ? OR callee_id = ?
		ORDER BY started_at DESC
		LIMIT ?`, userID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query call records: %w", err)
	}
	defer rows.Close()

	var records []*CallRecord
	for rows.Next() {
		var r CallRecord
		if err := rows.Scan(&r.ID, &r.CallerID, &r.CalleeID, &r.CallType,
			&r.Status, &r.Duration, &r.StartedAt, &r.EndedAt, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan call record: %w", err)
		}
		records = append(records, &r)
	}
	return records, rows.Err()
}

// GetCallRecordByID fetches a single call record.
func (m *MySQLStore) GetCallRecordByID(id string) (*CallRecord, error) {
	row := m.db.QueryRow(`
		SELECT id, caller_id, callee_id, call_type, status, duration, started_at, ended_at, created_at
		FROM call_records WHERE id = ?`, id)
	var r CallRecord
	err := row.Scan(&r.ID, &r.CallerID, &r.CalleeID, &r.CallType,
		&r.Status, &r.Duration, &r.StartedAt, &r.EndedAt, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get call record: %w", err)
	}
	return &r, nil
}

// ---- Report operations ----

// CreateReport inserts a new report.
func (m *MySQLStore) CreateReport(r *Report) error {
	_, err := m.db.Exec(`
		INSERT INTO reports (id, reporter_id, reported_id, reason, detail, status)
		VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.ReporterID, r.ReportedID, r.Reason, r.Detail, r.Status,
	)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	return nil
}

// GetReport fetches a report by ID.
func (m *MySQLStore) GetReport(id string) (*Report, error) {
	row := m.db.QueryRow(`
		SELECT id, reporter_id, reported_id, reason, detail, status, created_at, updated_at
		FROM reports WHERE id = ?`, id)
	var r Report
	err := row.Scan(&r.ID, &r.ReporterID, &r.ReportedID, &r.Reason,
		&r.Detail, &r.Status, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get report: %w", err)
	}
	return &r, nil
}

// UpdateReportStatus changes the status of a report.
func (m *MySQLStore) UpdateReportStatus(id, status string) error {
	_, err := m.db.Exec(`UPDATE reports SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("update report status: %w", err)
	}
	return nil
}

// ---- Blacklist operations (MySQL side; Redis is used for hot reads) ----

// AddBlacklist adds blockedUserID to userID's blacklist.
func (m *MySQLStore) AddBlacklist(userID, blockedUserID string) error {
	_, err := m.db.Exec(`INSERT IGNORE INTO blacklist (user_id, blocked_id) VALUES (?, ?)`,
		userID, blockedUserID)
	if err != nil {
		return fmt.Errorf("add blacklist: %w", err)
	}
	return nil
}

// RemoveBlacklist removes blockedUserID from userID's blacklist.
func (m *MySQLStore) RemoveBlacklist(userID, blockedUserID string) error {
	_, err := m.db.Exec(`DELETE FROM blacklist WHERE user_id = ? AND blocked_id = ?`,
		userID, blockedUserID)
	if err != nil {
		return fmt.Errorf("remove blacklist: %w", err)
	}
	return nil
}

// GetBlacklist returns all blocked user IDs for a user.
func (m *MySQLStore) GetBlacklist(userID string) ([]string, error) {
	rows, err := m.db.Query(`SELECT blocked_id FROM blacklist WHERE user_id = ?`, userID)
	if err != nil {
		return nil, fmt.Errorf("get blacklist: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan blacklist: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ---- Helpers ----

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
