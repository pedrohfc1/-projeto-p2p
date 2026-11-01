package main

import (
	"context"
	"embed"
	"flag"
	"fmt"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"projetop2p/internal/communities"
	"projetop2p/internal/community"
	"projetop2p/internal/desktop"
	"projetop2p/internal/localdb"
)

//go:embed all:frontend/dist
var assets embed.FS

type App struct {
	ctx   context.Context
	core  *desktop.Core
	err   error
	ready chan struct{}
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
	flag.Parse()
	if *version {
		fmt.Println("0.4.1")
		return
	}
	app := &App{ready: make(chan struct{})}
	err := wails.Run(&options.App{Title: "Projeto P2P", Width: 1180, Height: 790, MinWidth: 900, MinHeight: 640, BackgroundColour: options.NewRGB(15, 18, 29), AssetServer: &assetserver.Options{Assets: assets}, OnStartup: func(ctx context.Context) {
		app.ctx = ctx
		d := *dir
		var e error
		if d == "" {
			d, e = desktop.DefaultDir()
		}
		if e == nil {
			app.core, e = desktop.Open(ctx, d)
		}
		app.err = e
		close(app.ready)
	}, OnShutdown: func(context.Context) {
		_ = app.initialized()
		if app.core != nil {
			_ = app.core.Close()
		}
	}, Bind: []interface{}{app}})
	if err != nil {
		fmt.Println(err)
	}
}
