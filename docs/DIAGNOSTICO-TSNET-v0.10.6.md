# PeeTwoPee 0.10.6

O instalador privado inclui o painel de diagnóstico da chamada. O laboratório Tailscale é distribuído em um ZIP separado.

Abra **Diagnóstico** na barra da chamada. O painel coleta cerca de uma amostra por segundo e retém até 60 segundos, 120 eventos e oito resumos em memória. Use **Pausar visualização**, **Copiar**, **JSON** ou **TXT**. A exportação inclui os resumos da execução. Os identificadores do relatório são pseudônimos.

A fila de envio de tela mostra a profundidade, a capacidade e a espera da última retirada medidas no transmissor. Com mais de um espectador, mostra o maior valor observado. A ponte local de voz mostra a espera média no buffer WebRTC e os eventos de ocultação quando o WebView fornece os contadores necessários. Os demais campos e as distinções entre medições de rede e reprodução estão no [guia do diagnóstico 0.10.5](DIAGNOSTICO-TSNET-v0.10.5.md).

## Comparar conexões com a mesma carga

1. Em **Diagnóstico > Comparar conexões**, os dois participantes selecionam um ao outro. Quem recebe usa **Permitir teste deste participante**. A permissão vale para uma execução em até 90 segundos e pode ser revogada.
2. Quem inicia escolhe 10, 30 ou 60 segundos e confirma **Iniciar teste com carga limitada**. A sonda envia blocos de 4 kB com intervalo mínimo de 250 ms e mede os ecos do participante autorizado. O teto é de 128 kbit/s de payload por direção. Mantenha a mesma duração e condições de voz e vídeo nos testes.
3. No laboratório, use a mesma sonda com a opção explícita `-load`. Exporte o resultado pelo argumento `-report` e importe o JSON na comparação do PeeTwoPee. As instruções de autenticação e os comandos estão no [guia do laboratório](../experiments/tsnet/README.md).

O RTT sob carga nesta comparação é a latência dos ecos de 4 kB. A vazão é calculada pelos bytes confirmados na janela. A sonda mede uma carga limitada e não estima a capacidade máxima da rede. Gráficos de voz e vídeo do aplicativo continuam disponíveis para observação paralela.

## Validação e limites

- Dois processos locais do PeeTwoPee com EasyTier, áudio sintético Opus nos dois sentidos e chat simultâneo. O teste de diagnóstico recuperou RTT e contadores reais. O teste de carga de 10 segundos recebeu ecos e recusou tentativas sem permissão e após o uso da permissão única.
- O teste local de vídeo recebeu 150 de 150 quadros VP8 gerados em 720p a 30 FPS e manteve voz bidirecional. A recuperação de reset forçado falhou em uma rodada e passou na repetição. O teste de fila também passou com o detector de condições de corrida.
- Dois nós tsnet reais autenticaram perante o controle de teste local da Tailscale e transmitiram dados por caminhos direto e DERP. O laboratório verificou a rejeição de identidades e destinos não autorizados e a persistência do cadastro no controle de teste. A autenticação no serviço hospedado e a redescoberta após reinício entre computadores reais exigem testes adicionais.
- A sonda compartilhada de 10 segundos passou por DERP no ambiente local, com 156.000 bytes confirmados, 124,8 kbit/s por direção e RTT P95 de 5,2 ms. A autorização única rejeitou uma segunda tentativa.
- O SDK no laboratório manteve TUN desativado e Wintun descarregada durante a validação local. O código comum do SDK ainda inclui referência ao carregador Wintun. O executável principal permanece separado e passa pela auditoria estrita de PE. O seletor do aplicativo mantém a conexão existente e apresenta Tailscale como laboratório.
- A validação local não cobre login humano em uma tailnet real, peer relay em rede externa, tráfego do aplicativo principal via tsnet, captura física em dois PCs nem instalação completa no Windows de uso diário.

O beta Windows não tem assinatura Authenticode de editor. Consulte `VALIDACAO.json`, `payload-verificado.json` e `SHA256SUMS.txt` para conferir a entrega.
