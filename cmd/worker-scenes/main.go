// Deprecated: worker-scenes runs as the Python service under workers/scenes.
// Compose builds workers/scenes/Dockerfile. This binary remains so `go build ./...`
// and the production multi-binary image stay green; it only prints a hint.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "worker-scenes is the Python PySceneDetect service (workers/scenes). Use the compose worker-scenes image.")
	os.Exit(1)
}
