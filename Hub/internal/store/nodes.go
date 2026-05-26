package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"

	"time"

	"github.com/honeywire/hub/internal/models"
)

// CreateNode initializes a new UI-first node and returns the generated credentials.
func (s *SQLiteStore) CreateNode(alias string, tagsJSON string) (string, string, error) {
	nodeID := "node-" + uuid.New().String()[:8]

	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", err
	}
	apiKey := "hw_key_" + hex.EncodeToString(keyBytes)

	now := time.Now().UTC().Format(time.RFC3339)

	_, err := s.DB.Exec(`
		INSERT INTO nodes (id, alias, api_key, tags, pending_config, created_at, updated_at) 
		VALUES (?, ?, ?, ?, 0, ?, ?)`,
		nodeID, alias, apiKey, tagsJSON, now, now,
	)

	return nodeID, apiKey, err
}

// GetNodeByKey allows the agent/wizard to authenticate itself
func (s *SQLiteStore) GetNodeByKey(apiKey string) (string, error) {
	var nodeID string
	err := s.DB.QueryRow("SELECT id FROM nodes WHERE api_key = ?", apiKey).Scan(&nodeID)
	return nodeID, err
}

// UpdateNodeMeta allows the UI to edit the Node's Alias, Tags, and IP Addresses
func (s *SQLiteStore) UpdateNodeMeta(nodeID string, alias string, tagsJSON string, publicIP *string, privateIP *string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.Exec(`
		UPDATE nodes 
		SET alias = ?, tags = ?, public_ip = ?, private_ip = ?, updated_at = ? 
		WHERE id = ?`,
		alias, tagsJSON, publicIP, privateIP, now, nodeID,
	)
	return err
}

// Helper to determine if a node/sensor is alive based on the last heartbeat
func deriveStatus(lastHeartbeat *string) string {
	if lastHeartbeat == nil || *lastHeartbeat == "" {
		return "pending" // Never checked in
	}
	t, err := time.Parse(time.RFC3339, *lastHeartbeat)
	if err != nil {
		return "down"
	}
	// If heartbeat is older than 60 seconds, consider it offline
	if time.Now().UTC().Sub(t) > 60*time.Second {
		return "down"
	}
	return "up"
}

// deriveCompositeNodeStatus determines the overall node status based on its base network status and installed sensors
func deriveCompositeNodeStatus(baseStatus string, sensors []models.NodeSensor) string {
	if baseStatus != "up" || len(sensors) == 0 {
		return baseStatus
	}
	onlineCount := 0
	for _, s := range sensors {
		if s.Status == "up" {
			onlineCount++
		}
	}
	if onlineCount == 0 {
		return "down"
	}
	if onlineCount < len(sensors) {
		return "degraded"
	}
	return "up"
}

// GetNodes returns a list of all nodes and their installed sensors for the Fleet Dashboard
func (s *SQLiteStore) GetNodes() ([]models.Node, error) {
	rows, err := s.DB.Query(`
		SELECT id, alias, api_key, active_revision, desired_revision, public_ip, private_ip, tags, pending_config, last_heartbeat 
		FROM nodes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []models.Node
	for rows.Next() {
		var n models.Node
		var tagsJSON string
		var pendingInt int
		var activeRevision sql.NullString
		var desiredRevision sql.NullString

		if err := rows.Scan(&n.ID, &n.Alias, &n.APIKey, &activeRevision, &desiredRevision, &n.PublicIP, &n.PrivateIP, &tagsJSON, &pendingInt, &n.LastHeartbeat); err != nil {
			return nil, err
		}

		if activeRevision.Valid {
			n.ActiveRevision = activeRevision.String
		}
		if desiredRevision.Valid {
			n.DesiredRevision = desiredRevision.String
		}

		json.Unmarshal([]byte(tagsJSON), &n.Tags)
		n.HasPendingConfig = pendingInt == 1
		n.Status = deriveStatus(n.LastHeartbeat)
		n.InstalledSensors = []models.NodeSensor{}

		// Subquery to fetch sensors for this specific node
		sensorRows, err := s.DB.Query(`
			SELECT sensor_id, custom_name, config_values, is_silenced, last_heartbeat 
			FROM node_sensors WHERE node_id = ?`, n.ID)
		if err != nil {
			return nil, err
		}

		for sensorRows.Next() {
			var ns models.NodeSensor
			var configStr string
			var silencedInt int
			var lastHb sql.NullString

			if err := sensorRows.Scan(&ns.Name, &ns.Display, &configStr, &silencedInt, &lastHb); err != nil {
				sensorRows.Close()
				return nil, err
			}

			ns.ID = ns.Name // Use catalog ID for frontend keys
			ns.IsSilenced = silencedInt == 1
			if lastHb.Valid {
				ns.LastHeartbeat = &lastHb.String
			}
			ns.Status = deriveStatus(ns.LastHeartbeat)
			json.Unmarshal([]byte(configStr), &ns.EnvVars)
			n.InstalledSensors = append(n.InstalledSensors, ns)
		}
		sensorRows.Close()

		n.Status = deriveCompositeNodeStatus(n.Status, n.InstalledSensors)
		nodes = append(nodes, n)
	}

	if nodes == nil {
		nodes = []models.Node{} // Guarantee an empty array over null
	}

	return nodes, nil
}

// GetNodeDetails fetches the node, its installed sensors, and their recent event counts
func (s *SQLiteStore) GetNodeDetails(nodeID string) (*models.Node, error) {
	var node models.Node
	var tagsJSON string
	var pendingInt int
	var activeRevision sql.NullString
	var desiredRevision sql.NullString

	// Fetch Node Meta
	err := s.DB.QueryRow(`
		SELECT id, alias, api_key, active_revision, desired_revision, public_ip, private_ip, tags, pending_config, last_heartbeat 
		FROM nodes WHERE id = ?`, nodeID).
		Scan(&node.ID, &node.Alias, &node.APIKey, &activeRevision, &desiredRevision, &node.PublicIP, &node.PrivateIP, &tagsJSON, &pendingInt, &node.LastHeartbeat)
	if err != nil {
		return nil, err
	}
	if activeRevision.Valid {
		node.ActiveRevision = activeRevision.String
	}
	if desiredRevision.Valid {
		node.DesiredRevision = desiredRevision.String
	}

	json.Unmarshal([]byte(tagsJSON), &node.Tags)
	node.HasPendingConfig = pendingInt == 1
	node.Status = deriveStatus(node.LastHeartbeat)
	node.InstalledSensors = []models.NodeSensor{}

	// Fetch Installed Sensors (Added metadata and last_heartbeat)
	rows, err := s.DB.Query(`
		SELECT sensor_id, custom_name, config_values, metadata, is_silenced, last_heartbeat 
		FROM node_sensors WHERE node_id = ?`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ns models.NodeSensor
		var configStr string
		var metaStr string
		var silencedInt int
		var lastHb sql.NullString

		if err := rows.Scan(&ns.Name, &ns.Display, &configStr, &metaStr, &silencedInt, &lastHb); err != nil {
			return nil, err
		}

		ns.ID = ns.Name
		ns.NodeID = nodeID // Map the parent node ID for backend tracking
		ns.IsSilenced = silencedInt == 1

		if lastHb.Valid {
			ns.LastHeartbeat = &lastHb.String
		}

		// Derive status specifically for this container!
		ns.Status = deriveStatus(ns.LastHeartbeat)

		json.Unmarshal([]byte(configStr), &ns.EnvVars)
		json.Unmarshal([]byte(metaStr), &ns.Metadata)

		// Count events in the last 24h for this specific sensor
		yesterday := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
		if err := s.DB.QueryRow(`
			SELECT COUNT(*) FROM events 
			WHERE node_id = ? AND sensor_id = ? AND timestamp >= ?`,
			nodeID, ns.ID, yesterday).Scan(&ns.Events24h); err != nil {
			return nil, err
		}

		node.InstalledSensors = append(node.InstalledSensors, ns)
	}

	node.Status = deriveCompositeNodeStatus(node.Status, node.InstalledSensors)

	return &node, nil
}

// AddSensorToNode securely inserts a configured sensor and flags the node as pending sync.
func (s *SQLiteStore) AddSensorToNode(nodeID, sensorID, customName string, configValues map[string]interface{}) error {
	configJSON, err := json.Marshal(configValues)
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO node_sensors (node_id, sensor_id, custom_name, config_values, is_silenced, created_at, updated_at) 
		VALUES (?, ?, ?, ?, 0, ?, ?)`,
		nodeID, sensorID, customName, string(configJSON), now, now,
	)
	if err != nil {
		tx.Rollback()
		return err // Likely hits UNIQUE constraint
	}

	_, err = tx.Exec(`UPDATE nodes SET pending_config = 1, desired_revision = NULL, updated_at = ? WHERE id = ?`, now, nodeID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// UpdateNodeSensor updates the config/alias and sets pending_config = 1
func (s *SQLiteStore) UpdateNodeSensor(nodeID, sensorID, customName string, configValues map[string]interface{}) error {
	configJSON, err := json.Marshal(configValues)
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE node_sensors 
		SET custom_name = ?, config_values = ?, updated_at = ? 
		WHERE node_id = ? AND sensor_id = ?`,
		customName, string(configJSON), now, nodeID, sensorID,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`UPDATE nodes SET pending_config = 1, desired_revision = NULL, updated_at = ? WHERE id = ?`, now, nodeID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// RemoveNodeSensor deletes the sensor and sets pending_config = 1
func (s *SQLiteStore) RemoveNodeSensor(nodeID, sensorID string) error {
	now := time.Now().Format(time.RFC3339)
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM node_sensors WHERE node_id = ? AND sensor_id = ?`, nodeID, sensorID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`UPDATE nodes SET pending_config = 1, desired_revision = NULL, updated_at = ? WHERE id = ?`, now, nodeID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// DeleteNode permanently removes a node and cascades to delete all its sensors, events, and heartbeats
func (s *SQLiteStore) DeleteNode(nodeID string) error {
	_, err := s.DB.Exec("DELETE FROM nodes WHERE id = ?", nodeID)
	return err
}

// ApplyNodeRevision saves the newly generated compose revision and clears the pending flag
func (s *SQLiteStore) SetNodeDesiredRevision(nodeID, revision string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.Exec(`
		UPDATE nodes 
		SET desired_revision = ?, pending_config = 1, updated_at = ? 
		WHERE id = ?`,
		revision, now, nodeID,
	)
	return err
}

func (s *SQLiteStore) ClearNodePendingConfig(nodeID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.Exec(`
		UPDATE nodes 
		SET pending_config = 0, desired_revision = NULL, updated_at = ? 
		WHERE id = ?`,
		now, nodeID,
	)
	return err
}

func (s *SQLiteStore) ApplyNodeRevision(nodeID, revision string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.Exec(`
		UPDATE nodes 
		SET active_revision = ?, pending_config = 0, updated_at = ? 
		WHERE id = ?`,
		revision, now, nodeID,
	)
	return err
}
