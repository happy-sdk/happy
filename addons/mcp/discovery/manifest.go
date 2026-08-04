// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package discovery

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/goccy/go-yaml"
)

// ManifestVersion is the only .happy/mcp.yaml schema version understood.
const ManifestVersion = "1"

var (
	namespacePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	toolNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

// Manifest is a repository's .happy/mcp.yaml.
type Manifest struct {
	Version     string      `yaml:"version"`
	Namespace   string      `yaml:"namespace"`
	Description string      `yaml:"description"`
	Tools       []Tool      `yaml:"tools"`
	Server      *ServerSpec `yaml:"server"`
}

// Tool is one declaratively defined tool: run a command, return its output.
type Tool struct {
	Name        string         `yaml:"name"`
	Title       string         `yaml:"title"`
	Description string         `yaml:"description"`
	Input       map[string]any `yaml:"input"`
	Exec        ExecSpec       `yaml:"exec"`
	Output      string         `yaml:"output"`
}

// ExecSpec is how a declarative tool runs. Command is argv, executed without a
// shell; inputs are interpolated into separate argv entries and never
// concatenated into a command line.
type ExecSpec struct {
	Command []string  `yaml:"command"`
	Cwd     string    `yaml:"cwd"`
	Args    []ArgRule `yaml:"args"`
	Timeout Duration  `yaml:"timeout"`
}

// ArgRule appends argv entries when a condition interpolates to a non-empty
// value, which is how optional inputs stay optional without a shell.
type ArgRule struct {
	If  string   `yaml:"if"`
	Add []string `yaml:"add"`
}

// ServerSpec delegates to a server the repository ships itself. The org server
// spawns it and merges its tools in beneath the namespace.
type ServerSpec struct {
	Command   []string          `yaml:"command"`
	Transport string            `yaml:"transport"`
	Cwd       string            `yaml:"cwd"`
	Env       map[string]string `yaml:"env"`
}

// Duration is a time.Duration that unmarshals from YAML strings like "5m".
type Duration time.Duration

func (d *Duration) UnmarshalYAML(b []byte) error {
	var s string
	if err := yaml.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("%w: timeout %q: %s", ErrInvalidValue, s, err.Error())
	}
	*d = Duration(parsed)
	return nil
}

// Duration returns the value as a time.Duration, substituting fallback when
// unset so every tool has a bound.
func (d Duration) Duration(fallback time.Duration) time.Duration {
	if d == 0 {
		return fallback
	}
	return time.Duration(d)
}

func loadManifest(r *Repo, path string) (*Manifest, []Issue) {
	issue := func(format string, a ...any) []Issue {
		return []Issue{{Repo: r.Name, Path: relTo(r.Dir, path), Message: fmt.Sprintf(format, a...)}}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, issue("reading manifest: %s", err.Error())
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, issue("parsing manifest: %s", err.Error())
	}

	if m.Version != ManifestVersion {
		return nil, issue("unsupported version %q, want %q", m.Version, ManifestVersion)
	}
	if m.Namespace != "" && !namespacePattern.MatchString(m.Namespace) {
		return nil, issue("namespace %q must match %s", m.Namespace, namespacePattern)
	}

	var issues []Issue
	seen := make(map[string]bool, len(m.Tools))
	tools := m.Tools[:0]
	for _, t := range m.Tools {
		if err := validateTool(t); err != nil {
			issues = append(issues, Issue{Repo: r.Name, Path: relTo(r.Dir, path), Message: err.Error()})
			continue
		}
		if seen[t.Name] {
			issues = append(issues, Issue{Repo: r.Name, Path: relTo(r.Dir, path), Message: fmt.Sprintf("duplicate tool %q", t.Name)})
			continue
		}
		seen[t.Name] = true
		tools = append(tools, t)
	}
	m.Tools = tools

	if m.Server != nil && len(m.Server.Command) == 0 {
		issues = append(issues, Issue{Repo: r.Name, Path: relTo(r.Dir, path), Message: "server declared with no command"})
		m.Server = nil
	}

	return &m, issues
}

func validateTool(t Tool) error {
	if !toolNamePattern.MatchString(t.Name) {
		return fmt.Errorf("tool name %q must match %s", t.Name, toolNamePattern)
	}
	if t.Description == "" {
		return fmt.Errorf("tool %q has no description; it is the only signal for when to call a tool", t.Name)
	}
	if len(t.Exec.Command) == 0 {
		return fmt.Errorf("tool %q has no exec.command", t.Name)
	}
	switch t.Output {
	case "", "text", "json":
	default:
		return fmt.Errorf("tool %q has output %q, want \"text\" or \"json\"", t.Name, t.Output)
	}
	return nil
}

// DeclaredInputs is the set of input names the tool's schema declares.
//
// It is what separates a manifest bug from an omitted optional argument: a
// placeholder naming something outside this set is a typo that should fail
// loudly, while one naming a declared input the caller did not supply is
// simply absent and renders empty.
func (t Tool) DeclaredInputs() map[string]bool {
	props, ok := t.InputSchema()["properties"].(map[string]any)
	if !ok {
		return map[string]bool{}
	}
	declared := make(map[string]bool, len(props))
	for name := range props {
		declared[name] = true
	}
	return declared
}

// InputSchema returns the tool's JSON Schema, defaulting to an empty object
// schema. The MCP specification requires a non-nil object schema even for a
// tool that takes no arguments.
func (t Tool) InputSchema() map[string]any {
	if len(t.Input) == 0 {
		return map[string]any{"type": "object"}
	}
	if _, ok := t.Input["type"]; !ok {
		schema := make(map[string]any, len(t.Input)+1)
		for k, v := range t.Input {
			schema[k] = v
		}
		schema["type"] = "object"
		return schema
	}
	return t.Input
}
