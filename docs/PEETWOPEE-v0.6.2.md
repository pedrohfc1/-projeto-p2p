# PeeTwoPee 0.6.2 — conectividade de comunidades (beta privado)

## Causa e correção

O modo nativo escutava TCP em porta dinâmica e seus anúncios incluíam loopback/LAN. Os convites de comunidade guardavam esse retrato; a réplica reutilizava os endpoints sem acompanhar a rota integrada e o peerstore conservava endereços anteriores. A descoberta de membros dependia de um diretório do proprietário em memória, com validade local de dez minutos. Isso explicava o dial para `127.0.0.1:60129`/LAN e o backoff, sem constituir uma negação de acesso.

No modo integrado, o listener userspace e o anúncio agora usam uma porta estável derivada da identidade (12000–59999) e um IPv6 lógico derivado do Peer ID. Esses valores não são rotas do Windows/Linux. O dialer conecta por SOCKS a `server.etcat` dentro da rede privada autenticada do destinatário. Um convite antigo com porta anterior é resolvido para o serviço estável quando existe a capacidade EasyTier previamente autenticada. O ajuste de porta do modo nativo não muda esse endpoint integrado; a porta efetiva aparece no diagnóstico de escuta.

Uma conexão libp2p já autenticada com o proprietário fixado também pode fornecer o caminho atual após uma troca manual de transporte; o escopo antigo do convite não altera as permissões da comunidade. Isso preserva o uso manual de interfaces/Radmin sem confiar somente no IP.

A rota EasyTier continua privada e é persistente no perfil. Não existe senha global compartilhada. O token nunca substitui a identidade Ed25519: libp2p/Noise precisa autenticar o Peer ID esperado. Só endereços de convites assinados por esse mesmo participante são aceitos no diretório. No modo integrado, IPs físicos antigos são substituídos pelo endpoint lógico, sem fallback silencioso para a LAN. Falta de capacidade autenticada é erro de transporte, não permissão para inventar uma rota.

Comunidades antigas podem utilizar uma capacidade integrada conhecida do mesmo proprietário sem recriar comunidade ou identidade. Quando essa capacidade nunca foi recebida, é necessário importar uma vez um convite integrado atual do proprietário, verificando o mesmo Peer ID. Não é possível descobrir uma rede privada apenas com um ID e sem qualquer contato/capacidade/infraestrutura alcançável.

No modo nativo, chamadas de comunidade filtram link-local, loopback de listener não local e redes privadas fora das sub-redes locais. Loopback permanece disponível para listeners explicitamente locais; interfaces LAN/VPN selecionadas continuam funcionando. Endereços públicos/relay compatíveis e conexões autenticadas existentes permanecem disponíveis. A lista de candidatos inclui endereços atuais do peerstore; candidatos incompatíveis são removidos também do cache de reconexão. O backoff é liberado em tentativas limitadas a uma por cinco segundos por peer, sem loop de discagem agressivo. Isso não torna um computador atrás de NAT publicamente alcançável.

## Comunidades e dados locais

Rotas de membros são verificadas, limitadas a membros atuais, seladas em disco e recuperadas após reiniciar. Os membros trocam seus próprios anúncios assinados e retransmitem anúncios assinados de outros membros. Um anúncio com validade mais antiga não substitui um mais novo já conhecido. O anúncio do proprietário atualiza o endpoint da comunidade sem alterar raiz, identidade, permissões ou histórico.

A sincronização entre membros também pode transportar até quatro eventos de controle por resposta. Cada evento continua exigindo assinatura do proprietário, raiz fixada, cadeia contígua e validação local. Não se confia no papel ou snapshot que um membro afirma ter. As respostas têm limites de tamanho e cursores; conteúdo não confirmado não é tratado como recibo do proprietário.

A leitura do histórico autorizado é local. Texto nas comunidades novas já utiliza diário assinado pelos autores. Nas comunidades legadas, uma falha de conectividade durante o envio pode usar o mesmo texto/ID da operação em um diário entre membros, mantendo a outbox para confirmação posterior. A interface elimina duplicatas entre diário, outbox e registro do proprietário. Assinatura do autor não é confirmação do proprietário. Ingressos, emissão/revogação de convites e alterações administrativas continuam dependentes da autoridade; as permissões não são ampliadas pelo fallback.

Falhas de transporte, endereço inválido, tempo esgotado, autoridade não alcançável e revogação confirmada têm estados distintos. Autoridade não alcançável não comprova que o programa esteja desligado. A interface informa sincronização pendente sem desabilitar o histórico e o texto autorizado entre membros.

## Voz durante falha do proprietário

O transporte de áudio funcional é preservado: Opus/20 ms, streams libp2p criptografados, até seis participantes, mute/deaf e volume individual. Sessões existentes recebem do proprietário uma autorização assinada válida por **até 15 minutos**. A presença mantida pela autoridade online continua expirando em 45 segundos sem renovação. Uma chamada já autorizada pode continuar e reconstruir seus streams dentro da janela assinada mesmo se a sincronização do proprietário estiver bloqueada. A manutenção da malha não espera a resposta da autoridade.

A proteção contra replay cobre toda a janela assinada, com orçamento de nonces separado por participante; um membro não consome o orçamento dos demais.

Não há renovação local ou assinatura falsa: ao esgotar a janela, a mídia pausa; voltar a alcançar a autoridade permite renovar. Novas chamadas/sessões ainda precisam do proprietário. Remoções, alterações de permissões e arquivamento conhecidos são aplicados pela validação local e podem interromper a mídia antes do prazo. Não existe revogação instantânea entre réplicas totalmente isoladas: uma autorização já emitida pode continuar utilizável durante sua janela. Texto offline usa o último registro de autorização verificado e pode ter propagação de revogação atrasada. Isso é uma limitação de consistência offline, não autorização para ingressar ou alterar permissões.

## Diagnóstico

Logs resumidos `community_connectivity` identificam transporte, tipo do caminho, quantidade de endpoints e resultado (`connected`, `timeout`, `dial_backoff`, `dial_failed`, falta de rota). Estados repetidos são limitados a um registro por 30 segundos por peer; mudanças podem aparecer imediatamente. Não registram tokens, convites, conteúdo de texto/áudio ou IDs completos. A tela de estado da rede mostra endpoints/porta reais da conexão; uma rota lógica ou socket SOCKS não comprova conexão direta. Direto/relay só é exibido quando informado pelo auxiliar de forma verificável.

## Testes disponíveis

- Frontend: preferências/volume, monitor local, permissões e polling; build TypeScript/Vite.
- Go: suíte completa, vet, resolução integrada a partir de caches LAN/portas antigas, separação loopback/internet e persistência selada de capacidades.
- Legado: três nós locais, proprietário offline, texto entregue entre membros, reinício/identidades preservadas e confirmação posterior sem duplicação; paginação e outbox preservadas.
- EasyTier real: três nós autorizados, cache da comunidade e peerstore contaminados com endpoints antigos, proprietário offline por mais de 65 segundos, voz bidirecional e texto entre membros, interrupção da conexão de mídia, retorno do proprietário com ajuste de porta alterado e ingresso de outro participante pelo convite compacto multiuso original.
- Regressão de voz: assinatura, mute/deaf, reparo de ponte local, reconexão, texto simultâneo e revogação. Processos/áudio sintéticos não equivalem a microfones físicos.
- Instalador: metadados, hashes e conteúdo; PE sem dependência Packet.dll/Npcap; helper e fontes correspondentes/licenças preservados.

Resultados finais ficam em `PEETWOPEE-v0.6.2-VALIDACAO.json`. Testes entre casas, mudança real de IP WAN, CGNAT, queda física de relay público, instalação/atualização completa e microfones físicos permanecem pendentes. Nenhuma configuração de firewall, VPN, TUN, DNS ou rotas foi alterada. A porta estável precisa estar livre localmente; conflito causa falha explícita, sem mudar silenciosamente o anúncio.

## Testar em PCs de redes diferentes

1. Feche todas as instâncias e instale o beta em todos os PCs. Preserve `%APPDATA%/ProjetoP2P`; confira Peer IDs e histórico antes/depois. Não use perfis de teste sobre o perfil real.
2. Selecione **EasyTier integrado** nos PCs. Não é necessário Radmin/EasyTier externo. Aguarde o auxiliar e a conexão; se o relay comunitário estiver indisponível, registre a falha e utilize apenas alternativas autorizadas/configuradas. Não abra portas nem mude firewall automaticamente.
3. O proprietário abre a comunidade existente e gera um convite compacto multiuso atual para novos membros. Compartilhe-o por canal privado. Membros já cadastrados com rota antiga incompatível podem importar um convite integrado de contato do mesmo proprietário; não recriem a comunidade nem a identidade.
4. Com três PCs, confirme ingressos assinados, texto em canais e chat individual. Inicie a voz por ação explícita, com fones. Feche o proprietário por 70 segundos: os dois membros devem manter texto e a chamada existente; uma nova chamada não é prometida offline.
5. Interrompa a rede de um membro brevemente, restaure e confirme recuperação da mídia/chat. Reabra o proprietário, sincronize, confirme histórico/IDs e experimente o mesmo convite ainda válido. Repita após mudar a conexão WAN do proprietário. A rota EasyTier depende da mesma identidade/capacidade persistente e de infraestrutura alcançável.
6. Confira `Estado da rede`, `authorityState` e os resultados de reconexão. Não confunda ícone de adaptador, histórico local ou presença em sala com mídia funcionando. Teste entrada/remoção de um membro e confirme que permissões revogadas não voltam por convite antigo.

## Distribuição

Somente instalador beta privado; nenhuma publicação automática. DHT legado vulnerável permanece em quarentena; alertas Rust não resolvidos continuam bloqueando release pública. O auxiliar auditado é o mesmo, com hash revisado, fontes correspondentes e licenças incluídas. Não se redistribui Npcap. Todos os participantes devem atualizar para 0.6.2: clientes antigos não conhecem a nova porta integrada nem a janela de voz offline.
