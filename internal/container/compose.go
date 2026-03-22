package container

import "errors"

// WriteComposeFragment writes a gsdw.compose.yaml fragment to dir.
// If force is false and the file already exists, returns an error.
// Returns the path to the written file.
func WriteComposeFragment(dir string, cfg ContainerConfig, force bool) (string, error) {
	return "", errors.New("not implemented")
}
