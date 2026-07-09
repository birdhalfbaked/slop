package models

import (
	"database/sql"
	"fmt"
)

// ListUsers returns all admin users.
func ListUsers(db *sql.DB) ([]*User, error) {
	rows, err := db.Query(`SELECT id, username, password_hash, must_change, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// ListSessions returns all server-side sessions.
func ListSessions(db *sql.DB) ([]*Session, error) {
	rows, err := db.Query(`SELECT token, user_id, expires_at, created_at FROM sessions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Session
	for rows.Next() {
		s := &Session{}
		var exp, created int64
		if err := rows.Scan(&s.Token, &s.UserID, &exp, &created); err != nil {
			return nil, err
		}
		s.ExpiresAt = fromUnix(exp)
		s.CreatedAt = fromUnix(created)
		out = append(out, s)
	}
	return out, rows.Err()
}

// RestoreAuth replaces users and sessions in db with the provided snapshots.
// Used after a backup restore to keep the current admin credentials.
func RestoreAuth(db *sql.DB, users []*User, sessions []*Session) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM sessions`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM users`); err != nil {
		return err
	}
	for _, u := range users {
		_, err := tx.Exec(`INSERT INTO users(id, username, password_hash, must_change, created_at) VALUES(?, ?, ?, ?, ?)`,
			u.ID, u.Username, u.PasswordHash, boolToInt(u.MustChange), unix(u.CreatedAt))
		if err != nil {
			return fmt.Errorf("restoring user %q: %w", u.Username, err)
		}
	}
	if len(users) > 0 {
		if _, err := tx.Exec(`DELETE FROM sqlite_sequence WHERE name = 'users'`); err != nil {
			return err
		}
		var maxID int64
		for _, u := range users {
			if u.ID > maxID {
				maxID = u.ID
			}
		}
		if _, err := tx.Exec(`INSERT INTO sqlite_sequence(name, seq) VALUES('users', ?)`, maxID); err != nil {
			return err
		}
	}
	for _, s := range sessions {
		_, err := tx.Exec(`INSERT INTO sessions(token, user_id, expires_at, created_at) VALUES(?, ?, ?, ?)`,
			s.Token, s.UserID, unix(s.ExpiresAt), unix(s.CreatedAt))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
