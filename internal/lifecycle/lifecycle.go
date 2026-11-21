// Package lifecycle coordinates app instances and installation updates without touching profile contents.
package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"projetop2p/internal/installtxn"
	"sync"
)

type Lock struct {
	file *os.File
	once sync.Once
}

func acquire(path string, exclusive bool) (*Lock, error) {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, e
	}
	if _, e := installtxn.Safe(filepath.Dir(path), filepath.Base(path)); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = lockFile(f, exclusive); e != nil {
		f.Close()
		return nil, e
	}
	return &Lock{file: f}, nil
}
func (l *Lock) Close() {
	if l != nil {
		l.once.Do(func() { unlockFile(l.file); l.file.Close() })
	}
}
func Exclusive(path string) (*Lock, error) { return acquire(path, true) }
func Shared(path string) (*Lock, error)    { return acquire(path, false) }

type Guard struct {
	profile, run *Lock
	stop         <-chan struct{}
	closeEvent   func()
}

func (g *Guard) StopRequests() <-chan struct{} { return g.stop }
func (g *Guard) Close() {
	if g == nil {
		return
	}
	if g.closeEvent != nil {
		g.closeEvent()
	}
	g.run.Close()
	g.profile.Close()
}
func Acquire(profile, install string) (*Guard, error) {
	g := &Guard{}
	var e error
	g.profile, e = Exclusive(filepath.Join(profile, ".p2p-instance.lock"))
	if e != nil {
		return nil, errors.New("este perfil já está aberto; confira a bandeja antes de iniciar outra instância")
	}
	fail := func(e error) (*Guard, error) { g.Close(); return nil, e }
	if _, e = os.Stat(filepath.Join(install, installtxn.ManifestName)); e == nil {
		startup, e := Shared(filepath.Join(install, ".p2p-update.lock"))
		if e != nil {
			return fail(errors.New("atualização em andamento; aguarde o instalador"))
		}
		defer startup.Close()
		g.run, e = Shared(filepath.Join(install, ".p2p-running.lock"))
		if e != nil {
			return fail(errors.New("atualização em andamento; aguarde o instalador"))
		}
		if e = installtxn.VerifyRuntime(install); e != nil {
			return fail(e)
		}
	}
	if _, e = os.Stat(filepath.Join(install, installtxn.JournalName)); e == nil {
		return fail(errors.New("atualização interrompida; execute o instalador para recuperar"))
	}
	g.stop, g.closeEvent, e = watchStop(install)
	if e != nil {
		return fail(e)
	}
	return g, nil
}
