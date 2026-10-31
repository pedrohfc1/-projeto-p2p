package main

import (
	"os"
	"projetop2p/internal/installwin"
	"strconv"
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--cleanup" {
		n, e := strconv.ParseUint(os.Args[2], 10, 32)
		if e == nil {
			_ = installwin.Cleanup(uint32(n))
		}
		return
	}
	silent := len(os.Args) > 1 && os.Args[1] == "--silent"
	if !silent && !installwin.Message("Desinstalar o Projeto P2P? Identidade, contatos, mensagens e configurações serão preservados.", true) {
		return
	}
	if e := installwin.Uninstall(); e != nil {
		if !silent {
			installwin.Message(e.Error(), false)
		}
		os.Exit(1)
	}
	if !silent {
		installwin.Message("Aplicativo removido. Seus dados locais foram preservados.", false)
	}
}
