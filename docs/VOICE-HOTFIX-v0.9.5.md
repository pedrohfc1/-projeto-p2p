# PeeTwoPee 0.9.5 — hotfix privado de autorização e reconexão de voz

Atualize todos os participantes, incluindo o proprietário. Instale sobre a versão anterior e preserve %APPDATA%/ProjetoP2P. Não recrie a identidade nem a comunidade para testar. Nenhuma release pública.

## Causas reproduzidas no código

1. O ciclo de renovação reutilizava um único prazo de dez segundos para sincronizar toda a comunidade e depois pedir a autorização. Uma sincronização bloqueada consumia o prazo antes do pedido assinado. Agora a renovação ocorre primeiro, com o mesmo prazo de dez segundos; a sincronização tem um orçamento separado de dois segundos. Não aumentamos indiscriminadamente os prazos de transporte.
2. A guarda modificava Expires de uma autorização assinada para zero. Isso corrompia a própria prova que deveria ser mantida para diagnóstico/recuperação. A prova permanece imutável; um estado separado pausa a mídia, fecha streams inválidos e conserva a participação até recuperação ou saída/revogação efetiva.
3. O roster reutilizava a versão antiga do convite/endereço quando o Peer ID e a sessão permaneciam iguais. Agora cada roster assinado atualiza a rota. Resultados de tentativas antigas são rejeitados por objeto do peer, sessão, geração, prazo e autorização; tentativas removidas são canceladas. Há uma tentativa por peer, com recuo de 1/2/4/8 segundos.
4. Respostas de renovação atrasadas podiam sobrescrever uma prova mais nova recebida durante a sincronização. A instalação de leases é monotônica por geração/sequência; uma geração diferente requer expiração posterior, além da verificação integral de assinatura, estado e vínculo das duas sessões. Uma prova válida recebida de outro membro pode recuperar o estado local sem mintar permissões ou ampliar a validade assinada.
5. A inicialização de um helper SOCKS segurava o mutex global do EasyTier durante a espera por readiness. Isso serializava peers independentes e atrasava renovação/reconexão. A inicialização agora é única por peer e ocorre fora do mutex global; fechamento aguarda as inicializações em curso. Helpers de dial com falha são reciclados sob cooldown, sem modificar o Radmin, rotas, DNS ou firewall.
6. Connectedness de libp2p não garantia um multiplexador utilizável. Somente falhas de abertura de stream por deadline invalidam a conexão, com cooldown de cinco segundos. Endereços integrados são novamente resolvidos pela identidade/rota autenticada e substituem o cache antigo; Noise continua validando o Peer ID.

Essas causas foram exercitadas por regressões locais. Sem os logs completos dos dois computadores, não se pode afirmar que expliquem isoladamente todos os erros observados em redes externas.

## Segurança e continuidade

A validade assinada de quinze minutos já existente não foi aumentada. Com o proprietário offline, uma sessão já autorizada pode continuar somente até a expiração da prova verificável e respeitando o estado autorizado local. Não há novos ingressos, renovação inventada ou extensão de prazo sem o proprietário. Revogação conhecida ou prova inválida pausa a mídia. A retomada exige prova válida, identidade e sessões corretas. Replay, sessão antiga, cadeia/raiz/epoch incorretos continuam rejeitados.

Um join explícito da mesma identidade autenticada pode substituir uma sessão anterior que caiu; renew da sessão antiga não recupera a ocupação. Participação, autorização, conectividade e erros de mídia têm estados separados. O codec Opus/20 ms e o transporte libp2p/TCP sobre EasyTier são preservados, assim como WebRTC local e a arquitetura de vídeo.

## Como comparar os logs

Compare flow nas duas pontas: é um resumo da mesma requisição, não uma credencial. source_session e target_session são resumos das sessões; stream identifica o stream local. origin=local registra ações locais; origin=remote aparece somente quando o erro do transporte identifica o lado remoto. EOF/timeout ambíguos permanecem classificados como falha de transporte. O fechamento do consumidor e o reset libp2p subsequente são eventos distintos, para não confundir limpeza com a causa inicial.

O painel exibe separadamente ponte local e pacotes TX/RX da rede. Captura ativa não é transmissão confirmada. RX é recepção pelo próprio peer; compare com TX e flow da outra ponta, pois o transporte de voz não adiciona confirmação de reprodução remota. Logs não incluem conteúdo de áudio, mensagens, SDP, convites nem tokens. O console aplica ocultação e aviso de cópia já existentes.

O WebView nem sempre fornece um contador bruto de frames capturados. Esse campo é declarado n/d; duração de amostras e pacotes Opus da ponte local são medições distintas e não substituem esse contador. Não inventamos frames. Os contadores de rede são efetivos; não são a contagem de pacotes produzidos apenas no navegador.

## Validação e limites

Consulte VOICE-HOTFIX-v0.9.5-VALIDACAO.json para resultados e hashes. Testes de integração usam processos/cores reais, criptografia e helpers auditados, mas mídia sintética e uma única máquina. Eles não comprovam dois microfones físicos, duas casas, CGNAT nem qualidade perceptiva. TCP pode sofrer head-of-line blocking em redes instáveis; nenhum resultado de latência do loopback equivale a captura até reprodução entre casas.

Teste manual: com todos na 0.9.5, compare sessões/flows nos dois lados, entre em chamada com dois e depois três participantes, interrompa e restaure a conectividade, deixe o proprietário temporariamente offline e confirme que somente provas ainda válidas mantêm a mídia. Confira TX/RX reais e o retorno do áudio sem recriar a comunidade. Não modifique o firewall automaticamente.
