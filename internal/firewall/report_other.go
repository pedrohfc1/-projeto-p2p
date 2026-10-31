//go:build !windows

package firewall

func Inspect(port int, exe string) Report {
	return Report{Port: port, Summary: "Diagnóstico de Windows Defender Firewall disponível somente no Windows. O teste P2P verifica o caminho TCP."}
}
