package artifact

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestDeterministicArchive(t *testing.T) {
	input := []byte("apiVersion: v1\nkind: ConfigMap\n")
	first, err := deterministicArchive(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := deterministicArchive(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("archive is not deterministic")
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	tarReader := tar.NewReader(gzipReader)
	header, err := tarReader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if header.Name != "manifests.yaml" || header.Mode != 0o644 {
		t.Fatalf("header = %#v", header)
	}
	content, err := io.ReadAll(tarReader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, input) {
		t.Fatalf("content = %q", content)
	}
}
