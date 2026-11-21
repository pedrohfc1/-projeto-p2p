# PeeTwoPee 0.9.4 — console local (beta privada)

O botão de terminal no canto inferior direito abre e fecha o console. Ele inicia fechado. Arraste o cabeçalho para mover; use o canto inferior direito para redimensionar. O console permanece montado ao mudar de tela. Nenhuma chamada é iniciada por ele.

Clique na área dos eventos para usar PgUp/PgDown; a roda do mouse e a barra de rolagem também funcionam. Rolar para cima desliga Auto. Clique Auto para ir ao final e reativar o acompanhamento. Selecionar texto suspende a atualização visual enquanto houver seleção, preservando os nós DOM; a coleta continua. Ctrl+C pede confirmação antes de copiar a seleção. Copiar tudo pede confirmação e copia os eventos retidos do filtro escolhido. Pausar congela apenas a visualização. Limpar remove somente eventos de diagnóstico, nunca dados do perfil.

Filtros: Todos, Voz, Vídeo, Rede e Erros. Horários são locais com milissegundos. INFO/WARN/ERROR indicam o evento emitido pelo componente; ausência de um evento não comprova ausência de falha.

## Origem e limites

Backend: snapshots reais de voz/tela/rede a cada três segundos, transições de participantes por slot, estado do EasyTier e contadores libp2p. Eventos slog operacionais explicitamente permitidos (voz e conectividade de comunidades) são capturados; não há interceptação irrestrita de stdout/stderr. O caminho de mídia não espera pelo mutex do console: sob contenção, um evento de transporte pode ser descartado. Os contadores periódicos continuam disponíveis.

Frontend: captura de microfone, seleção de dispositivos, erros reais de aquisição, retorno local, ponte WebRTC/ICE, encoder Opus e contadores RTP a cada cinco segundos; captura/configuração de tela, estados da visualização e métricas do encoder/decoder/renderização a cada três segundos quando disponíveis. Os valores indisponíveis permanecem nulos/não informados. FPS solicitado não é FPS real. Não há um log por pacote.

Cada origem usa um buffer circular de até 2.000 eventos. Cada texto tem tamanho limitado. A API entrega até 250 eventos por consulta a cada três segundos; eventos sobrescritos são informados. A interface atualiza no máximo a cada 300 ms, somente se o buffer mudou, sem seleção, aberta e visível. Linhas antigas são memoizadas. Não há dependência nova, gravação de mídia, arquivo de log novo ou envio automático. O teste local de microfone continua exclusivamente local.

Não são coletados corpos de mensagens, amostras de áudio/vídeo, títulos de janelas compartilhadas, SDP ou credenciais. Peer IDs, IPv4, convites e padrões de tokens/chaves são ocultados; nomes de dispositivos e informações operacionais podem identificar o ambiente. Revise o conteúdo antes de compartilhar, mesmo após a confirmação. A ocultação é defensiva, não uma garantia de anonimato.

O console usa as sessões existentes; não modifica transporte, firewall, rotas, DNS ou permissões. A atualização preserva o perfil e a instalação transacional anterior. Nenhuma release pública.

## Roteiro manual em Windows

1. Abra o app: o console deve estar fechado. Entre numa chamada pelos controles usuais, com consentimento para microfone. Abra o terminal: confira eventos de captura/Opus e contadores reais crescendo a cada 3–5 s.
2. Arraste o cabeçalho e redimensione pelo canto. Navegue para outra conversa/comunidade; posição, filtro e pausa devem permanecer. O botão X fecha apenas o console.
3. Com eventos suficientes, use roda, barra, PgUp/PgDown. Ao subir, Auto desliga. Novos eventos continuam no buffer. Selecione texto com o mouse: as linhas selecionadas não devem se mover. Ctrl+C deve mostrar aviso antes de copiar. Teste cancelar.
4. Pause, aguarde métricas, retome e confira novos eventos. Teste filtros Voz/Vídeo/Rede/Erros, Copiar tudo e o aviso, e Limpar. Nenhum contato, mensagem ou identidade deve ser removido.
5. Inicie uma tela por seleção explícita, prévia e Iniciar. Compare resolução real, FPS e bitrate com o painel de mídia. Pare e reconecte pelos controles normais; confira motivos/estados registrados.
6. Minimize/restaure: voz deve continuar e tela seguir a política existente de interrupção; console deve continuar utilizável ao restaurar. Finalize a chamada: capture/encoder devem parar e os contadores de sessão deixar de crescer.

Não envie logs a terceiros sem revisão. A validação automatizada é local e utiliza mídia sintética; não comprova dispositivos físicos, testes entre casas, nem toda a interação nativa do console.

A cópia tenta primeiro a API de clipboard do WebView durante o gesto confirmado, e depois a API nativa dedicada (limite de 12 MiB). Se o sistema negar ambas, o console oferece o snapshot em um campo selecionável para cópia manual, sem indicar sucesso falso e sem alterar permissões.

Se o monitor ou a janela do aplicativo estiverem sendo compartilhados, o console visível também pode aparecer na imagem escolhida pelo usuário. Feche o console antes de mostrar dados de diagnóstico na transmissão. Isso não é exportação automática de logs pelo aplicativo.
