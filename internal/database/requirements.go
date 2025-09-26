package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Requirement represents a requirement in the database
type Requirement struct {
	ID                   string    `json:"id"`
	ProjectID            string    `json:"project_id"`
	ComponentID          string    `json:"component_id"`
	PhaseID              *string   `json:"phase_id,omitempty"`
	ParentRequirementID  *string   `json:"parent_requirement_id,omitempty"`
	RequirementKey       string    `json:"requirement_key"`
	RequirementType      string    `json:"requirement_type"`
	Title                string    `json:"title"`
	Description          *string   `json:"description,omitempty"`
	Category             string    `json:"category"`
	Priority             string    `json:"priority"`
	Status               string    `json:"status"`
	AcceptanceCriteria   []string  `json:"acceptance_criteria,omitempty"`
	CreatedAt            string    `json:"created_at"`
	UpdatedAt            string    `json:"updated_at"`
}

// CreateRequirement creates a new requirement in the database
func (db *DB) CreateRequirement(req *Requirement) error {
	// Generate a unique ID if not provided
	if req.ID == "" {
		req.ID = generateID()
	}

	// Set timestamps
	now := time.Now().UTC().Format(time.RFC3339)
	req.CreatedAt = now
	req.UpdatedAt = now

	// Convert acceptance criteria to JSON
	acceptanceCriteriaJSON := "[]"
	if len(req.AcceptanceCriteria) > 0 {
		data, err := json.Marshal(req.AcceptanceCriteria)
		if err != nil {
			return fmt.Errorf("failed to marshal acceptance criteria: %w", err)
		}
		acceptanceCriteriaJSON = string(data)
	}

	query := `
		INSERT INTO requirements (
			id, project_id, component_id, phase_id, parent_requirement_id,
			requirement_key, requirement_type, title, description,
			category, priority, status, acceptance_criteria,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query,
		req.ID, req.ProjectID, req.ComponentID, req.PhaseID, req.ParentRequirementID,
		req.RequirementKey, req.RequirementType, req.Title, req.Description,
		req.Category, req.Priority, req.Status, acceptanceCriteriaJSON,
		req.CreatedAt, req.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create requirement: %w", err)
	}

	// Log the change in audit trail
	return db.logRequirementChange(req.ID, "created", nil, req)
}

// UpdateRequirement updates an existing requirement
func (db *DB) UpdateRequirement(req *Requirement) error {
	// Get the old requirement for audit logging
	oldReq, err := db.GetRequirementByID(req.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing requirement: %w", err)
	}

	// Update timestamp
	req.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Convert acceptance criteria to JSON
	acceptanceCriteriaJSON := "[]"
	if len(req.AcceptanceCriteria) > 0 {
		data, err := json.Marshal(req.AcceptanceCriteria)
		if err != nil {
			return fmt.Errorf("failed to marshal acceptance criteria: %w", err)
		}
		acceptanceCriteriaJSON = string(data)
	}

	query := `
		UPDATE requirements SET
			title = ?, description = ?, category = ?,
			priority = ?, status = ?, acceptance_criteria = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := db.Exec(query,
		req.Title, req.Description, req.Category,
		req.Priority, req.Status, acceptanceCriteriaJSON,
		req.UpdatedAt, req.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update requirement: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("requirement not found: %s", req.ID)
	}

	// Log the change in audit trail
	return db.logRequirementChange(req.ID, "updated", oldReq, req)
}

// UpdateRequirementDescription updates only the description of a requirement
func (db *DB) UpdateRequirementDescription(requirementID string, description string) error {
	// Get the old requirement for audit logging
	oldReq, err := db.GetRequirementByID(requirementID)
	if err != nil {
		return fmt.Errorf("failed to get existing requirement: %w", err)
	}

	query := `
		UPDATE requirements SET
			description = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := db.Exec(query, description, time.Now().UTC().Format(time.RFC3339), requirementID)
	if err != nil {
		return fmt.Errorf("failed to update requirement description: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("requirement not found: %s", requirementID)
	}

	// Create a copy of the old requirement with the new description for logging
	newReq := *oldReq
	newReq.Description = &description

	return db.logRequirementChange(requirementID, "updated", oldReq, &newReq)
}

// DeleteRequirement deletes a requirement and all its children
func (db *DB) DeleteRequirement(requirementID string) error {
	// Get the requirement for audit logging
	req, err := db.GetRequirementByID(requirementID)
	if err != nil {
		return fmt.Errorf("failed to get requirement: %w", err)
	}

	// Delete the requirement (CASCADE will handle children and related data)
	query := `DELETE FROM requirements WHERE id = ?`
	result, err := db.Exec(query, requirementID)
	if err != nil {
		return fmt.Errorf("failed to delete requirement: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("requirement not found: %s", requirementID)
	}

	// Log the deletion
	return db.logRequirementChange(requirementID, "deleted", req, nil)
}

// GetRequirementByID retrieves a requirement by its ID
func (db *DB) GetRequirementByID(requirementID string) (*Requirement, error) {
	var req Requirement
	var acceptanceCriteriaJSON string

	query := `
		SELECT id, project_id, component_id, phase_id, parent_requirement_id,
			requirement_key, requirement_type, title, description,
			category, priority, status, acceptance_criteria,
			created_at, updated_at
		FROM requirements
		WHERE id = ?
	`

	err := db.QueryRow(query, requirementID).Scan(
		&req.ID, &req.ProjectID, &req.ComponentID, &req.PhaseID, &req.ParentRequirementID,
		&req.RequirementKey, &req.RequirementType, &req.Title, &req.Description,
		&req.Category, &req.Priority, &req.Status, &acceptanceCriteriaJSON,
		&req.CreatedAt, &req.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("requirement not found: %s", requirementID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get requirement: %w", err)
	}

	// Parse acceptance criteria
	if acceptanceCriteriaJSON != "" && acceptanceCriteriaJSON != "[]" {
		err = json.Unmarshal([]byte(acceptanceCriteriaJSON), &req.AcceptanceCriteria)
		if err != nil {
			return nil, fmt.Errorf("failed to parse acceptance criteria: %w", err)
		}
	}

	return &req, nil
}

// GetRequirementByKey retrieves a requirement by its key and project ID
func (db *DB) GetRequirementByKey(projectID, requirementKey string) (*Requirement, error) {
	var req Requirement
	var acceptanceCriteriaJSON string

	query := `
		SELECT id, project_id, component_id, phase_id, parent_requirement_id,
			requirement_key, requirement_type, title, description,
			category, priority, status, acceptance_criteria,
			created_at, updated_at
		FROM requirements
		WHERE project_id = ? AND requirement_key = ?
	`

	err := db.QueryRow(query, projectID, requirementKey).Scan(
		&req.ID, &req.ProjectID, &req.ComponentID, &req.PhaseID, &req.ParentRequirementID,
		&req.RequirementKey, &req.RequirementType, &req.Title, &req.Description,
		&req.Category, &req.Priority, &req.Status, &acceptanceCriteriaJSON,
		&req.CreatedAt, &req.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("requirement not found: %s", requirementKey)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get requirement: %w", err)
	}

	// Parse acceptance criteria
	if acceptanceCriteriaJSON != "" && acceptanceCriteriaJSON != "[]" {
		err = json.Unmarshal([]byte(acceptanceCriteriaJSON), &req.AcceptanceCriteria)
		if err != nil {
			return nil, fmt.Errorf("failed to parse acceptance criteria: %w", err)
		}
	}

	return &req, nil
}

// GetRequirementsByProject retrieves all requirements for a project
func (db *DB) GetRequirementsByProject(projectID string) ([]*Requirement, error) {
	query := `
		SELECT id, project_id, component_id, phase_id, parent_requirement_id,
			requirement_key, requirement_type, title, description,
			category, priority, status, acceptance_criteria,
			created_at, updated_at
		FROM requirements
		WHERE project_id = ?
		ORDER BY requirement_key
	`

	rows, err := db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get requirements: %w", err)
	}
	defer rows.Close()

	var requirements []*Requirement
	for rows.Next() {
		var req Requirement
		var acceptanceCriteriaJSON string

		err := rows.Scan(
			&req.ID, &req.ProjectID, &req.ComponentID, &req.PhaseID, &req.ParentRequirementID,
			&req.RequirementKey, &req.RequirementType, &req.Title, &req.Description,
			&req.Category, &req.Priority, &req.Status, &acceptanceCriteriaJSON,
			&req.CreatedAt, &req.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan requirement: %w", err)
		}

		// Parse acceptance criteria
		if acceptanceCriteriaJSON != "" && acceptanceCriteriaJSON != "[]" {
			err = json.Unmarshal([]byte(acceptanceCriteriaJSON), &req.AcceptanceCriteria)
			if err != nil {
				return nil, fmt.Errorf("failed to parse acceptance criteria: %w", err)
			}
		}

		requirements = append(requirements, &req)
	}

	return requirements, nil
}

// GetChildRequirements retrieves all child requirements of a parent
func (db *DB) GetChildRequirements(parentRequirementID string) ([]*Requirement, error) {
	query := `
		SELECT id, project_id, component_id, phase_id, parent_requirement_id,
			requirement_key, requirement_type, title, description,
			category, priority, status, acceptance_criteria,
			created_at, updated_at
		FROM requirements
		WHERE parent_requirement_id = ?
		ORDER BY requirement_key
	`

	rows, err := db.Query(query, parentRequirementID)
	if err != nil {
		return nil, fmt.Errorf("failed to get child requirements: %w", err)
	}
	defer rows.Close()

	var requirements []*Requirement
	for rows.Next() {
		var req Requirement
		var acceptanceCriteriaJSON string

		err := rows.Scan(
			&req.ID, &req.ProjectID, &req.ComponentID, &req.PhaseID, &req.ParentRequirementID,
			&req.RequirementKey, &req.RequirementType, &req.Title, &req.Description,
			&req.Category, &req.Priority, &req.Status, &acceptanceCriteriaJSON,
			&req.CreatedAt, &req.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan requirement: %w", err)
		}

		// Parse acceptance criteria
		if acceptanceCriteriaJSON != "" && acceptanceCriteriaJSON != "[]" {
			err = json.Unmarshal([]byte(acceptanceCriteriaJSON), &req.AcceptanceCriteria)
			if err != nil {
				return nil, fmt.Errorf("failed to parse acceptance criteria: %w", err)
			}
		}

		requirements = append(requirements, &req)
	}

	return requirements, nil
}

// GenerateNextRequirementKey generates the next requirement key based on type and parent
func (db *DB) GenerateNextRequirementKey(projectID, componentID, requirementType string, parentID *string) (string, error) {
	var prefix string
	var query string
	var args []interface{}

	switch strings.ToLower(requirementType) {
	case "scope":
		prefix = "SCOPE-"
		query = `
			SELECT requirement_key FROM requirements
			WHERE project_id = ? AND component_id = ? AND requirement_type = 'scope'
			ORDER BY requirement_key DESC LIMIT 1
		`
		args = []interface{}{projectID, componentID}

	case "user_story":
		if parentID == nil {
			return "", fmt.Errorf("parent requirement ID required for user stories")
		}
		// Get parent scope key
		var parentKey string
		err := db.QueryRow("SELECT requirement_key FROM requirements WHERE id = ?", *parentID).Scan(&parentKey)
		if err != nil {
			return "", fmt.Errorf("failed to get parent requirement: %w", err)
		}
		prefix = parentKey + "-US-"
		query = `
			SELECT requirement_key FROM requirements
			WHERE project_id = ? AND parent_requirement_id = ? AND requirement_type = 'user_story'
			ORDER BY requirement_key DESC LIMIT 1
		`
		args = []interface{}{projectID, *parentID}

	case "tech_spec":
		if parentID == nil {
			return "", fmt.Errorf("parent requirement ID required for tech specs")
		}
		// Get parent story key
		var parentKey string
		err := db.QueryRow("SELECT requirement_key FROM requirements WHERE id = ?", *parentID).Scan(&parentKey)
		if err != nil {
			return "", fmt.Errorf("failed to get parent requirement: %w", err)
		}
		prefix = parentKey + "-TS-"
		query = `
			SELECT requirement_key FROM requirements
			WHERE project_id = ? AND parent_requirement_id = ? AND requirement_type = 'tech_spec'
			ORDER BY requirement_key DESC LIMIT 1
		`
		args = []interface{}{projectID, *parentID}

	default:
		return "", fmt.Errorf("invalid requirement type: %s", requirementType)
	}

	var lastKey sql.NullString
	err := db.QueryRow(query, args...).Scan(&lastKey)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to get last requirement key: %w", err)
	}

	// Generate new key
	if !lastKey.Valid || lastKey.String == "" {
		return prefix + "1", nil
	}

	// Extract the number from the last key and increment
	var lastNum int
	if requirementType == "scope" {
		fmt.Sscanf(lastKey.String, "SCOPE-%d", &lastNum)
	} else {
		// For user stories and tech specs, extract the last number after the last dash
		parts := []rune(lastKey.String)
		numStr := ""
		for i := len(parts) - 1; i >= 0 && parts[i] >= '0' && parts[i] <= '9'; i-- {
			numStr = string(parts[i]) + numStr
		}
		if numStr != "" {
			fmt.Sscanf(numStr, "%d", &lastNum)
		}
	}

	return fmt.Sprintf("%s%d", prefix, lastNum+1), nil
}

// logRequirementChange logs changes to the requirement_changes table
func (db *DB) logRequirementChange(requirementID, changeType string, oldReq, newReq *Requirement) error {
	var oldValuesJSON, newValuesJSON string

	if oldReq != nil {
		data, err := json.Marshal(oldReq)
		if err != nil {
			return fmt.Errorf("failed to marshal old values: %w", err)
		}
		oldValuesJSON = string(data)
	}

	if newReq != nil {
		data, err := json.Marshal(newReq)
		if err != nil {
			return fmt.Errorf("failed to marshal new values: %w", err)
		}
		newValuesJSON = string(data)
	}

	query := `
		INSERT INTO requirement_changes (
			id, requirement_id, change_type, old_values, new_values,
			changed_by, change_reason, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query,
		generateID(), requirementID, changeType, oldValuesJSON, newValuesJSON,
		"system", "", time.Now().UTC().Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to log requirement change: %w", err)
	}

	return nil
}

// generateID generates a unique ID (simplified version, could use UUID)
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}