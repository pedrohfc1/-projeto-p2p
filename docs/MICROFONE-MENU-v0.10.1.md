# PeeTwoPee 0.10.1 — microfone e menu de participantes (beta privada)

Configurações → Voz e vídeo → Voz: Supressão de ruído Ativado/Desativado. Usa o processamento nativo do WebView (`noiseSuppression`), mantém o cancelamento de eco/AGC existentes e persiste no perfil. Perfis anteriores continuam com a supressão ativada. O status distingue solicitação, aplicação confirmada, ausência de suporte e falha do dispositivo. A alteração aplica constraints às capturas locais existentes; o retorno local é retomado com a mesma preferência, saída e volume. Não aplica processamento a voz recebida ou áudio WASAPI e não grava.

A enumeração inicial e `devicechange` não pedem acesso nem capturam. Aviso não modal informa ausência de entrada, seleção indisponível, desconexão e falha de captura/permissão. Pode ser fechado sem bloquear o app; a mesma falha não reabre repetidamente até recuperação ou outra causa. Abrir configurações, atualizar e tentar novamente são ações distintas. Listar um dispositivo não apaga uma falha de permissão. Reconectar o hardware atualiza a lista; uma nova captura exige a ação explícita de tentar novamente/selecionar o microfone.

Sem microfone, entrar na sala prepara apenas recepção. O primeiro transceiver permanece sendrecv sem track e os quatro adicionais recvonly. Um microfone real pode ser anexado depois com replaceTrack, sem outra sessão ou codec. Essa situação é diagnosticada como receive-only, nunca como captura ativa.

## Menu compartilhado

Clique direito em participantes de voz, membros, contatos ou mensagens. Nos membros e no roster de voz, Shift+F10 também abre o menu focado. Setas navegam; no slider, esquerda/direita e Home/End ajustam o volume. Esc/clique fora fecham. A posição respeita a janela e menus longos têm rolagem.

Perfil, menção, mensagem privada de contatos existentes e copiar ID aparecem conforme o contexto. Volume 0–200% e silenciamento local usam o mixer e a persistência por Peer ID já existentes; compressor/limitador permanece ativo. Ajustar o slider não fecha o menu nem muda o volume da transmissão. O mute do menu não modifica o áudio enviado pelo participante nem o de outros ouvintes.

Em comunidades, o cargo e a remoção de membros reutilizam os fluxos existentes: confirmação antes de expulsar, operação assinada e validação de identidade/papel/registro no backend. Proprietário/moderador não pode remover indevidamente a si mesmo, o proprietário ou outro papel não autorizado. Uma mudança posterior de permissão continua sendo verificada pelo backend.

Ações sem backend existente ficam ocultas: amizade sem convite de contato, ignorar/bloquear/desbloquear, banimento, apelido de terceiros, mover/desconectar/mutar no servidor e efeitos sonoros por usuário. Não há controles administrativos fictícios ou validação somente no frontend. O menu não promete que essas funções foram implementadas.

## Validação e limites

95 testes frontend, build TypeScript/Vite, Go completo/vet, CLI Windows/Linux e smoke de três processos passaram. O teste novo de RTP/Opus em loopback verifica escuta sem microfone e posterior envio sem renegociação. Testes locais reais com 2/3 processos usaram EasyTier/libp2p e áudio gerado. Também passaram vídeo + voz + áudio da transmissão e recuperação de três peers com proprietário temporariamente offline/reset de stream.

A janela nativa Windows/WebView2 foi inspecionada com perfil isolado: supressão selecionável; menu de membro autorizado, volume 146% por mouse/teclado sem fechar, mute, Shift+F10, Esc e limites da janela. Após encerramento normal e reabertura do banco, preferência de supressão, volume/mute, identidade e histórico foram confirmados. Isso não comprova áudio audível.

O navegador headless não iniciou (Crashpad/pipe); o teste de interface automatizado não foi contado como aprovado. Testes acústicos, permissão/remover/reconectar hardware, dois PCs, Radmin real, chamada física ativa durante ajustes, dez minutos de captura física, instalação gráfica completa/upgrade/reboot e GUI Linux seguem pendentes. As métricas sintéticas não comprovam FPS de captura/renderização, qualidade perceptiva ou latência boca-ouvido.

Veja MICROFONE-MENU-v0.10.1-VALIDACAO.json. Não há publicação pública; restrições anteriores do beta permanecem. Rollback do desenvolvimento: ../work/rollback-pre-v0101 contém a fonte 0.10.0 e o diff anterior às alterações. Perfis reais não foram usados.

## Teste manual em dois PCs Windows

1. Instale o beta privado em ambos, preservando o perfil. Confirme o Peer ID e o histórico antigos. Use a conectividade integrada ou a mesma rede Radmin já configurada.
2. Entre na mesma sala. No PC A alterne a supressão em Voz; confirme o status e peça a B avaliar ruído/voz. Não confunda microfone local com rede TX/RX e reprodução remota.
3. Feche/reabra e confira a preferência. Retire o microfone, feche o aviso, navegue e atualize: a mesma falha não deve reabrir continuamente. Reconecte, atualize e clique Tentar novamente. Teste também permissão negada e dispositivo ocupado.
4. Entre sem entrada disponível: o outro usuário deve continuar audível, sem envio local. Depois conecte/seleciona um microfone e confira envio nos dois sentidos.
5. Clique direito em B, ajuste de 0 a 200%, teste mute local e reconexão. Confira que o outro usuário continua audível a terceiros. Reabra o app e verifique a preferência por identidade.
6. Teste perfil, menção, conversa de contato existente, teclado e clique fora. A remoção só deve aparecer para o papel apropriado, exigir confirmação e passar pela autoridade; ações não suportadas não aparecem.
7. Faça uma chamada com voz e tela/áudio WASAPI simultâneos, minimização/restauração e dez minutos contínuos. Verifique saída, mute/deaf, volume independente da transmissão, recursos e logs. Não conceda acesso de captura que não queira autorizar.
