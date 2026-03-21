package graph

import "time"

// Bead represents a beads graph item (issue/task/epic) matching bd v0.61.0 JSON schema.
type Bead struct {
	ID                 string         `json:"id"`
	Title              string         `json:"title"`
	Description        string         `json:"description,omitempty"`
	AcceptanceCriteria string         `json:"acceptance_criteria,omitempty"`
	Status             string         `json:"status"`
	Priority           int            `json:"priority"`
	IssueType          string         `json:"issue_type"`
	Assignee           string         `json:"assignee,omitempty"`
	Owner              string         `json:"owner,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	CreatedBy          string         `json:"created_by,omitempty"`
	UpdatedAt          time.Time      `json:"updated_at"`
	ClosedAt           *time.Time     `json:"closed_at,omitempty"`
	CloseReason        string         `json:"close_reason,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	Labels             []string       `json:"labels,omitempty"`
	Dependencies       []Dependency   `json:"dependencies,omitempty"`
	Dependents         []BeadSummary  `json:"dependents,omitempty"`
	DependencyCount    int            `json:"dependency_count,omitempty"`
	DependentCount     int            `json:"dependent_count,omitempty"`
	CommentCount       int            `json:"comment_count,omitempty"`
	Parent             string         `json:"parent,omitempty"`
}

// Dependency represents a dependency relationship between beads.
// Note: "metadata" in dependency is a JSON string, not object.
type Dependency struct {
	IssueID     string    `json:"issue_id"`
	DependsOnID string    `json:"depends_on_id"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	Metadata    string    `json:"metadata"`
}

// BeadSummary appears in the "dependents" array from bd show --json.
type BeadSummary struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	IssueType      string `json:"issue_type"`
	DependencyType string `json:"dependency_type"`
}
