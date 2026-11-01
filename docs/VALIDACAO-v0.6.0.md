# Validação e segurança — beta privado 0.6.0

Executado em Windows AMD64, em 1 de novembro de 2026. Código em feature/v0.6.0-voice; beta 0.5.2 preservado. Não houve push, tag ou publicação.

## Resultados

- Frontend: npm ci, 10 testes, TypeScript e build Vite aprovados. Os testes de renderização verificam escape de conteúdo; não são testes de áudio físico.
- Go: go test ./... e go vet ./... aprovados. Testes opcionais são executados separadamente com o auxiliar real; não confundir skip da suíte padrão com integração realizada.
- Comunidades: políticas de convites, legado de uso único, expiração, adulteração, revogação; exatamente 5 ingressos aceitos em 20 concorrentes contra cota 5. Registro autenticado com 3/5/10/20/100 membros e recusa do 101º. Limites de 50 canais de texto e 20 de voz, edição de sala assinada. Migração preserva configurações, histórico e índice de prefixos de schema 2; downgrade destrutivo recusado.
- Texto entre membros: dois nós continuam escrevendo sem o nó proprietário após apresentações autenticadas. Assinatura adulterada recusada; repetição não duplica. O caminho legado ordenado pelo proprietário segue coberto por regressão própria.
- Três processos reais sobre EasyTier integrado: ingresso, papéis, canais, diário de texto, reinício, recuperação, deduplicação, revogação e chat privado preservados; mesmo PC. Cenário Python aprovado em cerca de 60 s.
- UDP real do auxiliar: eco transforma o payload, valida sequência e drena aquecimento para evitar falso positivo por loop local. 50 datagramas de 160 bytes, 0 perdidos, RTT médio 7,188 ms e jitter de RTT 1,177 ms nesta amostra. Não é latência boca-ouvido nem teste CGNAT.
- Pion/DTLS-SRTP sobre UDP EasyTier: 50 quadros Opus DTX sintéticos. Negociação ICE+DTLS de aproximadamente 1,805 s no teste de transporte isolado.
- Dois processos independentes do backend de produção: sinalização libp2p autenticada e ponte de mídia nos dois sentidos; 50/50 quadros por direção. Tempo da negociação remota (após preparar as ofertas locais): 8,456 s. Jitter RTP medido de 0,470 e 0,475 ms; perda estimada 0. RTT RTCP ainda não disponível na amostra curta, representado como null.
- Serviço real de voz: mute/deaf bloqueiam entrega de RTP, texto funciona durante chamada, tentativa de substituir sessão ativa/replay ou glare recusada, assinatura de autorização adulterada e identidade declarada falsa recusadas pelo stream autenticado. Remover membro fecha mídia. Derrubar os auxiliares de mídia, mantendo sinalização/chat, recuperou áudio com nova negociação em 24,692 s na amostra específica de segurança.
- Supervisor real: recuperação de auxiliar e credenciais reutilizáveis continuam funcionando. Amostra de um helper: RSS 23.453.696 bytes e CPU acumulada 0,172 s, não pico nem garantia em outro PC.
- CLI Windows/Linux AMD64 compilados; smoke CLI com três processos, bootstrap/fallback, queda, identidade e endereço recuperados aprovado. Regressão legacydht aprovada isoladamente; esse código não é ativado no beta distribuído.
- Desktop Windows compilado com Wails. WebView2 inicializou com perfil isolado. Inspeção visual automatizada da janela não foi concluída. Microfone, saída física, seleção efetiva de dispositivos e minimizar/navegar com áudio real exigem teste manual.
- Instalador: --verify-payload e installer-smoke aprovados, com desktop/helper/fontes/licenças/bootstrapper WebView2/desinstalador e hashes correspondentes. Isso não equivale a instalar/atualizar em máquina limpa. Setup e app não têm assinatura Authenticode de distribuição.
- Race detector: tentativa feita, mas o linker Windows recebeu Permission denied ao escrever binários instrumentados em diretório temporário. Não houve execução dos testes -race. Nenhuma proteção do Windows foi desativada. Desktop Linux e integração de áudio Linux não foram compilados/testados nesta máquina; somente CLI Linux.

O ensaio final de dois processos mediu RSS dos processos Go em aproximadamente 42,1 e 42,3 MiB; CPU de 4,58% e 2,74%, onde 100% equivale a um núcleo lógico. É uma janela curta com áudio DTX sintético, sem captura/codec de voz real. Não inclui WebView2 nem CPU/RAM dos helpers no total do aplicativo. As métricas TCP existentes não contam os datagramas de mídia; os contadores RTP separados mediram 750 bytes por direção (50 pacotes de 15 bytes), excluindo overhead. Não usar esse bitrate de silêncio sintético para dimensionar fala ou banda do relay. Dados estruturados em METRICAS-v0.6.0.json.

## Segurança e limites

A mídia é autenticada por associação entre Peer ID libp2p, lease do proprietário e SDP/fingerprint DTLS. Os processos Go são endpoints da ponte WebRTC local e remota. O proprietário coordena autorização, mas não é retransmissor obrigatório do áudio. Autorizações expiram em até 45 s; revogação conhecida interrompe imediatamente no próximo ciclo de guarda (500 ms, mais atualização da interface). Uma partição pode atrasar conhecimento da revogação até a expiração. Não há gravação, captura antes de Entrar/teste explícito, E2EE de grupo, consenso de texto nem remoção remota de cópias históricas.

No binário Go padrão, govulncheck retornou zero vulnerabilidades alcançáveis e zero em pacotes importados. Oito achados continuam em módulos exigidos, mas não alcançados. A DHT provider afetada por GO-2024-3218 está em quarentena, não foi corrigida upstream nem teve aviso suprimido. Consulte SEGURANCA-DHT-BETA.md. O warning operacional de keepalive TCP aparece no adaptador SOCKS porque o wrapper net.Conn não expõe SetKeepAlive ao libp2p; a conexão autenticada funciona, mas esse ajuste específico não pode ser aplicado pelo wrapper. O log não foi ocultado e não é o aviso Sybil da DHT.

O auxiliar d25540cadfb76697c53a44a9d981fc36e5e56eed09dffef0f4a86c6ff4674a32 é recompilado, sem TUN/capture/UPnP. A inspeção de imports PE normais, delay imports e referências ASCII/UTF-16 nos executáveis finais não encontrou Packet.dll/Npcap/Wintun/WinDivert. O desktop importa kernel32.dll; o helper importa bibliotecas normais do Windows/CRT. Isso foi verificado no executável, não apenas pela ausência de arquivos no instalador. Npcap não é redistribuído.

O Cargo.lock não mudou em relação ao auxiliar anterior; os achados Rust registrados em third_party/etcat/security-inventory.json permanecem (incluindo hickory/russh e crates sem manutenção). Não foi declarado que são inexploráveis. O scanner Go não analisa Rust. PRIVATE_BETA_ONLY e o gate de publicação continuam bloqueando a release pública. Fontes correspondentes LGPL completas, vendor, patches, Cargo.lock e notices acompanham o instalador; a licença MIT cobre somente o código original.

Limites: no máximo duas identidades ativas por sala, um peer remoto de mídia por processo, SDP de até 32 KiB, frames de sinalização de até 64 KiB, payload RTP de até 1200 bytes, portas não privilegiadas e ACL UDP limitada ao serviço de mídia. Controles de streams/conexões do libp2p e cotas de banco permanecem. Endpoint virtual ativo não comprova conectividade; só negociação e tráfego verificados o demonstram.

## Pendências de aceitação manual

Voz real com microfone/alto-falante em dois PCs Windows entre casas, CGNAT, troca de caminho/relay indisponível com alternativa, teste prolongado de retransmissão/head-of-line blocking, consumo total e atualização em instalação limpa. Conexão entre casas do chat foi informada pelo usuário na fase anterior; não é prova de voz. Não houve execução de v0.6.1, volume individual nem grupos de 3–6.

Roteiro: [BETA-v0.6.0.md](BETA-v0.6.0.md). Não publicar este beta no GitHub Release público enquanto os bloqueios permanecerem.
