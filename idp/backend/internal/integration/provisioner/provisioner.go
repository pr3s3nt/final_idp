// Package provisioner is the Provisioner Adapter abstraction and its Terraform
// implementation (the Terraform/OpenTofu Runner of the design).
package provisioner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"deploy/internal/domain"
)

// Request identifies one Resource Instance workspace and the module that
// provisions it. Variables are passed to the module as-is.
type Request struct {
	InstanceID           string
	ProvisionerReference string
	Variables            map[string]any
}

type Result struct {
	InfrastructureReference string
	ProviderStateReference  string
}

type Provisioner interface {
	Reconcile(ctx context.Context, req Request) (*Result, error)
	Destroy(ctx context.Context, req Request) error
	// Outputs reads the current outputs from provider state; values are never
	// persisted by the IDP.
	Outputs(ctx context.Context, instanceID string) (map[string]string, error)
}

// Terraform runs one workspace per Resource Instance. The module named by
// provisioner_reference is copied into the workspace on every apply, so the
// catalog stays authoritative at execution time.
type Terraform struct {
	ModulesDir string
	DataDir    string
	Log        io.Writer

	mu sync.Mutex
}

func (t *Terraform) workspace(instanceID string) string {
	return filepath.Join(t.DataDir, "terraform", "workspaces", instanceID)
}

func (t *Terraform) moduleDir(ref string) (string, error) {
	name, ok := strings.CutPrefix(ref, "terraform://modules/")
	if !ok || name == "" || strings.ContainsAny(name, `/\.`) {
		return "", fmt.Errorf("unsupported provisioner reference %q", ref)
	}
	dir := filepath.Join(t.ModulesDir, name)
	if _, err := os.Stat(filepath.Join(dir, "main.tf")); err != nil {
		return "", fmt.Errorf("module for %q not found: %w", ref, err)
	}
	return dir, nil
}

func (t *Terraform) prepare(req Request) (string, error) {
	src, err := t.moduleDir(req.ProvisionerReference)
	if err != nil {
		return "", err
	}
	ws := t.workspace(req.InstanceID)
	if err := os.MkdirAll(ws, 0o700); err != nil {
		return "", err
	}
	old, _ := filepath.Glob(filepath.Join(ws, "*.tf"))
	for _, f := range old {
		os.Remove(f)
	}
	files, _ := filepath.Glob(filepath.Join(src, "*.tf"))
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(ws, filepath.Base(f)), body, 0o600); err != nil {
			return "", err
		}
	}
	vars, err := json.MarshalIndent(req.Variables, "", "  ")
	if err != nil {
		return "", err
	}
	// Variables can include outputs of required resources (for example a
	// kubeconfig); the file is private to the workspace and never committed.
	return ws, os.WriteFile(filepath.Join(ws, "terraform.tfvars.json"), vars, 0o600)
}

func (t *Terraform) run(ctx context.Context, ws string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Dir = ws
	cmd.Env = append(os.Environ(), "TF_IN_AUTOMATION=1", "TF_INPUT=0",
		"TF_PLUGIN_CACHE_DIR="+filepath.Join(t.DataDir, "terraform", "plugin-cache"))
	var out bytes.Buffer
	logFile, err := os.OpenFile(filepath.Join(ws, "terraform.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	defer logFile.Close()
	fmt.Fprintf(logFile, "\n$ terraform %s\n", strings.Join(args, " "))
	writers := []io.Writer{&out, logFile}
	if t.Log != nil {
		writers = append(writers, t.Log)
	}
	cmd.Stdout = io.MultiWriter(writers...)
	cmd.Stderr = cmd.Stdout
	if err := cmd.Run(); err != nil {
		return out.Bytes(), fmt.Errorf("terraform %s failed: %v: %s", args[0], err, tail(out.String(), 15))
	}
	return out.Bytes(), nil
}

func (t *Terraform) Reconcile(ctx context.Context, req Request) (*Result, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := os.MkdirAll(filepath.Join(t.DataDir, "terraform", "plugin-cache"), 0o755); err != nil {
		return nil, err
	}
	ws, err := t.prepare(req)
	if err != nil {
		return nil, err
	}
	if _, err := t.run(ctx, ws, "init", "-input=false", "-no-color", "-upgrade=false"); err != nil {
		return nil, err
	}
	if _, err := t.run(ctx, ws, "apply", "-auto-approve", "-input=false", "-no-color"); err != nil {
		return nil, err
	}
	state := filepath.Join(ws, "terraform.tfstate")
	return &Result{InfrastructureReference: "terraform-workspace://" + req.InstanceID, ProviderStateReference: state}, nil
}

// Destroy destroys with the variables of the last apply, which the workspace
// keeps, so a required resource's outputs need not be re-read.
func (t *Terraform) Destroy(ctx context.Context, req Request) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	ws := t.workspace(req.InstanceID)
	if _, err := os.Stat(filepath.Join(ws, "terraform.tfvars.json")); err != nil {
		if _, err := t.prepare(req); err != nil {
			return err
		}
	} else if src, err := t.moduleDir(req.ProvisionerReference); err == nil {
		files, _ := filepath.Glob(filepath.Join(src, "*.tf"))
		for _, f := range files {
			body, _ := os.ReadFile(f)
			os.WriteFile(filepath.Join(ws, filepath.Base(f)), body, 0o600)
		}
	}
	if _, err := t.run(ctx, ws, "init", "-input=false", "-no-color"); err != nil {
		return err
	}
	_, err := t.run(ctx, ws, "destroy", "-auto-approve", "-input=false", "-no-color")
	return err
}

func (t *Terraform) Outputs(ctx context.Context, instanceID string) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "terraform", "output", "-json", "-no-color")
	cmd.Dir = t.workspace(instanceID)
	raw, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("read outputs of %s: %w", instanceID, err)
	}
	var parsed map[string]struct {
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for k, v := range parsed {
		var s string
		if err := json.Unmarshal(v.Value, &s); err == nil {
			out[k] = s
		} else {
			out[k] = string(v.Value)
		}
	}
	return out, nil
}

// tail prefers Terraform's "Error:" blocks; otherwise the last lines.
func tail(s string, lines int) string {
	if i := strings.Index(s, "Error: "); i >= 0 {
		s = s[i:]
		parts := strings.Split(strings.TrimSpace(s), "\n")
		if len(parts) > lines {
			parts = parts[:lines]
		}
		return strings.Join(parts, "\n")
	}
	parts := strings.Split(strings.TrimSpace(s), "\n")
	if len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	return strings.Join(parts, "\n")
}

// Snake converts a resource type such as k8s-cluster to a Terraform variable name.
func Snake(s string) string {
	return strings.ToLower(strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(s))
}

// ensure domain import is used for documentation links in godoc.
var _ = domain.ActionCreate
