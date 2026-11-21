package installwin

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"projetop2p/internal/lifecycle"
	"strings"
	"time"
	"unsafe"
)

type ownedProcess struct {
	PID     uint32
	Created uint64
	Path    string
}

func samePath(a, b string) bool {
	a = strings.TrimPrefix(a, `\\?\`)
	b = strings.TrimPrefix(b, `\\?\`)
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
func ownedHandle(h windows.Handle, dir string) (string, bool) {
	buf := make([]uint16, 32768)
	n := uint32(len(buf))
	if windows.QueryFullProcessImageName(h, 0, &buf[0], &n) != nil {
		return "", false
	}
	path := windows.UTF16ToString(buf[:n])
	known := false
	for _, name := range []string{"PeeTwoPee.exe", "ProjetoP2P.exe", "easytier/etcat-p2p.exe", "uninstall.exe"} {
		if samePath(path, filepath.Join(dir, filepath.FromSlash(name))) {
			known = true
		}
	}
	if !known {
		return "", false
	}
	var token windows.Token
	if windows.OpenProcessToken(h, windows.TOKEN_QUERY, &token) != nil {
		return "", false
	}
	defer token.Close()
	user, e := token.GetTokenUser()
	self, se := windows.GetCurrentProcessToken().GetTokenUser()
	return path, e == nil && se == nil && user.User.Sid.String() == self.User.Sid.String()
}
func processes(dir string) ([]ownedProcess, error) {
	snap, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		return nil, e
	}
	defer windows.CloseHandle(snap)
	item := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	out := []ownedProcess{}
	for e = windows.Process32First(snap, &item); e == nil; e = windows.Process32Next(snap, &item) {
		if item.ProcessID == uint32(os.Getpid()) {
			continue
		}
		h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, item.ProcessID)
		if err != nil {
			continue
		}
		path, own := ownedHandle(h, dir)
		created, ce := creationTime(h)
		windows.CloseHandle(h)
		if own && ce == nil {
			out = append(out, ownedProcess{PID: item.ProcessID, Path: path, Created: created})
		}
	}
	if e != windows.ERROR_NO_MORE_FILES {
		return nil, e
	}
	return out, nil
}
func terminateOwned(dir string, items []ownedProcess) error {
	for _, p := range items {
		h, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, p.PID)
		if e == windows.ERROR_INVALID_PARAMETER {
			continue
		}
		if e != nil {
			return e
		}
		path, owned := ownedHandle(h, dir)
		created, ce := creationTime(h)
		if !owned || !samePath(path, p.Path) || ce != nil || created != p.Created {
			windows.CloseHandle(h)
			return fmt.Errorf("identidade do processo mudou; nada encerrado")
		}
		e = windows.TerminateProcess(h, 0)
		if e == nil {
			_, e = windows.WaitForSingleObject(h, 10000)
		}
		windows.CloseHandle(h)
		if e != nil {
			return e
		}
	}
	return nil
}

// holdUpdate blocks concurrent installers AND new application startup. Process termination
// is scoped to exact installation paths and current-user handles; never process names alone.
func holdUpdate(dir string, confirm func(string) bool) (func(), error) {
	update, e := lifecycle.Exclusive(filepath.Join(dir, ".p2p-update.lock"))
	if e != nil {
		return nil, fmt.Errorf("outro instalador está em execução: %w", e)
	}
	fail := func(e error) (func(), error) { update.Close(); return nil, e }
	items, e := processes(dir)
	if e != nil {
		return fail(e)
	}
	if len(items) > 0 {
		if !confirm("PeeTwoPee ou seu auxiliar está ativo, possivelmente na bandeja. Encerrar as instâncias desta instalação para atualizar? Chamadas serão encerradas; dados salvos serão preservados.") {
			return fail(fmt.Errorf("atualização cancelada; arquivos preservados"))
		}
	}
	reset, e := lifecycle.RequestStop(dir)
	if e != nil {
		return fail(e)
	}
	// New versions receive the application-specific graceful shutdown event.
	until := time.Now().Add(8 * time.Second)
	for len(items) > 0 && time.Now().Before(until) {
		time.Sleep(100 * time.Millisecond)
		items, e = processes(dir)
		if e != nil {
			reset()
			return fail(e)
		}
	}
	if len(items) > 0 {
		// Older versions did not implement shutdown IPC. Ask separately before escalation.
		list := ""
		for _, p := range items {
			list += fmt.Sprintf("\n%s (PID %d)", p.Path, p.PID)
		}
		if !confirm("A versão anterior não respondeu ao encerramento. Você pode cancelar e usar Encerrar aplicativo na bandeja. Forçar somente estes processos desta instalação? Alterações ainda não salvas podem ser perdidas." + list) {
			reset()
			return fail(fmt.Errorf("encerre a versão anterior pela bandeja e tente novamente"))
		}
		if e = terminateOwned(dir, items); e != nil {
			reset()
			return fail(e)
		}
	}
	run, e := lifecycle.WaitExclusive(filepath.Join(dir, ".p2p-running.lock"), 10*time.Second)
	if e != nil {
		reset()
		return fail(fmt.Errorf("instância ainda ativa; nenhum arquivo substituído: %w", e))
	}
	items, e = processes(dir)
	if e != nil || len(items) > 0 {
		run.Close()
		reset()
		return fail(fmt.Errorf("processo desta instalação ainda está em execução"))
	}
	return func() { reset(); run.Close(); update.Close() }, nil
}

func creationTime(h windows.Handle) (uint64, error) {
	var created, exited, kernel, user windows.Filetime
	e := windows.GetProcessTimes(h, &created, &exited, &kernel, &user)
	return uint64(created.HighDateTime)<<32 | uint64(created.LowDateTime), e
}
