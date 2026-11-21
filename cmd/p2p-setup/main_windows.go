package main

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/json"
	"io"
	"os"
	"projetop2p/internal/installwin"
)

//go:embed payload/*
var payload embed.FS

func main() {
	if len(os.Args) == 1 {
		if err := wizard(); err != nil {
			installwin.Message("Assistente indisponível: "+err.Error(), false)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--rollback" {
		if !installwin.Message("Restaurar os arquivos da instalação anterior? O aplicativo será encerrado após confirmação. Identidade, histórico e configurações não serão alterados.", true) {
			return
		}
		if e := installwin.Rollback(); e != nil {
			installwin.Message("Rollback não concluído: "+e.Error(), false)
			os.Exit(1)
		}
		installwin.Message("Arquivos anteriores restaurados. Perfil preservado.", false)
		return
	}

	compressed, e := payload.ReadFile("payload/PeeTwoPee.exe.gz")
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
	helper, e := payload.ReadFile("payload/etcat-p2p.exe")
	if e != nil {
		panic(e)
	}
	source, e := payload.ReadFile("payload/EasyTier-corresponding-source.zip")
	if e != nil {
		panic(e)
	}
	if len(os.Args) > 1 && os.Args[1] == "--verify-payload" {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"version": installwin.Version, "transactionalUpdate": true, "processConsent": true, "rollbackSupported": true, "licenseSHA256": installwin.Hash(license), "thirdPartySHA256": installwin.Hash(thirdParty), "appBytes": len(app), "appSHA256": installwin.Hash(app), "uninstallerBytes": len(un), "runtimeBootstrapBytes": len(runtime), "hamachiBundled": false, "radminBundled": false, "vpnBundled": false, "integratedHelperBundled": true, "helperSHA256": installwin.Hash(helper), "correspondingSourceSHA256": installwin.Hash(source)})
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "--apply-options" {
		o, err := readOptions(os.Args[2])
		if err == nil {
			err = installwin.EnsureRuntime(runtime)
		}
		if err == nil {
			err = installwin.InstallTo(o.Directory, app, un, license, thirdParty, helper, source, o.Desktop, o.StartMenu, &o.Startup)
		}
		result := map[string]string{}
		if err != nil {
			result["error"] = err.Error()
		}
		raw, _ := json.Marshal(result)
		if writeErr := os.WriteFile(os.Args[2]+".result", raw, 0600); writeErr != nil || err != nil {
			os.Exit(1)
		}
		return
	}
	if e = wizard(); e != nil {
		installwin.Message("Não foi possível abrir o assistente: "+e.Error(), false)
		os.Exit(1)
	}
}
