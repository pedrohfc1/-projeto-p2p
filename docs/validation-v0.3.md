# Resultados 0.3 — 31/10/2026

Ambiente: Windows AMD64 isolado, Go 1.27.3, Wails v2.17.0, Node 24 e Vite 7.3.7. Hamachi é dependência externa opcional e não foi instalado/autenticado pelo agente.

## Executado com sucesso

| Verificação | Resultado |
|---|---|
| `go test ./... -v -count=1 -timeout=180s` | PASS, 18 funções de teste |
| `go vet ./...` | PASS, sem diagnósticos |
| `go mod verify` | Módulos verificados |
| TypeScript + Vite | PASS |
| Build Wails Windows AMD64, bindings incluídos | PASS |
| Build CLI Windows/Linux AMD64 | PASS; Linux por cross-build |
| Build setup/desinstalador Windows 64 bits | PASS |
| Manifest asInvoker e execução `--verify-payload` | PASS |
| SHA-256 do app descomprimido no setup | Igual ao executável desktop da entrega |
| Três processos CLI e recuperação | PASS |
| Operador Circuit Relay v2 e probe de reserva | PASS |
| API desktop real, persistência e reinício | PASS, harness de QA sem mocks de rede/banco |
| Enumeração real de adaptadores Windows | PASS; Hamachi detectado |
| Host vinculado ao IPv4 do Hamachi real | PASS; somente endereço selecionado, sem DHT/relay |

A suíte cobre driver Hamachi com adaptador renomeado; rejeição de nome amigável sem evidência; leitura/seleção de IP; interface ausente, inativa ou sem IP; seleção manual; convite assinado e escopo; adulteração/expiração; multiaddr bruto válido/inválido; conexão por endereço conhecido; mensagens; identidade/contatos/histórico após reinício; queda/retorno do adaptador; mudança de IP com bind inválido e recuperação sem fallback público; exclusão de 25/8 e privados da descoberta pública; regras de porta de firewall; e os testes anteriores de assinatura, replay, limites, DHT e relay.

O teste de transporte por interface usa loopback explicitamente selecionado em dois perfis e portas próprias, com TCP/Noise reais. Não finge que loopback é uma conexão Hamachi entre casas. O teste real de adaptador confirma metadados do Windows e bind no IPv4 virtual sem alterar a VPN, mas não comunica com um peer remoto.

## Não aprovado / pendente

**Troca por IPv4 virtual entre dois processos no mesmo PC:** houve timeout antes do handshake libp2p. Uma conexão TCP simples, sem libp2p, entre listener e cliente nesse mesmo endereço virtual também expirou. A causa não foi isolada entre política de rede/firewall, comportamento de loopback virtual ou isolamento do ambiente. Não foi corrigida desativando proteções nem criando regras sem autorização. Não declaramos sucesso nesse cenário.

O script opcional `python tests/adapter_host_smoke.py dist/projetop2p-windows-amd64.exe --messages` mantém a tentativa de mensagem real e falha se não houver entrega. Sem `--messages`, testa somente enumeração e bind e informa isso explicitamente. O relato do timeout está em `docs/evidence/v0.3/virtual-tcp-observation.txt`; IPs reais não foram publicados no pacote.

**Entre casas:** não houve segundo computador remoto disponível no mesmo Hamachi para execução do roteiro. A conversa em duas redes residenciais, a associação à mesma rede virtual e a ausência de encaminhamento de portas precisam ser confirmadas pelos testadores. Roteiro em `hamachi-guide.md`.

**Instalação completa:** o setup real foi compilado e executado somente no modo de verificação de payload, que não instala. Não foram executados instalação no perfil do usuário, atalhos, atualização por cima ou desinstalação num Windows comum. O código implementa essas operações, mas isso não equivale a um teste de aceitação delas. O instalador não é assinado e não há rollback completo de uma instalação parcialmente concluída. O pacote preserva o perfil por desenho; testes de persistência do serviço foram executados separadamente.

**Janela nativa:** abertura e inspeção visual do WebView2 ainda precisam de validação fora deste ambiente isolado (o runtime havia falhado na versão anterior ao criar o controlador). Frontend e bindings compilam; testes da API não comprovam interação visual/clipboard.

**Linux Mint:** CLI compilado; execução nativa e desktop GTK/WebKit ainda pendentes. Workflow Linux atualizado e não executado remotamente. Detecção enriquecida por driver foi priorizada no Windows; seleção manual mantém a fronteira portátil.

Outros pendentes: teste `-race` local (sem compilador C), testes prolongados de reconexão, carga adversarial e auditoria independente. Não há benchmark ou garantia de disponibilidade. O modo Hamachi depende da infraestrutura e política do fornecedor, e conexões libp2p diretas sobre a VPN podem atravessar relays Hamachi.

## Evidências e empacotamento

Logs em `docs/evidence/v0.3`: Go, vet, módulos, build do frontend/desktop/setup, três nós, relay, API desktop e bind de adaptador real. `setup-payload.json` informa versão, tamanho/hash do app, tamanho do desinstalador/runtime e `hamachiBundled: false`. O instalador final é `ProjetoP2P-Setup-v0.3.exe`; SHA-256 dos arquivos de entrega acompanha o pacote.

Nenhuma chave, perfil, banco, senha Hamachi, IP virtual real, node_modules ou compilador foi incluído. As fontes e logs históricos 0.1/0.2 foram preservados. Nenhuma regra de firewall, configuração de adaptador, autenticação Hamachi, servidor público ou conta externa foi alterada.
