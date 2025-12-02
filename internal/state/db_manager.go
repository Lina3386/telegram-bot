package state

import (
	"context"
	"database/sql"
	"encoding/json"
)

type DBStateManager struct {
	db *sql.DB
}

func (m *DBStateManager) GetSession(ctx context.Context, userID int64) (*UserSession, error) {
	var session UserSession
	var tempDataJSON []byte

	err := m.db.QueryRowContext(ctx,
		"SELECT user_id, state, temp_data FROM sessions WHERE user_id = $1",
		userID,
	).Scan(&session.UserID, &session.State, &tempDataJSON)

	if err == sql.ErrNoRows {
		session.UserID = userID
		session.State = StateIdle
		session.TempData = make(map[string]string)
		return &session, nil
	}

	json.Unmarshal(tempDataJSON, &session.TempData)
	return &session, nil
}
