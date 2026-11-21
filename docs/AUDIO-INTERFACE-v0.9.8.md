# PeeTwoPee 0.9.8 — áudio e interface (beta privada)

## Correções e causas verificadas no código
A entrada aguardava getUserMedia antes de criar/iniciar o elemento de reprodução. Captura e mixer remoto também compartilhavam o mesmo AudioContext. Isso deixava a ativação sujeita a autoplay após a perda do gesto original e vinculava dois ciclos de vida. Agora os contextos de microfone e voz remota são independentes e ambos, junto do reprodutor, são ativados sincronamente no clique de entrada, antes de qualquer await. Falha de saída é recuperável sem abandonar a sala, não muda silenciosamente o dispositivo e permanece silenciada até um roteamento válido. A ação de recuperar reprodução retoma os contextos e o elemento, verificando estado real e com espera limitada. Nenhuma fonte WASAPI participa da ativação da voz.

Isso corrige os caminhos de falha encontrados; não comprova que toda ocorrência relatada em outros computadores tinha a mesma causa. Não foram alterados Opus, protocolo de voz, autorização, EasyTier ou transporte de vídeo.

## Compartilhamento e privacidade
A seleção explícita de janela/monitor é precedida por aviso de vídeo e áudio e inicia a publicação automaticamente. O seletor do Windows continua obrigatório; nenhuma permissão é concedida automaticamente. Vídeo e player existentes são preservados. Sem gravação.

Para janela, o catálogo nativo inclui título local em UTF-8, associado ao PID e à criação do processo já verificados pelo backend. Só um título exatamente igual e único ao rótulo da faixa permite seleção automática. Rótulos opacos, várias janelas com mesmo título, abas ou metadados ausentes falham de forma segura: vídeo continua, fonte manual disponível em Detalhes técnicos do áudio. Não há fallback de uma janela para captura global. Não são registrados títulos no console.

Para monitor, a fonte é a saída padrão do Windows. WASAPI de dispositivo não consegue excluir seletivamente o PeeTwoPee. Para não retransmitir a voz recebida, a captura automática exige saída da chamada comprovadamente em outro grupo físico, conforme enumerateDevices. Se essa separação não puder ser verificada, o som automático permanece indisponível e há aviso; o vídeo continua. Selecionar uma saída separada e compartilhar novamente permite a tentativa automática. Mudanças de saída/dispositivos interrompem a captura de saída inteira, sem parar vídeo, para impedir eco por roteamento alterado. A alternativa avançada 'Todos os aplicativos, exceto PeeTwoPee' exclui a árvore local, mas pode capturar outros dispositivos de saída; não equivale a somente a saída padrão. O modo legado de saída inteira exige confirmação específica sobre eco.

## Interface
Detalhes técnicos recolhíveis conservam qualidade, limites, codecs, métricas e seleção manual WASAPI. Esses controles permanecem montados; recolher não reinicia mídia. Prévia local, AO VIVO, espectadores e player persistente recebem prioridade. Participantes de voz usam avatares compactos, indicadores de mute/ensurdecer/fala e transmissão, usuário local destacado, recolhimento e convite sujeito à autorização existente. Membros organizados por voz/online/offline e cargo; identidade completa e gerenciamento ficam no perfil. Contextos e volumes individuais permanecem.

## Procedimento obrigatório em dois PCs
1. Instalar o beta em ambos; verificar mesma identidade e histórico. Usar rede integrada e repetir com Radmin já configurado, sem alterações de firewall.
2. Criar/entrar em sala por clique e autorizar microfone pessoalmente. Sem abrir tela/WASAPI, confirmar áudio bidirecional, TX/RX por peer e reprodução. Captura local não comprova envio remoto.
3. Testar mute, ensurdecer, volumes e troca de microfone/saída; testar retorno local, sem gravação nem envio do teste.
4. Escolher janela de jogo e verificar associação segura. Confirmar som e imagem recebidos. Quando metadados forem opacos, conferir erro claro e selecionar fonte avançada explicitamente.
5. Para monitor, usar chamada em outra saída física e conferir aviso de abrangência. Mesmo dispositivo deve bloquear captura automática para evitar eco. Verificar volume da transmissão independente da chamada.
6. Recolher detalhes, navegar, assistir via AO VIVO, entrar/sair espectadores; minimizar/restaurar por dez minutos e parar pela bandeja. Conferir encerramento de captura e integridade do chat/comunidades/convites.
7. Desconectar/restaurar rede; comparar contadores capturados, ponte local, rede TX/RX e reprodução nos dois consoles. Não coletar conteúdo da chamada.

## Limites de validação
Testes Go/Pion/EasyTier usam mídia gerada e fonte WASAPI substituída, não microfones/alto-falantes físicos. Testes de política/ativação do frontend usam mocks; não comprovam autoplay em cada WebView2. Edge headless falhou no ambiente de execução; a revisão automática de aprovação rejeitou o lançamento nativo com perfil isolado (aprovação exigida, indisponível nesta sessão). Isso não conta como teste visual aprovado. Dois PCs, áudio audível, WASAPI físico, eco, minimização real, desempenho de captura/renderização e 1080p60 exigem validação externa. Fontes correspondentes e licenças EasyTier acompanham o beta. Publicação bloqueada.
