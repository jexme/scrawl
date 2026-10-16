//go:build !unix

package fonts

import "os"

func readFile(path string) ([]byte, error) { return os.ReadFile(path) }
