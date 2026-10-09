package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jack-barr3tt/bouncer/internal/limit"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	Alphabet       = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	UserSessionTTL = 30 * 24 * time.Hour
)

var (
	ErrNotFound       = errors.New("not found")
	ErrInvalid        = errors.New("invalid")
	ErrUsernameTaken  = errors.New("username taken")
	ErrLastAdmin      = errors.New("last admin")
	ErrInvalidCode    = errors.New("invalid code")
	ErrCodeFull       = errors.New("code full")
	ErrRevokedAccount = errors.New("revoked account")
	ErrBadNickname    = errors.New("bad nickname")
	ErrLocked         = errors.New("locked")
)

type User struct {
	ID       uuid.UUID
	Username string
	Role     string
	Disabled bool
	Apps     []string
}

type Code struct {
	ID          uuid.UUID
	Code        string
	Label       string
	ExpiresAt   time.Time
	MaxSignups  int
	SignupCount int
	RevokedAt   *time.Time
	Apps        []string
	CreatedAt   time.Time
}

type TempAccount struct {
	ID        uuid.UUID
	Nickname  string
	CreatedIP string
	LastIP    string
	CreatedAt time.Time
	RevokedAt *time.Time
}

type Principal struct {
	SessionID uuid.UUID
	Kind      string
	UserID    *uuid.UUID
	TempID    *uuid.UUID
	Username  string
	Nickname  string
	Role      string
	Apps      []string
	ExpiresAt time.Time
	Key       string
}

type Store struct {
	pool      *pgxpool.Pool
	cost      int
	dummyHash []byte
}

func New(pool *pgxpool.Pool, cost int) (*Store, error) {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	dummy, err := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), cost)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool, cost: cost, dummyHash: dummy}, nil
}

func (s *Store) UserCount(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) EnsureAdmin(ctx context.Context, username, password string) error {
	n, err := s.UserCount(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = s.insertUser(ctx, username, password, "admin")
	return err
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func HashFingerprint(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func NormalizeCode(raw string) (string, bool) {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "-", "")
	raw = strings.ReplaceAll(raw, " ", "")
	if len(raw) != 8 {
		return "", false
	}
	for _, r := range raw {
		if !strings.ContainsRune(Alphabet, r) {
			return "", false
		}
	}
	return raw, true
}

func ValidNickname(raw string) (string, bool) {
	name := strings.TrimSpace(raw)
	n := len([]rune(name))
	if n < 1 || n > 40 {
		return "", false
	}
	return name, true
}

func (s *Store) Authenticate(ctx context.Context, username, password string) (User, error) {
	var user User
	var hash string
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, role, disabled, password_hash
		FROM users WHERE username_key = $1`, strings.ToLower(strings.TrimSpace(username)),
	).Scan(&user.ID, &user.Username, &user.Role, &user.Disabled, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil || user.Disabled {
		return User{}, ErrNotFound
	}
	user.Apps, err = s.userApps(ctx, user.ID)
	return user, err
}

func (s *Store) CreateSession(ctx context.Context, userID, tempID *uuid.UUID, expires time.Time) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO sessions (id, token_hash, user_id, temporary_account_id, expires_at)
		VALUES ($1, $2, $3, $4, $5)`,
		uuid.New(), HashToken(token), userID, tempID, expires)
	return token, err
}

func (s *Store) DeleteSessionToken(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, HashToken(token))
	return err
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (s *Store) DeleteTempSessions(ctx context.Context, tempID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE temporary_account_id = $1`, tempID)
	return err
}

func (s *Store) Principal(ctx context.Context, token string, registry []string) (*Principal, error) {
	if token == "" {
		return nil, ErrNotFound
	}
	var (
		p           Principal
		userID      *uuid.UUID
		tempID      *uuid.UUID
		username    *string
		role        *string
		disabled    *bool
		nickname    *string
		tempRevoked *time.Time
		codeExpires *time.Time
		codeID      *uuid.UUID
	)
	err := s.pool.QueryRow(ctx, `
		SELECT s.id, s.expires_at, s.user_id, s.temporary_account_id,
			u.username, u.role, u.disabled,
			t.nickname, t.revoked_at, t.access_code_id, c.expires_at
		FROM sessions s
		LEFT JOIN users u ON u.id = s.user_id
		LEFT JOIN temporary_accounts t ON t.id = s.temporary_account_id
		LEFT JOIN access_codes c ON c.id = t.access_code_id
		WHERE s.token_hash = $1`, HashToken(token),
	).Scan(&p.SessionID, &p.ExpiresAt, &userID, &tempID, &username, &role, &disabled, &nickname, &tempRevoked, &codeID, &codeExpires)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !p.ExpiresAt.After(time.Now()) {
		_ = s.DeleteSessionToken(ctx, token)
		return nil, ErrNotFound
	}
	if userID != nil {
		if disabled != nil && *disabled {
			return nil, ErrNotFound
		}
		p.Kind = "user"
		p.UserID = userID
		p.Username = *username
		p.Role = *role
		p.Key = "user:" + userID.String()
		if p.Role == "admin" {
			p.Apps = append([]string(nil), registry...)
			return &p, nil
		}
		p.Apps, err = s.userApps(ctx, *userID)
		return &p, err
	}
	if tempID == nil || tempRevoked != nil || codeExpires == nil || !codeExpires.After(time.Now()) {
		return nil, ErrNotFound
	}
	if codeExpires.Before(p.ExpiresAt) {
		p.ExpiresAt = *codeExpires
	}
	p.Kind = "temporary"
	p.TempID = tempID
	p.Nickname = *nickname
	p.Key = "temp:" + tempID.String()
	p.Apps, err = s.codeApps(ctx, *codeID)
	return &p, err
}

func (s *Store) TouchTempIP(ctx context.Context, tempID uuid.UUID, ip string) error {
	_, err := s.pool.Exec(ctx, `UPDATE temporary_accounts SET last_ip = $2 WHERE id = $1`, tempID, ip)
	return err
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.username, u.role, u.disabled,
			COALESCE(array_agg(a.slug ORDER BY a.slug) FILTER (WHERE a.slug IS NOT NULL), '{}')
		FROM users u
		LEFT JOIN user_apps a ON a.user_id = u.id
		GROUP BY u.id
		ORDER BY lower(u.username)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.Disabled, &user.Apps); err != nil {
			return nil, err
		}
		if user.Apps == nil {
			user.Apps = []string{}
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) CreateUser(ctx context.Context, username, password string) (User, error) {
	return s.insertUser(ctx, username, password, "user")
}

func (s *Store) UpdateUser(ctx context.Context, id uuid.UUID, password *string, disabled *bool, role *string) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current User
	err = tx.QueryRow(ctx, `SELECT id, username, role, disabled FROM users WHERE id = $1 FOR UPDATE`, id).
		Scan(&current.ID, &current.Username, &current.Role, &current.Disabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	if password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*password), s.cost)
		if err != nil {
			return User{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, id, string(hash)); err != nil {
			return User{}, err
		}
	}
	nextRole := current.Role
	nextDisabled := current.Disabled
	if role != nil {
		nextRole = *role
	}
	if disabled != nil {
		nextDisabled = *disabled
	}
	wasAdmin := current.Role == "admin" && !current.Disabled
	stillAdmin := nextRole == "admin" && !nextDisabled
	if wasAdmin && !stillAdmin {
		var others int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM users
			WHERE role = 'admin' AND disabled = false AND id <> $1`, id).Scan(&others); err != nil {
			return User{}, err
		}
		if others == 0 {
			return User{}, ErrLastAdmin
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET role = $2, disabled = $3 WHERE id = $1`, id, nextRole, nextDisabled); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	if nextDisabled {
		_ = s.DeleteUserSessions(ctx, id)
	}
	return s.getUser(ctx, id)
}

func (s *Store) SetUserApps(ctx context.Context, id uuid.UUID, slugs []string) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, id).Scan(&exists); err != nil {
		return User{}, err
	}
	if !exists {
		return User{}, ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_apps WHERE user_id = $1`, id); err != nil {
		return User{}, err
	}
	for _, slug := range slugs {
		if _, err := tx.Exec(ctx, `INSERT INTO user_apps (user_id, slug) VALUES ($1, $2)`, id, slug); err != nil {
			return User{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return s.getUser(ctx, id)
}

func (s *Store) CreateCode(ctx context.Context, createdBy uuid.UUID, label string, expires time.Time, maxSignups int, slugs []string) (Code, error) {
	var last error
	for range 5 {
		code, err := randomCode()
		if err != nil {
			return Code{}, err
		}
		id := uuid.New()
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return Code{}, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO access_codes (id, code, label, expires_at, max_signups, created_by)
			VALUES ($1, $2, $3, $4, $5, $6)`, id, code, label, expires, maxSignups, createdBy)
		if err != nil {
			_ = tx.Rollback(ctx)
			last = err
			continue
		}
		for _, slug := range slugs {
			if _, err := tx.Exec(ctx, `INSERT INTO access_code_apps (code_id, slug) VALUES ($1, $2)`, id, slug); err != nil {
				_ = tx.Rollback(ctx)
				return Code{}, err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return Code{}, err
		}
		return s.GetCode(ctx, id)
	}
	return Code{}, fmt.Errorf("create access code: %w", last)
}

func (s *Store) ListCodes(ctx context.Context) ([]Code, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.code, c.label, c.expires_at, c.max_signups, c.revoked_at, c.created_at,
			(SELECT count(*) FROM temporary_accounts t WHERE t.access_code_id = c.id),
			COALESCE((
				SELECT array_agg(a.slug ORDER BY a.slug) FROM access_code_apps a WHERE a.code_id = c.id
			), '{}')
		FROM access_codes c
		ORDER BY c.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var codes []Code
	for rows.Next() {
		code, err := scanCode(rows)
		if err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

func (s *Store) GetCode(ctx context.Context, id uuid.UUID) (Code, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT c.id, c.code, c.label, c.expires_at, c.max_signups, c.revoked_at, c.created_at,
			(SELECT count(*) FROM temporary_accounts t WHERE t.access_code_id = c.id),
			COALESCE((
				SELECT array_agg(a.slug ORDER BY a.slug) FROM access_code_apps a WHERE a.code_id = c.id
			), '{}')
		FROM access_codes c WHERE c.id = $1`, id)
	code, err := scanCode(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Code{}, ErrNotFound
	}
	return code, err
}

func (s *Store) RevokeCode(ctx context.Context, id uuid.UUID) (Code, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE access_codes SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return Code{}, err
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.GetCode(ctx, id); err != nil {
			return Code{}, err
		}
	}
	return s.GetCode(ctx, id)
}

func (s *Store) ListTemps(ctx context.Context, codeID uuid.UUID) ([]TempAccount, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM access_codes WHERE id = $1)`, codeID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, nickname, created_ip, last_ip, created_at, revoked_at
		FROM temporary_accounts WHERE access_code_id = $1
		ORDER BY created_at`, codeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var accounts []TempAccount
	for rows.Next() {
		var account TempAccount
		if err := rows.Scan(&account.ID, &account.Nickname, &account.CreatedIP, &account.LastIP, &account.CreatedAt, &account.RevokedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (s *Store) RevokeTemp(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE temporary_accounts SET revoked_at = COALESCE(revoked_at, now()) WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return s.DeleteTempSessions(ctx, id)
}

func (s *Store) Redeem(ctx context.Context, code, nickname, fpHash, ip string) (uuid.UUID, time.Time, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		codeID  uuid.UUID
		expires time.Time
		max     int
		revoked *time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT id, expires_at, max_signups, revoked_at
		FROM access_codes WHERE code = $1 FOR UPDATE`, code).Scan(&codeID, &expires, &max, &revoked)
	if errors.Is(err, pgx.ErrNoRows) || revoked != nil || !expires.After(time.Now()) {
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, time.Time{}, err
		}
		return uuid.Nil, time.Time{}, ErrInvalidCode
	}
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}
	name, ok := ValidNickname(nickname)
	if !ok {
		return uuid.Nil, time.Time{}, ErrBadNickname
	}

	var tempID uuid.UUID
	var tempRevoked *time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, revoked_at FROM temporary_accounts
		WHERE access_code_id = $1 AND fingerprint_hash = $2`, codeID, fpHash).Scan(&tempID, &tempRevoked)
	if err == nil {
		if tempRevoked != nil {
			return uuid.Nil, time.Time{}, ErrRevokedAccount
		}
		if _, err := tx.Exec(ctx, `UPDATE temporary_accounts SET last_ip = $2 WHERE id = $1`, tempID, ip); err != nil {
			return uuid.Nil, time.Time{}, err
		}
	} else if errors.Is(err, pgx.ErrNoRows) {
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM temporary_accounts WHERE access_code_id = $1`, codeID).Scan(&count); err != nil {
			return uuid.Nil, time.Time{}, err
		}
		if count >= max {
			return uuid.Nil, time.Time{}, ErrCodeFull
		}
		tempID = uuid.New()
		if _, err := tx.Exec(ctx, `
			INSERT INTO temporary_accounts (id, access_code_id, nickname, fingerprint_hash, created_ip, last_ip)
			VALUES ($1, $2, $3, $4, $5, $5)`, tempID, codeID, name, fpHash, ip); err != nil {
			return uuid.Nil, time.Time{}, err
		}
	} else {
		return uuid.Nil, time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, time.Time{}, err
	}
	return tempID, expires, nil
}

type Bucket struct {
	Name      string
	Threshold int
}

func (s *Store) Locked(ctx context.Context, names []string) (bool, error) {
	for _, name := range names {
		var until *time.Time
		err := s.pool.QueryRow(ctx, `SELECT locked_until FROM auth_attempts WHERE bucket = $1`, name).Scan(&until)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return false, err
		}
		if until != nil && until.After(time.Now()) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) Fail(ctx context.Context, buckets []Bucket) error {
	for _, bucket := range buckets {
		var failures int
		if err := s.pool.QueryRow(ctx, `
			INSERT INTO auth_attempts (bucket, failures, updated_at)
			VALUES ($1, 1, now())
			ON CONFLICT (bucket) DO UPDATE
				SET failures = auth_attempts.failures + 1, updated_at = now()
			RETURNING failures`, bucket.Name).Scan(&failures); err != nil {
			return err
		}
		if bucket.Threshold > 0 && failures%bucket.Threshold == 0 {
			until := time.Now().Add(limit.Duration(failures / bucket.Threshold))
			if _, err := s.pool.Exec(ctx, `UPDATE auth_attempts SET locked_until = $2 WHERE bucket = $1`, bucket.Name, until); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) Clear(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM auth_attempts WHERE bucket = ANY($1)`, names)
	return err
}

func (s *Store) insertUser(ctx context.Context, username, password, role string) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return User{}, err
	}
	user := User{ID: uuid.New(), Username: strings.TrimSpace(username), Role: role, Apps: []string{}}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO users (id, username, username_key, password_hash, role)
		VALUES ($1, $2, $3, $4, $5)`,
		user.ID, user.Username, strings.ToLower(user.Username), string(hash), role)
	if err != nil && strings.Contains(err.Error(), "username_key") {
		return User{}, ErrUsernameTaken
	}
	return user, err
}

func (s *Store) getUser(ctx context.Context, id uuid.UUID) (User, error) {
	var user User
	err := s.pool.QueryRow(ctx, `SELECT id, username, role, disabled FROM users WHERE id = $1`, id).
		Scan(&user.ID, &user.Username, &user.Role, &user.Disabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	user.Apps, err = s.userApps(ctx, id)
	return user, err
}

func (s *Store) userApps(ctx context.Context, id uuid.UUID) ([]string, error) {
	return s.slugs(ctx, `SELECT slug FROM user_apps WHERE user_id = $1 ORDER BY slug`, id)
}

func (s *Store) codeApps(ctx context.Context, id uuid.UUID) ([]string, error) {
	return s.slugs(ctx, `SELECT slug FROM access_code_apps WHERE code_id = $1 ORDER BY slug`, id)
}

func (s *Store) slugs(ctx context.Context, query string, id uuid.UUID) ([]string, error) {
	rows, err := s.pool.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	slugs := []string{}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		slugs = append(slugs, slug)
	}
	return slugs, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCode(row scanner) (Code, error) {
	var code Code
	err := row.Scan(&code.ID, &code.Code, &code.Label, &code.ExpiresAt, &code.MaxSignups, &code.RevokedAt, &code.CreatedAt, &code.SignupCount, &code.Apps)
	if code.Apps == nil {
		code.Apps = []string{}
	}
	return code, err
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, 8)
	for i, b := range buf {
		out[i] = Alphabet[int(b)%len(Alphabet)]
	}
	return string(out), nil
}
