package deliveryrepo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pr3s3nt/final_idp/idp/backend/internal/integration/secretstore"
)

// GitHub creates the delivery repository of an application on GitHub and gives
// it a key pair of its own: a write key the IDP pushes with and a read key the
// CD system is given. The Git hosting credential is read from the Secret Store
// on every call; it is never held in the database, in a manifest or in a log
// line, and neither are the generated keys.
type GitHub struct {
	Pattern        string // owner/idp-<app>-gitops
	Branch         string
	TokenReference string // idpsecret://platform/git-hosting-token
	Secrets        secretstore.Store
	WorkDir        string // keys are generated here, never under /tmp
	APIBase        string // https://api.github.com
	SSHHost        string // github.com
	HTTP           *http.Client
}

const (
	writeKeyTitle = "idp-delivery-write"
	readKeyTitle  = "idp-delivery-read"
)

// EnsureDeliveryRepository returns the delivery repository of an application,
// creating it the first time. A repository that already carries the name the
// platform pattern produces is reused instead of reported as an error.
func (g *GitHub) EnsureDeliveryRepository(ctx context.Context, app Application) (Repository, error) {
	owner, name, err := RepositoryName(g.Pattern, app.Name)
	if err != nil {
		return Repository{}, err
	}
	token, err := g.token(ctx)
	if err != nil {
		return Repository{}, err
	}
	if err := g.ensureRepository(ctx, token, owner, name, app.Name); err != nil {
		return Repository{}, err
	}
	write, read, err := g.ensureKeyPair(ctx, token, owner, name, app.Name)
	if err != nil {
		return Repository{}, err
	}
	return Repository{
		URL:               fmt.Sprintf("git@%s:%s/%s.git", g.sshHost(), owner, name),
		Branch:            g.branch(),
		WriteKeyReference: write,
		ReadKeyReference:  read,
	}, nil
}

func (g *GitHub) ensureRepository(ctx context.Context, token, owner, name, application string) error {
	status, _, err := g.api(ctx, token, http.MethodGet, fmt.Sprintf("/repos/%s/%s", owner, name), nil)
	if err != nil {
		return err
	}
	if status == http.StatusOK {
		return nil
	}
	if status != http.StatusNotFound {
		return fmt.Errorf("look up delivery repository %s/%s: unexpected status %d", owner, name, status)
	}
	// auto_init creates the first commit so the branch exists and can be cloned.
	body := map[string]any{
		"name": name, "private": true, "auto_init": true,
		"description": fmt.Sprintf("Desired state of application %s, managed by the IDP", application),
	}
	path := "/user/repos"
	if login, err := g.login(ctx, token); err != nil {
		return err
	} else if !strings.EqualFold(login, owner) {
		path = "/orgs/" + owner + "/repos"
	}
	status, resp, err := g.api(ctx, token, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("create delivery repository %s/%s: status %d: %s", owner, name, status, firstLine(resp))
	}
	return nil
}

// ensureKeyPair generates a key pair for this application, stores both private
// keys in the Secret Store and attaches the public keys to the repository. Key
// pairs the IDP attached before are replaced, so a repository never accumulates
// keys nobody holds any more.
func (g *GitHub) ensureKeyPair(ctx context.Context, token, owner, name, application string) (writeRef, readRef string, err error) {
	dir, err := os.MkdirTemp(g.WorkDir, "keys-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(dir)

	writePriv, writePub, err := sshKeyPair(dir, "write", "idp-"+application+"-write")
	if err != nil {
		return "", "", err
	}
	readPriv, readPub, err := sshKeyPair(dir, "read", "idp-"+application+"-read")
	if err != nil {
		return "", "", err
	}
	if err := g.removeKeys(ctx, token, owner, name); err != nil {
		return "", "", err
	}
	for _, k := range []struct {
		title    string
		pub      []byte
		readOnly bool
	}{{writeKeyTitle, writePub, false}, {readKeyTitle, readPub, true}} {
		status, resp, err := g.api(ctx, token, http.MethodPost, fmt.Sprintf("/repos/%s/%s/keys", owner, name),
			map[string]any{"title": k.title, "key": strings.TrimSpace(string(k.pub)), "read_only": k.readOnly})
		if err != nil {
			return "", "", err
		}
		if status != http.StatusCreated {
			return "", "", fmt.Errorf("attach %s key to %s/%s: status %d: %s", k.title, owner, name, status, firstLine(resp))
		}
	}
	base := "delivery/" + slug(application)
	if writeRef, err = g.Secrets.Put(ctx, base+"/write-key", writePriv); err != nil {
		return "", "", err
	}
	if readRef, err = g.Secrets.Put(ctx, base+"/read-key", readPriv); err != nil {
		return "", "", err
	}
	return writeRef, readRef, nil
}

func (g *GitHub) removeKeys(ctx context.Context, token, owner, name string) error {
	status, resp, err := g.api(ctx, token, http.MethodGet, fmt.Sprintf("/repos/%s/%s/keys", owner, name), nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("list deploy keys of %s/%s: status %d: %s", owner, name, status, firstLine(resp))
	}
	var keys []struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(resp, &keys); err != nil {
		return err
	}
	for _, k := range keys {
		if k.Title != writeKeyTitle && k.Title != readKeyTitle {
			continue
		}
		status, resp, err := g.api(ctx, token, http.MethodDelete, fmt.Sprintf("/repos/%s/%s/keys/%d", owner, name, k.ID), nil)
		if err != nil {
			return err
		}
		if status != http.StatusNoContent {
			return fmt.Errorf("remove deploy key %d of %s/%s: status %d: %s", k.ID, owner, name, status, firstLine(resp))
		}
	}
	return nil
}

func (g *GitHub) login(ctx context.Context, token string) (string, error) {
	status, resp, err := g.api(ctx, token, http.MethodGet, "/user", nil)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("read the Git hosting account: status %d: %s", status, firstLine(resp))
	}
	var user struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(resp, &user); err != nil {
		return "", err
	}
	return user.Login, nil
}

func (g *GitHub) token(ctx context.Context) (string, error) {
	if g.TokenReference == "" {
		return "", fmt.Errorf("no Git hosting credential is configured for delivery repositories")
	}
	v, err := g.Secrets.Get(ctx, g.TokenReference)
	if err != nil {
		return "", fmt.Errorf("Git hosting credential: %w", err)
	}
	return strings.TrimSpace(string(v)), nil
}

// api performs one Git hosting API call. The credential travels in the header
// only; no error message it returns contains it.
func (g *GitHub) api(ctx context.Context, token, method, path string, body any) (int, []byte, error) {
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, g.apiBase()+path, payload)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := g.client().Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, out, err
}

func (g *GitHub) client() *http.Client {
	if g.HTTP != nil {
		return g.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (g *GitHub) apiBase() string {
	if g.APIBase != "" {
		return strings.TrimSuffix(g.APIBase, "/")
	}
	return "https://api.github.com"
}

func (g *GitHub) sshHost() string {
	if g.SSHHost != "" {
		return g.SSHHost
	}
	return "github.com"
}

func (g *GitHub) branch() string {
	if g.Branch != "" {
		return g.Branch
	}
	return "main"
}

// sshKeyPair generates one ed25519 key pair inside dir and returns it. The
// files stay in dir, which the caller removes.
func sshKeyPair(dir, name, comment string) (private, public []byte, err error) {
	path := filepath.Join(dir, name)
	cmd := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", comment, "-f", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("generate %s key: %v: %s", name, err, strings.TrimSpace(string(out)))
	}
	if private, err = os.ReadFile(path); err != nil {
		return nil, nil, err
	}
	if public, err = os.ReadFile(path + ".pub"); err != nil {
		return nil, nil, err
	}
	return private, public, nil
}

// firstLine keeps an API error readable in a step summary.
func firstLine(body []byte) string {
	var e struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &e) == nil && e.Message != "" {
		return e.Message
	}
	s := strings.TrimSpace(string(body))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
