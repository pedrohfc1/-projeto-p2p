# Plano técnico 0.6 — voz

Ponto de partida: comunidades 0.5, papéis assinados e canais `voice` sem mídia. A 0.6 não deve incluir vídeo, tela ou blockchain. Implementar em branch própria, preservando DM, perfis e protocolos anteriores. Entregar primeiro duas pessoas e depois aceitar pequenos grupos de 4–6; não anunciar escalabilidade ilimitada.

## Caminho de mídia

Primeira hipótese: WebRTC do WebView, Opus e tratamento de eco/ruído do engine. Validar capacidades reais do Edge WebView2 Windows e WebKitGTK Linux Mint antes de fixar desenho. Alternativa: [Pion WebRTC](https://github.com/pion/webrtc) no backend, que exige ainda avaliar captura/reprodução/Opus nativos; Pion sozinho não fornece toda a experiência de dispositivo de áudio. Fazer prova isolada e decidir com métricas, licenças e compatibilidade documentadas.

Topologia inicial mesh limitada a 4–6 usuários, com orçamento explícito de upload/CPU. Radmin pode facilitar alcance IP, mas não comprova passagem UDP. Testar ICE nas duas redes e caminho nativo; STUN/TURN somente configurados e informados ao usuário, sem servidor oculto obrigatório. Circuit Relay de streams TCP não deve ser apresentado como transporte WebRTC UDP. Encaminhamento cooperativo/SFU ficará para outra avaliação com permissões e custos próprios.

## Sinalização e autorização

Novo protocolo versionado de sinalização libp2p, separado de texto: sessão aleatória efêmera, comunidade/canal, Peer IDs dos participantes, nonce, sequência, expiração curta e assinatura de SDP/fingerprint DTLS e candidatos. Vincular resposta à sessão e chave autenticada; rejeitar replay, troca de canal, candidatos excessivos e remetente que não é membro ativo. Não confiar em papel enviado pelo frontend.

Entrada/saída exige política de autorização atual do grupo; projetar lease curto emitido pelo dono com expiração e política explícita se ele cair. Na primeira entrega, acesso novo depende do dono; expiração deve encerrar sessão sem renovação. Remoção, troca de canal e encerramento revogam sessão local e notificam peers; cada peer valida a autorização, não somente a UI. Não existe revogação retroativa de áudio já ouvido. Evitar persistir SDP/IPs em histórico público.

## Experiência e recursos

Microfone apenas após ação explícita e autorização do SO/WebView. Mostrar entrada, saída, mute/deafen, participantes e estado real da conexão. Seleção de saída depende de `setSinkId`/capacidade disponível; oferecer mensagem clara quando ausente. Indicador de fala baseado em nível local, sem gravar/transcrever. Volume com limites, push-to-talk opcional após estabilidade. Nenhum dispositivo aberto em repouso.

Sair fecha todos os tracks, AudioContexts, PeerConnections, timers e sinalização; crash/reconexão não pode criar duas sessões. Limitar salas simultâneas, participantes, bitrate, candidatos, tempo de ICE e tentativas. Não executar captura de tela nem pedir permissões de câmera nesta fase.

## Etapas e aceitação

1. Prova de capacidade Windows/Mint, captura/reprodução local, liberação de recursos e decisão WebView/Pion documentada.
2. Duas pessoas, mesma VPN e redes diferentes: sinalização autenticada, Opus/DTLS-SRTP, mute e saída segura.
3. Membership/remoção/lease/replay e reconexão com falhas de rede; nenhuma sessão sem autorização.
4. Mesh 4–6 com teto real; medir e reduzir limite se necessário.
5. Testes de 30 minutos entre dois PCs, depois 4–6, entrada/saída repetida, perda/jitter simulados, dono offline, troca de dispositivo, suspensão e cancelamento de permissão. Repetir no Mint nativo.

Registrar CPU e RSS da árvore inteira (Go + WebView), bitrate total/por participante, tempo até áudio, RTT, jitter, perda e concealment obtidos de `getStats`/RTCP quando disponíveis. Não inferir perda UDP de TCP. Comparar repouso com baseline 0.5, medir latência fim a fim com método descrito e manter DM responsiva. Verificar que sair retorna recursos ao repouso sem microfone ativo. Liberar somente após testes reais, revisão de segurança e novo instalador; publicação depende de autorização e dos gates existentes.
