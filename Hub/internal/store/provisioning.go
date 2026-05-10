package store

import "time"

func (s *SQLiteStore) InsertPairingToken(token string, expiresAt time.Time, createdAt time.Time) error {
	_, err := s.DB.Exec(
		"INSERT INTO pairing_tokens (token, expires_at, created_at) VALUES (?, ?, ?)",
		token, expiresAt.Format(time.RFC3339), createdAt.Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) ValidatePairingToken(token string) (bool, error) {
	var createdAt string
	err := s.DB.QueryRow(
		"SELECT created_at FROM pairing_tokens WHERE token = ? AND expires_at > datetime('now')",
		token,
	).Scan(&createdAt)
	if err != nil {
		return false, err
	}

	// Delete token immediately (single-use)
	s.DB.Exec("DELETE FROM pairing_tokens WHERE token = ?", token)
	return true, nil
}

func (s *SQLiteStore) CreateNode(nodeID, alias, nodeKey, ipAddress, nowStr string) error {
	_, err := s.DB.Exec(
		"INSERT INTO nodes (node_id, alias, node_key, ip_address, first_seen, last_seen) VALUES (?, ?, ?, ?, ?, ?)",
		nodeID, alias, nodeKey, ipAddress, nowStr, nowStr,
	)
	return err
}

func (s *SQLiteStore) GetNodeKey(nodeID string) (string, error) {
	var nodeKey string
	err := s.DB.QueryRow("SELECT node_key FROM nodes WHERE node_id = ?", nodeID).Scan(&nodeKey)
	return nodeKey, err
}
