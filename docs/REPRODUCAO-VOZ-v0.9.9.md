# Reprodução de voz — beta privada 0.9.9

## Caminho identificado e grau de certeza
Na 0.9.8, ontrack da voz conectava as faixas RTC somente a MediaStreamAudioSourceNode, ao mixer e a um MediaStreamAudioDestinationNode. O elemento de saída recebia o stream gerado pelo Web Audio, nunca a faixa WebRTC diretamente, e era criado fora do DOM. Retomar somente AudioContext não inicializava explicitamente o consumidor RTC original.

A seleção de aplicativo WASAPI altera apenas PID/estado React. O evento adicional é marcar a opção de áudio: SystemAudio.toggle chama prepare mesmo sem publicação de vídeo. ScreenSoundBridge.connect cria uma ponte, executa setRemoteDescription e seu ontrack associa a faixa RTC diretamente ao elemento de áudio montado; play() configura a saída e inicia esse consumidor. Captura WASAPI nativa continua dependente da publicação de vídeo; não é necessária para esse caminho de preparação.

Isso identifica a diferença de inicialização no código e explica a dependência indireta observada. A atribuição ao renderer do WebView2 afetado ainda precisa de reprodução física: Edge headless não pôde iniciar neste ambiente (processo encerrado/pipe Crashpad). Não foi observado aqui o áudio dos dois computadores antes/depois do checkbox, portanto não se declara causa física comprovada.

O Chromium mantém estados de reprodução próprios para seu renderer WebRTC, separados do processamento Web Audio. Referências primárias: [WebRtcAudioRenderer](https://raw.githubusercontent.com/chromium/chromium/main/third_party/blink/renderer/modules/webrtc/webrtc_audio_renderer.cc), [WebAudioMediaStreamAudioSink](https://raw.githubusercontent.com/chromium/chromium/main/third_party/blink/renderer/modules/mediastream/webaudio_media_stream_audio_sink.cc), [política de autoplay](https://developer.chrome.com/blog/web-audio-autoplay). A aplicação dessa diferença ao sintoma é uma inferência do fluxo local e dessa implementação; não uma medição do engine instalado no computador do usuário.

## Correção isolada
Cada slot remoto possui agora um consumidor RTC próprio, montado, permanentemente mutado e com volume zero. Ele recebe a faixa exata antes de criar o source Web Audio. Não há caminho audível bruto: ganhos individuais, mute local, ensurdecer, saída e volume continuam exclusivamente no mixer/player da chamada. A saída da chamada também é um elemento persistente no VoiceProvider, independente da árvore do compartilhamento. Entrar ativa o contexto/player pelo clique; trilha recebida ativa seu renderer sem abrir WASAPI.

Rebind remove consumidores antigos antes de substituir tracks. Slots e contexto têm limpeza idempotente, sem parar faixas não pertencentes ao controlador nem criar contextos por consumidor. Retentativa e restauração retomam consumidores RTC e mixer; falhas reais de play ficam no aviso existente. A seleção de saída é aplicada também ao valor Padrão para não reutilizar sink de uma chamada anterior. O áudio da tela, sua automação e seus arquivos não foram modificados.

## Diagnóstico local, sem conteúdo
A cada três segundos, o PC atual informa recepção e emissão RTP por slot, amostras emitidas pelo jitter buffer, amostras decodificadas/concealment quando expostas pelo runtime e estado por participante. Frames Opus ficam null quando a API não os fornece; não são inventados a partir de pacotes.

Um AudioWorklet transparente mede somente quantidades de amostras/quanta encaminhadas ao stream de saída, com contagem de amostras não silenciosas. Não transmite PCM ao JavaScript, não guarda áudio e só publica contadores a cada três segundos. Se indisponível, a cadeia de áudio original continua e o contador fica indisponível. Estados do AudioContext, tracks, elementos, mute, volume, correspondência da saída escolhida e motivos de bloqueio são registrados sem IDs brutos nem nomes de aplicativos.

Amostras na cadeia não provam som físico: a saída pode estar pausada, mutada, com volume zero ou com problema no dispositivo. Estes estados são separados. O WebView não mede o som entregue ao ouvido. O diagnóstico usa o PC corrente, inclusive após rebind, sem manter timers presos ao PC anterior.

## Teste físico pendente
1. Em dois PCs, entrar por clique e autorizar microfone pessoalmente. Deixar compartilhamento fechado e checkbox de áudio desligado. Conferir voz bidirecional.
2. Comparar nos dois consoles eventos rtc-renderer-playing, contexto running, tracks live/unmuted, rede RX por participante e contadores de saída. Verificar outputGraph sem confundir captura com reprodução.
3. Abrir detalhes, escolher aplicativo e ligar/desligar opção de áudio: voz deve permanecer independente. A seleção não autoriza captura antes da publicação de vídeo.
4. Trocar saída, testar Padrão e voltar à saída salva; testar volumes, mute e ensurdecer. Desconectar/restaurar rede sem acumular consumidores.
5. Repetir com três participantes e verificar vídeo/som automático autorizado, navegação, minimização e parada de captura. Nenhuma permissão é concedida automaticamente.

Testes de processo usam mídia gerada e Pion, não alto-falantes físicos. Testes de controlador/worklet usam mocks e não comprovam autoplay no WebView2. Instalação/atualização gráfica completa, WASAPI real, eco e dois PCs continuam pendentes. Beta privado; sem publicação pública. Quarentena DHT e bloqueios existentes de distribuição pública preservados.
