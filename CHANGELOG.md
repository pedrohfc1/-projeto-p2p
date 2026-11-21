## 0.11.0 — perfis, imagens e sons

- Perfis Ed25519 com revisão, avatar otimizado por hash e fontes locais.
- Imagens de comunidades e salas com autorização do proprietário e cache local.
- Efeitos sonoros opcionais para eventos de voz e compartilhamento.

## 0.10.7 — estabilidade de voz e vídeo

- O envio de vídeo descarta quadros antigos e dá prioridade ao Opus quando a fila de áudio cresce.
- A publicação recupera a ponte de vídeo e pede um keyframe novo depois de substituir o stream.
- A captura mantém perfis definidos, mede as dimensões observadas e exige confirmação após a prévia.
- O diagnóstico oferece histórico de 30 minutos e importação de relatórios para comparar transportes.

## 0.10.6 — diagnóstico e validação Tailscale

- Medição da fila de envio de tela e teste de carga limitada com consentimento dos dois participantes.
- O laboratório verifica a presença do destino na tailnet e usa a pilha userspace do SDK fixado para discar.

# 0.10.5 — diagnóstico de voz e laboratório tsnet

- Painel de diagnóstico local na chamada: RTT, contadores reais, gráficos de 60 s, eventos e exportação consentida.
- Comparação de resumos em RAM e importação numérica do laboratório; métricas não disponíveis permanecem n/d.
- Módulo tsnet separado, sem substituir a conexão padrão; login real e transporte completo do app permanecem pendentes.
- Contadores do worklet por consulta, sem conteúdo de áudio nem nova captura; protocolos e codec preservados.

# 0.10.4 — beta privado

- Refinamento visual de mensagens diretas, comunidades, salas de voz e transmissões.
- Assistente Windows em português com pasta personalizada, atalhos e inicialização opcionais; snapshot/rollback das opções.
- Radmin opcional pelo site oficial, sem redistribuição ou instalação automática de driver.
- Sem mudança de protocolos, codecs ou esquema de dados; validação e limites em docs/ETAPA3-v0.10.4.md.

## 0.10.3 — tela de voz, PTT e sensibilidade

Teste único com retorno local, modo de entrada persistente, tecla PTT em foco, atraso0–500ms, sensibilidade automática/manual e gate AudioWorklet. Paleta grafite/índigo; ganho, dispositivos, mute/deaf, vídeo, WASAPI e transportes preservados. Testes físicos em dois PCs pendentes.

## 0.10.2 — diagnóstico de rede, etapa parcial

RTT autenticado com histórico limitado, quantis/variação, gráficos, JSON sem identidades e contadores libp2p por peer. Sem troca de transporte, testes de carga, métricas inventadas ou causa de gargalo comprovada. Instrumentação avançada e validação física pendentes.

## 0.10.1 — microfone e menu de participantes

Supressão nativa persistente e aplicável ao microfone local, avisos deduplicados sem bloquear o app e entrada em escuta sem captura. Menu compartilhado com volume/mute local persistente, perfil/menção/conversas existentes, teclado e remoção autorizada com confirmação. Funções sem backend ficam ocultas. Codec, transportes, vídeo e WASAPI preservados. Validação física em dois PCs pendente.

## 0.9.7 — beta privada

## 0.10.0 — central de configurações e frontend

Uma engrenagem no perfil, central lateral com 15 seções e abas de mídia. Formulários portados sem recriar sessões, padrões de transmissão locais, categorias recolhíveis, cartões de perfil, membros e chat refinados. Reutiliza as ações/permissões existentes; soundboard e novos DSP permanecem futuros.

## 0.9.9 — reprodução RTC da chamada

Consumidores RTC remotos próprios e inaudíveis, saída persistente fora do compartilhamento, limpeza no rebind, retomada independente e diagnóstico real de rede/processamento/saída. Vídeo, WASAPI, codec e transportes preservados. Confirmação acústica no WebView2 e entre PCs permanece pendente.

## 0.9.8 — beta privada

Ativação de voz antecipada e independente do WASAPI, recuperação de saída sem sair da sala; seleção de tela inicia publicação e tenta áudio seguro; detalhes técnicos recolhíveis, participantes de voz compactos e membros por presença/cargo. Associação ambígua de janela ou saída sem separação física mantém vídeo e bloqueia som automático para evitar captura indevida/eco. Validação física pendente; sem publicação pública.
- Voz: tentativa reversa autenticada, desempate de streams e contadores reais por peer. Liveness da mídia desacoplada do mutex de consultas da UI.
- Áudio da tela: WASAPI por aplicativo/exclusão da árvore do PeeTwoPee, consentimento e fluxo Opus separado; opção legada global explícita com aviso de eco; volume independente e proteção contra comandos de sessão antiga.
- Vídeo e dados preservados. Testes físicos de captura/jogo/eco/Radmin e duas casas pendentes.

## 0.9.6 — beta privada
- Continuidade de tela com controles na bandeja Windows; validação da tela fora do mutex de áudio; menos reconfigurações redundantes do encoder; viewer persistente 16:9 com controles sobrepostos e métricas reais.
- Preservados Opus, VP8, libp2p/EasyTier, autorizações, identidade e dados. Validação física permanece pendente.

# 0.9.5 — hotfix de voz

- Renovação assinada independente do orçamento de sincronização; provas imutáveis e atualizações monotônicas.
- Rotas atualizadas por roster, reconexão por peer com cancelamento/fencing e helpers EasyTier sem espera sob mutex global.
- Continuidade offline limitada à prova válida; separação entre participação, autorização e mídia.
- Correlação de sessões/streams e contadores distintos de ponte local e rede; captura não indica transmissão.

# 0.9.4 — console local de diagnóstico

- Console flutuante movível e redimensionável, fechado por padrão e persistente na navegação.
- Eventos operacionais reais do frontend/backend; filtros, pausa, seleção, cópia com aviso e rolagem manual/automática.
- Buffers circulares de 2.000 eventos, mensagens limitadas, atualização visual reduzida e métricas de mídia a cada 3–5 segundos.
- Conteúdo de chat/mídia, convites e credenciais não são coletados; identificadores são ocultados.

# 0.9.3 — interface compacta

- Superfícies em cinza carvão, canais compactos e cabeçalho com tópico.
- Mensagens em linhas, boas-vindas do canal e participantes com avatares circulares.
- Rodapé de perfil/voz integrado; foco por teclado e movimento reduzido preservados.
- Transporte de mídia, autorizações e dados não alterados.

## 0.9.2 — correção de atualizações, beta privado

- Substituição sequencial sem reversão trocada por staging, inventário SHA256, backup, diário com checksum, recuperação e rollback.
- Bloqueio por perfil e por instalação; saída consentida da bandeja por IPC, legado com confirmação adicional e escopo estrito de caminho/usuário/processo.
- Cache HTTP/código/GPU movido sem remover permissões, armazenamento local ou perfil; frontend/backend verificados e revisão por hash do EXE.
- Mensagens de instalação deixam de exibir0.5.2; Registro/atalhos têm snapshot e reversão. Módulos funcionais de voz, vídeo e EasyTier preservados.
- Testes Windows de arquivos/processos, perfil isolado, rollback e .lnk; Registro bloqueado no sandbox. Validação física e instalação completa pendentes.

## 0.9.1 — hotfix privado de áudio/tela

- Slots de áudio por transceiver quando stream ID está ausente; contexto desbloqueado no ingresso e compartilhado entre ganho/mixer, preservado na recuperação.
- Intenção de assistir separada de conexão; perda temporária não encerra ou recolhe o player.
- Removidos teto3Mbps e limites720p30/45FPS; perfis até1080p60 solicitados e limite total2–12Mbps; VP8 preservado.
- Fila de dois quadros, janela adaptativa com512KiB em voo, descarte por idade e prioridade de voz; ACK não conta como frame.
- Telemetria por contadores e diagnóstico de descartes; hardware e captura real60FPS não comprovados; H.264 não implementado.

## 0.9.0 — experiência de uso e transmissões, beta privado

- Indicadores AO VIVO, prévia/Assistir, player persistente compacto/ampliado, desligamento e autorização preservados; codificação pausada sem espectadores.
- Respostas, edições próprias, exclusão lógica, reações, digitação e não lidas; atividades Ed25519 e sincronização incremental com retomada offline.
- Referências privadas autor+ID, ACL por comunidade/época, limites de recursos e cursores persistentes.
- Presença de mute/ensurdecer vinculada à sessão e notificações nativas opt-in com privacidade e silenciamento.
- 38 testes frontend e regressões locais reais de chat/comunidades/Opus/VP8 sobre EasyTier; validação física e entre casas pendente.

## 0.8.2 — painel persistente de chamada, beta privado

- Painel inferior esquerdo com canal/comunidade do estado local verificado, conexão, duração, participantes e perfil/avatar atual.
- Ações compactas de microfone, ensurdecer, tela, desconectar e configurações; câmera desabilitada com tooltip, sem captura.
- Atalho de tela chama diretamente o controlador existente para preservar a ação explícita do seletor/prévia; parada permanece disponível durante transmissão.
- Indicadores imediatos de mute/ensurdecer/tela; estilos relacionados de sala, participantes e tela alinhados ao painel, sem novos assets proprietários.
- Saída/cancelamento limpa o estado ocupado e descarta respostas antigas de consulta/ingresso, impedindo reaparecer como conectado por snapshot atrasado.
- Testes SSR de controles/tooltips/estado/câmera e regressões reais locais de mídia; limites físicos em docs/PEETWOPEE-v0.8.2.md.

## 0.8.1 — configurações durante chamadas e rede padrão, beta privado

- Operações de dispositivo separadas da entrada em chamada; enumeração deduplicada/com prazo deixa de reter o estado ocupado. Retorno, mute e troca de fonte preservados.
- Espera de confirmação de chat não retém o mutex global dos controles de voz; teste com ACK atrasado cobre a regressão.
- Removido Encerrar PeeTwoPee do áudio; bandeja e fechamento permanecem.
- Cabeçalhos reutilizáveis e estilos unificados de formulários, perfis, mensagens, convites e painéis; navegação/painel de membros persistem no perfil WebView.
- Novos perfis e perfis sem campo de modo usam integrada; escolhas explícitas antigas, incluindo auto legado, são respeitadas. Seleção de rede é salva antes da tentativa de iniciar, sem fallback silencioso.
- Status distingue conectando, reconectando, indisponível e peers conectados. Enumeração de adaptadores no modo integrado limitada a uma vez a cada 30 segundos, sem interferir no acompanhamento nativo de IP/interface.
- Regressões e limites em docs/PEETWOPEE-v0.8.1.md; nenhuma publicação pública.

## 0.8.0 — interface e retorno de microfone, beta privado

- Retorno local mantém a intenção ao trocar entrada/saída; dispositivo anterior é liberado, falha não aparece como áudio ativo e há recuperação explícita.
- Dispositivos lado a lado, volumes de entrada/saída persistentes, medidor e retorno independente; captura continua explícita e local no teste.
- Paleta própria, navegação por comunidades/canais, perfil fixo e voz persistente, participantes com conectividade observada e indicador de fala.
- Menus contextuais reutilizáveis com teclado, foco de modais, tooltips, estilos compactos e redução de movimento.
- Envelope binário autossuficiente peetwopee://join/ com política compactada e endpoint integrado derivável, mantendo payload/assinatura Ed25519 e leitores JSON/c2 antigos. Sem serviço central nem código offline fictício.
- Regressões de mídia/comunidades e limites de validação em docs/PEETWOPEE-v0.8.0.md. Perfil e identidade preservados.

## 0.7.0 — microfone e compartilhamento de tela, beta privado

- Dispositivos de áudio com consentimento explícito, mensagens de erro, troca de microfone sem sair da sala e retorno exclusivamente local.
- Build Windows com política WebView2 de consentimento padrão, verificada e reproduzível; builds não verificados não iniciam a interface.
- Compartilhamento explícito de monitor/janela com prévia, VP8 até 720p/30, limite de banda, espectador autorizado e tela cheia.
- Protocolo de vídeo separado sobre libp2p/EasyTier, filas/ACK limitados, reconexão e prioridade para voz; parada ao ocultar/sair.
- Testes sintéticos de vídeo decodificado, Opus e chat simultâneos; testes físicos e desempenho completo pendentes conforme docs/PEETWOPEE-v0.7.0.md.

## 0.6.2 — conectividade das comunidades, beta privado

- Serviço EasyTier estável por Peer ID e resolução sem IP LAN/porta antiga; cache e backoff tratados sem trocar identidade ou transporte.
- Diretório assinado entre membros, persistência selada e retransmissão de eventos de controle assinados pelo proprietário.
- Texto legado com fallback e deduplicação; histórico local e funções autorizadas independentes de erro da autoridade.
- Voz existente com janela offline assinada de até 15 minutos e reconexão independente da sincronização. Novas sessões e mudanças administrativas exigem proprietário.
- Diagnósticos distinguem endereço, transporte, tempo esgotado, autoridade não alcançável e revogação; testes/limites em docs/PEETWOPEE-v0.6.2.md.
## 0.6.1 — PeeTwoPee, beta privado

- Nova marca, SVG/PNG/ICO originais e instalador compatível com o perfil legado.
- Malha de voz até seis peers, trilhas isoladas, volume/mute local por identidade, indicador de fala e preferências persistentes.
- Bandeja Windows/StatusNotifier Linux, configuração de fechamento e redução de trabalho visual em segundo plano, preservando mídia.
- Ingresso pendente/convite consumido deixam de aparecer como negação; revogação permanece autenticada e persistente.
- Testes e limites em docs/PEETWOPEE-v0.6.1.md; nenhuma publicação pública.
## 0.6.0 — beta privado de voz para duas pessoas (2026-11-01)

- Convites compactos com políticas assinadas, cotas, expiração, status e revogação; legados continuam de uso único.
- Novas comunidades com 100 membros registrados, 50 canais de texto e 20 de voz; controle de salas e permissões no backend.
- Diário de texto assinado pelos autores e replicado entre membros nas novas comunidades, separado do registro de controle do proprietário.
- Captura explícita de microfone, seleção de dispositivos, Opus, Pion/DTLS-SRTP e sinalização autenticada libp2p; barra de chamada persistente, mute/deaf e saída.
- Encaminhador UDP EasyTier userspace dedicado à mídia, sem TUN/Npcap; reconexão e encerramento por revogação/expiração.
- Testes locais com dois processos e mídia sintética; testes de dispositivos físicos, instalação limpa e voz entre casas ainda pendentes. Malha de 3–6 e volume individual ficam para 0.6.1.
- Migração schema 3 preserva dados e impede downgrade silencioso. Release público continua bloqueado por avisos Rust e validações pendentes.

## 0.5.2 — beta privado integrado (2026-11-01)

- Chat e comunidades sobre libp2p/Noise via SOCKS/EasyTier sem TUN, com helper embutido, credenciais protegidas e convites assinados.
- Colisões de clientes etcat corrigidas com instâncias independentes e credencial de convite explicitamente reutilizável; permissões de comunidades preservadas.
- Reconexão e ACK idempotente após gravação; espera inicial ajustada à inicialização da malha.
- DHT provider afetada em quarentena por padrão; legado preservado com tag explícita, sem suprimir aviso de segurança.
- Sem Packet.dll/Npcap, drivers ou UPnP; fontes correspondentes e licenças no instalador. Publicação bloqueada e validação entre casas pendente.

# Histórico de mudanças

O projeto usa versionamento semântico. As versões abaixo descrevem desenvolvimento local; não significam que existam releases públicas no GitHub.

## [Não lançado]

- Infraestrutura de distribuição: CI Windows/Linux, Release por tag e build manual sem publicação, Actions fixadas por SHA, checksums e inventários de licenças.
- Auditoria de candidatos Git, gitleaks, análise de vulnerabilidades e documentação pública em português.
- Licença MIT confirmada, testes do frontend e consistência de versão.

## [0.5.2] — 2026-11-01 — experimental, não lançado

- Laboratório EasyTier 2.6.4 no-TUN/SOCKS5, com stream libp2p Noise isolado e identidade do perfil preservada.
- UI experimental, diagnóstico de peers/custos, métricas Go/auxiliar e credenciais somente na sessão.
- Processos com hashes fixos, Job Object Windows, limites, reinício limitado e encerramento após crash do hospedeiro.
- Integração local real: echo autenticado, canário de porta alheia bloqueado, Peer ID incorreto recusado e relay(2) com três auxiliares.
- Auxiliares/driver não redistribuídos no pacote; preparador explícito verificado para desenvolvimento. LGPL/notices/fontes correspondentes ainda precisam de fechamento para distribuição turnkey.
- Chat/comunidades continuam na rede anterior; sem alegação de teste entre casas, CGNAT, Linux nativo ou produto integrado pronto.

## [0.5.1] — 2026-11-01

- Envelope compacto Protobuf/Base64URL para contatos e comunidades, com assinatura e validação preservadas.
- Leitor legado, conversão para 0.5.0, QR Code local e testes entre backends antigo/novo.
- Estrutura local de códigos imprevisíveis, registros assinados, mirrors autorizados e revogação; sem resolução DHT/global.
- Exemplos medidos: contato Radmin 553→193 caracteres; comunidade 869→327. Perfis e esquema 2 preservados.

## [0.5.0] — 2026-11-01

- Comunidades com identidade única, metadados, membros, owner/moderator/member e convites assinados de uso único.
- Canais de texto, definições de voz sem áudio, histórico paginado e registro autenticado com cursor, hash anterior e deduplicação.
- Outbox durável, recuperação após reinício, revogação e controle de conflitos; dono é sequenciador online nesta etapa.
- Interface com barra de comunidades, canais, chat, participantes e configurações; DM e configurações VPN preservados.
- Migração bbolt versionada 1→2 com testes de preservação e downgrade seguro condicionado a buckets vazios.
- Correção de replay de ID no histórico individual, limites de leitura da chave e contatos não solicitados, CSP e file drop desativado.
- Telemetria inicial, testes de integração Radmin locais, preparação opcional de assinatura Windows e documentação E2EE/voz.
- Build Windows de testes sem assinatura; sem publicação. GO-2024-3218 mantém gate bloqueado. Comunidades ainda sem E2EE de grupo.

## [0.4.1] — 2026-10-31

- Escolha explícita da VPN ao gerar/receber convites; aviso de VPN detectada mas não selecionada no modo nativo.
- Persistência do GUID e porta da interface selecionada, erro claro quando a configuração foi salva mas o nó não iniciou.
- Filtros de endereço comuns entre convite, discagem e reconexão: rejeição de APIPA, separação Hamachi/Radmin e anúncio apenas da porta real de escuta.
- Diagnóstico estruturado com modo, interface, endereço, porta, destino, handshake e ping.
- Testes de regressão e smoke opcional com adaptador Radmin real, dois perfis locais, reinício e mensagem persistida.

## [0.4.0] — 2026-10-31

- Arquitetura genérica de adaptadores virtuais Windows com Hamachi e Radmin VPN independentes.
- Seleção automática do adaptador da VPN escolhida ou seleção manual; exibição de interface, índice, IPv4 e estado.
- Escuta, origem TCP e convites vinculados ao endereço selecionado, com escopo assinado e ping autenticado.
- Preservação de identidade, contatos, histórico e modo libp2p nativo.

## [0.3.0] — 2026-10-31

- Integração opcional com Hamachi, monitoramento da interface e instalador por usuário em Go 64 bits.

## [0.2.0] — 2026-10-31

- Desktop Wails/React, histórico e contatos bbolt, convites assinados e diagnóstico de Circuit Relay v2.

## [0.1.0] — 2026-10-31

- Núcleo Go/libp2p, identidade Ed25519 persistente, DHT, bootstraps, texto assinado, limites de recursos e testes com três nós.
