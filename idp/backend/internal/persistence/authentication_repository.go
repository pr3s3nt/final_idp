package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"idp/internal/authentication"
)

type AuthenticationRepository struct{ DB *DB }

func (r *AuthenticationRepository) FindCredential(ctx context.Context, username string) (*authentication.CredentialRecord, error) {
	var out authentication.CredentialRecord
	err := r.DB.Pool.QueryRow(ctx, `SELECT u.user_id, u.username, u.display_name, u.status, c.password_hash
		FROM user_account u JOIN local_credential c ON c.user_id = u.user_id WHERE u.username = $1`, username).
		Scan(&out.UserID, &out.Username, &out.DisplayName, &out.Status, &out.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &out, err
}

func (r *AuthenticationRepository) BlockedUntil(ctx context.Context, accountKey, sourceKey []byte, now time.Time) (time.Time, error) {
	if _, err := r.DB.Pool.Exec(ctx, `DELETE FROM login_attempt WHERE expires_at <= $1`, now); err != nil {
		return time.Time{}, err
	}
	var until *time.Time
	err := r.DB.Pool.QueryRow(ctx, `SELECT max(blocked_until) FROM login_attempt
		WHERE ((scope = 'ACCOUNT' AND key_hash = $1) OR (scope = 'SOURCE' AND key_hash = $2))
		AND blocked_until > $3`, accountKey, sourceKey, now).Scan(&until)
	if err != nil || until == nil {
		return time.Time{}, err
	}
	return until.UTC(), nil
}

func (r *AuthenticationRepository) RecordFailure(ctx context.Context, scope string, key []byte, limit int, now time.Time, window time.Duration) (time.Time, error) {
	windowStartCutoff := now.Add(-window)
	expires := now.Add(2 * window)
	var blocked *time.Time
	err := r.DB.Pool.QueryRow(ctx, `INSERT INTO login_attempt
		(login_attempt_id, scope, key_hash, window_started_at, failure_count, blocked_until, expires_at)
		VALUES ($1, $2, $3, $4::timestamptz, 1,
			CASE WHEN 1 >= $5 THEN $4::timestamptz + make_interval(secs => $6) END, $7)
		ON CONFLICT (scope, key_hash) DO UPDATE SET
		window_started_at = CASE WHEN login_attempt.window_started_at <= $8 THEN $4::timestamptz ELSE login_attempt.window_started_at END,
		failure_count = CASE WHEN login_attempt.window_started_at <= $8 THEN 1 ELSE login_attempt.failure_count + 1 END,
		blocked_until = CASE WHEN (CASE WHEN login_attempt.window_started_at <= $8 THEN 1 ELSE login_attempt.failure_count + 1 END) >= $5
			THEN $4::timestamptz + make_interval(secs => $6) ELSE NULL END,
		expires_at = $7
		RETURNING blocked_until`, uuid.NewString(), scope, key, now, limit, int(window.Seconds()), expires, windowStartCutoff).Scan(&blocked)
	if err != nil || blocked == nil {
		return time.Time{}, err
	}
	return blocked.UTC(), nil
}

func (r *AuthenticationRepository) CreateSessionAndClearFailures(ctx context.Context, session authentication.SessionRecord, accountKey []byte) error {
	return r.DB.InTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO auth_session
			(session_id, user_id, token_hash, csrf_token_hash, created_at, last_seen_at, expires_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`, session.SessionID, session.UserID, session.TokenHash,
			session.CSRFHash, session.CreatedAt, session.LastSeenAt, session.ExpiresAt); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM login_attempt WHERE scope = 'ACCOUNT' AND key_hash = $1`, accountKey)
		return err
	})
}

func (r *AuthenticationRepository) AuthenticateSession(ctx context.Context, tokenHash []byte, now, idleCutoff time.Time) (*authentication.AuthenticatedSession, error) {
	var out authentication.AuthenticatedSession
	err := r.DB.Pool.QueryRow(ctx, `UPDATE auth_session s SET last_seen_at = $2
		FROM user_account u
		WHERE s.user_id = u.user_id AND s.token_hash = $1 AND s.revoked_at IS NULL
		AND s.expires_at > $2 AND s.last_seen_at > $3 AND u.status = 'ACTIVE'
		RETURNING s.session_id, u.user_id, u.username, u.display_name, s.csrf_token_hash`, tokenHash, now, idleCutoff).
		Scan(&out.SessionID, &out.UserID, &out.Username, &out.DisplayName, &out.CSRFHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &out, err
}

func (r *AuthenticationRepository) RevokeSession(ctx context.Context, tokenHash []byte, now time.Time) error {
	_, err := r.DB.Pool.Exec(ctx, `UPDATE auth_session SET revoked_at = COALESCE(revoked_at, $2)
		WHERE token_hash = $1`, tokenHash, now)
	return err
}

func (r *AuthenticationRepository) CreateLocalUser(ctx context.Context, username, displayName, passwordHash string, now time.Time) error {
	err := r.DB.InTx(ctx, func(tx pgx.Tx) error {
		userID := uuid.NewString()
		if _, err := tx.Exec(ctx, `INSERT INTO user_account
			(user_id, username, display_name, status, created_at, updated_at)
			VALUES ($1,$2,$3,'ACTIVE',$4,$4)`, userID, username, displayName, now); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO local_credential (user_id, password_hash, password_changed_at)
			VALUES ($1,$2,$3)`, userID, passwordHash, now)
		return err
	})
	if isUniqueViolation(err) {
		return authentication.ErrUsernameTaken
	}
	return err
}

func (r *AuthenticationRepository) ResetLocalPassword(ctx context.Context, username, passwordHash string, now time.Time) error {
	return r.DB.InTx(ctx, func(tx pgx.Tx) error {
		var userID string
		if err := tx.QueryRow(ctx, `SELECT user_id FROM user_account WHERE username = $1 FOR UPDATE`, username).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
			return authentication.ErrUserNotFound
		} else if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE local_credential SET password_hash = $2, password_changed_at = $3 WHERE user_id = $1`, userID, passwordHash, now); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE auth_session SET revoked_at = COALESCE(revoked_at, $2) WHERE user_id = $1`, userID, now)
		return err
	})
}

func (r *AuthenticationRepository) SetLocalUserStatus(ctx context.Context, username, status string, now time.Time) error {
	return r.DB.InTx(ctx, func(tx pgx.Tx) error {
		var userID string
		if err := tx.QueryRow(ctx, `UPDATE user_account SET status = $2, updated_at = $3 WHERE username = $1 RETURNING user_id`, username, status, now).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
			return authentication.ErrUserNotFound
		} else if err != nil {
			return err
		}
		if status == "DISABLED" {
			_, err := tx.Exec(ctx, `UPDATE auth_session SET revoked_at = COALESCE(revoked_at, $2) WHERE user_id = $1`, userID, now)
			return err
		}
		return nil
	})
}
