# PeeTwoPee 0.9.1 — hotfix privado de áudio e tela

## Escopo e limite da entrega
Correções no áudio, estado do player e transporte VP8 existente; chat, comunidades, EasyTier, identidade e perfil preservados. Não há release pública. Atualize os dois PCs para0.9.1; vídeo1080p não é compatível com o limite de720p dos clientes anteriores. Voz e chat continuam nos protocolos existentes.

**A meta completa de captura de jogos/desktop1080p60 por H.264 com NVENC/AMF/Quick Sync NÃO está implementada nesta entrega.** O vídeo continua VP8 pelo WebView, com a implementação de encoder/decoder informada quando o runtime fornece estatísticas. Há perfis1080p/60 solicitados e ensaios de transporte de mídia gerada, sem certificação de captura, encoder ou renderização reais nessa taxa. Não confunda suporte de resolução do transporte com desempenho ponta a ponta.

## Comparação e voz
O último commit local é c4e90b2 (beta0.6.0). Versões posteriores estão no diretório de trabalho sem commits intermediários. Foram comparados esse commit e os arquivos dos pacotes-fonte privados0.8.2/0.9.0; não existe histórico suficiente para apontar um commit intermediário exato ou executar bisect dessas alterações.

No caminho antigo de dois participantes, uma track remota era ligada diretamente ao áudio. O mixer posterior usava slot0 como fallback quando o WebView não fornecia stream ID: várias tracks sem ID podiam sobrescrever o mesmo slot com uma track não utilizada. Agora usa o transceiver negociado e rejeita slots desconhecidos. O roster inicial é aplicado imediatamente. Testes reproduzem o mapeamento de cinco tracks sem ID.

O AudioContext de reprodução é criado/retomado no clique explícito de ingresso, antes das esperas de captura e IPC. Captura com ganho e mixer compartilham o contexto da chamada, sem ligação entre microfone e saída local. Trocar microfone fecha somente o ganho/track anterior; recuperar a ponte mantém o contexto e não o fecha enquanto a captura ainda o utiliza. Contexto suspenso não deixa a entrada pendente indefinidamente; a interface continua oferecendo ativação de reprodução quando necessária. Nenhuma permissão é concedida automaticamente. Transporte Opus20ms/libp2p/EasyTier preservado, vídeo em sessão/protocolo separado.

Essas falhas foram identificadas no código/reproduzidas em testes de lógica; não foi reproduzido o defeito físico específico dos PCs do usuário. Não se declara audibilidade física comprovada.

## Player e reconexão
Perda de stream de vídeo deixou de ser tratada como anúncio de Parar. O estado distingue anúncio confirmado, transporte indisponível, fonte encerrada e membro removido da autorização assinada. A intenção de assistir e o componente do player permanecem durante reconexões; o quadro antigo fica oculto e aparece Reconectando. Ao recuperar a ponte local, a seleção autorizada é reaplicada. Saída/revogação real ou Parar encerra o acesso. Canal/chat não desmontam o painel global da chamada.

Não foi adicionado evento de foco/blur que minimize a janela ou ative a bandeja. Captura continua parando quando a janela é realmente ocultada/minimizada, conforme a proteção existente; voz permanece. Assistir pode ser retomado na restauração sem abrir nova fonte automaticamente. Compartilhar novamente exige seleção explícita, prévia e Iniciar.

## FPS, qualidade e bitrate
Foram encontrados quatro limites: captura720p30, guarda de resolução720p, guarda de45 quadros/s e teto3.000.000bit/s no backend/UI. Foram substituídos por perfis Automático(720p30),720p30,720p60,1080p30 e1080p60.60FPS é experimental. Faixa configurável2–12Mbps de upload TOTAL, dividido entre espectadores; limitações menores do protocolo anterior continuam aceitas para compatibilidade. Padrões conservadores3/6/10Mbps conforme perfil. Não há duplicação de quadros na implementação.

Foi removida a fórmula que derrubava arbitrariamente FPS em função de RTT(2700/RTT). Uma perda isolada deixou de reduzir permanentemente a captura a540p. O controle de bitrate continua reduzindo sob descartes e recuperando gradualmente; CPU/congestionamento sustentados reduzem resolução/FPS, com prioridade para áudio.

A fila de quadros completos passou de1 para2, absorvendo pequenos agrupamentos do escalonador sem crescimento ilimitado. A janela de ACK acompanha o RTT, com máximo16 quadros E512KiB em voo. Validade150ms antes de envio, descarte de frames atrasados no receptor e prazo200ms de escrita preservados; referência perdida exige novo keyframe. Houve correção na contagem: ACKs não são mais contados como quadros de vídeo. A guarda de entrada admite até90 quadros/s e2MB/s por stream, mantendo quadros limitados a384KiB e superfície máxima1920×1080.

Transmissor sem espectadores desacopla a track do sender, mantendo captura/fonte autorizada e estado de publicação, e retoma a codificação quando houver assinante. Inicializar/parar vídeo não fecha o contexto, microfone ou ponte de áudio.

## Telemetria e transporte
A interface calcula FPS de encoder/decoder e bitrate por diferenças de contadores, não pelo valor configurado. Captura usa framesPerSecond da fonte quando fornecido; renderização usa requestVideoFrameCallback. Contadores ausentes ficam n/d. Envio/recepção são entregas agregadas do backend; fanout pode ultrapassar o FPS de uma única fonte sem significar quadros fabricados. São mostrados implementação/eficiência do encoder e decoder quando o runtime informa, RTT de ACK e descartes por fila, janela, atraso/autorização, prioridade de áudio/banda e espera de keyframe.

Perda RTP da ponte local é separada de perda no túnel: TCP não fornece aqui uma medida de perda UDP da rede. Não há percentagem inventada. O transporte continua TCP/libp2p sobre SOCKS EasyTier e sofre head-of-line blocking. Foram medidos payload/latência local e testadas filas/janelas/prazos, mas não há validação de desempenho em WAN com perda. O limite de escrita não cancela retransmissões já realizadas pelo kernel. Os perfis altos não são promessa de baixa latência na internet. Não foi adicionada associação UDP, novo relay, TUN ou alteração de firewall/rotas/DNS.

## Testes disponíveis e limitações
Consulte VALIDACAO.json para resultados e métricas. Um ensaio anterior passou o trecho de600s, mas falhou depois ao reproduzir a fixture antes da confirmação de assinatura na reconexão; o teste foi corrigido para aguardar essa confirmação e a prontidão dos dois sentidos de voz. A fixture não responde a PLI como o encoder real. Duas repetições curtas passaram; o relatório registra também a repetição completa. Nenhuma falha foi ocultada. Testes reais locais de backend usam libp2p criptografado/EasyTier, VP8 já codificado e Opus gerado. Clipe gerado1080p60 possui quadros distintos no ensaio curto; o teste de duração repete o clipe como carga sintética, não simula captura60FPS. Isso não comprova FPS de captura/encoder/renderização ou hardware.

A tentativa adicional de teste Chromium com PCM e os componentes reais de ganho/mixer não pôde iniciar o Edge no sandbox; nenhuma reprodução audível de navegador é declarada. Testes frontend verificam slots, contexto compartilhado, encerramento de recursos, estado do player e telemetria; não substituem WebView2 nativo.

Pendentes: teste físico entre dois PCs/casas, microfone/alto-falantes, janela/monitor reais, minimizar/restaurar nativo, hardware encoder/decoder, acessibilidade/resoluções, instalação/atualização completa e consumo do conjunto app+WebView2+auxiliar. H.264/NVENC/AMF/Quick Sync, captura WGC direta e transporte de vídeo não confiável continuam trabalho não entregue. getDisplayMedia é a API de captura existente; a escolha da API Windows interna é do runtime e não foi comprovada.

## Procedimento em dois PCs Windows
1. Instale a beta privada em ambos sem apagar %APPDATA%/ProjetoP2P. Não mude permissões de microfone automaticamente.
2. Conecte pelo convite existente/EasyTier e entre na sala por clique. Confira entrada/saída, volumes, mute/ensurdecer e reprodução. Teste antes sem vídeo.
3. Compartilhe em720p30 com prévia/Iniciar; o outro participante clica AO VIVO e Assistir. Converse e use chat durante dez minutos. Troque canal visual, modos compacto/normal/tela cheia e minimize/restaure.
4. Pare/inicie vídeo mantendo a voz. Interrompa/reponha a rede; a intenção de assistir deve permanecer com Reconectando, sem minimizar sozinha. Teste saída/revogação real.
5. Suba gradualmente para720p60/1080p30/1080p60 e6–12Mbps apenas se houver capacidade. Registre TODOS os FPS, bitrate real e descartes; n/d não é60FPS. Observe CPU/RAM do app, WebView2 e helper. Não declare a meta alcançada só porque o perfil foi selecionado.

Instalador sem assinatura de editor, fontes/licenças completas anexadas, nenhuma publicação automática. Bloqueios Rust e DHT legado em quarentena mantidos. Sem Npcap/Packet.dll, gravação, câmera ou mudanças de configuração global.

Referências consultadas: [estatísticas WebRTC](https://www.w3.org/TR/webrtc-stats/), [API WebRTC](https://www.w3.org/TR/webrtc/), [EasyTier no-TUN](https://doc.easytier.cn/en/guide/network/no-root.html), [port forwarding](https://easytier.cn/en/guide/network/port-forward.html). A capacidade upstream de encaminhar UDP não comprova que o auxiliar SOCKS integrado ofereça associação UDP ou capacidade WAN suficiente.
