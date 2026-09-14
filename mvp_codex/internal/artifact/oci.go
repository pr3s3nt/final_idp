package artifact

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"strings"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/worker"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

const (
	manifestLayerMediaType = "application/vnd.oci.image.layer.v1.tar+gzip"
	artifactType           = "application/vnd.idp.kubernetes.manifests.v1"
)

type OCI struct {
	Repository         string
	SourceRepository   string
	PlainHTTP          bool
}

func (p *OCI) Publish(ctx context.Context, deploymentID string, manifests []byte) (worker.Artifact, error) {
	if p.Repository == "" || deploymentID == "" || len(manifests) == 0 {
		return worker.Artifact{}, fmt.Errorf("INVALID_ARTIFACT: repository, deployment ID and manifests are required")
	}
	repositoryName := strings.TrimPrefix(strings.TrimPrefix(p.Repository, "oci://"), "https://")
	repositoryName = strings.TrimPrefix(repositoryName, "http://")
	repositoryName = strings.TrimSuffix(repositoryName, "/")
	repository, err := remote.NewRepository(repositoryName)
	if err != nil {
		return worker.Artifact{}, fmt.Errorf("create OCI repository client: %w", err)
	}
	repository.PlainHTTP = p.PlainHTTP
	store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err != nil {
		return worker.Artifact{}, fmt.Errorf("open Docker credential store: %w", err)
	}
	repository.Client = &auth.Client{Cache: auth.NewCache(), Credential: credentials.Credential(store)}

	layer, err := deterministicArchive(manifests)
	if err != nil {
		return worker.Artifact{}, err
	}
	layerDescriptor, err := oras.PushBytes(ctx, repository, manifestLayerMediaType, layer)
	if err != nil {
		return worker.Artifact{}, fmt.Errorf("push OCI manifest layer: %w", err)
	}
	descriptor, err := oras.PackManifest(ctx, repository, oras.PackManifestVersion1_0, artifactType, oras.PackManifestOptions{
		Layers: []ocispec.Descriptor{layerDescriptor},
		ManifestAnnotations: map[string]string{
			ocispec.AnnotationCreated: "1970-01-01T00:00:00Z",
			"io.idp.deployment-id":    deploymentID,
		},
	})
	if err != nil {
		return worker.Artifact{}, fmt.Errorf("pack OCI manifest: %w", err)
	}
	tag := "deployment-" + strings.ToLower(deploymentID)
	if err := repository.Tag(ctx, descriptor, tag); err != nil {
		return worker.Artifact{}, fmt.Errorf("tag OCI manifest: %w", err)
	}
	sourceRepository := strings.TrimSuffix(strings.TrimPrefix(p.SourceRepository, "oci://"), "/")
	if sourceRepository == "" {
		sourceRepository = repositoryName
	}
	return worker.Artifact{URI: "oci://" + sourceRepository, Digest: descriptor.Digest.String()}, nil
}

func deterministicArchive(manifests []byte) ([]byte, error) {
	var buffer bytes.Buffer
	gzipWriter, err := gzip.NewWriterLevel(&buffer, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	gzipWriter.Header.ModTime = time.Unix(0, 0).UTC()
	gzipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)
	header := &tar.Header{
		Name:     "manifests.yaml",
		Mode:     0o644,
		Size:     int64(len(manifests)),
		ModTime:  time.Unix(0, 0).UTC(),
		Typeflag: tar.TypeReg,
		Format:   tar.FormatUSTAR,
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := tarWriter.Write(manifests); err != nil {
		return nil, err
	}
	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
