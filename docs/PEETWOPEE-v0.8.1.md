# PeeTwoPee 0.8.1 — áudio, interface e seleção de rede

Beta privado. Preserve `%APPDATA%/ProjetoP2P`. Voz, chat, comunidades, convites c2/JSON/compactos, EasyTier e tela continuam utilizando os serviços existentes. Sem publicação pública, novas dependências ou mudanças de firewall/VPN/DNS/rotas.

## Configurações de áudio durante chamadas

O estado ocupado de ingresso/sinalização não deve bloquear controles de uma chamada ativa. Enumeração de dispositivos era aguardada no caminho de ingresso, podia manter esse estado preso e não tinha prazo. Agora há estados separados de troca de entrada/autorização; só a aquisição inicial ainda em preparação e a troca em andamento impedem outra aquisição concorrente. Saída, volumes e atualização permanecem acessíveis. A enumeração é deduplicada, tem prazo de quatro segundos, ignora resultado tardio e permite tentar novamente, sem fechar a chamada.

A troca mantém `replaceTrack`, libera a captura anterior após commit da nova e preserva mute/ensurdecer e volumes individuais. A faixa processada permanece desabilitada durante a operação assíncrona e só é ativada conforme a intenção atual de mute depois de atualizar a referência de captura. Falhas preservam a chamada/fonte anterior quando possível e mostram erro. Retorno/teste local reutilizam a captura por clone, sem transmissão ou gravação e sem mudar o mute da rede. Use fones. Sensibilidade avançada continua indicada como futura; os recursos existentes de eco/ruído do navegador não ganharam processamento pesado.

Também havia um mutex global retido enquanto o chat aguardava a confirmação de um peer. Isso bloqueava acesso ao serviço de áudio/estado. A espera remota agora ocorre fora desse mutex, mantendo a verificação de contato, assinatura com a identidade persistente e gravação do resultado. O teste de ACK atrasado comprova que VoiceState e State respondem durante a espera.

Foi removido **Encerrar PeeTwoPee** da página Voz e Áudio. Encerrar pela bandeja e minimizar/fechar para a bandeja mantêm seu comportamento.

## Padrão de rede e persistência

Primeiro início e perfis sem campo de modo usam **Rede P2P integrada** e iniciam o auxiliar incluído. Preferências explícitas auto/Radmin/Hamachi/interface/integrada permanecem exatamente como salvas. Um campo antigo explicitamente vazio continua significando auto legado. Perfis anteriores ao campo mode com bootstraps/relays nativos configurados também continuam nativos, evitando trocar uma configuração existente.

A seleção e o adaptador passam por `SelectNetwork`, que carrega o perfil atual, altera somente esses campos e grava antes de iniciar. Se o adaptador/helper/rede falhar, a escolha permanece salva; não há fallback automático. Nome e opções avançadas ainda usam Salvar configurações. A interface mostra conectado/conectando/reconectando/indisponível e não confunde auxiliar ativo com peer autenticado conectado. Identidade, mensagens, contatos e comunidades são preservados. Trocar o transporte efetivo pode encerrar uma chamada; isso é diferente de trocar dispositivos de áudio.

No modo integrado, não há necessidade de enumerar adaptadores Windows a cada consulta da interface. O snapshot informativo fica em cache por 30 segundos; o auxiliar continua acompanhando a rede WAN e o dialer usa o endpoint lógico por identidade. Os modos nativos/interface mantêm o monitoramento existente de IP e GUID. Selecionar outro modo busca seus adaptadores normalmente. A lista informativa pode levar até 30 segundos para refletir mudanças quando integrada; ela não é usada para decidir se um peer está conectado.

Redescoberta autenticada, cache certificado, porta estável do serviço, diretório assinado e reconexão existente permanecem. Testamos novamente endereço antigo, proprietário reiniciando e voz/texto com proprietário temporariamente offline. A autorização offline é finita (até 15 minutos); permissões, novos ingressos e novas sessões ainda dependem da autoridade. Não há confiança em IP substituindo Ed25519.

## Interface e desempenho

Cabeçalhos de configurações e estados de conexão são componentes reutilizáveis. Formulários, sliders, campos, perfis, mensagens privadas, amigos/participantes, convites, gerenciamento e painéis usam a mesma linguagem visual da área de áudio, com cores próprias e ícones atuais. Navegação e painel de membros persistem no armazenamento do WebView do perfil; limpar esse cache pode redefinir apenas essas preferências visuais. Preferências de áudio/rede continuam no banco existente.

Polling visual continua pausado enquanto oculto; transporte, áudio, autorização e recebimento continuam ativos. Não há nova captura automática ao restaurar. A tela continua exigindo seleção/prévia/início e para ao ocultar/minimizar/sair conforme as regras existentes.

Medição disponível: seis backends Go independentes no mesmo PC, 32,33–33,92 MiB RSS e 4,31–6,38% de um núcleo lógico por backend no intervalo amostrado. São pacotes gerados com pontes Pion; não incluem auxiliares, WebView, microfone, codecs do renderer nem tela. Entrega de RTP entre pontes: média10,60ms/máximo21,21ms; não é latência boca-ouvido ou WAN. O teste completo durou95,21s, com10s de mídia medida, saída/reentrada e chat. Não é teste de cinco minutos. CPU/RAM da árvore completa com janela aberta/minimizada e captura física **não foram medidos**.

## Validação

Veja `PEETWOPEE-v0.8.1-VALIDACAO.json`. Passaram31 testes frontend, TypeScript/Vite, Go/vet, padrões/migração de rede, ACK sem bloquear áudio, três processos de comunidades, primeiro início integrado real/persistência/reinício, malha6, proprietário offline/reinício e regressões de voz/tela/chat com EasyTier real. Mídia foi gerada; não declarar testes físicos entre casas ou microfones audíveis a partir disso.

Permanecem pendentes renderização WebView2/layout em resoluções reais, dispositivos físicos, tela física/GPU, bandeja/minimização nativa, instalação completa, duas casas/CGNAT/relay real, consumo da árvore completa e Linux desktop/race. O ambiente atual bloqueia o início do navegador automatizado/alcance do localhost isolado; o harness visual preparado na versão anterior pode ser executado em um host QA apropriado, sem autorizar dispositivos reais automaticamente.

Teste físico: em perfil novo, confira integrada selecionada e estado do auxiliar. Escolha outro modo, reinicie e confira a mesma escolha mesmo se indisponível. Em perfil existente, confira a preferência antiga. Com dois PCs autorizados em chamada, troque entrada/saída, ajuste volumes/mute individual, atualize a lista, ative retorno e desconecte/reconecte o USB. Confirme que a chamada continua e não há eco ou segunda captura. Navegue por mensagens/comunidades, use convites antigos/novos e compartilhe uma janela de teste com seleção explícita. Minimize/restaure: voz permanece e tela não recomeça sozinha. Meça a árvore inteira de processos em janela aberta, minimizada, voz e voz+tela.

## Build

Compile frontend e testes, depois Go/vet. Windows: `scripts/build-desktop.ps1 -Output PeeTwoPee-v0.8.1.exe`, `scripts/package-windows.ps1 -DesktopExecutable build/bin/PeeTwoPee-v0.8.1.exe`, `tests/installer-smoke.ps1 -DesktopExecutable build/bin/PeeTwoPee-v0.8.1.exe`. A política revisada WebView2 Default/prompt-v1 permanece obrigatória. Setup privado sem assinatura de editor; payload/hash/PE verificados, sem alegar instalação/upgrade completo. Licenças e fontes LGPL correspondentes completas acompanham. Sem Npcap/Packet.dll/TUN. O bloqueio público por avisos Rust e a quarentena DHT continuam.
