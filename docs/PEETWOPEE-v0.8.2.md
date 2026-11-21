# PeeTwoPee 0.8.2 — painel de chamada de voz

Beta privado. Preserve `%APPDATA%/ProjetoP2P`. Voz, chat, comunidades, identidades, convites, EasyTier, preferências e transporte de tela permanecem nos serviços existentes. Sem publicação pública.

## Painel e ações

A chamada aparece acima do perfil no canto inferior esquerdo, dentro da região lateral. Mostra canal e comunidade do estado local verificado, estado de conexão, microfone/saída, duração, quantidade de participantes, nome/avatar atual. Não depende do canal de texto ou tela visitada. Quando há outros participantes e a mídia cai, mostra Conectando/Reconectando voz. Sozinho, Voz conectada significa ingresso na sala local, não confirmação de áudio remoto.

Os seis botões são microfone, áudio/fone, tela, câmera futura, desconectar e configurações. Todos têm tooltip; há estados de mute/ensurdecer/tela ativos e foco por teclado. Câmera tem tooltip **Câmera em breve**, está desabilitada e não possui handler de captura. Nenhuma permissão de câmera é solicitada. Mantemos ícones Lucide e identidade visual própria, sem assets proprietários do Discord.

O atalho de tela chama diretamente o controlador React existente, dentro do gesto do clique. Isso preserva a ativação exigida pelo seletor nativo. Escolher monitor/janela produz apenas prévia; o botão Iniciar continua separado. Ao transmitir, o atalho vira Parar e permanece utilizável para interromper. Autenticação, limites de banda/filas, indicação permanente e parada ao ocultar/sair continuam. Não existe início silencioso ou retomada automática de captura.

Desconectar cancela a sessão local, encerra captura/retorno, PC/áudio e painel de tela, e sai do backend sem fechar a janela. Respostas antigas de consulta/ingresso são descartadas por geração, evitando voltar a mostrar um estado conectado depois de sair. O estado ocupado também é limpo ao cancelar uma preparação. Mute/ensurdecer atualizam o painel imediatamente e continuam usando controles associados à sessão.

Configurações abre diretamente Voz e Áudio e mantém a chamada. Troca de dispositivos, volumes individuais e retorno local seguem a implementação anterior. O realce da sala e os estilos dos participantes, perfil e tela acompanham o painel. O espaço lateral é reservado para não esconder os últimos canais. O relógio usa tempo monotônico e só atualiza visualmente quando visível; restaurar inclui o período minimizado sem reiniciar a contagem. Não é um histórico persistente de chamadas.

## Testes disponíveis

Veja `PEETWOPEE-v0.8.2-VALIDACAO.json`. Passaram 35 testes frontend, TypeScript/Vite e Go/vet. Os testes do painel renderizam seu componente React, verificam nomes/estado/duração/contagem, seis botões/tooltips, callbacks corretos, câmera desabilitada sem captura e indicação de microfone silenciado pelo ensurdecer. Não são cliques em um WebView nativo.

Regressões reais locais: voz bidirecional pelo EasyTier/libp2p, mute/deaf, reparo da ponte, reconexão, texto e revogação; vídeo VP8 gerado recebido/decodificado, preview sem envio, permissões, voz/chat simultâneos, recuperação e parada por expiração de visibilidade; três backends de comunidades com convites, permissões, texto, reinício e preservação de chat. O rótulo amigável da comunidade/canal é conferido no teste de serviço de voz.

Continuam pendentes o posicionamento/renderização e os cliques físicos no WebView2, seletor real de monitor/janela pelo novo atalho, microfones audíveis, minimizar/restaurar nativo, duas casas, instalação/upgrade completo e Linux desktop/race. O ambiente de QA atual tem limitações de inicialização do navegador/alcance do localhost registradas nas versões anteriores. Nenhum teste físico foi declarado realizado a partir de SSR ou mídia gerada.

## Procedimento Windows

Atualize ambos os PCs, entre em uma comunidade autorizada e em uma sala de voz. Confira Voz conectada ou o estado real de reconexão, nomes, perfil e duração. Navegue por mensagens privadas, rede e outras comunidades: o painel deve permanecer. Teste mute e ensurdecer, os indicadores e o volume por participante. Abra a engrenagem e troque dispositivos/volumes sem sair. A câmera deve estar desabilitada e nenhum pedido de câmera deve surgir.

No atalho de tela, selecione uma janela sem dados privados, confirme a prévia e só então inicie. O outro participante autorizado escolhe Assistir. Confira estado ativo e pare pelo atalho; repita por monitor se desejar. Minimize para a bandeja: voz continua, tela para e não volta ao restaurar. Desconecte pelo painel: captura e reprodução devem terminar, mantendo o app aberto. Confira larguras de 760/900/1280/1600, tooltips, foco e ausência de sobreposição de canais. Esses passos ainda exigem validação física.

## Build e distribuição

Use `scripts/build-desktop.ps1 -Output PeeTwoPee-v0.8.2.exe`, `scripts/package-windows.ps1 -DesktopExecutable build/bin/PeeTwoPee-v0.8.2.exe` e `tests/installer-smoke.ps1 -DesktopExecutable build/bin/PeeTwoPee-v0.8.2.exe`. Mantém a política WebView2 Default/prompt-v1, sem concessão automática. Setup privado sem assinatura de editor; verificação de payload não equivale a instalação completa. Licenças e fontes LGPL correspondentes completas acompanham. Sem Npcap/Packet.dll/TUN, alterações de firewall/rotas/DNS ou novo processamento pesado. Rust/DHT continuam bloqueando release pública.
