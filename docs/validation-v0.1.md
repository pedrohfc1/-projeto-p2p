# Validação — 31/10/2026

Ambiente: Windows AMD64, Go 1.27.3 instalado localmente na pasta de trabalho. Download do compilador conferido contra o SHA-256 publicado no catálogo oficial. Dependências fixadas em go.mod/go.sum. Testes feitos em loopback, sem rede externa entre participantes.

| Verificação | Resultado |
|---|---|
| `go test ./... -v -count=1 -timeout=150s` | PASS |
| `go vet ./...` | PASS, sem diagnósticos |
| `go mod verify` | Todos os módulos verificados |
| Build Windows AMD64, CGO=0 | PASS |
| Build Linux AMD64, CGO=0 | PASS, compilação cruzada |
| `python tests/cli_smoke.py dist/projetop2p-windows-amd64.exe` | PASS, três processos reais |

Testes automatizados: persistência e rejeição de identidade corrompida; adulteração, origem/destinatário inválidos, validade temporal e texto excessivo; rejeição de JSON inválido, frame excessivo e assinatura inválida; replay e limite de frequência/cache; fallback entre identidades de bootstrap independentes com uma offline; descoberta mútua B/C com apenas A configurado; mensagens; desligamento inesperado; reinício com mesmo ID e novo endereço; recuperação e reconexão após fechamento de conexão.

O teste de processos confirma mensagens após a interrupção do bootstrap A e recuperação de C com identidade preservada em outra porta usando B como bootstrap vivo. O cenário de instabilidade automatizado cobre interrupções discretas, não perda de pacotes/latência artificial ou partições prolongadas.

Não executados: execução nativa em Linux Mint, redes físicas distintas, CGNAT/NAT simétrico, reserva/encaminhamento real de Circuit Relay, hole punching real, ensaios de carga/abuso, auditoria independente e detector `-race` (não há compilador C disponível no ambiente). CI Linux/Windows foi preparada, não executada remotamente. Os módulos storage/registry são contratos futuros, não implementações validadas de distribuição ou consenso.

Os logs da execução estão em `docs/evidence`. SHA-256 dos executáveis entregues consta em `dist/SHA256SUMS.txt`. O pacote não contém identidades de testes, caches de dependências ou o compilador Go.
