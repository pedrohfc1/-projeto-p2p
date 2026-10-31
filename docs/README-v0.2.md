# Projeto P2P — Desktop 0.2

**Infraestrutura Bootstrap + Relay:** modo CLI existente confirmado e acrescido de `--announce`, `--check-relay`, `--status-every` e `/status`. Veja [iniciar e diagnosticar um nó público](docs/relay-quickstart.md). Interface gráfica preservada nesta atualização.

Mensageiro desktop Windows 10/11 AMD64 com React + TypeScript e backend Go/Wails. O núcleo libp2p da versão 0.1 foi preservado, incluindo identidade Ed25519 e o protocolo de mensagens. Linux Mint é alvo de build nativo preparado no workflow.

## Começar no Windows

O pacote inclui `build/bin/ProjetoP2P.exe`: abra com duplo clique, sem terminal. Ele é um executável desktop, **não um instalador**. É necessário Microsoft Edge WebView2 Runtime. O executável foi compilado com a estratégia Wails de bootstrapper embutido; se o runtime não existir, sua instalação precisa de internet e das permissões normais do Windows.

O instalador real será gerado como `ProjetoP2P-Setup-v0.2.exe` pelo workflow `.github/workflows/desktop.yml`. Não existe um setup validado neste pacote: o compilador NSIS não executa no ambiente isolado usado para esta entrega. Veja `docs/installation.md` para gerar e baixar o artefato do workflow. O aplicativo Windows compilou, mas a abertura nativa não pôde ser validada aqui: WebView2 retornou 0x8000ffff ao criar o controlador. Não desative mecanismos de proteção para contornar isso.

Na interface:

1. Em **Configurações**, escolha seu nome.
2. Em **Rede comunitária**, informe os bootstraps e relays de operadores conhecidos. Um endereço completo por linha, com `/p2p/ID` ao final. Nenhum serviço público arbitrário está pré-configurado.
3. Mantenha **Modo assistido para NAT / CGNAT** habilitado para solicitar reservas sem esperar a classificação automática de alcance. Salve as configurações.
4. Em **Estado da rede**, aguarde os endereços com reserva aparecerem. Sem relays disponíveis, a interface informa que não há caminho de relay anunciado.
5. Gere seu convite e compartilhe por outro canal. Peça ao contato para fazer o mesmo e use **Adicionar contato**.
6. Selecione o contato e envie mensagens. O cabeçalho mostra `direta`, `relay` ou desconectado. O histórico fica neste perfil.

Convites têm assinatura verificável, validade de sete dias, nome, chave pública, ID e endereços. Não carregam chave privada. A assinatura comprova controle da identidade P2P; confira por outro canal quem é a pessoa. Endereços/relays dentro de um convite não são autorização para alterar silenciosamente suas configurações de infraestrutura. Um convite antigo pode precisar ser substituído se os endereços mudarem.

## Internet sem configurar o roteador do usuário

A aplicação tenta mapeamento NAT, usa Identify, AutoNAT no modo automático, AutoRelay, Circuit Relay v2 e DCUtR/hole punching. No modo assistido, declara-se cliente privado para iniciar reservas de imediato; essa indicação é política local, não uma medição de CGNAT.

**Dois participantes atrás de NAT/CGNAT ainda precisam de algum participante auxiliar alcançável.** Um relay comunitário em máquina pública atende esse papel. Ele não controla identidades, permissões ou conteúdo, e pode ser substituído. Configure dois operadores independentes para redundância. Nenhum endereço de relay foi inventado ou incluído no programa.

AutoRelay procura candidatos configurados e registros DHT do namespace de operadores, valida suporte ao protocolo, solicita até duas reservas e gerencia renovação e substituição. O operador publica a capacidade na DHT da aplicação. Convites usam os endereços ativos publicados pelo host. O libp2p pode promover uma conexão por relay para direta quando a rede permitir. Não há garantia de hole punching em todas as redes.

Veja `docs/relay-operations.md` e `docs/internet-tests.md`. O usuário final não precisa encaminhar portas, mas o operador do relay precisa de conectividade pública e firewall apropriado.

## Dados locais e atualização

Windows: `%APPDATA%\ProjetoP2P`. Linux: `$XDG_CONFIG_HOME/ProjetoP2P` ou `~/.config/ProjetoP2P`. Arquivos: `identity.key` e `chat.db`. Windows mantém a chave protegida por DPAPI; Linux usa 0700/0600. A base bbolt é equivalente transacional embutida para contatos, configurações e histórico, sem serviço de banco externo e sem CGO no núcleo.

O instalador/atualizador não deve apagar esse perfil. A desinstalação preserva os dados. **O histórico local não é criptografado pelo aplicativo.** Processos do mesmo usuário e administradores estão fora da proteção da chave. Não compartilhe a pasta de perfil. Duas instâncias simultâneas precisam de perfis separados; o banco impede uso simultâneo do mesmo arquivo.

A versão 0.1 usava diretórios informados por `--data`. Para reutilizar a identidade, feche o nó antigo e abra o desktop com `--data "CAMINHO_DO_PERFIL_ANTIGO"`. Exemplo em PowerShell: `./build/bin/ProjetoP2P.exe --data ./data/a`. O nome do arquivo de identidade e o formato protegido foram preservados. Não execute CLI e desktop sobre o mesmo perfil ao mesmo tempo. Copiar uma chave DPAPI para outro usuário/computador não é uma migração suportada.

## Testar dois perfis no mesmo computador

Abra dois PowerShells na raiz do projeto somente para iniciar as instâncias:

```powershell
./build/bin/ProjetoP2P.exe --data ./data/desktop-a
./build/bin/ProjetoP2P.exe --data ./data/desktop-b
```

Cada linha pertence a um terminal separado. Depois use apenas as janelas. Gere e troque os dois convites. Conexão direta local não exige relay. Reinicie cada aplicativo e confira ID, contatos e histórico. Em computadores distintos na LAN, autorize o aplicativo no firewall da rede usada; convites precisam incluir o endereço LAN alcançável.

## Compilar

Ferramentas usadas: Go 1.27.3 (módulo exige >=1.26), Node 24, Wails v2.17.0, React 19.2.0, TypeScript 5.9.3, Vite 7.3.7, bbolt v1.5.0. Núcleo: libp2p v0.51.0 e kad-dht v0.43.0. Versões e hashes estão em go.mod/go.sum e package-lock.json. Consulte ADR 0002 para justificativa.

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
cd frontend
npm ci
npm run build
cd ..
go test ./... -count=1 -timeout=180s
go vet ./...
./scripts/desktop.ps1
# Com NSIS no PATH:
./scripts/desktop.ps1 -Installer
# CLI para Windows e Linux:
./scripts/build.ps1
```

No Linux Mint 22, instale `build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev`, Go e Node; compile frontend e instale Wails como acima. Execute `go test -tags webkit2_41 ./... -timeout=180s`, `go vet -tags webkit2_41 ./...` e `sh scripts/desktop-linux.sh`. O workflow Ubuntu 24.04 gera o binário desktop e os CLIs. Execução nativa em Mint ainda depende de validação real. Mint 21 exige confirmar a disponibilidade de WebKit2GTK 4.1 nos repositórios utilizados.

O frontend `dist` está incluído para permitir testes Go sem Node. Depois de alterar o React, gere-o novamente. Os bindings são expostos pelo IPC do Wails; a interface não usa um servidor HTTP na versão desktop.

## CLI e operador de relay

A CLI existente permanece em `cmd/projetop2p`. Flags adicionais:

- `--private`: solicita relay sem esperar AutoNAT.
- `--public`: declaração explícita de alcance público por um operador.
- `--daemon`: não encerra ao receber EOF em stdin.
- `--relay-server`: oferece serviço Circuit Relay v2.

`--bootstrap`, `--relay` e `--listen` continuam repetíveis. A descoberta de relays usa a DHT privada do projeto. Nenhum bootstrap IPFS está embutido. DHT armazena referências e presença, não o histórico de chat.

## Testes e limites

`go test ./...` cobre identidade, convites, adulteração, replay, limites, banco transacional, dois perfis desktop, histórico/reinício, três hosts, recuperação, encaminhamento por relay e failover automático para outro relay com caminhos diretos bloqueados pelo gater do teste. Isso não simula fielmente CGNAT.

Teste de três processos CLI: `python tests/cli_smoke.py dist/projetop2p-windows-amd64.exe` (no Linux, `python3` e o binário Linux). `tests/webui` é um harness de desenvolvimento em loopback, com token por execução e sem uso no produto. Ele permite verificar as operações reais do backend por HTTP em ambiente de QA; não é uma interface pública nem alternativa de instalação.

Leia os resultados em `docs/validation.md`. A CI foi preparada; não foi executada remotamente nesta entrega. Testes em duas redes físicas, CGNAT real, renovação durante 20 minutos, carga adversarial e abertura nativa no Mint ainda estão pendentes.

Sem voz, blockchain, comunidades, grupos, entrega offline ou armazenamento distribuído de conteúdo. Transporte autenticado não equivale a uma arquitetura de conteúdo ponta a ponta para histórico e replicação futura. O ACK desktop é enviado após gravação local; não confirma leitura humana. Falha/timeout após envio significa resultado incerto, e a interface informa que pode ter havido recebimento.

## Código e documentação

- `main.go`, `frontend/`: janela Wails e React.
- `internal/desktop/`: casos de uso, perfis e integração.
- `internal/localdb/`: banco transacional e retenção (10 mil mensagens totais; 300 exibidas por conversa).
- `internal/invite/`: convites públicos assinados.
- `internal/network/`, `discovery/`, `identity/`, `messaging/`, `security/`: núcleo P2P.
- `internal/storage/`, `registry/`: fronteiras futuras, sem blockchain ou replicação implementadas.
- `docs/installation.md`, `relay-operations.md`, `internet-tests.md`, `validation.md`: operação e validação.
- Documentação 0.1 preservada em `docs/README-v0.1.md` e `docs/validation-v0.1.md`.
