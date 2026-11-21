# PeeTwoPee 0.7.0 — microfone e tela (beta privado)

## Microfone

A interface anterior desabilitava a seleção de entrada e os testes durante uma chamada. A enumeração antes da autorização podia não apresentar nomes/dispositivos, e um ID salvo de microfone removido era exigido com `exact`, impedindo captura pelo padrão do sistema. Falhas apareciam pouco claramente atrás da janela de configurações. O encerramento de uma faixa também encerrava toda a participação na sala.

Os controles agora permanecem utilizáveis durante a chamada. `Atualizar dispositivos` apenas enumera; `Autorizar e detectar microfones`, `Testar microfone local`, `Ouvir minha voz` e a entrada na sala são ações explícitas de captura. Permissão negada não dispara novas tentativas com outros dispositivos. Um ID removido pode usar o padrão do sistema após a ação explícita, com aviso. Há mensagens para negação, dispositivo ausente/ocupado e falha de inicialização. Mudanças de dispositivos atualizam as listas; reconectar um microfone encerrado exige clicar em Reconectar.

A troca usa `replaceTrack` e só encerra a captura anterior após concluir a substituição na mesma chamada/PC. Cancelamento, saída ou erro durante a troca libera a nova captura sem reabrir o microfone. O retorno/teste durante a chamada utiliza uma cópia da faixa existente, sem segunda abertura de hardware, sem alterar o mute enviado aos demais. O clone é exclusivamente local, não é adicionado ao transporte. Trocar dispositivos, fechar configurações, ocultar ou encerrar limpa o teste e os recursos; a voz normal é preservada ao ocultar. Use fones para evitar feedback.

### Consentimento Windows

A dependência Wails 2.17.0 usava `SetGlobalPermission(...Allow)` no frontend Windows. O build revisado troca somente essa política por `...Default`: o WebView2 segue o pedido/decisão do usuário e as permissões já existentes. Não se altera Privacidade do Windows, registro, firewall ou perfis do navegador. Não se remove nem ignora uma negação do sistema. Quando necessário, o usuário deve conferir manualmente **Configurações → Privacidade e segurança → Microfone**, inclusive acesso por aplicativos desktop. Listas vazias não são prova de ausência de hardware: autorize o teste explicitamente e atualize.

O build verifica versão/hash do arquivo original, copia o SDK para `build/permission-policy/wails`, aplica a mudança e restaura go.mod/go.sum ao terminar. O cache de módulos não é editado. `--permission-policy` deve informar `prompt-v1`; o empacotador rejeita um build sem essa marca, e o app Windows não inicia captura/GUI em builds comuns não verificados. A geração de bindings não inicializa a GUI. Isso não comprova interação física com o diálogo: o teste nativo de consentimento ainda está pendente.

## Compartilhar tela

1. Entre explicitamente em uma sala de voz autorizada.
2. Em **Tela na sala**, clique em **Compartilhar tela**. Escolha o monitor inteiro ou a janela exata no seletor do sistema.
3. Confira a **prévia local**. A captura da prévia não é enviada. Clique separadamente em **Iniciar compartilhamento**.
4. Os participantes da mesma sessão autorizada podem escolher **Assistir** e usar **Tela cheia**. Cada app assiste a uma fonte de cada vez para limitar decodificação e RAM.
5. Use **Parar compartilhamento**, encerre a fonte no seletor do sistema ou saia da sala. Não há retomada automática de captura após cancelamento ou troca de sessão.

Há indicação permanente de transmissão e botão Parar, mesmo com o painel recolhido. O áudio da sala continua em seu protocolo próprio. Não se captura áudio do sistema nem câmera. Nenhum vídeo é gravado ou armazenado. O título da fonte é exibido apenas na prévia local, sem registro em logs ou envio como metadado de rede. A seleção usa `getDisplayMedia`, com troca automática de superfície excluída; não há seleção silenciosa de outra janela ou monitor.

Ao ocultar/fechar para a bandeja, a captura da interface é encerrada e o backend interrompe a sessão de tela antes de ocultar. O heartbeat também verifica o estado minimizado da janela nativa, mesmo se flags de áudio mantiverem o WebView ativo. A voz não é encerrada. Se a interface parar de renovar a visibilidade, o backend deixa de transmitir em até três segundos. Há ainda prazos de escrita/in-flight; não se promete cancelamento de bytes que já chegaram ao destinatário. A retomada abre apenas a ponte de recepção: compartilhar novamente exige selecionar a fonte e iniciar explicitamente.

## Arquitetura e limites

- Captura e codificação WebRTC no WebView; ponte local WebRTC/Pion apenas de vídeo, com codec VP8. O runtime usa aceleração disponível para esse codec; a implementação do encoder é mostrada quando verificável. VP8 pode usar CPU. Não se promete uso de GPU/H.264 nem suporte 1080p/60 nesta versão.
- Pela internet/EasyTier, vídeo VP8 **comprimido** e empacotado em frames completos percorre `/peetwopee/screen-media/1.0.0`, em streams libp2p autenticados/criptografados, separados da voz e do chat. Não se supõe que UDP direto entre casas funcione. O teste disponível comprova essa camada sobre o EasyTier real.
- A autorização usa a identidade, comunidade, canal e sessões atuais da sala de voz, com autorização assinada pelo proprietário. Um membro cadastrado que não está na chamada não pode receber a tela. Expiração/revogação conhecida e saída removem os streams. A janela offline de voz continua limitada a 15 minutos; nenhuma autorização é inventada para vídeo.
- Inicial: até 1280×720 e 30 FPS. Banda total configurável de 256–3000 kbps, padrão 1500 kbps, com 1–5 espectadores. O limite total é dividido entre os espectadores admitidos. O algoritmo reduz bitrate/FPS conforme descarte, RTT e limitação de CPU; pode baixar captura para 960×540/20 FPS. Preferência de visualização é local.
- O backend verifica dimensões VP8 de keyframes, tamanho comprimido (384 KiB/frame), até 512 pacotes/frame e limites de taxa. Um quadro incompleto, fora de ordem ou atrasado é descartado; referências antigas não são reapresentadas, e um keyframe é solicitado com limite de frequência.
- Fila de um frame por stream e janela de até três frames aguardando ACK cumulativo. ACK mede entrega até o backend remoto, não renderização. Sem confirmação por um segundo, a sessão se reconecta. A taxa de controle também é limitada. Não há crescimento ilimitado de filas.
- Escrita com prazo de 200 ms, descarte de fila após 150 ms e de atraso relativo recebido acima de 200 ms; montagem local limitada a 250 ms. Relógios entre casas não são usados como medida absoluta de latência. O transporte TCP continua sujeito a retransmissão e bloqueio da conexão; os limites reduzem acúmulo, sem garantir latência em qualquer rede.
- Vídeo pausa quando a voz está reconectando ou sua fila ultrapassa 40 ms. Isso prioriza a chamada, mas não é uma garantia de desempenho em hardware/rede insuficientes. A carga de captura/encoder/WebView precisa de medição física.

A política HTTP também permite `display-capture=(self)` para o pedido explícito do seletor; câmera e geolocalização continuam bloqueadas. Isso habilita a API na origem local, sem conceder consentimento.

## Testes disponíveis e pendentes

`PEETWOPEE-v0.7.0-VALIDACAO.json` contém resultados e medidas com escopo. Os testes automatizados abrangem negação sem repetição, dispositivo salvo removido, enumeração sem captura, retorno local, cancelamento de handover, fonte de tela encerrada/pedido tardio, bitrate, limites de frames/resolução/ACK/fanout, estado nativo minimizado, permissões e limpeza.

O teste de integração usa duas instâncias do núcleo de produção e Pion no mesmo processo de teste, além de um terceiro membro para teste de rejeição, com **EasyTier real** e vídeo gerado a 1280×720/30. Os frames recebidos são reconstruídos e decodificados por FFmpeg, utilizado apenas no teste, sem ser instalado ou incluído no app. Também há Opus bidirecional e chat simultâneos, prévia sem envio, interrupção/reconexão, expiração de visibilidade e encerramento de vídeo sem sair da voz. Não é uma captura física de monitor ou microfone.

CPU/RAM dessa medição são do processo Go de teste, incluindo núcleos e pontes Pion simuladas. Não incluem WebView2, captura, encoder de hardware/software ou os processos auxiliares. Latência medida vai do envio de RTP codificado à montagem do frame remoto, sem aquisição, codificação nativa ou exibição. Não usar essas medidas como consumo total ou latência entre casas.

Permanecem pendentes: microfones/saídas/USB reais, diálogo de consentimento e seletor de monitor/janela no WebView2, prévia e tela cheia no desktop real, voz durante minimizar/restaurar nativo, medição de GPU/CPU/RAM da árvore completa, duas casas/CGNAT, falha física de relay e Linux desktop/portal. Compilar Windows/Linux CLI não comprova todos esses ambientes. Não se automatiza nem concede permissões do sistema para contornar os testes pendentes.

## Procedimento físico Windows

- Atualize os PCs com o beta, preservando `%APPDATA%/ProjetoP2P`, Peer ID, contatos e histórico. Use o EasyTier integrado já configurado e uma comunidade com autorização válida.
- Abra Configurações → Áudio; atualize os dispositivos, autorize explicitamente a captura se desejar, selecione entrada/saída e teste nível/retorno com fones. Negue o pedido uma vez e confirme a mensagem; altere permissões manualmente e repita. Desconecte/reconecte o USB e confirme que a sala não é encerrada por isso.
- Em chamada, troque microfone, teste retorno e feche configurações. Confirme áudio nos dois sentidos e ausência de retorno enviado ao outro PC. Minimize/restaure e confirme que a voz continua.
- Selecione primeiro uma janela de teste sem dados privados, confira a prévia e confirme que o outro PC ainda não recebe vídeo. Inicie, assista em tela cheia e continue conversando. Repita selecionando um monitor por escolha explícita.
- Pare, feche a fonte, saia da sala e minimize para a bandeja: o outro PC deve deixar de receber a tela. Restaurar não deve reiniciar captura. Interrompa a rede brevemente, restaure e confirme recuperação de voz e da sessão autorizada; após parar/minimizar, selecione novamente para transmitir.
- Compare CPU/RAM/rede com voz apenas e voz+vídeo usando a árvore inteira do app (WebView/helper). Confira encoder reportado e FPS real; reduza a banda se a voz degradar. Um adaptador ou presença em sala não comprova mídia funcionando.

## Build e distribuição

No Windows: `npm --prefix frontend ci`, testes/build, `go test ./...`, `go vet ./...`, `scripts/desktop.ps1` (ou `scripts/build-desktop.ps1 -Output PeeTwoPee-v0.7.0.exe`), depois `scripts/package-windows.ps1 -DesktopExecutable build/bin/PeeTwoPee-v0.7.0.exe`. O script de desktop garante consentimento e restaura o manifesto de dependências. Não usar `wails build` puro para distribuir o app Windows.

Os testes de integração opcionais usam `P2P_ETCAT_TEST_DIR` apontando para o auxiliar auditado e `P2P_SCREEN_IVF` apontando para um IVF gerado, por exemplo com FFmpeg `testsrc2`, VP8/libvpx, 1280×720 a 30 FPS. Execute `go test ./internal/desktop -run TestRealScreenAndVoiceOverIntegratedTransport -count=1 -v -timeout 4m`. Nunca use perfis reais para esses testes.

Somente beta privado, sem publicação automática. O helper EasyTier auditado, licenças e fontes correspondentes continuam incluídos. Não há Npcap/Packet.dll, novos drivers, mudanças de firewall, TUN, rotas ou DNS. Alertas Rust não resolvidos e a quarentena do DHT legado continuam bloqueando release pública. Identidade e protocolos de voz/chat existentes são preservados.

Referências de API: [getDisplayMedia e exigências de consentimento](https://developer.mozilla.org/en-US/docs/Web/API/MediaDevices/getDisplayMedia), [WebView2 ScreenCaptureStarting](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/overview-features-apis). São referências, não evidência de teste físico no computador do usuário.
