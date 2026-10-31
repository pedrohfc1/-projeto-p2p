# Projeto P2P 0.3 — Hamachi opcional

Aplicativo desktop Windows com identidade Ed25519, chat libp2p autenticado, histórico e contatos persistentes. A 0.3 adiciona detecção de Hamachi e seleção de interface; bootstrap, DHT e Circuit Relay v2 continuam disponíveis no modo automático.

## Instalar e conectar dois computadores

1. Em cada computador Windows, instale/autentique o Hamachi manualmente e entre na mesma rede virtual. O Projeto P2P não instala, autentica ou altera o Hamachi.
2. Feche a versão anterior e execute **ProjetoP2P-Setup-v0.3.exe**. Ele instala por usuário, cria atalho no menu Iniciar, oferece atalho de área de trabalho e instala WebView2 se necessário. Não inclui Hamachi. Identidade, contatos, histórico e configurações ficam preservados no perfil existente.
3. Abra o aplicativo e vá a **Configurações → Modo de conectividade → Hamachi**. Selecione detecção automática ou um adaptador confirmado, mantenha TCP 4001 e salve. Em dois computadores diferentes a mesma porta pode ser usada. Duas instâncias no mesmo PC precisam de perfis e portas diferentes.
4. Se a detecção falhar, escolha **Seleção manual de interface de rede** e selecione o adaptador virtual correto. Não selecione Wi-Fi/Ethernet por engano.
5. Abra **Estado da rede → Conectividade — Hamachi**. Confira adaptador, IPv4 virtual, estado, Peer ID e multiaddr. Interface ativa ainda não significa peer conectado.
6. No computador A, clique **Copiar convite**. No B, use **Conectar por convite**, cole e confirme. O programa verifica a identidade e abre a conversa. Também aceita multiaddr completo; nesse caso a identidade é verificada durante o handshake autenticado.
7. Troque mensagens. Para que ambos tenham o nome de exibição assinado do outro, troque os dois convites. Um contato que chega sem convite recíproco é identificado inicialmente pelo Peer ID.
8. Se necessário, selecione o contato e use **Testar conectividade** no painel de rede. O resultado verifica conexão e ping libp2p; no Windows inclui leitura consultiva das regras de firewall. Não cria nem altera regras.

**No modo Hamachi não é necessário configurar bootstrap público, relay libp2p, VPS ou encaminhamento de portas do roteador.** As configurações antigas desses serviços permanecem salvas, mas não são usadas no modo de interface. A rede Hamachi e o firewall ainda precisam permitir o TCP entre os participantes. O Hamachi pode usar coordenação/retransmissão de seu fornecedor; uma conexão libp2p direta sobre a VPN não comprova caminho físico direto.

Guia detalhado e roteiro entre casas: `docs/hamachi-guide.md`. Resultados executados, inclusive limitações, em `docs/validation-v0.3.md`.

## Detecção e privacidade

No Windows, o backend usa interfaces do Go, IP Helper GetAdaptersAddresses e leitura de identificadores do driver no registro. Combina GUID/índice, estado operacional, IPv4 e evidências do driver. Nome amigável sozinho ou um endereço 25.x sozinho não confirmam Hamachi. Se a leitura estiver indisponível, a seleção manual continua possível. Nenhuma configuração do Windows é escrita por essa detecção.

O adaptador e o IP são monitorados a cada três segundos, além das atualizações da interface. Se ele desaparecer, ficar inativo ou perder IPv4, o nó desse modo é encerrado e a identidade permanece; a aplicação não muda silenciosamente para a rede pública. Ao voltar, o host é recriado na interface selecionada. Contatos conhecidos são reconectados, e endereços recebidos de peers autenticados são atualizados no cache. Se todos mudarem seus endereços sem caminho conhecido, gere novos convites.

No modo Hamachi/interface: transporte TCP com endereço de escuta **e origem de discagem** vinculados ao IPv4 escolhido; Noise; somente esse endereço anunciado; DHT/AutoRelay/NAT mapping/DCUtR libp2p desabilitados nesse host. Convites recebem escopo assinado e não incluem relays. Convites virtuais não são importados silenciosamente no modo público. No modo automático do desktop, endereços 25/8 são excluídos do anúncio nativo e referências DHT são filtradas para endereços públicos.

Não há varredura IP ou API inventada de lista de membros Hamachi. O mecanismo principal é convite + contatos persistentes. mDNS não foi habilitado: exige multicast que não foi validado nessa rede virtual. Consulte a decisão arquitetural 0003.

## Dados e atualizações

Windows: `%APPDATA%\ProjetoP2P`; Linux: `$XDG_CONFIG_HOME/ProjetoP2P` ou `~/.config/ProjetoP2P`. O banco bbolt e `identity.key` da 0.2 continuam no mesmo formato/caminho. Novos campos de conectividade têm valores padrão compatíveis com configurações antigas. Windows usa DPAPI para a chave; Linux usa permissões 0700/0600. O histórico não é criptografado em repouso pelo aplicativo.

Instalação por usuário em `%LOCALAPPDATA%\Programs\ProjetoP2P`. O desinstalador remove programa, atalhos e entrada de desinstalação, preservando o perfil. Feche o aplicativo antes de instalar por cima. O instalador 0.3 é implementado em Go 64 bits, com manifest asInvoker, diálogos nativos e bootstrapper WebView2 assinado pela Microsoft. Não é apenas um executável portátil renomeado. Ele e o aplicativo ainda não possuem certificado de assinatura de distribuição.

A geração e o payload do setup foram verificados. Instalação, atualização, atalhos e desinstalação em um Windows comum ainda precisam de validação de aceitação; não foram executados sobre o seu perfil. Veja `docs/installation.md`.

## Build

Dependências fixadas: Go >=1.26 (usado 1.27.3), Wails v2.17.0, React 19.2.0, TypeScript 5.9.3, Vite 7.3.7, libp2p v0.51.0, kad-dht v0.43.0 e bbolt v1.5.0. Sem SDK Hamachi ou dependência Go nova para a VPN.

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
cd frontend
npm ci
npm run build
cd ..
go test ./... -count=1 -timeout=180s
go vet ./...
./scripts/desktop.ps1 -Installer
./scripts/build.ps1
```

O empacotamento 0.3 usa `scripts/package-windows.ps1`, sem NSIS como dependência de build. Gera desinstalador, comprime o aplicativo, confere assinatura do bootstrapper Microsoft, incorpora os arquivos e verifica o SHA-256 do aplicativo dentro do setup. `tests/installer-smoke.ps1` verifica o payload sem instalar o produto. Fontes NSIS antigas permanecem como referência, mas não são o caminho canônico da 0.3.

Mint 22: instale `build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev`, Go e Node. Compile o frontend, instale Wails, execute `go test -tags webkit2_41 ./...` e `sh scripts/desktop-linux.sh`. O modo automático e a seleção manual são compatíveis; a identificação por driver Windows é específica do Windows. O workflow Desktop v0.3 prepara build nativo Linux; não foi executado remotamente nesta entrega. O CLI Linux foi cross-compilado no Windows.

## Diagnóstico CLI

```powershell
./dist/projetop2p-windows-amd64.exe --interfaces
./dist/projetop2p-windows-amd64.exe --data data/vpn --interface-ip IP_VIRTUAL_REAL --listen /ip4/IP_VIRTUAL_REAL/tcp/4001
```

`--interfaces` é somente leitura e mostra os adaptadores/evidências. `--interface-ip` é um diagnóstico explícito, sem detecção de membros da VPN; não aceite endereços de desconhecidos. `/status` mostra `localOnly: true` nesse modo. A infraestrutura nativa está documentada em `docs/relay-quickstart.md`.

## Validação e limites

A suíte automatizada cobre detecção por fixtures, adaptador renomeado, IP, estado inativo/ausente, seleção manual, convites/escopo/assinatura, mensagens por interface conhecida, reconexão, reinício e persistência. Testes reais de enumeração e bind no adaptador Hamachi local foram possíveis. Uma tentativa entre dois processos pelo IPv4 virtual local deu timeout; um TCP simples também falhou antes do libp2p. Esse resultado foi registrado, não tratado como sucesso de comunicação Hamachi.

Não foram realizados testes entre casas, configuração de firewall, autenticação do Hamachi, instalação completa do produto ou execução nativa Linux Mint. A janela WebView2 também precisa de validação fora do ambiente isolado. Não há garantia de conexão em qualquer política de rede. Sem voz, blockchain ou armazenamento distribuído de conteúdo.
