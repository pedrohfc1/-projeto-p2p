//go:build !windows || nosystemaudio

package loopback

import (
	"context"
	"errors"
)

func Sources(context.Context, string) ([]Source, error) {
	return nil, errors.New("captura de áudio do sistema disponível somente no Windows")
}
func Start(context.Context, string, string, int) (*Session, error) {
	return nil, errors.New("captura de áudio do sistema disponível somente no Windows")
}
