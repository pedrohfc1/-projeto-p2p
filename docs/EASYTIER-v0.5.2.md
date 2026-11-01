# EasyTier 0.5.2 — laboratório experimental

Branch `experimental/v0.5.2-easytier`, após validar e separar `feature/v0.5.1-compact-invites`. Em 2026-11-01, a API oficial de releases apontou **EasyTier 2.6.4** como stable/latest, commit `8428a89d2dabc94c97d370ec607c6ca142473626`. Fonte: https://github.com/EasyTier/EasyTier/releases/tag/v2.6.4 . Não usamos nightly nem um install script remoto executado pelo shell.

## O que funciona e o que ainda não é produto

Há protótipo funcional Windows AMD64, sem TUN/driver, com dois hosts libp2p e auxiliares EasyTier reais: stream de texto/echo autenticado por Noise, descoberta via nós conhecidos, reconexão, teste de identidade, relay privado e diagnóstico. A UI oferece **Rede P2P integrada (laboratório experimental)** em Configurações. Mostra processo, reinícios, peers, custos de rota, latência, contadores e diagnóstico; processo ativo não é apresentado como peer conectado.

**O laboratório usa um stream separado. Ele ainda não transporta o chat individual nem as comunidades.** Esses recursos continuam usando Radmin, Hamachi ou libp2p nativo conforme a configuração anterior. O seletor abre o laboratório, sem persistir um modo incompleto na configuração antiga. Essa separação mantém o protótipo isolado e evita misturar lógica de comunidades com VPN. Convites `projetop2p-lab:e1:` são aceitos somente pelo teste de stream, não concedem ingresso em comunidade.

O objetivo definitivo de instalar apenas o Projeto P2P e conversar pela internet está **em desenvolvimento**. O protótipo não é uma alegação de conectividade universal, CGNAT resolvido ou implantação pronta para leigos. Não há nó público autorizado disponível nesta sessão; o usuário confirmou que ainda não possui um. Nenhum nó público vem embutido. Testes entre casas/CGNAT permanecem pendentes.

## Integração escolhida

O release estável avaliado fornece core Rust, CLI/RPC local e SOCKS5/no-TUN; não adotamos uma ABI Go estável para esse release. A árvore de desenvolvimento atual já mostra iniciativas `easytier-go`, mas isso não comprova API/artefato estável compatível com 2.6.4. FFI exigiria build Rust/C, manutenção de ABI e revisão de licenças. Optamos por processo auxiliar separado, configuração limitada e CLI somente de leitura. Fonte versionada: https://github.com/EasyTier/EasyTier/tree/v2.6.4 .

Go libp2p TCP → SOCKS5 em 127.0.0.1 → pilha userspace EasyTier → conexão direta ou relay privado → EasyTier remoto → **somente porta loopback reservada ao host libp2p do laboratório**. Noise e o Peer ID esperado continuam obrigatórios. A chave Ed25519 não é entregue ao EasyTier: ele recebe credencial de transporte separada. O teste verificou que o laboratório reutiliza o mesmo Peer ID do perfil existente.

O host libp2p reserva a porta de aplicação antes de iniciar o auxiliar. A whitelist TCP contém somente essa porta; UDP de aplicação usa whitelist sem portas utilizáveis (0), sem subnet proxy, exit node, port forwarding, DNS integrado ou rota global. UPnP, IPv6 do overlay e relay de broadcast são explicitamente desativados na configuração final. RPC e SOCKS ficam em loopback. Rede privada com nome/segredo próprios e `private_mode=true`; não há uma rede global compartilhada por todos os usuários. A autenticação EasyTier não concede permissões de comunidade.

O primeiro ensaio falhou porque a seleção automática do EasyTier tentou bind em APIPA. O protótipo define `bind_device=false`, deixando o SO escolher a origem para o destino, sem modificar adaptadores. Isso corrigiu o teste loopback. Não prova seleção ótima de underlay em todos os PCs; diagnóstico e testes externos ainda necessários.

## Execução local e instalação do protótipo

Para uso normal e convites compactos, use o instalador **0.5.1**. Faça backup privado com o aplicativo fechado. Não envie perfil, banco ou chave ao amigo.

Para desenvolvimento 0.5.2:

1. Extraia o pacote de laboratório numa pasta própria. Não execute simultaneamente usando o mesmo perfil da instalação pessoal.
2. O pacote de entrega **não redistribui os executáveis EasyTier**. Faça a preparação explícita abaixo a partir do release oficial. O script verifica o ZIP e ambos os executáveis por SHA-256 e extrai core/CLI e a DLL de usuário Packet.dll exigida pelo binário oficial; não extrai WinDivert64.sys nem wintun.dll. Packet.dll tem licença própria Npcap, que deve ser revisada antes do uso. Nenhum driver é instalado. Não executa os arquivos baixados.
3. Abra o app com perfil isolado. É necessário WebView2, sem Go/Node para executar o app; Python 3 é necessário apenas para o preparador de desenvolvimento.

```powershell
python scripts/prepare-easytier.py --output ./easytier
./ProjetoP2P.exe --data ./perfil-laboratorio
```

O app procura `easytier` junto do seu executável; alternativamente informe o caminho absoluto dos auxiliares e Packet.dll no painel. Não há download de código pelo app. O preparador é uma etapa manual de desenvolvimento, não um instalador de VPN. Em máquina offline pode usar `--archive CAMINHO_ZIP_OFICIAL`, com a mesma verificação.

4. Em Configurações, escolha Rede P2P integrada. Gere nome e segredo aleatórios; envie por canal privado ao outro participante. Não envie o segredo no chat de suporte. O app não salva essa credencial no banco: ela fica na memória e no ambiente do processo auxiliar, fora da linha de comando. Após iniciar, a caixa de segredo é limpa; para reiniciar após parar será necessário reintroduzi-lo.
5. Use IPs distintos `10.203.0.1`, `.2`, etc. no mesmo /24. No mesmo PC, use quatro portas diferentes por instância; entre PCs, as portas podem coincidir. Padrões: app 45101, transporte 45110, SOCKS 45180, RPC 45188. SOCKS/RPC não devem ser expostos.
6. Para teste só neste PC mantenha escuta 127.0.0.1. Para receber de outro PC escolha escuta externa, sem alterar firewall automaticamente. O nó B inclui `tcp://IP_ALCANCAVEL_DO_A:PORTA_TRANSPORTE`. O endereço de underlay não é o IP virtual 10.203. Não use 127.0.0.1 entre PCs.
7. Inicie os dois laboratórios, copie o convite experimental de A para o campo de teste de B e execute o probe. Confira Peer ID autenticado e retorno do texto; processo em execução ou peer na tabela sozinho não comprova isso.
8. Clique Parar antes de remover a pasta. O app fecha o auxiliar e o host do laboratório. Não instala serviço, adaptador ou driver. Remova apenas a pasta extraída e o perfil experimental explicitamente escolhido; preserve o perfil pessoal.

Não há instalador/desinstalador completo do componente EasyTier nesta entrega. O setup 0.5.2 do aplicativo pode ser compilado/verificado, mas não embute EasyTier. Integração ao instalador exige concluir fonte correspondente/notices e aceitação em VM. A validação de payload não equivale a instalação/atualização/desinstalação completas.

## Nós conhecidos, NAT e relay

Até oito URIs TCP/UDP conhecidas são aceitas; o EasyTier faz descoberta de rotas, tentativas de enlace e reconexão. Suporte de NAT traversal existe no componente, mas depende de disponibilidade de peers/STUN/relay e características da rede. STUN só é usado se configurado explicitamente; listas vazias substituem os defaults. Configure somente serviços realmente autorizados, sem copiar endpoints aleatórios de listas públicas.

Para duas casas atrás de CGNAT, o próximo ensaio precisa de pelo menos um host público autorizado. O dono desse host instala a versão oficial verificada e opera rede privada no mesmo nome/segredo. Listener TCP/UDP numa porta alta, sem TUN, sem subnet/exit node/DNS, RPC somente loopback, sem clientes estrangeiros. O operador precisa autorizar exposição da porta no **servidor**; o app não faz isso nem exige abrir portas nos roteadores dos clientes. Ambos os clientes usam o endereço público desse host como nó conhecido. Um segundo host autorizado permite testar indisponibilidade/fallback.

O relay pode ver metadados e consumir banda/CPU; capacidade não é ilimitada. A escolha definitiva deve avaliar segregação de credenciais por comunidade, admissão por dispositivo, quotas e disponibilidade. O segredo de transporte compartilhado desta prova não substitui a identidade/assinaturas do app. Não publicar um segredo único para todos os usuários.

## Limites e ciclo de vida

- Core e CLI 2.6.4 mais Packet.dll fixados por hash; hash conferido antes de execução e reinícios. Arquivos diferentes são recusados. Hash garante correspondência ao download fixado, não ausência de vulnerabilidades nem assinatura do publicador.
- Processo sem shell/elevação, ambiente ET_* herdado removido, logs do auxiliar descartados para evitar gravação de configuração/segredo. Diagnóstico vem somente de `peer`/`connector`, nunca `node config`.
- Windows Job Object: kill-on-close, memória de processo 256 MiB e teto de CPU 25% da máquina. CPU mostrada nas métricas usa 100%=um núcleo, portanto são escalas diferentes. Se não conseguir aplicar isolamento, falha sem manter o processo.
- No máximo três reinícios após falha, com espera progressiva; o EasyTier reconecta enlaces internamente. Parar cancela reinícios. Queda do hospedeiro fecha o Job Object e encerra o auxiliar.
- Sistema libp2p: 32 conexões/streams, orçamento de memória 32 MiB, quatro handlers de echo concorrentes, frame até 4096 bytes, prazos e um probe por vez. Contadores globais não criam mapa ilimitado por Peer ID.
- Limite de recebimento configurado no EasyTier: 1.048.576 BPS; não há prova de shaping completo de upload e memória de todas as estruturas. Não anunciar esses limites como proteção total contra DDoS/Sybil.

O socket de loopback continua acessível a processos locais; não protege contra malware ou outro processo com acesso ao usuário. O segredo permanece na memória/ambiente do auxiliar durante a sessão. Não promete zeroização de todas as cópias. Não existe E2EE de grupo novo. A verificação de chave esperada continua em Noise; um nó que apenas sabe o segredo não pode se passar pelo Peer ID do convite.

## Testes e desempenho

Testes opcionais reais (sem drivers):

```powershell
$env:EASYTIER_LAB_BIN = (Resolve-Path ./easytier).Path
go test ./internal/easytierlab -v -count=1 -timeout=150s
go build -buildvcs=false -o work-webui.exe ./tests/webui
python tests/easytier_lifecycle_smoke.py ./work-webui.exe ./easytier
```

Passaram: dois nós e echo Noise, identidade esperada, canário TCP de outro serviço local inacessível, Peer ID forjado recusado, crash/reinício do auxiliar com retorno do stream e encerramento normal. Passou relay privado com três auxiliares, A/B com P2P automático desativado; o diagnóstico marcou **relay(2)**. Passou matar o hospedeiro e verificar o PID conhecido do auxiliar encerrado. Não foi alterado firewall nem configuração das VPNs.

Medição inicial: echo com conexão inicial **4,603 ms**, relay **5,117 ms**, ambos no mesmo PC, não internet. Auxiliares: aproximadamente **25,41 e 25,85 MiB RSS**. Uma amostra curta registrou CPU 0% nos auxiliares, insuficiente para inferir consumo sustentado zero. O processo `go test` hospedava dois hosts e registrou cerca de **69,8 MiB RSS**; os dois campos Go do JSON são o mesmo processo, não devem ser somados. WebView2 não foi medido. Tráfego registrado pelo libp2p de cada host: 489 bytes enviados/recebidos nessa amostra; não representa todo overhead EasyTier. Não houve benchmark de vazamento de horas ou comparação controlada de otimização.

CLI libp2p pode registrar que o wrapper SOCKS não expõe SetKeepAlive. O stream autenticado passou e usa deadlines; não suprimimos o diagnóstico. Resolver ergonomia/keepalive e observar conexões longas é parte da integração definitiva.

Suíte Go/vet/mod verify e frontend TypeScript/testes passaram, além de regressões de Radmin/DM/comunidades e compatibilidade de convites. Dois testes de helper são opcionais na suíte normal e foram executados separadamente com binários reais. `govulncheck` continua com GO-2024-3218 alcançável no DHT nativo; sem supressão e sem release. `npm audit` não reportou vulnerabilidades. Não foi feito cargo-audit das dependências Rust; ausência de advisory publicado na página oficial não comprova ausência de vulnerabilidades: https://github.com/EasyTier/EasyTier/security .

Pendências: dois PCs/casas, CGNAT real, troca de rede, suspensão, UDP/NAT traversal externo, relay público autorizado/fallback, longa duração, desktop Mint, inspeção visual nativa, instalação/desinstalação em VM e auditoria independente. A prova Windows no-TUN não valida uma implantação Linux ou a totalidade do isolamento contra um peer malicioso.

## LGPL e distribuição

EasyTier 2.6.4 é LGPL-3.0; fonte oficial e texto: https://github.com/EasyTier/EasyTier/blob/v2.6.4/LICENSE . A LGPL incorpora disposições GPLv3: https://www.gnu.org/licenses/lgpl-3.0.html e https://www.gnu.org/licenses/gpl-3.0.html . O app próprio continua MIT; executar componente separado não remove as obrigações de redistribuição do componente.

Antes de embutir e distribuir core/CLI: preservar avisos e licença LGPL/GPL; identificar mudanças; fornecer fonte correspondente da versão efetiva, instruções/scripts e dependências exigidas; garantir direitos de modificação/substituição e não impor proibição de engenharia reversa para depuração das modificações. O fonte upstream com Cargo.lock foi obtido para avaliação, mas isso sozinho não demonstra completude de todas as fontes/notices das dependências do binário pré-compilado. Não declaramos auditoria de redistribuição concluída.

Além da LGPL, o binário oficial Windows depende de **Packet.dll do Npcap**, identificado nos metadados como Insecure.Com LLC, versão 1.79. Sem a DLL, --version termina com 0xc0000135; com ela, o teste no-TUN funciona. Isso não comprova ausência de todos os drivers já instalados na máquina: a consulta CIM foi indisponível. Não instalamos nem iniciamos deliberadamente um driver. A DLL não deve ser tratada como LGPL: a edição gratuita/demo normalmente não permite redistribuição; conferir condições aplicáveis com o fornecedor antes de embutir. Fontes primárias: https://npcap.com/license.html e https://npcap.com/oem/redist . A prioridade para o produto é um build mínimo EasyTier sem fake-TCP/pcap, não comprar ou aceitar licença silenciosamente.

**Por essa razão, os pacotes entregues de aplicativo/fonte não incluem os executáveis EasyTier nem drivers.** O preparador obtém a cópia oficial para uso local de desenvolvimento, verifica hashes e não executa scripts do fornecedor. Antes de uma distribuição turnkey, fazer inventário de crates/licenças, fonte correspondente/offline build e revisão da forma concreta de entrega. Substituir helper modificado exige também atualizar o hash no fonte do app e recompilar, preservando possibilidade de modificação sem aceitar código desconhecido silenciosamente.

## Plano para integração definitiva

1. Produzir e testar build mínimo sem Packet.dll/Npcap e sem drivers de captura, ou resolver expressamente a licença do componente. Concluir inventário LGPL/notices/fontes e auditoria de dependências Rust, fixar artefatos de Windows/Linux e CI de proveniência. Preparar entrega do componente com atualização assinada/manifesto, sem segredo no Git.
2. Definir interface de transporte no núcleo, movendo somente dial/listen/endereços para o adaptador EasyTier. Comunidades mantêm autorização/sincronização independentes. Provar compatibilidade de DM/histórico/outbox sem mover identidade.
3. Persistir configurações não secretas versionadas e credenciais protegidas por DPAPI/secret store, com migração reversível. Não salvar senha em JSON claro. Admissão criptográfica por dispositivo e isolamento de rede por grupo antes de uso amplo.
4. Configurar dois nós de operadores autorizados e testar direto, relay, queda/fallback e CGNAT em duas casas. Registrar evidência de caminho real, latência, perda quando mensurável e orçamento total (Go+WebView+EasyTier).
5. Aceitar instalação/upgrade/desinstalação em Windows limpo, job/filhos após crash e eventual atualização do componente sem processo antigo. Avaliar Linux Mint nativamente, sem pedir root/driver no modo userspace.
6. Só então habilitar Rede P2P integrada como modo persistente de chat/comunidades e preparar release autorizada. O gate DHT e assinatura/reputação do instalador continuam questões independentes. Voz, tela e blockchain não fazem parte desta tarefa.
