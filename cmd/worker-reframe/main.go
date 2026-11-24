// Deprecated: worker-reframe runs as the Python service under workers/reframe.
// Compose builds workers/reframe/Dockerfile. This binary remains so `go build ./...`
// and the production multi-binary image stay green; it only prints a hint.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "worker-reframe is the Python MediaPipe service (workers/reframe). Use the compose worker-reframe image.")
	os.Exit(1)
}
