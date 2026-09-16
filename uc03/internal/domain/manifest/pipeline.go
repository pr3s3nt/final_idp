// Package manifest holds the manifest pipeline components of UC-03: Resolved
// Specification Generator, Manifest Generator / Score Renderer, Target Manifest
// Adapter, Environment Configuration Materializer and Secret Materializer.
package manifest

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/targetadapter"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/cd"
)

// ResolvedConfiguration is the transient result of resolving one workload's
// Environment Configuration.
type ResolvedConfiguration struct {
	Variables map[string]string
	Secrets   map[string]string
}

// ResolvedSpecification is the Score workload input: image, ports and
// non-secret variables. Secrets are never part of it.
type ResolvedSpecification struct {
	Score []byte
}

// GenerateResolvedApplicationSpecification builds the resolved Score spec.
func GenerateResolvedApplicationSpecification(w *domain.Workload, wd domain.WorkloadDeployment, rc *ResolvedConfiguration) (*ResolvedSpecification, error) {
	container := map[string]any{"image": wd.ImageRef()}
	if len(rc.Variables) > 0 {
		vars := map[string]any{}
		for k, v := range rc.Variables {
			// score-k8s interpolates ${...}; a literal $ must be written as $$.
			vars[k] = strings.ReplaceAll(v, "$", "$$")
		}
		container["variables"] = vars
	}
	spec := map[string]any{
		"apiVersion": "score.dev/v1b1",
		"metadata":   map[string]any{"name": w.Name},
		"containers": map[string]any{"main": container},
	}
	if w.Port != nil {
		spec["service"] = map[string]any{"ports": map[string]any{"web": map[string]any{"port": *w.Port, "targetPort": *w.Port}}}
	}
	body, err := yaml.Marshal(spec)
	return &ResolvedSpecification{Score: body}, err
}

// ScoreRenderer calls the score-k8s CLI.
type ScoreRenderer struct {
	Binary string
}

// GenerateKubernetesManifest renders base manifests with score-k8s.
func (r *ScoreRenderer) GenerateKubernetesManifest(ctx context.Context, spec *ResolvedSpecification, namespace string) ([]map[string]any, error) {
	dir, err := os.MkdirTemp("", "idp-score-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "score.yaml"), spec.Score, 0o600); err != nil {
		return nil, err
	}
	bin := r.Binary
	if bin == "" {
		bin = "score-k8s"
	}
	for _, args := range [][]string{{"init", "--no-sample"}, {"generate", "score.yaml", "--namespace", namespace}} {
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("score-k8s %s: %v: %s", args[0], err, strings.TrimSpace(string(out)))
		}
	}
	raw, err := os.ReadFile(filepath.Join(dir, "manifests.yaml"))
	if err != nil {
		return nil, err
	}
	var objs []map[string]any
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	for {
		var obj map[string]any
		if err := dec.Decode(&obj); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		if obj != nil {
			objs = append(objs, obj)
		}
	}
	if len(objs) == 0 {
		return nil, errors.New("score-k8s produced no manifests")
	}
	return objs, nil
}

// TargetInfo carries what the Target Manifest Adapter needs about the target.
type TargetInfo struct {
	Application    string
	Environment    string
	Namespace      string
	RegistryMirror string
}

// AdaptManifestForTarget sets the namespace, maps the image registry to the
// target cluster's registry, replaces score-k8s' random instance label with a
// stable one (so selectors never change between renders), labels objects with
// the stable workload ID and adds a readiness probe on the workload port.
func AdaptManifestForTarget(objs []map[string]any, w *domain.Workload, wd domain.WorkloadDeployment, t TargetInfo) ([]map[string]any, error) {
	// The marker is deliberately in the idp.dev namespace, not
	// app.kubernetes.io/managed-by: a CD system that deploys through Helm (Fleet
	// does) owns that well-known label and sets it to its own value, so claiming
	// it here makes every object differ from Git forever and the delivery never
	// reports Ready.
	stable := map[string]any{
		"app.kubernetes.io/name": w.Name, "app.kubernetes.io/instance": w.Name, "idp.dev/managed-by": "idp",
		"idp.dev/workload-id": wd.WorkloadID, "idp.dev/application": t.Application, "idp.dev/environment": strings.ToLower(t.Environment),
	}
	selector := map[string]any{"idp.dev/workload-id": wd.WorkloadID}
	for _, obj := range objs {
		delete(obj, "status")
		meta := child(obj, "metadata")
		meta["namespace"] = t.Namespace
		meta["labels"] = copyMap(stable)
		switch obj["kind"] {
		case "Deployment":
			spec := child(obj, "spec")
			spec["selector"] = map[string]any{"matchLabels": copyMap(selector)}
			tmpl := child(spec, "template")
			child(tmpl, "metadata")["labels"] = copyMap(stable)
			for _, c := range containers(obj) {
				c["image"] = targetadapter.ResolveImage(wd.ImageRepository, wd.ImageVersion, t.RegistryMirror)
				c["imagePullPolicy"] = "IfNotPresent"
				if w.Port != nil {
					c["readinessProbe"] = map[string]any{"tcpSocket": map[string]any{"port": *w.Port}, "periodSeconds": 3, "failureThreshold": 2}
				}
			}
		case "Service":
			child(obj, "spec")["selector"] = copyMap(selector)
		}
	}
	return objs, nil
}

// MaterializeEnvironmentConfiguration moves literal environment variables into
// a ConfigMap and references it, with a hash annotation so a value change rolls
// the pods.
func MaterializeEnvironmentConfiguration(objs []map[string]any, w *domain.Workload, wd domain.WorkloadDeployment, t TargetInfo, hashKey []byte) ([]map[string]any, error) {
	name := w.Name + "-config"
	data := map[string]any{}
	for _, c := range containers(objs...) {
		env, _ := c["env"].([]any)
		for i, e := range env {
			m := e.(map[string]any)
			if v, ok := m["value"]; ok {
				key := fmt.Sprint(m["name"])
				data[key] = fmt.Sprint(v)
				env[i] = map[string]any{"name": key, "valueFrom": map[string]any{"configMapKeyRef": map[string]any{"name": name, "key": key}}}
			}
		}
	}
	cm := map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": name, "namespace": t.Namespace, "labels": map[string]any{
			"idp.dev/workload-id": wd.WorkloadID, "idp.dev/managed-by": "idp"}},
		"data": data,
	}
	annotate(objs, "idp.dev/config-hash", hmacOf(hashKey, data))
	return append(objs, cm), nil
}

// MaterializeSecretConfiguration references a Secret for every resolved secret
// and returns the Secret object to apply out of band. The manifests only carry
// a keyed hash, never a value.
func MaterializeSecretConfiguration(objs []map[string]any, w *domain.Workload, wd domain.WorkloadDeployment, rc *ResolvedConfiguration, hashKey []byte) ([]map[string]any, []cd.SecretObject, error) {
	if len(rc.Secrets) == 0 {
		return objs, nil, nil
	}
	name := w.Name + "-secrets"
	keys := make([]string, 0, len(rc.Secrets))
	for k := range rc.Secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	data := map[string][]byte{}
	hashInput := map[string]any{}
	for _, k := range keys {
		data[k] = []byte(rc.Secrets[k])
		hashInput[k] = rc.Secrets[k]
	}
	for _, c := range containers(objs...) {
		env, _ := c["env"].([]any)
		for _, k := range keys {
			env = append(env, map[string]any{"name": k, "valueFrom": map[string]any{"secretKeyRef": map[string]any{"name": name, "key": k}}})
		}
		c["env"] = env
	}
	annotate(objs, "idp.dev/secret-hash", hmacOf(hashKey, hashInput))
	secret := cd.SecretObject{Name: name, Data: data, Labels: map[string]string{"idp.dev/workload-id": wd.WorkloadID, "idp.dev/managed-by": "idp"}}
	return objs, []cd.SecretObject{secret}, nil
}

// Encode serializes manifests deterministically (sorted keys, sorted objects).
func Encode(objs []map[string]any) ([]byte, error) {
	sort.SliceStable(objs, func(i, j int) bool {
		return fmt.Sprint(objs[i]["kind"], child(objs[i], "metadata")["name"]) < fmt.Sprint(objs[j]["kind"], child(objs[j], "metadata")["name"])
	})
	var buf bytes.Buffer
	for _, o := range objs {
		buf.WriteString("---\n")
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(o); err != nil {
			return nil, err
		}
		enc.Close()
	}
	return buf.Bytes(), nil
}

func child(m map[string]any, key string) map[string]any {
	v, ok := m[key].(map[string]any)
	if !ok {
		v = map[string]any{}
		m[key] = v
	}
	return v
}

func containers(objs ...map[string]any) []map[string]any {
	var out []map[string]any
	for _, obj := range objs {
		if obj["kind"] != "Deployment" {
			continue
		}
		podSpec := child(child(child(obj, "spec"), "template"), "spec")
		list, _ := podSpec["containers"].([]any)
		for _, c := range list {
			if m, ok := c.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func annotate(objs []map[string]any, key, value string) {
	for _, obj := range objs {
		if obj["kind"] == "Deployment" {
			child(child(child(child(obj, "spec"), "template"), "metadata"), "annotations")[key] = value
		}
	}
}

func hmacOf(key []byte, v any) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(domain.CanonicalJSON(v))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
