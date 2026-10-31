# Validação da infraestrutura Bootstrap + Relay — 31/10/2026

## Análise e alterações

O CLI já oferecia DHT Kademlia em modo servidor e Circuit Relay v2 com as flags `--relay-server --public --daemon`. Nenhum servidor público externo ou endereço de infraestrutura está embutido. O que faltava era diagnóstico operacional. Foram adicionados:

- `--announce`: transporte TCP público fixo, validado e acrescentado aos endereços do host.
- `/status` e `--status-every`: serviço de relay realmente habilitado, tabela DHT, reservas, circuitos e bytes observados.
- `--check-relay`: probe de reserva real com identidade temporária, prazo de 15 segundos, validação do voucher/IDs/expiração e saída JSON.
- Script de execução Linux, unit systemd e guia com os valores para os dois campos do desktop.

`main.go` e todos os arquivos de frontend da 0.2 foram comparados byte a byte com o ZIP fonte anterior: **10 arquivos iguais**. A interface gráfica não foi alterada nem recompilada nesta atualização. O protocolo de streams, identidade e quotas também foram preservados.

## Resultados executados

Ambiente Windows AMD64, Go 1.27.3, participantes locais em loopback. Houve uma tentativa inicial sem permissão de rede do sandbox, que falhou com acesso a socket proibido; ela foi interrompida. Após concedida a permissão, foram executados:

| Verificação | Resultado |
|---|---|
| `go test ./... -v -count=1 -timeout=180s` | PASS, 12 testes |
| `go vet ./...` | PASS, sem diagnósticos |
| Build CLI Windows AMD64 | PASS |
| Build CLI Linux AMD64, CGO=0 | PASS, cross-build |
| `tests/relay_cli_smoke.py` com CLI Windows | PASS |
| `tests/cli_smoke.py` com CLI Windows | PASS |
| Comparação dos arquivos GUI com o fonte anterior | Sem alterações |

O teste de relay bloqueia o caminho direto entre participantes, usa Circuit Relay v2 real, verifica contadores de reservas/circuitos/bytes e recupera a entrega após queda de um operador. Um participante descobre candidatos por DHT, sem relays estáticos próprios. O teste de probe confirma voucher assinado, ID do relay, ID temporário do cliente, expiração e quotas, e rejeita um nó comum que não oferece o protocolo. O teste de anúncio usa um domínio de exemplo somente como dado de teste; a conexão real é feita pelo listener loopback, sem alegar alcance desse domínio.

O smoke de operador inicia o executável com `--daemon` e stdin fechado, verifica o serviço ativo nos logs JSON, executa um segundo processo `--check-relay` e observa a reserva nos contadores do operador. O probe encerra sua conexão ao terminar; não mantém reserva para o desktop.

Logs: `docs/evidence/relay-ops`. O buildinfo Linux confirma alvo `GOOS=linux`, `GOARCH=amd64`, `CGO_ENABLED=0`. Os SHA-256 dos novos CLIs constam no pacote de infraestrutura.

## Limites

Não foram executados: binário Linux nativamente, implantação systemd, firewall de hospedagem, IP público real, dois computadores em casas distintas, NAT/CGNAT físico ou acompanhamento de renovação de reservas por 20 minutos. Não foi provisionada uma máquina pública. Os roteiros estão em `relay-quickstart.md`.

`--public` e `--announce` são declarações do operador, não provas de alcance. Contadores são telemetria por processo e podem refletir transições concorrentes; não substituem o probe remoto nem o teste de mensagem. Reserva aceita não comprova que o chat foi recebido ou lido. Quotas de bytes/duração são por circuito e não constituem limite global de Mbps.
