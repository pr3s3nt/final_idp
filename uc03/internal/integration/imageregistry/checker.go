// Package imageregistry verifies that an image tag exists before a deployment
// is accepted.
package imageregistry

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type Checker struct {
	Client *http.Client
}

var ecrHost = regexp.MustCompile(`^(\d+)\.dkr\.ecr\.([a-z0-9-]+)\.amazonaws\.com$`)

// ImageExists checks host/repository:tag in a plain HTTP local registry or in ECR.
func (c Checker) ImageExists(ctx context.Context, ref string) error {
	slash := strings.Index(ref, "/")
	colon := strings.LastIndex(ref, ":")
	if slash < 0 || colon < slash {
		return fmt.Errorf("image reference %q needs a registry host and a tag", ref)
	}
	host, repo, tag := ref[:slash], ref[slash+1:colon], ref[colon+1:]
	if m := ecrHost.FindStringSubmatch(host); m != nil {
		cmd := exec.CommandContext(ctx, "aws", "ecr", "describe-images", "--region", m[2], "--registry-id", m[1],
			"--repository-name", repo, "--image-ids", "imageTag="+tag, "--output", "json")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("not found in ECR: %s", strings.TrimSpace(lastLine(string(out))))
		}
		return nil
	}
	if !strings.HasPrefix(host, "localhost:") && !strings.HasPrefix(host, "127.0.0.1:") {
		return fmt.Errorf("registry %s is not supported for verification", host)
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, fmt.Sprintf("http://%s/v2/%s/manifests/%s", host, repo, tag), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.oci.image.index.v1+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.list.v2+json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry answered %s", resp.Status)
	}
	return nil
}

// Tags lists tags of a local registry repository for the form's suggestions.
func (c Checker) Tags(ctx context.Context, hostRepo string) []string {
	slash := strings.Index(hostRepo, "/")
	if slash < 0 || !strings.HasPrefix(hostRepo, "localhost:") {
		return nil
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/v2/%s/tags/list", hostRepo[:slash], hostRepo[slash+1:]), nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var body struct {
		Tags []string `json:"tags"`
	}
	if err := decodeJSON(resp, &body); err != nil {
		return nil
	}
	return body.Tags
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}
