//go:build envtest

package kubernetes_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var cfg *rest.Config

func TestMain(m *testing.M) {
	wd, _ := os.Getwd()
	crdDir := filepath.Clean(filepath.Join(wd, "../../../../../testdata/crds/network"))
	testenv := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{crdDir},
		DownloadBinaryAssets:  true,
		BinaryAssetsDirectory: filepath.Join(os.TempDir(), "envtest-binaries"),
	}

	var err error
	cfg, err = testenv.Start()
	if err != nil {
		slog.Error("failed to start test environment", "error", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := testenv.Stop(); err != nil {
		slog.Error("failed to stop test environment", "error", err)
	}

	os.Exit(code)
}
