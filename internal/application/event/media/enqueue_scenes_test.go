package media_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkerDI_ProxyReadyDoesNotEnqueueScenesOrReframe(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "cmd", "worker", "di", "container.go")
	b, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if strings.Contains(src, `"enqueue_scenes_on_proxy_ready"`) {
		t.Fatal("scenes must not be registered on proxy_ready")
	}
	if strings.Contains(src, `"enqueue_reframe_on_proxy_ready"`) {
		t.Fatal("reframe must not be registered on proxy_ready")
	}
}
