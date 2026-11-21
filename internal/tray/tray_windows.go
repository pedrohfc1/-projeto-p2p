package tray

import (
	"errors"
	"golang.org/x/sys/windows"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

var user = windows.NewLazySystemDLL("user32.dll")
var shell = windows.NewLazySystemDLL("shell32.dll")

type notifyIcon struct {
	Size                uint32
	Window              windows.Handle
	ID, Flags, Callback uint32
	Icon                windows.Handle
	Tip                 [128]uint16
	State, StateMask    uint32
	Info                [256]uint16
	Version             uint32
	InfoTitle           [64]uint16
	InfoFlags           uint32
	Guid                windows.GUID
	Balloon             windows.Handle
}
type wndClass struct {
	Size, Style                        uint32
	Proc                               uintptr
	ClassExtra, WindowExtra            int32
	Instance, Icon, Cursor, Background windows.Handle
	MenuName, ClassName                *uint16
	SmallIcon                          windows.Handle
}
type point struct{ X, Y int32 }
type message struct {
	Window         windows.Handle
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Point          point
	Private        uint32
}
type Tray struct {
	mu            sync.Mutex
	window        windows.Handle
	normal, muted windows.Handle
	cb            Callbacks
	state         State
	done          chan struct{}
	closed        atomic.Bool
	healthy       atomic.Bool
}

func Start(normal, muted string, cb Callbacks) (*Tray, error) {
	t := &Tray{cb: cb, done: make(chan struct{})}
	ready := make(chan error, 1)
	go t.run(normal, muted, ready)
	if e := <-ready; e != nil {
		return nil, e
	}
	return t, nil
}
func (t *Tray) run(normal, muted string, ready chan error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(t.done)
	instance, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetModuleHandleW").Call(0)
	load := func(path string) windows.Handle {
		p, _ := windows.UTF16PtrFromString(path)
		h, _, _ := user.NewProc("LoadImageW").Call(0, uintptr(unsafe.Pointer(p)), 1, 32, 32, 0x10)
		return windows.Handle(h)
	}
	t.normal = load(normal)
	t.muted = load(muted)
	if t.normal == 0 || t.muted == 0 {
		ready <- errors.New("tray icon could not be loaded")
		return
	}
	defer user.NewProc("DestroyIcon").Call(uintptr(t.normal))
	defer user.NewProc("DestroyIcon").Call(uintptr(t.muted))
	taskbar, _ := windows.UTF16PtrFromString("TaskbarCreated")
	taskbarMessage, _, _ := user.NewProc("RegisterWindowMessageW").Call(uintptr(unsafe.Pointer(taskbar)))
	proc := windows.NewCallback(func(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
		switch msg {
		case 0x8001:
			switch uint32(l) & 0xffff {
			case 0x203, 0x400, 0x401:
				if t.cb.Open != nil {
					go t.cb.Open()
				}
			case 0x205, 0x7b:
				t.menu()
			}
			return 0
		case 0x8002:
			t.update()
			return 0
		case 0x111:
			t.action(int(w & 0xffff))
			return 0
		case 0x10:
			user.NewProc("DestroyWindow").Call(hwnd)
			return 0
		case 0x2:
			t.notify(2)
			user.NewProc("PostQuitMessage").Call(0)
			return 0
		}
		if msg == uint32(taskbarMessage) {
			t.notify(0)
			return 0
		}
		r, _, _ := user.NewProc("DefWindowProcW").Call(hwnd, uintptr(msg), w, l)
		return r
	})
	className, _ := windows.UTF16PtrFromString("PeeTwoPee.Tray")
	wc := wndClass{Size: uint32(unsafe.Sizeof(wndClass{})), Proc: proc, Instance: windows.Handle(instance), ClassName: className}
	registered, _, e := user.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
	if registered == 0 {
		ready <- e
		return
	}
	defer user.NewProc("UnregisterClassW").Call(uintptr(unsafe.Pointer(className)), instance)
	hwnd, _, e := user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(className)), 0, 0, 0, 0, 0, 0, 0, instance, 0)
	if hwnd == 0 {
		ready <- e
		return
	}
	t.window = windows.Handle(hwnd)
	if !t.notify(0) {
		user.NewProc("DestroyWindow").Call(hwnd)
		ready <- errors.New("notification area unavailable")
		return
	}
	ready <- nil
	var m message
	for {
		r, _, _ := user.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		user.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&m)))
		user.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&m)))
	}
}
func (t *Tray) notify(command uintptr) bool {
	t.mu.Lock()
	state := t.state
	t.mu.Unlock()
	icon := t.normal
	if state.Joined && state.Muted {
		icon = t.muted
	}
	n := notifyIcon{Size: uint32(unsafe.Sizeof(notifyIcon{})), Window: t.window, ID: 1, Flags: 1 | 2 | 4 | 0x80, Callback: 0x8001, Icon: icon}
	tip, _ := windows.UTF16FromString(state.Tooltip())
	copy(n.Tip[:], tip)
	r, _, _ := shell.NewProc("Shell_NotifyIconW").Call(command, uintptr(unsafe.Pointer(&n)))
	if command == 0 && r != 0 {
		n.Version = 4
		shell.NewProc("Shell_NotifyIconW").Call(4, uintptr(unsafe.Pointer(&n)))
	}
	t.healthy.Store(r != 0 && command != 2)
	return r != 0
}
func (t *Tray) Available() bool { return !t.closed.Load() && t.healthy.Load() }
func (t *Tray) update()         { t.notify(1) }
func (t *Tray) Update(s State) {
	if t.closed.Load() {
		return
	}
	t.mu.Lock()
	// Refresh an active sharing icon so an Explorer/tray failure fails closed.
	changed := t.state != s || s.Sharing
	t.state = s
	t.mu.Unlock()
	if changed {
		user.NewProc("PostMessageW").Call(uintptr(t.window), 0x8002, 0, 0)
	}
}
func (t *Tray) menu() {
	h, _, _ := user.NewProc("CreatePopupMenu").Call()
	if h == 0 {
		return
	}
	defer user.NewProc("DestroyMenu").Call(h)
	t.mu.Lock()
	s := t.state
	t.mu.Unlock()
	labels := []string{"Abrir PeeTwoPee", "Silenciar microfone", "Sair da chamada", "Encerrar aplicativo", "Parar compartilhamento de tela"}
	if s.Muted {
		labels[1] = "Ativar microfone"
	}
	for i, label := range labels {
		p, _ := windows.UTF16PtrFromString(label)
		flags := uintptr(0)
		if ((i == 1 || i == 2) && !s.Joined) || (i == 4 && !s.Sharing) {
			flags = 1
		}
		user.NewProc("AppendMenuW").Call(h, flags, uintptr(i+1), uintptr(unsafe.Pointer(p)))
	}
	var at point
	user.NewProc("GetCursorPos").Call(uintptr(unsafe.Pointer(&at)))
	user.NewProc("SetForegroundWindow").Call(uintptr(t.window))
	id, _, _ := user.NewProc("TrackPopupMenu").Call(h, 0x100|2, uintptr(at.X), uintptr(at.Y), 0, uintptr(t.window), 0)
	if id != 0 {
		t.action(int(id))
	}
	user.NewProc("PostMessageW").Call(uintptr(t.window), 0, 0, 0)
}
func (t *Tray) action(id int) {
	var cb func()
	switch id {
	case 1:
		cb = t.cb.Open
	case 2:
		cb = t.cb.Mute
	case 3:
		cb = t.cb.Leave
	case 4:
		cb = t.cb.Quit
	case 5:
		cb = t.cb.StopScreen
	}
	if cb != nil {
		go cb()
	}
}
func (t *Tray) Close() {
	if t.closed.CompareAndSwap(false, true) {
		user.NewProc("PostMessageW").Call(uintptr(t.window), 0x10, 0, 0)
	}
	<-t.done
}

func (t *Tray) Notify(title, body string) bool {
	if t.closed.Load() {
		return false
	}
	n := notifyIcon{Size: uint32(unsafe.Sizeof(notifyIcon{})), Window: t.window, ID: 1, Flags: 0x10, InfoFlags: 1 | 0x10 | 0x80}
	title = string([]rune(title)[:min(30, len([]rune(title)))])
	body = string([]rune(body)[:min(120, len([]rune(body)))])
	a, _ := windows.UTF16FromString(title)
	b, _ := windows.UTF16FromString(body)
	copy(n.InfoTitle[:], a)
	copy(n.Info[:], b)
	r, _, _ := shell.NewProc("Shell_NotifyIconW").Call(1, uintptr(unsafe.Pointer(&n)))
	return r != 0
}
