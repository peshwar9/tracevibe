package models

import (
	"encoding/json"
)

// RTMData represents the complete RTM structure from YAML/JSON import
type RTMData struct {
	Project          Project            `json:"project" yaml:"project"`
	SystemComponents []SystemComponent  `json:"system_components" yaml:"system_components"`
	Requirements     []Requirement      `json:"requirements" yaml:"requirements"`
	APIEndpoints     []APIEndpoint      `json:"api_endpoints,omitempty" yaml:"api_endpoints,omitempty"`
}

type Project struct {
	ID          string      `json:"id" yaml:"id"`
	Name        string      `json:"name" yaml:"name"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	TechStack   interface{} `json:"tech_stack,omitempty" yaml:"tech_stack,omitempty"`
	Repository  string      `json:"repository,omitempty" yaml:"repository,omitempty"`
	Version     string      `json:"version,omitempty" yaml:"version,omitempty"`
	LastUpdated string      `json:"last_updated,omitempty" yaml:"last_updated,omitempty"`
}

type SystemComponent struct {
	ID            string `json:"id" yaml:"id"`
	Name          string `json:"name" yaml:"name"`
	ComponentType string `json:"component_type" yaml:"component_type"`
	Technology    string `json:"technology,omitempty" yaml:"technology,omitempty"`
	Description   string `json:"description,omitempty" yaml:"description,omitempty"`
	BasePath      string `json:"base_path,omitempty" yaml:"base_path,omitempty"`
}

type Requirement struct {
	ID               string                `json:"id" yaml:"id"`
	ComponentID      string                `json:"component_id" yaml:"component_id"`
	RequirementType  string                `json:"requirement_type" yaml:"requirement_type"` // scope, user_story, tech_spec
	Title            string                `json:"title" yaml:"title"`
	Description      string                `json:"description,omitempty" yaml:"description,omitempty"`
	Category         string                `json:"category" yaml:"category"`
	Priority         string                `json:"priority,omitempty" yaml:"priority,omitempty"`
	Status           string                `json:"status,omitempty" yaml:"status,omitempty"`
	AcceptanceCriteria []string            `json:"acceptance_criteria,omitempty" yaml:"acceptance_criteria,omitempty"`
	Children         []Requirement         `json:"children,omitempty" yaml:"children,omitempty"`
	Implementation   *Implementation       `json:"implementation,omitempty" yaml:"implementation,omitempty"`
	Tests            *TestCoverage         `json:"tests,omitempty" yaml:"tests,omitempty"`
}

type Implementation struct {
	Backend  *BackendImpl  `json:"backend,omitempty" yaml:"backend,omitempty"`
	Frontend *FrontendImpl `json:"frontend,omitempty" yaml:"frontend,omitempty"`
	Database *DatabaseImpl `json:"database,omitempty" yaml:"database,omitempty"`
}

type BackendImpl struct {
	Files []FileImpl `json:"files" yaml:"files"`
}

type FrontendImpl struct {
	Files    []FileImpl `json:"files" yaml:"files"`
	APICalls []APICall  `json:"api_calls,omitempty" yaml:"api_calls,omitempty"`
}

type DatabaseImpl struct {
	Files  []FileImpl `json:"files" yaml:"files"`
	Tables []string   `json:"tables,omitempty" yaml:"tables,omitempty"`
}

type FileImpl struct {
	Path      string   `json:"path" yaml:"path"`
	Functions []string `json:"functions,omitempty" yaml:"functions,omitempty"`
}

type APICall struct {
	Method   string `json:"method" yaml:"method"`
	Endpoint string `json:"endpoint" yaml:"endpoint"`
}

type TestCoverage struct {
	Backend  []TestFile `json:"backend,omitempty" yaml:"backend,omitempty"`
	Frontend []TestFile `json:"frontend,omitempty" yaml:"frontend,omitempty"`
}

type TestFile struct {
	File      string   `json:"file" yaml:"file"`
	Functions []string `json:"functions" yaml:"functions"`
}

type APIEndpoint struct {
	Method      string `json:"method" yaml:"method"`
	Path        string `json:"path" yaml:"path"`
	Handler     string `json:"handler,omitempty" yaml:"handler,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Helper methods for JSON marshaling/unmarshaling of arrays
func (r *Requirement) MarshalAcceptanceCriteriaJSON() (string, error) {
	if len(r.AcceptanceCriteria) == 0 {
		return "[]", nil
	}
	bytes, err := json.Marshal(r.AcceptanceCriteria)
	return string(bytes), err
}

func (r *Requirement) UnmarshalAcceptanceCriteriaJSON(data string) error {
	if data == "" || data == "[]" {
		r.AcceptanceCriteria = nil
		return nil
	}
	return json.Unmarshal([]byte(data), &r.AcceptanceCriteria)
}

func MarshalStringSliceJSON(slice []string) (string, error) {
	if len(slice) == 0 {
		return "[]", nil
	}
	bytes, err := json.Marshal(slice)
	return string(bytes), err
}

func UnmarshalStringSliceJSON(data string) ([]string, error) {
	if data == "" || data == "[]" {
		return nil, nil
	}
	var slice []string
	err := json.Unmarshal([]byte(data), &slice)
	return slice, err
}