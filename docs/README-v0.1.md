# Projeto P2P — núcleo 0.1

Aplicativo de terminal em Go para comunicação direta entre identidades Ed25519 através de libp2p. Windows AMD64 e Linux AMD64, incluindo Linux Mint como alvo. Não há interface gráfica, voz, blockchain ou serviço central obrigatório.

## Estado funcional

- Identidade criada automaticamente e reutilizada. DPAPI no Windows; permissões restritas no Linux.
- Host libp2p, transporte autenticado, streams versionados e mensagens assinadas.
- Kademlia DHT própria, rendezvous e múltiplos bootstraps configuráveis.
- Participantes conectados, detecção de desconexão e reconexão com recuo e aleatoriedade.
- Limites de conexões, streams, quadros, texto, filas, frequência e cache de replay.
- NAT mapping, hole punching, cliente Circuit Relay v2 e candidatos estáticos de AutoRelay habilitáveis.
- Serviço voluntário de relay opcional, sujeito à avaliação de alcance público do libp2p.

A rede começa isolada se nenhum endereço de outro participante for informado. Pelo menos um participante conhecido precisa estar alcançável para ingressar em uma rede existente. Não há descoberta mágica através da internet. Bootstraps são pares comuns, substituíveis e sem autoridade sobre mensagens ou identidades.

## Dependências e compilação

Instale Go 1.26 ou superior; esta entrega foi validada com Go 1.27.3. Git é opcional para compilar o arquivo ZIP. Python 3 é necessário somente para o teste adicional de processos. As dependências estão fixadas em `go.mod` e `go.sum`: libp2p v0.51.0, kad-dht v0.43.0.

Na raiz `projeto-p2p`:

```powershell
go mod download
go test ./... -count=1 -timeout=150s
go vet ./...
./scripts/build.ps1
```

Linux Mint:

```sh
go mod download
go test ./... -count=1 -timeout=150s
go vet ./...
sh scripts/build.sh
chmod +x dist/projetop2p-linux-amd64
```

Os scripts geram ambos os executáveis em `dist`, com CGO desativado. A compilação Linux feita no Windows não comprova execução em Linux Mint. Para executar o detector de corridas, use `go test -race ./... -timeout=180s` em ambiente com compilador C compatível.

## Três nós na mesma máquina

Abra três terminais na raiz do projeto. Cada nó precisa de diretório próprio. No Windows:

```powershell
# Terminal A
./dist/projetop2p-windows-amd64.exe --data data/a --listen /ip4/127.0.0.1/tcp/4001
```

A saída mostra `Peer ID: 12D3...` e um endereço completo terminado em `/p2p/12D3...`. Copie esse endereço para B e C; substitua `ID_A` abaixo pelo ID real, sem reticências:

```powershell
# Terminal B
./dist/projetop2p-windows-amd64.exe --data data/b --listen /ip4/127.0.0.1/tcp/4002 --bootstrap /ip4/127.0.0.1/tcp/4001/p2p/ID_A

# Terminal C
./dist/projetop2p-windows-amd64.exe --data data/c --listen /ip4/127.0.0.1/tcp/4003 --bootstrap /ip4/127.0.0.1/tcp/4001/p2p/ID_A
```

No Linux, use `./dist/projetop2p-linux-amd64` com as mesmas flags. Após alguns segundos, use `/peers`. B e C recebem somente o endereço de A; a descoberta mútua ocorre pelo rendezvous DHT.

Comandos interativos:

```text
/peers
/addresses
/connect /ip4/127.0.0.1/tcp/4002/p2p/ID_B
/send ID_C Olá, mensagem do nó B
/quit
```

`/send` recebe o ID, não o multiendereço. A mensagem aparece no terminal do destinatário; o remetente recebe confirmação da fila de recepção. Não há histórico persistente, entrega offline, grupos, retransmissão de mensagens de aplicação ou garantia de leitura humana. EOF no terminal encerra o nó; mantenha stdin aberto ao automatizar processos.

Para testar múltiplos bootstraps, repita `--bootstrap ENDERECO_A --bootstrap ENDERECO_B` ao iniciar C. Para testar autonomia, encerre A depois que B e C estiverem conectados e envie mensagens entre eles. Encerre C de forma abrupta, observe a desconexão em B e reinicie C usando `data/c`; o ID deve permanecer igual. Para mudar a porta, reinicie C em outra porta e informe um bootstrap vivo. Se todos os endereços conhecidos mudarem e ninguém conseguir alcançar ninguém, será necessário fornecer novamente um endereço atual.

## Dois computadores em redes diferentes

1. Em A, execute com `--listen /ip4/0.0.0.0/tcp/4001`. Anote seu ID e identifique um endereço externo alcançável. `0.0.0.0`, `127.0.0.1` e endereços privados não servem como destino pela internet.
2. Autorize TCP 4001 no firewall de A. Se houver roteador com IP público, configure encaminhamento TCP 4001 para A ou verifique se o mapeamento automático funciona. Não alteramos seu firewall automaticamente.
3. Em B, use `--bootstrap /ip4/IP_PUBLICO_A/tcp/4001/p2p/ID_A`, diretório próprio e sua porta local. Confirme `/peers` e envie mensagens nos dois sentidos.
4. Repita com reinício, mudança de porta e interrupções temporárias da rede. Registre latência, logs, topologia e resultados; relógios precisam estar razoavelmente sincronizados para a janela de validade das mensagens.

Se ambos estiverem atrás de NAT restritivo/CGNAT, conexão direta pode falhar. Para esse caso, disponibilize um terceiro participante R em rede com endereço público, firewall adequado e porta fixa:

```text
projetop2p --data data/relay --listen /ip4/0.0.0.0/tcp/4001 --relay-server
```

Em A e B, informe `--relay /ip4/IP_PUBLICO_R/tcp/4001/p2p/ID_R` e um bootstrap alcançável. É possível usar R como bootstrap também. Aguarde AutoNAT/AutoRelay detectar a situação; `/addresses` pode mostrar um endereço com `/p2p-circuit`. Compartilhe o endereço completo do participante, incluindo relay e ID final, usando `/connect` se necessário. Hole punching poderá tentar promover a conexão para direta. NAT simétrico, políticas de firewall, ausência de observadores externos e quotas do relay podem impedir sucesso. A configuração não garante conectividade universal.

Relays transportam tráfego criptografado e não são repositórios de mensagens. R deve ter banda, recursos e consentimento do operador. Não há relays públicos embutidos. **NAT traversal e relay não foram validados entre redes reais nesta entrega.**

## Testes automatizados

`go test ./...` inclui identidade persistente/corrompida, adulteração de assinatura, expiração, limites, replay, três hosts independentes, descoberta pela DHT, envio, queda, reinício e recuperação. Os hosts do teste usam loopback e diretórios independentes.

Teste adicional com três processos reais, incluindo falha de bootstrap e reinício com novo endereço:

```powershell
python tests/cli_smoke.py dist/projetop2p-windows-amd64.exe
```

```sh
python3 tests/cli_smoke.py dist/projetop2p-linux-amd64
```

Veja `docs/validation.md` para resultados efetivamente obtidos. O workflow em `.github/workflows/validate.yml` prepara testes em Windows/Linux e detector de corridas no Linux; ainda não foi executado em um serviço de CI.

## Estrutura

```text
cmd/projetop2p/       CLI
internal/identity/   identidade persistente e proteção por plataforma
internal/network/    host, streams e recuperação de conexões
internal/discovery/  rendezvous sobre Kademlia
internal/messaging/  formato e assinatura de mensagens
internal/security/   replay e limitação de frequência
internal/registry/   contratos de registro futuros
internal/storage/    contrato para ciphertext futuro
configs/             referência das flags
docs/                arquitetura, decisões, ameaças e validação
scripts/             builds Windows/Linux
tests/               teste adicional com processos
```

## Limites e evolução

Transporte seguro não equivale a um sistema de mensagens com criptografia ponta a ponta para armazenamento e encaminhamento futuros. Nesta versão, a aplicação envia diretamente ao destinatário autenticado; o conteúdo fica em memória e aparece no terminal. Não entregue chaves privadas a outros participantes. Faça backup seguro do perfil/identidade e não compartilhe diretórios `data`.

Registro de participantes significa aqui observação temporária de pares e registros distribuídos de provedores do rendezvous. Não há cadastro global verificável, permissões de comunidades ou resistência a Sybil. Persistência e expiração das observações locais serão uma próxima etapa.

Consulte `docs/security/threat-model.md`, `docs/architecture/roadmap.md`, `docs/decisions/0001-core.md` e `docs/decisions/blockchain-options.md`. Blockchain e voz não foram implementadas. O armazenamento futuro usará conteúdo criptografado, hashes, replicação e reparação; a DHT guardará localização e referências.
