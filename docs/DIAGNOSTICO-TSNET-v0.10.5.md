# PeeTwoPee 0.10.5 — diagnóstico na sala e laboratório Tailscale

Beta privado. Nenhuma publicação pública foi feita.

## Como usar o diagnóstico

Entre na chamada e clique no botão Diagnóstico (ícone de atividade) na barra inferior. O painel lateral pode ser fechado sem sair da sala. São coletadas amostras aproximadamente a cada segundo, com no máximo uma requisição de coleta em andamento. Fechar o painel interrompe as sondas; Pausar visualização congela somente a apresentação.

A coleta reúne:
- RTT de uma sonda libp2p pequena em peers já conectados da sala, média/P95 da janela e variação sucessiva de RTT. Não é RTT do codec, perda UDP nem jitter RTP.
- Conexões/streams ativos, caminho confirmado por multiaddr Circuit Relay quando disponível, transporte, duração de NewStream e contadores acumulados de tentativas/falhas/timeouts/resets. Um reset reportado não identifica automaticamente o iniciador; EOF normal é contado separadamente.
- Bytes/pacotes de mídia TX/RX por peer, fila de saída e horário local do último pacote recebido. TX local não comprova RX no outro computador. Silêncio, PTT e DTX podem não enviar mídia; tempo sem RX não é automaticamente perda ou desconexão.
- Autorização real da sala e mudanças observadas na expiração/renovação. Sem autorização válida não se cria novo caminho de mídia. O diagnóstico não altera permissões.
- Blocos/amostras reais processados pelo worklet do microfone, estado do gate, captura e contexto; duração capturada, pacotes da ponte local e amostras/frames decodificados somente quando WebRTC os fornece. Blocos de processamento não são apresentados como frames Opus de 20 ms. Nenhuma amostra de áudio é gravada ou enviada no diagnóstico.
- FPS de captura/encode/envio/recepção/decode/render, bitrate configurado e deltas reais, descartes e tempos médios por frame derivados dos contadores totalEncodeTime/totalDecodeTime quando presentes. Ausência de contador fica n/d; não se altera codec, bitrate, fonte ou FPS.
- CPU/RAM do processo Go para correlação, sem incluir WebView2 ou EasyTier externos. Isso não mede CPU do encoder. Coletas entre estágios são assíncronas e não provam causalidade.

O caminho TCP libp2p pode parecer direto dentro de um túnel EasyTier que esteja usando relay. Quando o underlay não é confirmado, a interface o declara desconhecido. Não se atribui lentidão automaticamente a libp2p/EasyTier.

## Retenção e privacidade

Histórico limitado a 60 segundos/61 amostras, 120 eventos e oito resumos em RAM nesta execução. Dados não persistem entre reinícios. Gráficos usam horários reais; snapshots do cache não duplicam amostras de RTT. Identificadores de peer/sessão são pseudonimizados; não se exportam nomes, IPs, convites, mensagens, credenciais ou conteúdo de mídia.

Copiar/Exportar pedem confirmação antes de compartilhar identificadores e horários. JSON e TXT contêm o mesmo relatório legível em texto; a exportação nativa abre o seletor de arquivo, nunca envia a terceiros. O tamanho é limitado a 2 MiB no backend. O console flutuante e seus logs existentes continuam disponíveis.

Análises diferenciam evidências (autorização pausada, reprodução local pausada) de hipóteses (ponte avançando sem TX). Reprodução no grafo não comprova som audível no dispositivo físico. Métricas de perda UDP, buffer físico, ACK remoto de áudio e filas de vídeo sem contadores apropriados permanecem n/d.

## Comparação

Em Comparar conexões, registre janelas de 10, 30 ou 60 segundos **sem carga adicional**, selecione dois resumos ou importe o resumo numérico do laboratório tsnet. Os campos importados são validados e normalizados, descartando texto arbitrário, credenciais e identificadores. Compare apenas condições equivalentes. A sonda não mede throughput máximo, latência sob carga ou perda UDP.

O botão de teste de carga permanece desabilitado e explica que ainda não foi validado; não se inicia carga durante chamadas ou transmissões. Resumos do laboratório não têm mídia/FPS, e não comprovam que o aplicativo utilizou Tailscale.

## Tailscale: entrega isolada autorizada

A alternativa prevista no pedido foi adotada: diagnóstico funcional no app e protótipo tsnet separado. `tailscale.com/tsnet v1.104.7` exige Go 1.27.4; o app mantém Go 1.26 e suas dependências. Não se atualizou o transporte estável para obter uma integração ainda não validada entre tailnets reais.

O ZIP separado contém PeeTwoPee-Tailscale-Lab.exe, README, avisos e inventário de 107 módulos efetivamente compilados (nenhum aviso ausente). O SDK é BSD-3-Clause e o módulo está fixado. O app/instalador principal **não incluem o SDK/protótipo**: a opção Tailscale na central é apresentada como laboratório separado e fica desabilitada. Libp2p/EasyTier/Radmin permanecem funcionais como antes. Não há troca automática nem fallback oculto.

O protótipo usa listener/dialer userspace tsnet e um adaptador limitado para Noise/libp2p, preservando a autenticação Ed25519. Testes verificaram identidade autorizada e rejeição de outra sobre underlay simulado. O estado fica em diretório separado com proteção de acesso; não há auth key embutida. Login só ocorre por URL oficial e ação da própria pessoa. Uma tailnet e suas ACLs são requisitos separados do convite PeeTwoPee. Instruções completas estão em experiments/tsnet/README.md, também incluído no pacote.

A referência a wintun.dll no código comum do SDK é explicitamente registrada no relatório PE do laboratório. A auditoria estrita do app não foi relaxada. Não há essa DLL no pacote do laboratório, nem importação PE estática/delay dela. O SDK real foi inicializado e reportou tun=false, wintunLoaded=false, NeedsLogin. Isso valida sua inicialização userspace, não uma sessão autenticada ou transferência entre casas. Não se declara ausência da string no executável. O opt-out oficial do SDK e a tag ts_omit_logtail impedem envio automático de logs do laboratório.

## Validação disponível

- Testes automatizados frontend, TypeScript/Vite, fixtures browser de diagnóstico/cópia/exportação/pausa/comparação em 1280×720, 1600×900 e 1920×1080.
- Regressões frontend de chat, comunidades, perfil, controles de voz, AO VIVO, player persistente e tela cheia.
- go test ./... e go vet ./..., build Wails Windows e verificação de payload/licenças/versões.
- Dois processos reais de produção em perfis isolados, EasyTier local integrado e navegador com PCM sintético: Opus bidirecional por 30 s, chat simultâneo, PTT, limpeza e diagnóstico real com RTT, RX/TX e último recebimento. Nenhum microfone físico ou ambiente externo foi capturado. Não é teste entre duas casas.
- Laboratório: testes de URL oficial, pasta exclusiva/ACL, ausência de fallback LAN/público, encerramento limitado, Noise/libp2p com gater e identidades persistentes. Inicialização do SDK sem TUN nem Wintun carregada, sem autenticar.

Pendentes: login humano/aprovação e persistência de sessão autenticada Tailscale, dois nós tsnet reais, direto/DERP/peer relay em redes distintas, chat/comunidades/voz/vídeo do app sobre tsnet, carga comparativa controlada, dispositivos físicos, estabilidade longa e teste completo do instalador em Windows limpo. Alguns contadores dependem do WebView/transportes e permanecem n/d. O beta não possui assinatura Authenticode de editor.

## Artefatos

Instalador privado: PeeTwoPee-Setup-v0.10.5-beta-privado.exe (diagnóstico, conexão padrão preservada).
Laboratório separado: PeeTwoPee-v0.10.5-Tailscale-Lab.zip.
Confira VALIDACAO.json, auditoria-laboratorio.json, GUIA.md e SHA256SUMS.txt na pasta de entrega.
Rollback de desenvolvimento: work/realtime-diagnostics-rollback.zip, fora do repositório. O rollback transacional do instalador permanece disponível.
