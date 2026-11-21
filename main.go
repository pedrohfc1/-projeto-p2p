package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"projetop2p/internal/diagnostics"
	"projetop2p/internal/installtxn"
	"projetop2p/internal/invite"
	"projetop2p/internal/lifecycle"
	"projetop2p/internal/tray"
	goruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	winoptions "github.com/wailsapp/wails/v2/pkg/options/windows"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"projetop2p/internal/communities"
	"projetop2p/internal/community"
	"projetop2p/internal/desktop"
	"projetop2p/internal/localdb"
	"projetop2p/internal/voice"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/brand/*.ico
var trayIcons embed.FS

var permissionPolicy = "unverified"

type App struct {
	screenVisible    atomic.Bool
	tray             *tray.Tray
	quitting         atomic.Bool
	backgroundCancel context.CancelFunc
	backgroundWG     sync.WaitGroup
	ctx              context.Context
	core             *desktop.Core
	err              error
	ready            chan struct{}
}

func (a *App) initialized() error { <-a.ready; return a.err }

func (a *App) State(id string) (desktop.State, error) {
	if err := a.initialized(); err != nil {
		return desktop.State{}, err
	}
	return a.core.State(id)
}
func (a *App) Send(id, text string) error {
	if err := a.initialized(); err != nil {
		return err
	}
	return a.core.Send(id, text)
}
func (a *App) AddContact(s string) error {
	if err := a.initialized(); err != nil {
		return err
	}
	return a.core.AddContact(s)
}
func (a *App) ConnectInvite(s string) (desktop.ConnectionTest, error) {
	if e := a.initialized(); e != nil {
		return desktop.ConnectionTest{}, e
	}
	return a.core.ConnectInvite(s)
}
func (a *App) TestConnectivity(id string) (desktop.ConnectionTest, error) {
	if e := a.initialized(); e != nil {
		return desktop.ConnectionTest{}, e
	}
	return a.core.TestConnectivity(id)
}
func (a *App) SaveSettings(s localdb.Settings) error {
	if err := a.initialized(); err != nil {
		return err
	}
	return a.core.SaveSettings(s)
}
func (a *App) Invite() (string, error) {
	if err := a.initialized(); err != nil {
		return "", err
	}
	return a.core.Invite()
}
func (a *App) LegacyInvite(raw string) (string, error) {
	if e := a.initialized(); e != nil {
		return "", e
	}
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, community.InvitePrefix) || strings.HasPrefix(raw, community.SmallInvitePrefix) {
		return community.LegacyInvite(raw)
	}
	return invite.Legacy(raw)
}
func (a *App) Copy(s string) error {
	if err := a.initialized(); err != nil {
		return err
	}
	if len(s) > 24000 {
		return fmt.Errorf("texto muito grande")
	}
	return wruntime.ClipboardSetText(a.ctx, s)
}
func (a *App) CreateCommunity(name, description, icon string) (string, error) {
	if e := a.initialized(); e != nil {
		return "", e
	}
	return a.core.CreateCommunity(name, description, icon)
}
func (a *App) CommunityState(id, channel string, before uint64) (communities.View, error) {
	if e := a.initialized(); e != nil {
		return communities.View{}, e
	}
	return a.core.CommunityState(id, channel, before)
}
func (a *App) CommunityAction(id, kind string, body community.Body) (communities.Receipt, error) {
	if e := a.initialized(); e != nil {
		return communities.Receipt{}, e
	}
	return a.core.CommunityAction(id, kind, body)
}
func (a *App) CommunityInvite(id string) (string, error) {
	if e := a.initialized(); e != nil {
		return "", e
	}
	return a.core.CommunityInvite(id)
}
func (a *App) JoinCommunity(raw string) (communities.Receipt, error) {
	if e := a.initialized(); e != nil {
		return communities.Receipt{}, e
	}
	return a.core.JoinCommunity(raw)
}
func (a *App) SyncCommunity(id string) error {
	if e := a.initialized(); e != nil {
		return e
	}
	return a.core.SyncCommunity(id)
}
func (a *App) DiscardCommunityPending(id, op string) error {
	if e := a.initialized(); e != nil {
		return e
	}
	return a.core.DiscardCommunityPending(id, op)
}

func main() {
	dir := flag.String("data", "", "Perfil separado para testes; padrão no diretório do usuário")
	version := flag.Bool("version", false, "Informar versão sem iniciar a janela ou o nó")
	policy := flag.Bool("permission-policy", false, "Informar politica de permissoes do build")
	flag.Parse()
	if *policy {
		fmt.Println(permissionPolicy)
		return
	}
	if *version {
		fmt.Println("0.11.0")
		return
	}
	if !bindingGeneration && goruntime.GOOS == "windows" && permissionPolicy != "prompt-v1" {
		fmt.Println("Build de desktop sem politica de consentimento verificada. Use scripts/build-desktop.ps1.")
		return
	}
	dataDir := *dir
	var resolveErr error
	if dataDir == "" {
		dataDir, resolveErr = desktop.DefaultDir()
	} else {
		dataDir, resolveErr = filepath.Abs(dataDir)
	}
	if resolveErr != nil {
		fmt.Println(resolveErr)
		return
	}
	var guard *lifecycle.Guard
	if !bindingGeneration {
		executable, e := os.Executable()
		if e != nil {
			lifecycle.ShowError(e.Error())
			return
		}
		guard, e = lifecycle.Acquire(dataDir, filepath.Dir(executable))
		if e != nil {
			lifecycle.ShowError(e.Error())
			return
		}
		defer guard.Close()
		verifiedBinary, e := installtxn.VerifyExecutable(executable, "0.11.0")
		if e != nil {
			lifecycle.ShowError(e.Error())
			return
		}
		if e = localdb.CheckProfileIdle(dataDir); e != nil {
			lifecycle.ShowError(e.Error())
			return
		}
		revision, e := lifecycle.AssetRevision(assets)
		if e != nil {
			lifecycle.ShowError(e.Error())
			return
		}
		revision = installtxn.Hash([]byte(revision + "|" + verifiedBinary))
		if e = lifecycle.RefreshCache(dataDir, revision); e != nil {
			lifecycle.ShowError(e.Error())
			return
		}
	}
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	defer shutdownCancel()
	if goruntime.GOOS == "windows" {
		_ = os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", os.Getenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS")+" --disable-background-timer-throttling --disable-renderer-backgrounding --disable-backgrounding-occluded-windows")
	}
	diagnostics.Install()
	app := &App{ready: make(chan struct{})}
	err := wails.Run(&options.App{Title: "PeeTwoPee", Width: 1180, Height: 790, MinWidth: 760, MinHeight: 600, BackgroundColour: options.NewRGB(15, 18, 29), DragAndDrop: &options.DragAndDrop{DisableWebViewDrop: true, EnableFileDrop: false}, AssetServer: &assetserver.Options{Assets: assets, Middleware: mediaPermissions}, Windows: &winoptions.Options{WebviewUserDataPath: filepath.Join(dataDir, "cache", "webview2")}, OnStartup: func(ctx context.Context) {
		app.ctx = ctx
		if guard != nil {
			go func() {
				select {
				case <-guard.StopRequests():
					app.quitting.Store(true)
					wruntime.Quit(ctx)
				case <-shutdownCtx.Done():
				}
			}()
		}
		app.screenVisible.Store(true)
		var e error
		app.core, e = desktop.Open(ctx, dataDir)
		app.err = e
		if e == nil {
			app.startTray(dataDir)
			app.startDiagnostics(shutdownCtx)
		}
		close(app.ready)
	}, OnBeforeClose: func(ctx context.Context) bool {
		if app.initialized() != nil {
			return false
		}
		p, e := app.core.AudioPreferences()
		if e == nil && tray.ShouldHide(p.CloseToTray, app.tray != nil, app.quitting.Load()) {
			wruntime.WindowExecJS(ctx, `window.dispatchEvent(new CustomEvent("peetwopee:visibility",{detail:{hidden:true}}))`)
			app.screenVisible.Store(false)
			app.core.SetChatForeground(false)
			if app.tray == nil || !app.tray.Available() {
				app.core.ScreenStop()
			}
			wruntime.WindowHide(ctx)
			return true
		}
		return false
	}, OnShutdown: func(context.Context) {
		shutdownCancel()
		_ = app.initialized()
		if app.backgroundCancel != nil {
			app.backgroundCancel()
		}
		app.backgroundWG.Wait()
		if app.tray != nil {
			app.tray.Close()
		}
		_ = app.initialized()
		if app.core != nil {
			_ = app.core.Close()
		}
	}, Bind: []interface{}{app}})
	if err != nil {
		fmt.Println(err)
	}
}

func (a *App) CommunityInvitePolicy(id string, options community.InviteOptions) (string, error) {
	if err := a.initialized(); err != nil {
		return "", err
	}
	return a.core.CommunityInvitePolicy(id, options)
}

func (a *App) CommunityInviteStatus(id, raw string) (community.InvitationStatus, error) {
	if err := a.initialized(); err != nil {
		return community.InvitationStatus{}, err
	}
	return a.core.CommunityInviteStatus(id, raw)
}

func (a *App) VoiceJoin(group, channel string, offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	if e := a.initialized(); e != nil {
		return webrtc.SessionDescription{}, e
	}
	return a.core.VoiceJoin(group, channel, offer)
}
func (a *App) VoiceState() (voice.View, error) {
	if e := a.initialized(); e != nil {
		return voice.View{}, e
	}
	return a.core.VoiceState(), nil
}
func (a *App) VoiceLeave() error {
	if e := a.initialized(); e != nil {
		return e
	}
	a.core.VoiceLeave()
	return nil
}
func (a *App) VoiceControls(muted, deaf bool) error {
	if e := a.initialized(); e != nil {
		return e
	}
	a.core.VoiceControls(muted, deaf)
	return nil
}

func (a *App) VoiceRebind(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	if e := a.initialized(); e != nil {
		return webrtc.SessionDescription{}, e
	}
	return a.core.VoiceRebind(offer)
}

func (a *App) AudioPreferences() (localdb.Preferences, error) {
	if e := a.initialized(); e != nil {
		return localdb.Preferences{}, e
	}
	return a.core.AudioPreferences()
}
func (a *App) SaveAudioPreferences(p localdb.Preferences) error {
	if e := a.initialized(); e != nil {
		return e
	}
	return a.core.SaveAudioPreferences(p)
}

func (a *App) startTray(dir string) {
	cache := filepath.Join(dir, "cache", "brand")
	if os.MkdirAll(cache, 0700) != nil {
		return
	}
	for _, name := range []string{"tray.ico", "tray-muted.ico"} {
		b, e := trayIcons.ReadFile("assets/brand/" + name)
		if e != nil {
			return
		}
		if os.WriteFile(filepath.Join(cache, name), b, 0600) != nil {
			return
		}
	}
	t, e := tray.Start(filepath.Join(cache, "tray.ico"), filepath.Join(cache, "tray-muted.ico"), tray.Callbacks{Open: func() {
		a.screenVisible.Store(true)
		a.core.SetChatForeground(true)
		wruntime.WindowShow(a.ctx)
		wruntime.WindowUnminimise(a.ctx)
		wruntime.WindowExecJS(a.ctx, `window.dispatchEvent(new CustomEvent("peetwopee:visibility",{detail:{hidden:false}}))`)
	}, Mute: func() {
		v := a.core.VoiceState()
		if v.Joined {
			muted := !(v.Muted || v.Deaf)
			deaf := v.Deaf
			if !muted {
				deaf = false
			}
			a.core.VoiceControls(muted, deaf)
		}
	}, StopScreen: func() {
		a.core.ScreenStop()
		wruntime.WindowExecJS(a.ctx, `window.dispatchEvent(new Event("peetwopee:screen-stop"))`)
	}, Leave: func() { a.core.VoiceLeave() }, Quit: func() { a.quitting.Store(true); wruntime.Quit(a.ctx) }})
	if e != nil {
		return
	}
	a.tray = t
	ctx, cancel := context.WithCancel(a.ctx)
	a.backgroundCancel = cancel
	a.backgroundWG.Add(1)
	go func() {
		defer a.backgroundWG.Done()
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if !a.screenWindowVisible() {
					a.core.SetChatForeground(false)
				}
				v := a.core.VoiceState()
				screen := a.core.ScreenState()
				t.Update(tray.State{Joined: v.Joined, Muted: v.Muted || v.Deaf, Sharing: screen.Publishing})
				// Native watchdog is independent of renderer timers while hidden.
				if screen.Active {
					if t.Available() {
						a.core.ScreenHeartbeat()
					} else {
						a.core.ScreenStop()
						wruntime.WindowExecJS(a.ctx, `window.dispatchEvent(new Event("peetwopee:screen-stop"))`)
					}
				}
				if notices, e := a.core.DrainNotices(); e == nil {
					for _, notice := range notices {
						t.Notify("PeeTwoPee", notice.Body)
					}
				}
			}
		}
	}()
}
func (a *App) QuitApp() error {
	if e := a.initialized(); e != nil {
		return e
	}
	a.quitting.Store(true)
	wruntime.Quit(a.ctx)
	return nil
}

// Capability only: never starts capture or grants a system permission.
func (a *App) ScreenBackgroundAllowed() bool {
	return a.tray != nil && a.tray.Available()
}

func (a *App) ScreenConnect(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	if e := a.initialized(); e != nil {
		return webrtc.SessionDescription{}, e
	}
	if !a.screenWindowVisible() {
		return webrtc.SessionDescription{}, fmt.Errorf("screen window is not visible")
	}
	return a.core.ScreenConnect(offer)
}
func (a *App) ScreenPublish(publish bool, bitrate, maxViewers int) error {
	if e := a.initialized(); e != nil {
		return e
	}
	if publish && !a.screenWindowVisible() {
		a.core.ScreenStop()
		return fmt.Errorf("screen sharing stopped while window is hidden or minimized")
	}
	return a.core.ScreenPublish(publish, bitrate, maxViewers)
}
func (a *App) ScreenWatch(id string) error {
	if e := a.initialized(); e != nil {
		return e
	}
	if id != "" && !a.screenWindowVisible() {
		return fmt.Errorf("screen viewing stopped while window is hidden or minimized")
	}
	return a.core.ScreenWatch(id)
}
func (a *App) ScreenStop() error {
	if e := a.initialized(); e != nil {
		return e
	}
	a.core.ScreenStop()
	return nil
}
func (a *App) ScreenHeartbeat() error {
	if e := a.initialized(); e != nil {
		return e
	}
	if !a.screenWindowVisible() && (a.tray == nil || !a.tray.Available()) {
		a.core.ScreenStop()
		wruntime.WindowExecJS(a.ctx, `window.dispatchEvent(new Event("peetwopee:screen-stop"))`)
		return nil
	}
	a.core.ScreenHeartbeat()
	return nil
}
func (a *App) ScreenState() (voice.ScreenView, error) {
	if e := a.initialized(); e != nil {
		return voice.ScreenView{}, e
	}
	return a.core.ScreenState(), nil
}

func screenHeartbeatAllowed(visible, minimized bool) bool { return visible && !minimized }
func (a *App) screenWindowVisible() bool {
	return a.ctx != nil && screenHeartbeatAllowed(a.screenVisible.Load(), wruntime.WindowIsMinimised(a.ctx))
}

func (a *App) VoiceControlsForSession(session string, muted, deaf bool) error {
	if e := a.initialized(); e != nil {
		return e
	}
	return a.core.VoiceControlsForSession(session, muted, deaf)
}

// Permit an explicit local media request; the runtime still requests user consent.
func mediaPermissions(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		w.Header().Set("Permissions-Policy", "microphone=(self), camera=(), display-capture=(self), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func (a *App) SelectNetwork(mode, interfaceID string) error {
	if e := a.initialized(); e != nil {
		return e
	}
	return a.core.SelectNetwork(mode, interfaceID)
}
