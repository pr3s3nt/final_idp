package cd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"deploy/internal/domain"
	"deploy/internal/integration/deliveryrepo"
	"deploy/internal/integration/kubernetes"
	"deploy/internal/integration/secretstore"
)

// Delivery is what every CD adapter needs: the application's delivery
// repository (found or prepared), the Secret Store the keys live in, and the
// cluster to materialize secrets into.
type Delivery struct {
	Provider       deliveryrepo.Provider   // prepares a repository the first time
	Registry       DeliveryRepositoryStore // remembers it afterwards
	Secrets        secretstore.Store
	Branch         string
	KnownHostsFile string
	WorkDir        string
	Kube           *kubernetes.Adapter
}

// gitDelivery is not a component of its own: it is the half of the Concrete CD
// Provider that does not depend on which CD system syncs the cluster. It runs
// the chain the UC-03 sequence gives to CD Integration — find the delivery
// repository of the application, have the Delivery Repository Provider prepare
// it the first time, record it in the Delivery Repository Registry — and then
// publishes the desired state. Both concrete providers embed it, so they differ
// only in the object they create inside the cluster and in how they read status.
type gitDelivery struct {
	cfg Delivery

	mu sync.Mutex // serializes publishes, which share one working copy

	repoMu sync.Mutex
	repos  map[string]*deliveryAccess
}

// deliveryAccess is everything needed to work with one application's delivery
// repository: where it is, the working copy, and the write key materialized as
// a file because that is what ssh needs.
type deliveryAccess struct {
	url          string
	branch       string
	dir          string
	writeKeyFile string
	readKeyRef   string
}

// deliveryName is the name of the CD object of one application + environment.
func deliveryName(ds DesiredState) string {
	return fmt.Sprintf("%s-%s", ds.Application, strings.ToLower(ds.Environment))
}

// environmentPath is where the desired state of one environment lives inside
// the application's own repository.
func environmentPath(ds DesiredState) string {
	return fmt.Sprintf("%s/%s", ds.Target, strings.ToLower(ds.Environment))
}

func (g *gitDelivery) branch() string {
	if g.cfg.Branch != "" {
		return g.cfg.Branch
	}
	return "main"
}

// delivery returns the delivery repository of the application, preparing it on
// the application's first deployment: the provider creates the repository and
// the key pair, and the registry records where they are.
func (g *gitDelivery) delivery(ctx context.Context, ds DesiredState) (*deliveryAccess, error) {
	g.repoMu.Lock()
	defer g.repoMu.Unlock()
	if ra, ok := g.repos[ds.ApplicationID]; ok {
		return ra, nil
	}
	if ds.ApplicationID == "" {
		return nil, fmt.Errorf("desired state without an application: no delivery repository can be resolved")
	}
	stored, err := g.cfg.Registry.Find(ctx, ds.ApplicationID)
	if err != nil {
		return nil, fmt.Errorf("look up the delivery repository of %s: %w", ds.Application, err)
	}
	if stored == nil {
		prepared, err := g.cfg.Provider.EnsureDeliveryRepository(ctx, deliveryrepo.Application{ID: ds.ApplicationID, Name: ds.Application})
		if err != nil {
			return nil, fmt.Errorf("prepare the delivery repository of %s: %w", ds.Application, err)
		}
		stored, err = g.cfg.Registry.Save(ctx, domain.DeliveryRepository{
			ApplicationID: ds.ApplicationID, RepositoryURL: prepared.URL, Branch: prepared.Branch,
			WriteKeyReference: prepared.WriteKeyReference, ReadKeyReference: prepared.ReadKeyReference,
		})
		if err != nil {
			return nil, fmt.Errorf("record the delivery repository of %s: %w", ds.Application, err)
		}
	}

	base := filepath.Join(g.cfg.WorkDir, safeDir(ds.Application))
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}
	key, err := g.cfg.Secrets.Get(ctx, stored.WriteKeyReference)
	if err != nil {
		return nil, fmt.Errorf("write key of %s: %w", ds.Application, err)
	}
	keyFile := filepath.Join(base, "write-key")
	if err := os.WriteFile(keyFile, key, 0o600); err != nil {
		return nil, err
	}
	if err := g.ensureKnownHosts(ctx, stored.RepositoryURL); err != nil {
		return nil, err
	}
	ra := &deliveryAccess{url: stored.RepositoryURL, branch: stored.Branch, dir: filepath.Join(base, "repo"),
		writeKeyFile: keyFile, readKeyRef: stored.ReadKeyReference}
	if ra.branch == "" {
		ra.branch = g.branch()
	}
	if g.repos == nil {
		g.repos = map[string]*deliveryAccess{}
	}
	g.repos[ds.ApplicationID] = ra
	return ra, nil
}

// readKey returns the read-only key of an application, which a CD system is
// given so it can read that application's repository and no other.
func (g *gitDelivery) readKey(ctx context.Context, ra *deliveryAccess, application string) ([]byte, error) {
	key, err := g.cfg.Secrets.Get(ctx, ra.readKeyRef)
	if err != nil {
		return nil, fmt.Errorf("read key of %s: %w", application, err)
	}
	return key, nil
}

// knownHosts returns the recorded host keys, which a CD system needs too when
// it verifies the Git host strictly.
func (g *gitDelivery) knownHosts() ([]byte, error) {
	return os.ReadFile(g.cfg.KnownHostsFile)
}

// ensureKnownHosts records the host key of the Git hosting the first time it is
// needed, so every later connection is checked strictly against it.
func (g *gitDelivery) ensureKnownHosts(ctx context.Context, repoURL string) error {
	if info, err := os.Stat(g.cfg.KnownHostsFile); err == nil && info.Size() > 0 {
		return nil
	}
	host := repoURL
	if _, rest, ok := strings.Cut(host, "@"); ok {
		host = rest
	}
	host, _, _ = strings.Cut(host, ":")
	if host == "" {
		return fmt.Errorf("cannot tell the Git host from %q", repoURL)
	}
	out, err := exec.CommandContext(ctx, "ssh-keyscan", "-t", "rsa,ecdsa,ed25519", host).Output()
	if err != nil || len(out) == 0 {
		return fmt.Errorf("read the host key of %s: %v", host, err)
	}
	if err := os.MkdirAll(filepath.Dir(g.cfg.KnownHostsFile), 0o700); err != nil {
		return err
	}
	return os.WriteFile(g.cfg.KnownHostsFile, out, 0o600)
}

func (g *gitDelivery) git(ctx context.Context, ra *deliveryAccess, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o IdentitiesOnly=yes -o UserKnownHostsFile=%s -o StrictHostKeyChecking=yes", ra.writeKeyFile, g.cfg.KnownHostsFile),
		"GIT_AUTHOR_NAME=idp-uc03", "GIT_AUTHOR_EMAIL=idp-uc03@users.noreply.github.com",
		"GIT_COMMITTER_NAME=idp-uc03", "GIT_COMMITTER_EMAIL=idp-uc03@users.noreply.github.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *gitDelivery) checkout(ctx context.Context, ra *deliveryAccess) (string, error) {
	if _, err := os.Stat(filepath.Join(ra.dir, ".git")); err != nil {
		if err := os.MkdirAll(filepath.Dir(ra.dir), 0o700); err != nil {
			return "", err
		}
		if _, err := g.git(ctx, ra, filepath.Dir(ra.dir), "clone", "--branch", ra.branch, ra.url, filepath.Base(ra.dir)); err != nil {
			return "", err
		}
	}
	if _, err := g.git(ctx, ra, ra.dir, "fetch", "origin", ra.branch); err != nil {
		return "", err
	}
	if _, err := g.git(ctx, ra, ra.dir, "reset", "--hard", "origin/"+ra.branch); err != nil {
		return "", err
	}
	return ra.dir, nil
}

// applyClusterObjects creates the namespace of the application + environment
// and materializes secret values straight into the cluster. Secret values are
// never written to Git, whichever CD system reads the repository.
func (g *gitDelivery) applyClusterObjects(ctx context.Context, ds DesiredState) error {
	labels := map[string]string{"idp.dev/application": ds.Application, "idp.dev/environment": strings.ToLower(ds.Environment)}
	if err := g.cfg.Kube.EnsureNamespace(ctx, ds.Cluster, ds.Namespace, labels); err != nil {
		return err
	}
	for _, w := range ds.Upsert {
		var keep []string
		for _, s := range w.Secrets {
			if err := g.cfg.Kube.ApplySecret(ctx, ds.Cluster, ds.Namespace, s.Name, s.Labels, s.Data); err != nil {
				return fmt.Errorf("apply secret %s: %w", s.Name, err)
			}
			keep = append(keep, s.Name)
		}
		if err := g.cfg.Kube.PruneWorkloadSecrets(ctx, ds.Cluster, ds.Namespace, w.WorkloadID, keep); err != nil {
			return fmt.Errorf("prune secrets of workload %s: %w", w.WorkloadID, err)
		}
	}
	return nil
}

// commitDesiredState writes the manifests of upserted workloads, drops removed
// ones and pushes. The commit SHA it returns is the delivery reference.
func (g *gitDelivery) commitDesiredState(ctx context.Context, ra *deliveryAccess, ds DesiredState) (string, error) {
	for attempt := 1; ; attempt++ {
		dir, err := g.checkout(ctx, ra)
		if err != nil {
			return "", err
		}
		base := filepath.Join(dir, filepath.FromSlash(environmentPath(ds)), "workloads")
		for _, w := range ds.Upsert {
			wd := filepath.Join(base, w.WorkloadID)
			if err := os.MkdirAll(wd, 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(filepath.Join(wd, "manifests.yaml"), w.Manifests, 0o644); err != nil {
				return "", err
			}
		}
		for _, id := range ds.Remove {
			if err := os.RemoveAll(filepath.Join(base, id)); err != nil {
				return "", err
			}
		}
		// Git does not keep empty directories; a fixed marker keeps the
		// environment path valid when every workload has been removed.
		marker := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: idp-desired-state
  namespace: %s
  labels:
    idp.dev/managed-by: idp
data:
  application: %s
  environment: %s
`, ds.Namespace, ds.Application, strings.ToLower(ds.Environment))
		if err := os.MkdirAll(filepath.Dir(base), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(base), "idp-app.yaml"), []byte(marker), 0o644); err != nil {
			return "", err
		}
		if _, err := g.git(ctx, ra, dir, "add", "-A"); err != nil {
			return "", err
		}
		if status, _ := g.git(ctx, ra, dir, "status", "--porcelain"); status != "" {
			msg := fmt.Sprintf("%s %s: deployment %s wave %d", ds.Application, ds.Environment, ds.DeploymentID, ds.Wave)
			if _, err := g.git(ctx, ra, dir, "commit", "-m", msg); err != nil {
				return "", err
			}
			if _, err := g.git(ctx, ra, dir, "push", "origin", "HEAD:"+ra.branch); err != nil {
				if attempt < 3 {
					continue // another publish moved the branch; rebuild on top of it
				}
				return "", err
			}
		}
		return g.git(ctx, ra, dir, "rev-parse", "HEAD")
	}
}

// removeEnvironmentPath drops the desired state of one environment from the
// application's repository. The repository and its key pair are kept: the
// application may be deployed again later.
func (g *gitDelivery) removeEnvironmentPath(ctx context.Context, ra *deliveryAccess, ds DesiredState) error {
	for attempt := 1; ; attempt++ {
		dir, err := g.checkout(ctx, ra)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(filepath.Join(dir, filepath.FromSlash(environmentPath(ds)))); err != nil {
			return err
		}
		if _, err := g.git(ctx, ra, dir, "add", "-A"); err != nil {
			return err
		}
		if status, _ := g.git(ctx, ra, dir, "status", "--porcelain"); status == "" {
			return nil
		}
		if _, err := g.git(ctx, ra, dir, "commit", "-m", fmt.Sprintf("%s %s: remove application", ds.Application, ds.Environment)); err != nil {
			return err
		}
		if _, err := g.git(ctx, ra, dir, "push", "origin", "HEAD:"+ra.branch); err != nil {
			if attempt < 3 {
				continue
			}
			return err
		}
		return nil
	}
}

// containsCommit reports whether the revision a CD system synced already
// contains the commit that was delivered.
func (g *gitDelivery) containsCommit(ctx context.Context, ds DesiredState, revision, sha string) bool {
	ra, err := g.delivery(ctx, ds)
	if err != nil {
		return false
	}
	g.git(ctx, ra, ra.dir, "fetch", "origin", ra.branch)
	_, err = g.git(ctx, ra, ra.dir, "merge-base", "--is-ancestor", sha, revision)
	return err == nil
}

// safeDir keeps an application recognisable in a path or a Kubernetes object
// name without letting its name escape the directory it belongs in.
func safeDir(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
