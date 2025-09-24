package models

import (
	"encoding/json"
)

// RTMData represents the complete RTM structure from YAML/JSON import
type RTMData struct {
	Metadata         RTMMetadata        `json:"metadata" yaml:"metadata"`
	Project          Project            `json:"project" yaml:"project"`
	SystemComponents []SystemComponent  `json:"components" yaml:"components"`
	Requirements     []Requirement      `json:"requirements" yaml:"requirements"`
	APIEndpoints     []APIEndpoint      `json:"api_endpoints,omitempty" yaml:"api_endpoints,omitempty"`
}

// RTMMetadata represents the metadata wrapper in the JSON
type RTMMetadata struct {
	GeneratedAt string  `json:"generated_at" yaml:"generated_at"`
	GeneratedBy string  `json:"generated_by" yaml:"generated_by"`
	Project     Project `json:"project" yaml:"project"`
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
	ComponentType string `json:"type" yaml:"type"`
	DeploymentUnit string `json:"deployment_unit,omitempty" yaml:"deployment_unit,omitempty"`
	Path          string `json:"path,omitempty" yaml:"path,omitempty"`
	Technology    string `json:"technology,omitempty" yaml:"technology,omitempty"`
	Description   string `json:"description,omitempty" yaml:"description,omitempty"`
	EntryPoint    string `json:"entry_point,omitempty" yaml:"entry_point,omitempty"`
	BasePath      string `json:"base_path,omitempty" yaml:"base_path,omitempty"`
}

type Requirement struct {
	ID               string                `json:"id" yaml:"id"`
	ComponentID      string                `json:"component_id" yaml:"component_id"`
	RequirementType  string                `json:"type" yaml:"type"` // scope, user_story, tech_spec
	Title            string                `json:"name" yaml:"name"`
	Description      string                `json:"description,omitempty" yaml:"description,omitempty"`
	Category         string                `json:"category" yaml:"category"`
	Priority         string                `json:"priority,omitempty" yaml:"priority,omitempty"`
	Status           string                `json:"status,omitempty" yaml:"status,omitempty"`
	AcceptanceCriteria []string            `json:"acceptance_criteria,omitempty" yaml:"acceptance_criteria,omitempty"`
	Children         []Requirement         `json:"children,omitempty" yaml:"children,omitempty"`
	Implementation   *Implementation       `json:"implementation,omitempty" yaml:"implementation,omitempty"`
	Tests            *TestCoverage         `json:"test_coverage,omitempty" yaml:"test_coverage,omitempty"`
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