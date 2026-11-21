# PeeTwoPee 0.8.0 — interface e retorno local

Beta privado. Não publicar automaticamente. Atualize ambos os PCs, preservando `%APPDATA%/ProjetoP2P`, identidade, contatos e históricos. Não há reconstrução do chat, comunidades, protocolos de voz/tela ou EasyTier.

## Microfone e retorno

A troca anterior chamava `stopTest()`, apagando a intenção de ouvir. Agora `MonitorSession` separa intenção, inicialização e reprodução efetiva. Trocar a entrada encerra o retorno anterior, tenta iniciar o novo e mantém saída/volume. Em chamada, a substituição do microfone conclui antes de emprestar uma cópia da captura ao retorno. Não se abre um segundo microfone para testar o áudio da chamada. A nova faixa de envio permanece desabilitada durante a substituição assíncrona e respeita a intenção mais recente de mute/ensurdecer após o commit, com regressão específica para mute durante a troca. Se falhar, a sala/captura anterior continuam quando possível, o retorno fica solicitado mas sem áudio e oferece recuperação. Uma seleção removida no retorno não captura silenciosamente outro microfone.

Fechar configurações, ocultar ou encerrar cancela a intenção e invalida aquisições pendentes; captura tardia é liberada. O encerramento físico da faixa limpa o medidor e mostra recuperação. Reconectar/trocar dispositivos exige uma ação explícita; a enumeração não abre microfone. Dispositivos e volumes persistem; o estado de captura não é reativado após reiniciar o app.

Entrada e saída aparecem lado a lado. Volume de microfone usa um ganho leve somente na chamada, sem nova aquisição; volume de alto-falante atua na reprodução local. O retorno tem seu próprio volume e usa o sinal capturado localmente. Zero é um volume válido; perfis anteriores recebem 100% de entrada/saída. Cancelamento de eco/supressão continuam usando os recursos existentes do navegador, sem novo processamento pesado. Fones são recomendados. O teste não transmite, grava ou armazena áudio.

## Organização visual

Paleta própria verde/violeta, rail de comunidades, canais, mensagens centrais, participantes e dock fixo de perfil/áudio. Voz continua no provider persistente enquanto se navega. O painel de tela existente continua separado, com prévia e confirmação de início. O input de mensagem não cresce desproporcionalmente durante chamadas.

Menus contextuais em comunidades, contatos, canais, mensagens e participantes reutilizam ações existentes e suas autorizações. Use botão direito ou o comando de contexto do teclado; setas/Home/End navegam, Enter executa e Escape fecha. Modais mantêm foco e fecham por Escape; botões têm nomes acessíveis e tooltips. Há estilos para movimento reduzido e larguras menores. Indicador online significa peer observado conectado; não significa identidade civil, participação autorizada por si só ou mídia comprovada. O indicador de fala usa nível de áudio, com atualizações quantizadas para reduzir renders.

Polling visual continua pausado ao ocultar, mantendo transporte, autoridade e voz. A bandeja e as regras de parada de tela ao minimizar/ocultar são preservadas. CPU/RAM de toda a árvore de WebView/helper ainda precisam de medição física; não há promessa de desempenho a partir de mocks.

## Convites menores e compatibilidade

`peetwopee://join/…` contém todos os dados necessários do convite, não um código de consulta. Não foi adicionado registro de protocolo no Windows; cole no campo de convite do app. JSON legado e protobuf c2 continuam aceitos; o botão Formato JSON permite compartilhar o envelope anterior. Clientes antigos podem exigir atualização para políticas novas.

O envelope versão 2 usa Base64URL, política/escopo binários e omite o endereço EasyTier somente quando ele é exatamente derivável da chave/Peer ID e porta estável. Reconstrói os mesmos campos originais antes de verificar a assinatura Ed25519. DEFLATE só é usado quando reduz tamanho. Decodificação limita entrada a 16.000 caracteres e expansão a 12.000 bytes; versão, números, campos, política e assinatura continuam validados. O nonce criptograficamente aleatório de 128 bits, cota, expiração, epoch e revogação existentes não mudam. Não há confiança em IP no lugar do Peer ID.

Nas fixtures, um convite nativo caiu de 869 caracteres JSON/327 c2 para 312; um integrado com política de 1.313 JSON/760 c2 para 659. O tamanho varia. Ainda são centenas de caracteres porque identidade, assinatura, raiz e conectividade precisam viajar. Não existe resolução distribuída segura para um código de poucos caracteres; não se cria servidor central obrigatório nem promessa de ingresso offline.

A validade e a cota são exibidas, o proprietário consulta utilizações/revogação e o ingresso verifica o registro assinado. Alterar opções limpa o convite gerado, evitando mostrar validade de outra política. Compartilhe apenas com pessoas autorizadas: a credencial de conectividade que já integra o convite é uma capacidade de acesso, não deve ir para logs públicos. A codificação não fornece confidencialidade do convite.

Resolução de proprietário continua priorizando rota autenticada por identidade, endereço integrado estável, diretório assinado e cache certificado; porta/IP antigos não substituem a validação Ed25519. Mudanças de permissão e ingresso continuam exigindo autoridade. Não há novo mecanismo central de descoberta.

## Validação e testes físicos

Veja `PEETWOPEE-v0.8.0-VALIDACAO.json`. Passaram testes unitários da interface, Go/vet, compatibilidade e limites de convites, três backends com EasyTier real, e regressões de voz/tela/chat com mídia gerada. O teste visual automatizado foi preparado em `tests/interface_browser_smoke.cjs`, mas Edge não iniciou no sandbox e o navegador integrado não alcançou o localhost isolado. Esses testes visuais NÃO foram declarados aprovados. As métricas de mídia são de um processo Go compartilhado com núcleos/Pion sintéticos, sem WebView/helper, encoder ou tela física.

Em um host QA com navegador apropriado, aponte `P2P_PLAYWRIGHT_PATH` para o Playwright de teste e, opcionalmente, `P2P_BROWSER_EXECUTABLE` para o executável do navegador. Compile `go build -o webui-qa.exe ./tests/webui` após o frontend e execute `node tests/interface_browser_smoke.cjs webui-qa.exe`. Esse harness usa oscilador e saídas simuladas, perfil temporário e verifica retorno/troca/erro/recuperação, chamada local, menus, 760/900/1280/1600 e ocultação simulada. Ele não pede acesso a dispositivos reais. Não substitui WebView2 ou teste entre casas.

Procedimento físico: abra Configurações de áudio, selecione entrada/saída, ajuste volumes, ative Ouvir minha voz e troque entre dois microfones. Confirme retorno contínuo no novo dispositivo, medidor e mesma saída/volume. Desconecte o selecionado, confirme estado sem áudio e recupere; feche configurações e confirme encerramento do retorno. Repita em chamada nos dois PCs, conversando e enviando texto. Gere/cole convites novos e antigos, teste cota/revogação com um terceiro participante. Em 760/900/1280/1600, verifique canais, perfil, menus, foco e legibilidade. Minimize/restaure: voz continua, tela para e não volta sem seleção explícita. Teste monitor e janela no compartilhamento, reconexão e saída. Meça a árvore completa de processos em voz apenas e voz+tela.

Continuam pendentes dispositivos físicos, layout/renderização real, instalação/atualização completa, duas casas/CGNAT/relay real, GPU e Linux desktop/race. Sem contornar consentimento do usuário.

## Build privado

Use `scripts/build-desktop.ps1 -Output PeeTwoPee-v0.8.0.exe` e `scripts/package-windows.ps1 -DesktopExecutable build/bin/PeeTwoPee-v0.8.0.exe`. Mantém o SDK Wails revisado com WebView2 PermissionStateDefault e verificação `prompt-v1`. Não usar build não verificado para distribuir. Instalador privado sem assinatura de editor; helper auditado, licenças e fontes correspondentes completas acompanham a entrega. Sem Npcap/Packet.dll/TUN, novas VPNs ou mudanças de firewall/rotas/DNS. Release pública segue bloqueada; nenhuma publicação foi autorizada.
