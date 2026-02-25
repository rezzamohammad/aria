package adapters

import "context"

// Capability maps to task role types for smart routing
type Capability string

const (
	CapCode     Capability = "code"
	CapResearch Capability = "research"
	CapRefactor Capability = "refactor"
	CapTest     Capability = "test"
	CapReview   Capability = "review"
	CapDocs     Capability = "docs"
	CapSecurity Capability = "security"
	CapRelease  Capability = "release"
)

// RoleToCapability maps agent roles to adapter capabilities
var RoleToCapability = map[string]Capability{
	"worker":      CapCode,
	"verifier":    CapReview,
	"architect":   CapRefactor,
	"researcher":  CapResearch,
	"qa":          CapTest,
	"security":    CapSecurity,
	"doc_system":  CapDocs,
	"release_ops": CapRelease,
	"librarian":   CapDocs,
	"pm":          CapResearch,
}

// AgentAdapter is the universal interface every CLI tool must satisfy
type AgentAdapter interface {
	Name() string
	IsAvailable() bool
	Execute(ctx context.Context, job Job) (*Result, error)
	Capabilities() []Capability
}

// Job is the unit of work dispatched to an adapter
type Job struct {
	TaskID      string
	TaskRole    string
	Prompt      string
	WorktreeDir string
	ClaudeMD    string // injected context file content
	MaxTokens   int
	TimeoutSecs int
}
