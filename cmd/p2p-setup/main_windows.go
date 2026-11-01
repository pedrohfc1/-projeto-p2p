package main

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"projetop2p/internal/installwin"
)

//go:embed payload/*
var payload embed.FS

func main() {
	compressed, e := payload.ReadFile("payload/ProjetoP2P.exe.gz")
	if e != nil {
		panic(e)
	}
	z, e := gzip.NewReader(bytes.NewReader(compressed))
	if e != nil {
		panic(e)
	}
	app, e := io.ReadAll(io.LimitReader(z, 128<<20))
	z.Close()
	if e != nil {
		panic(e)
	}
	un, e := payload.ReadFile("payload/uninstall.exe")
	if e != nil {
		panic(e)
	}
	runtime, e := payload.ReadFile("payload/MicrosoftEdgeWebview2Setup.exe")
	if e != nil {
		panic(e)
	}
	license, e := payload.ReadFile("payload/LICENSE.txt")
	if e != nil {
		panic(e)
	}
	thirdParty, e := payload.ReadFile("payload/THIRD_PARTY_LICENSES.zip")
	if e != nil {
		panic(e)
	}
	if len(os.Args) > 1 && os.Args[1] == "--verify-payload" {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"version": installwin.Version, "licenseSHA256": installwin.Hash(license), "thirdPartySHA256": installwin.Hash(thirdParty), "appBytes": len(app), "appSHA256": installwin.Hash(app), "uninstallerBytes": len(un), "runtimeBootstrapBytes": len(runtime), "hamachiBundled": false, "radminBundled": false, "vpnBundled": false})
		return
	}
	dir, e := installwin.Folder()
	if e == nil && !installwin.Message(fmt.Sprintf("Instalar Projeto P2P 0.5.2 em:\n%s\n\nFeche a versÃ£o anterior. O perfil existente serÃ¡ preservado. Nenhuma VPN serÃ¡ instalada. WebView2 serÃ¡ instalado se necessÃ¡rio (requer internet). Continuar?", dir), true) {
		return
	}
	desktop := false
	if e == nil {
		desktop = installwin.Message("Criar tambÃ©m um atalho na Ã¡rea de trabalho?", true)
		e = installwin.EnsureRuntime(runtime)
	}
	if e == nil {
		e = installwin.Install(app, un, license, thirdParty, desktop)
	}
	if e != nil {
		installwin.Message("InstalaÃ§Ã£o nÃ£o concluÃ­da: "+e.Error(), false)
		os.Exit(1)
	}
	installwin.Message("Projeto P2P 0.5.2 instalado. Abra pelo menu Iniciar. Seus dados anteriores foram preservados.", false)
}
