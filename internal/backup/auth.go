package backup

import (
	"os"

	"slop/internal/db"
	"slop/internal/models"
)

type authSnapshot struct {
	users    []*models.User
	sessions []*models.Session
}

func snapshotAuth(dbPath string) (*authSnapshot, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, nil
	}
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	users, err := models.ListUsers(database)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	sessions, err := models.ListSessions(database)
	if err != nil {
		return nil, err
	}
	return &authSnapshot{users: users, sessions: sessions}, nil
}

func restoreAuth(dbPath string, snap *authSnapshot) error {
	if snap == nil || len(snap.users) == 0 {
		return nil
	}
	database, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer database.Close()
	return models.RestoreAuth(database, snap.users, snap.sessions)
}

func replaceDatabasePreservingAuth(src, dest string) error {
	snap, err := snapshotAuth(dest)
	if err != nil {
		return err
	}
	if err := replaceDatabase(src, dest); err != nil {
		return err
	}
	return restoreAuth(dest, snap)
}
