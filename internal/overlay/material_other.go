//go:build !windows

package overlay

import "errors"

func privateMaterial(string, []byte) (string, func(), error) {
	return "", nil, errors.New("integrated beta requires Windows private material support")
}
