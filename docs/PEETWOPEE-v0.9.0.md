# PeeTwoPee 0.9.0 — experiência de uso e transmissões, beta privado

A criptografia Noise protege os streams entre peers; não foi adicionada E2EE de grupo independente do transporte.

A versão existente era 0.8.2; esta entrega usa 0.9.0 para evitar regressão de versão. Não há publicação pública. Preserve %APPDATA%/ProjetoP2P e atualize todos os participantes para as novas ações do chat.

## Transmissões e voz
O selo vermelho AO VIVO aparece para transmissores autorizados na sala atual. Clique para abrir a prévia identificada e use Assistir; anúncios não abrem vídeos automaticamente. Há modo compacto, ampliado, tela cheia e Parar de assistir. Navegar no chat não encerra o player. Fim da transmissão, saída ou revogação encerra o acesso; durante reconexão o quadro antigo fica oculto. Um player/decodificador por aplicativo, vários transmissores independentes, até cinco espectadores por transmissor.

A captura continua exigindo seleção explícita de monitor/janela, prévia e Iniciar. Começa em 720p/30, bitrate adaptativo limitado a 256 kbit/s–3 Mbit/s, fila de um quadro e janela de três ACKs. Áudio tem prioridade quando sua fila cresce ou reconecta. Sem espectadores o sender é desacoplado da track, suspendendo codificação/envio sem reabrir o seletor; a fonte autorizada permanece selecionada. Minimizar/ocultar encerra a captura pela política existente; voz permanece. Câmera continua desabilitada.

Presença de mute/ensurdecer é assinada, vinculada à sessão de voz autorizada e expira em seis segundos. É declaração do participante, não prova do estado físico do microfone. Volume individual persistente e painel inferior esquerdo preservados.

## Chat assinado
Responder, editar/excluir mensagens próprias, seis reações com alternância, não lidas e digitação usam o libp2p/EasyTier existente, com Ed25519/Peer ID, contexto, raiz e época da comunidade. Referências privadas incluem autor e ID, impedindo colisões de IDs entre remetentes. Apenas membros autorizados sincronizam comunidades; a assinatura não é substituída por confiança em IP.

O histórico original é imutável: excluir cria uma marca lógica e NÃO apaga cópias antigas, backups ou o texto original do diário. Edição também é uma projeção assinada. Clientes anteriores continuam com chat original mas não apresentam as novas ações. Sem dono, ações usam a autorização local válida e pares alcançáveis; ingressos e mudanças de permissões continuam exigindo autorização do dono. Atividades de época antiga não são importadas após mudança de autorização; não há promessa de convergência entre épocas distintas.

Sincronização incremental com cursores persistentes, recuperação após reinício e deduplicação. Limites: 5.000 atividades por contexto, 32 MiB combinados de diário/projeções/índices, projeção 128 KiB, últimas 100 respostas e projeção das últimas 300 mensagens consultadas. Mensagens antigas além dessa janela não recebem todas as projeções na interface. Atividades duráveis são aceitas por até 30 dias; a retomada não promete histórico de ações indefinidamente. Digitação expira em seis segundos e não é gravada; pode ser desativada nas configurações.

## Notificações e privacidade
Ative notificações nas configurações: padrão desligado, conteúdo oculto. Windows usa a API nativa da bandeja, respeitando política do sistema; não garante histórico na Central de Ações. Alertas para contatos conhecidos, menções exatas @{PEER_ID} e texto de convite reconhecido, sem abrir/aceitar convites. Prévia de conteúdo é opt-in. Menus permitem silenciar comunidade/canal/contato; não lidas continuam contabilizadas. Contexto visível e app em primeiro plano não notifica; deduplicação persistente e fila limitada. O sistema operacional pode suprimir alertas.

## Validação e teste manual pendente
Passaram 38 testes frontend/SSR, TypeScript/Vite, testes Go e vet, três processos reais locais com EasyTier (chat original, atividades assinadas, permissões, retomada offline e reinício), comunidades e mídia gerada VP8/Opus simultânea com reconexão/revogação/limpeza e dois transmissores simultâneos. Isso NÃO comprova captura física, UI nativa ou redes distintas.

Em dois PCs Windows, instale a beta privada sem apagar perfis, conecte pelo convite existente, entre na sala e inicie a transmissão explicitamente. No outro PC confirme que AO VIVO não abre sozinho, abra prévia/Assistir, navegue pelo chat, teste compacto/tela cheia/parada. Publique dos dois PCs e alterne a fonte assistida. Confira mute, volume e voz durante vídeo, desconecte/reconecte e revogue acesso. Teste ações do chat, retomada após reinício, notificações opt-in sem conteúdo e silenciamento. Meça no Gerenciador de Tarefas app + todos os WebView2 + auxiliar; logs automatizados medem somente backend Go.

Pendentes: dois computadores/redes distintas/CGNAT, microfone/tela físicos, múltiplos transmissores com hardware real, seleção e suspensão do encoder no WebView2, notificações do Windows, acessibilidade/layout/resoluções, minimizar/restaurar nativo e instalação/atualização completa. Métricas locais não garantem consumo em hardware real. Não há gravação de mídia. TCP pode acrescentar latência em perda/congestionamento.

## Distribuição
Instalador privado sem assinatura de editor; payload validado não equivale a instalação completa. Auxiliar EasyTier auditado, licenças e fontes LGPL completas acompanham a entrega. Sem Packet.dll/Npcap, TUN obrigatório, câmera ou alteração de firewall/rotas/DNS. Bloqueios Rust e quarentena do DHT legado permanecem; nenhuma release pública autorizada.
