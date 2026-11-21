# PeeTwoPee 0.10.3 — voz, PTT e tema (beta privada)

## Configurações > Voz e vídeo > Voz

Entrada/saída continuam lado a lado, com volumes separados, supressão nativa e atualização de dispositivos. Tema grafite com destaque índigo (#5865F2) e hover #4752C4. O teste tem um único botão: Testar microfone / Parar teste. Inicia a captura e o retorno local existentes, atualiza o medidor, preserva saída/volume e libera recursos ao parar, sair da aba ou minimizar. Volume de retorno e cuidados ficam recolhidos. Use fones; o teste não transmite à rede nem grava. Falha de retorno não é mostrada como áudio funcionando.

## Ativação por voz e sensibilidade

Ativação por voz é o padrão. Automático estima um piso de ruído por energia RMS, com limiar limitado; não é reconhecimento de fala nem substitui a supressão nativa. Manual oferece -60 a -10 dBFS (padrão -45). A prévia no teste mostra RMS real em escala logarítmica de -60 a 0 dBFS e marcador do limiar, sem a saturação visual anterior em RMS 0,125. A barra não comprova transmissão remota. Um hold de 150 ms reduz cortes entre sílabas; é necessário calibrar em ambiente real, especialmente com AGC do navegador.

O gate atua somente no PCM do microfone local antes do Opus, depois mantém o ganho de microfone separado. Não altera voz recebida, volume de participantes, WASAPI ou vídeo. Pacotes de silêncio/DTX podem continuar quando o gate está fechado; TX de rede não é prova de fala. Mute/deaf continuam desabilitando as faixas de captura e são prioritários.

## Aperte para falar

Selecione PTT. Clique Definir tecla; no campo focado pressione uma tecla, ou Esc para cancelar. F8 é o padrão e Redefinir tecla restaura F8. Letras, dígitos, F1–F12, espaço, CapsLock e Shift podem ser selecionados quando entregues pelo WebView. Atalhos reservados pelo Windows/WebView podem não chegar ao aplicativo; use uma letra alternativa. Ctrl/Alt/Meta não são aceitos como tecla PTT. O atraso de liberação é 0–500 ms, com suavização curta de ganho.

PTT funciona apenas enquanto o app recebe foco/teclas. Não é atalho global para jogos ou outros aplicativos. Digitação em input/textarea/select/contenteditable não abre o gate; focar um campo, perder foco, ocultar ou trocar a política limpa a tecla pressionada. Depois de reconectar/substituir o microfone, pressione a tecla novamente. Fechar configurações cancela a definição incompleta.

Modo, tecla, atraso e sensibilidade são persistidos no JSON de preferências existente do perfil, validado no backend. Perfis antigos usam ativação por voz automática. Nenhuma identidade, comunidade, contato ou histórico foi migrado/apagado. LocalStorage guarda uma cópia auxiliar; o perfil é a referência ao reabrir.

## Implementação e limitações

AudioWorklet processa o gate no relógio de áudio; não depende de timers React para a via principal. O nó solicita downmix explícito para mono, inclusive em microfones estéreo. O módulo é carregado uma vez por AudioContext e os nós/listeners são liberados ao trocar/sair. Se AudioWorklet não existir, há fallback de timer de 20 ms, com aviso no console e sem garantia de precisão em segundo plano. Se o módulo existir mas falhar ao carregar, a preparação falha explicitamente em vez de transmitir com gate ignorado. Codec, protocolos e transporte P2P não foram substituídos.

## Testes executados

107 testes frontend: políticas, limites, PTT/atraso/reset, energia/hold, ganho e limpeza, worklet processando PCM sintético, compatibilidade de teclado e regressões existentes. Go completo/vet e persistência/validação das preferências passaram. CLI Windows/Linux e smoke de três processos passaram.

Navegador Edge com dispositivos sintéticos e Core real: navegação, um botão de teste, retorno, seleção/troca, erro/recuperação, handover em chamada, tecla V, limites de atraso/sensibilidade, layout 760/900/1280/1600 e ocultação/restauração simuladas. Outro teste codificou PCM em Opus, atravessou dois cores/EasyTier/libp2p e verificou reprodução decodificada: PTT fechado, tecla pressionada, bloqueio ao digitar, retorno à ativação por voz e 30 segundos bidirecionais com chat/limpeza.

Regressões adicionais com 2/3 processos (70,10 s) e vídeo/voz/áudio da tela gerados (39,52 s), incluindo reconexão, passaram. A janela Windows foi inspecionada visualmente em perfil isolado e manteve o modo PTT ao reabrir. Isso não é validação acústica nem prova de teclado físico no WebView2. Testes de microfone/saída físicos, dois PCs, voz minimizada real, ruído ambiente, instalação gráfica completa/upgrade/reboot e GUI Linux continuam pendentes.

## Procedimento físico

1. Instale o beta privado nos dois PCs; confira Peer ID e histórico. Não publique a release.
2. Em Voz escolha dispositivos, ajuste volumes e inicie o teste único usando fones. Troque entrada/saída, pare e confirme liberação do indicador do Windows.
3. Em ativação por voz, teste automático e depois manual, usando a barra de limiar. Na sala confira fala recebida no outro PC; teste mute/deaf e a supressão existente.
4. Em PTT defina uma letra, confira que o outro PC só recebe fala enquanto pressionada e após o atraso escolhido. Teste 0/100/500 ms, texto, perda de foco, minimizar/restaurar e reconectar. PTT fora do foco do app não é suportado.
5. Reabra o app e confira preferências. Faça voz + tela/WASAPI simultâneos e confira que o gate não modifica o áudio da transmissão. Compare logs de rede com PCM audível, sem tratar contadores locais como prova remota.
