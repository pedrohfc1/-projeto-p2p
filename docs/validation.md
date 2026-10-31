# Validação da versão 0.2 — 31/10/2026

Ambiente de execução: Windows AMD64 isolado, Go 1.27.3, Node 24.19.0, npm 11.17.0. Núcleo libp2p v0.51.0 / kad-dht v0.43.0; Wails v2.17.0; bbolt v1.5.0. Testes de rede executados em loopback com perfis independentes.

## Resultados efetivamente obtidos

| Verificação | Resultado |
|---|---|
| `go test ./... -v -count=1 -timeout=180s` | PASS, 11 funções de teste |
| `go vet ./...` | PASS, sem diagnósticos |
| `go mod verify` | Todos os módulos verificados |
| TypeScript + Vite, `npm run build` | PASS |
| `npm audit` com Vite 7.3.7 | Zero vulnerabilidades reportadas na árvore resolvida |
| Build Wails Windows AMD64 + geração dos bindings | PASS, `build/bin/ProjetoP2P.exe` |
| Build CLI Windows AMD64 / Linux AMD64, CGO=0 | PASS |
| Três processos reais CLI | PASS |
| API desktop real em dois processos do harness de QA | PASS |
| Sintaxe PowerShell do smoke test de instalador | Válida; o script não foi executado |

### Rede e segurança

Persistência e corrupção de identidade; validação de remetente, destinatário, assinatura e janela temporal; rejeição de texto excessivo, JSON inválido e frame grande; replay e quotas de emissores; fallback entre bootstraps independentes; descoberta mútua com apenas um bootstrap; envio; desconexão; reinício com identidade preservada; recuperação de conectividade. Se a persistência do receptor falha, o remetente não recebe ACK de sucesso.

O teste Circuit Relay v2 usa dois operadores e dois participantes. Um participante conhece os operadores somente como bootstrap e precisa descobri-los pelo namespace DHT. Os gaters do teste bloqueiam caminhos diretos entre os participantes. Após a primeira entrega por circuito real, o primeiro relay é encerrado; a reconexão usa os endereços alternativos lembrados, sem discagem manual de um novo relay. A segunda mensagem é recebida. Isso comprova descoberta de candidatos, obtenção de reservas, streams limitados e failover local, não comportamento em CGNAT físico.

### Serviço desktop e persistência

Dois perfis desktop reais, convites assinados, nome de exibição, contatos, envio e histórico são testados no serviço Go. O teste adicional `webui_smoke.py` inicia dois processos de `tests/webui`, carrega os assets de produção, chama as operações reais do serviço, interrompe/reinicia um processo e verifica ID, contatos e mensagem em disco. Requisições sem token/origem ao harness são rejeitadas. Não há mocks de rede ou de banco nesse teste. Isso não é um teste de interação visual da janela Wails.

### Três processos CLI

O teste preservado da 0.1 foi reexecutado com os binários atualizados. Verificou DHT, mensagens, indisponibilidade do bootstrap, queda abrupta, novo endereço e recuperação com a mesma identidade.

## Limitações e testes pendentes

| Cenário | Situação |
|---|---|
| A — Dois usuários no mesmo computador | Backend/streams/processos aprovados; janelas nativas não confirmadas |
| B — Dois computadores físicos na LAN | Não executado; roteiro preparado |
| C — Redes distintas sem encaminhamento manual | Não executado; precisa de operadores públicos reais |
| D — NAT/CGNAT real com relay | Encaminhamento real validado em loopback; NAT físico pendente |
| E — Desconexão e recuperação | Automatizado localmente, incluindo queda de relay; instabilidade de operadora pendente |
| F — Reinício preservando dados | Aprovado no backend/processos; atualização por instalador pendente |

**Janela Windows:** a tentativa de abertura encontrou `WebView2 error creating controller with 8000ffff`, após criação do ambiente. Não foi possível inspecionar a interface nativa neste ambiente isolado. A tentativa de alcançar o harness por um navegador em outro contexto também expirou; não foi declarada validação visual. O frontend compila e o serviço passa nos testes, mas o fluxo de janela/clipboard ainda precisa de teste em Windows comum.

**Instalador:** NSIS 3.11 falhou ao iniciar com `Illegal System DLL Relocation`. Não foi gerado um setup validado. A configuração real NSIS e o workflow Windows estão entregues. O workflow inclui compilação do setup e smoke de instalação/atualização/desinstalação/preservação de arquivo de perfil em runner descartável; não foi executado remotamente aqui.

**Linux:** CLI AMD64 compilado por cross-build. Desktop precisa de build nativo GTK3/WebKit2GTK 4.1; workflow Ubuntu 24.04 e script Mint preparados. Nem o build nativo desktop Linux nem a execução em Mint foram realizados neste ambiente.

Outros pendentes: renovação contínua de reservas por mais de 20 minutos, DCUtR em NAT real, injeção controlada de perda/latência, carga adversarial, auditoria independente e detector `-race` (compilador C ausente localmente; etapa preparada no CI Linux). Não há medição de capacidade, RSS ou throughput. Limites por circuito não são teto agregado de Mbps.

## Evidências

Logs em `docs/evidence/v0.2`: suíte Go, vet, verificação dos módulos, frontend, audit npm, build Wails, tentativa NSIS, teste de processos e teste da API desktop. SHA-256 dos executáveis em `dist/SHA256SUMS.txt`. O ZIP não inclui perfis, identidades, bancos, node_modules, compiladores ou caches. Resultados anteriores preservados em `validation-v0.1.md`.
