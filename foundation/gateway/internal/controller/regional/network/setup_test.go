package network

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const TenantLabelKey = "secapi.cloud/tenant-id"

var cfg *rest.Config

// --- Envtest lifecycle ---
func TestMain(m *testing.M) {
	wd, _ := os.Getwd()
	crdDir := filepath.Clean(filepath.Join(wd, "../../../../../persistence/generated/crds/network"))
	testEnvironment := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{crdDir},
		DownloadBinaryAssets:  true,
		BinaryAssetsDirectory: filepath.Join(os.TempDir(), "envtest-binaries"),
	}
	var err error
	cfg, err = testEnvironment.Start()
	if err != nil {
		slog.Error("failed to start envtest", "error", err)
		os.Exit(1)
	}
	code := m.Run()
	if err := testEnvironment.Stop(); err != nil {
		slog.Error("failed to stop envtest", "error", err)
	}
	os.Exit(code)
}
