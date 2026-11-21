# Correção de áudio — 0.6.0, beta privado 2 (2026-11-04)

Instale o mesmo beta 2 nos dois computadores. Identidade, contatos, mensagens, comunidades e configurações existentes são preservados. O número-base continua 0.6.0; o nome do instalador identifica o beta 2. Não publicar Release pública.

## Correções

A versão anterior exigia UDP EasyTier entre processos e reutilizava o engine/interface da rede virtual na ponte local navegador↔Go. O frontend também chamava VoiceLeave quando a conexão WebRTC local entrava em failed; os timeouts ICE configurados (5 s + 12 s) são compatíveis com o abandono observado em cerca de vinte segundos. Essa relação foi identificada no código; não foi reproduzida com os dispositivos físicos do usuário.

O padrão agora transporta Opus mono de 20 ms em stream libp2p próprio, separado de chat e sinalização, reutilizando a conectividade EasyTier funcional. A identidade é autenticada pelo libp2p e a autorização de sala continua vinculada às sessões e à assinatura do proprietário. A ponte WebRTC local tem um engine independente, com candidatos locais utilizáveis pelo navegador. O caminho WebRTC/UDP entre peers permanece no código e nos testes para evolução, sem ser pré-requisito da nova mídia.

Presença e mídia têm estados separados: perder o stream não remove o usuário da sala; o iniciador reconecta automaticamente. A ponte local pode ser renegociada por VoiceRebind sem consumir outro ingresso, trocar sessão ou abrir uma nova captura. Revogação conhecida e expiração de autorização continuam encerrando a chamada por segurança; o proprietário precisa renovar a autorização de 45 s.

A fila de saída tem três quadros e descarta entradas com mais de 100 ms. Há jitter buffer de três quadros/20 ms, descarte por atraso relativo de transporte acima de 150 ms, limite de tamanho, validação da duração Opus, limite de taxa e deadlines de escrita/leitura. Mute/ensurdecer bloqueiam mídia no backend e limpam a fila de saída. O handshake preserva bytes de áudio recebidos junto ao JSON e os deadlines de mídia só passam a valer após o ACK.

TCP pode acumular atraso por retransmissão/head-of-line blocking. Os limites de aplicação não eliminam esse problema nem garantem uma latência absoluta de rede; o descarte relativo depende do atraso mínimo observado, sem pressupor relógios sincronizados. Uma escrita bloqueada por 200 ms encerra o stream para reconexão. O teste de backlog verifica descarte de quadros antigos e o teste de bloqueio verifica encerramento por deadline. Não foram medidas retransmissões sob CGNAT real ou perda induzida no enlace externo.

## Ouvir minha voz

Abra Configurações → Configurar e testar áudio. Selecione microfone/saída e clique em Ouvir minha voz. O botão muda para Parar de ouvir; o slider ajusta o volume e o medidor mostra o nível. Use fones para evitar feedback. A saída selecionada exige setSinkId do WebView; se indisponível, escolha o padrão do sistema.

Esse caminho é local, sem backend P2P, internet, envio ou gravação. Fechar as configurações, trocar dispositivo, entrar em chamada ou encerrar a janela interrompe o retorno e libera recursos. Pedidos de captura ainda pendentes são cancelados logicamente e suas tracks são encerradas quando chegam. O teste curto anterior continua disponível. Captura depende da ação explícita e das permissões do sistema.

## Diagnóstico e validação

A barra mostra presença e estado de mídia, transporte libp2p-stream, pacotes, fila e descartes. Logs do backend mostram estado e contadores a cada cinco segundos; console do frontend mostra captura, codec, pacotes e amostras decodificadas/reprodução quando informados pelo WebView. Não se registra áudio, SDP, credenciais ou conteúdo de mensagens.

Testes automatizados: 12 testes frontend (incluindo saída/volume/limpeza e cancelamento de captura local pendente), build TypeScript/Vite; suíte Go e vet; autenticação, assinatura falsa, replay/glare, mute/deaf, reconexão, reparação da ponte local mantendo a sessão, revogação e chat simultâneo. Regressão com três processos preserva comunidades/convites/chat EasyTier. O teste de cinco minutos usa dois processos de produção, frames Opus reais codificados de tom PCM e decodificação FFmpeg com RMS não nulo nos dois lados; não utiliza microfone físico. FFmpeg é ferramenta de teste instalada no ambiente e não é incluída nem necessária em runtime.

O script tests/audio_browser_smoke.cjs prepara testes de monitor/captura/codec/reprodução pelo navegador com PCM gerado localmente. A tentativa de executá-lo ficou bloqueada pela inicialização do Edge no ambiente de testes; não foi declarada aprovada. Portanto retorno audível a uma pessoa, microfone/saída físicos, WebView2 real, instalação limpa e voz entre casas continuam pendentes. As fontes e o instalador não demonstram esses testes físicos.

Execute os dois aplicativos, escolha EasyTier integrado, entre na mesma comunidade/sala e mantenha o proprietário online. Confira primeiro Ouvir minha voz em cada PC, depois conecte a chamada, alterne mute/deaf e troque mensagens. Anote estado/contadores caso não haja som; não envie gravações, credenciais ou bancos. Não alteramos firewall, roteador, VPN, DNS ou rotas do sistema.

A quarentena da DHT legada e o bloqueio público por avisos Rust existentes foram mantidos. O auxiliar EasyTier não mudou: sem Npcap/Packet.dll/TUN; fontes correspondentes e licenças seguem no setup.

## Resultados medidos neste ambiente

Ensaio final: 15.000/15.000 quadros por direção, 300 s nominais de áudio; duração total 323,45 s incluindo preparação/encerramento. Decodificação gerou 14.396.160 amostras PCM por lado, RMS 0,08744. Conexão remota: 3,205 s. Latência entre os endpoints de teste: média 38,918 ms, máximo 7.365,497 ms sob carga concorrente. Essa medição inclui WebRTC local, agendamento e I/O do gravador sintético do teste; não demonstra que o pico veio do TCP nem mede boca-ouvido. O outlier impede prometer latência máxima baixa. Contadores do stream: 15.000 pacotes de entrada/saída, 1.615.811 bytes RTP por direção, sem descarte nessa amostra. Jitter/perda/RTT não medidos ficam null.

CPU dos processos Go: 1,92% e 2,74% de um núcleo lógico; RSS aproximadamente 31,7 e 33,4 MiB. Não inclui WebView2/auxiliares no total. Regressão UDP separada continua aprovada. Testes de mídia local reparada, mute/deaf, reconexão e revogação aprovados; três processos com comunidades/chat integrado também aprovados. Go test/vet, frontend 12 testes/build e scanner Go aprovados. Inspeção PE continua sem referências a Packet.dll/Npcap/Wintun/WinDivert; nenhum driver novo foi incluído. Dados em AUDIO-BETA2-METRICAS.json.

## Reproduzir o ensaio de cinco minutos

Com Go, frontend já compilado, auxiliar verificado e FFmpeg de teste no PATH, execute no PowerShell na raiz do repositório:

```powershell
go build -o data/webui-audio.exe ./tests/webui
python tests/generate_audio_fixture.py data/audio-tone.frames
$env:P2P_ETCAT_TEST_DIR=(Resolve-Path build/bin/easytier).Path
$env:P2P_WEBUI_TEST_BIN=(Resolve-Path data/webui-audio.exe).Path
$env:P2P_VOICE_OPUS_FIXTURE=(Resolve-Path data/audio-tone.frames).Path
$env:P2P_VOICE_SOAK_SECONDS='300'
go test ./internal/desktop -run TestRealVoiceTwoBackendProcesses -count=1 -timeout 8m -v
```

Essas variáveis pertencem somente ao terminal de teste. Não inserir configurações privadas, bancos ou fixtures geradas no Git. O teste usa perfis isolados, não altera os perfis reais e não captura o microfone.
