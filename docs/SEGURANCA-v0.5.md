# Auditoria de segurança 0.5

Revisão de código e testes locais em 2026-10-31/2026-11-01. Não é auditoria externa nem certificação. Escopo: identidade, convites, protocolos, persistência, permissões, frontend e distribuição. [Validação e limitações](VALIDACAO-v0.5.md).

## Correções implementadas

| Achado | Correção e evidência |
|---|---|
| Um autor autenticado podia reapresentar ID de DM com conteúdo diferente após reiniciar o cache | O banco exige igualdade dos campos imutáveis antes de atualizar estado; teste de replay após reinício |
| Leitura de chave sem limite específico e aceitação implícita de tipo | Limite de 64 KiB, Ed25519 obrigatório, recusa de symlink final e preservação do arquivo inválido; testes |
| Remetentes desconhecidos podiam ocupar contatos locais | Novos contatos não solicitados identificados, máximo 16; contatos explicitamente adicionados continuam permitidos; teste de capacidade |
| Nova superfície de comunidades | Operações tipadas e assinadas, autor vinculado ao Peer ID Noise, papéis validados no backend, tickets limitados e revogáveis, leitura só por membros atuais, frames e filas limitados |
| Conteúdo remoto no frontend | React renderiza texto escapado, teste de injeção SSR, CSP, nenhum HTML remoto, upload ou acesso a mídia; file drop nativo desativado |
| Cache WebView2 compartilhado entre perfis | Cache dentro do perfil explicitamente resolvido; não muda chave/banco antigos. Inspeção visual nativa ainda pendente |

Ed25519 persiste no perfil. Windows usa DPAPI; Linux restringe permissões. Isso não protege contra malware no mesmo usuário, comprometimento do SO ou extração da sessão em execução. As permissões Windows também dependem das ACLs herdadas. O banco bbolt e seu conteúdo **não são criptografados em repouso**. Não foi adicionado SQLite.

Noise protege os enlaces libp2p e autentica chaves dos peers; nomes não são identidade civil. Convites devem ser conferidos por canal confiável. Radmin/Hamachi não são configurados pelo aplicativo e a existência do adaptador não comprova alcance. A seleção explícita e os filtros de endereços da 0.4.1 permanecem, assim como bootstrap, DHT e Circuit Relay no modo nativo.

## Protocolo de comunidade

Domínios de assinatura separados para operações, commits e convites. ID derivado do conteúdo, nonce aleatório, chave pública vinculada ao autor, proprietário e raiz fixados, sequência/hash anterior verificados. A assinatura é do conteúdo canônico produzido pelo código; JSON externo desconhecido e campos incompatíveis com a operação são recusados. Não se aceita snapshot remoto como prova de permissão.

O dono serializa commits e revalida papel/canal/época no recebimento. Membro não promove a si próprio; moderador não remove/promove dono ou outro moderador. Tickets dão papel membro, são assinados pelo dono, de uso único e expiram; mudanças de papéis/remoções invalidam a época anterior. Reentrada exige ticket atual. Usuário removido perde sincronização futura, mas mantém cópias anteriores.

Frames: requisição 32 KiB, resposta 256 KiB, leitura delimitada e prazo de 10 s. Portão limita 16 streams simultâneos, 2 por peer e 256 rastreadores, com expiração; rajada 64 e reposição 12/s por peer. Histórico, membros, canais, registro e filas têm quotas. Escritas são transacionais e só removem pendência após receber commit verificável. Essas medidas não eliminam DDoS, ataques Sybil ou esgotamento de disco por outros aplicativos. Não se presume que o dono seja honesto; assinatura de dono detecta adulteração, mas não resolve equivocações entre réplicas isoladas.

## Privacidade que ainda não existe

**Comunidades 0.5 não têm E2EE de grupo.** O dono e membros autorizados processam texto claro. Cada conexão tem Noise, mas o dono vê o conteúdo que sequencia. Não há segredo futuro/pós-comprometimento de grupo, rotação MLS nem arquivos distribuídos criptografados. [Plano E2EE](CRIPTOGRAFIA-E2EE-PLANO.md). Salas de voz não capturam microfone; nenhuma chamada, vídeo, tela ou blockchain foi implementada.

## Dependências e bloqueios

`go mod verify` e `npm audit` foram executados; npm não apontou vulnerabilidades. `govulncheck` retornou falha com **GO-2024-3218 alcançável em go-libp2p-kad-dht**, sem versão corrigida informada no aviso consultado. O gate existente continua bloqueando publicação; não foi suprimido. No modo VPN o DHT permanece desligado, o que reduz essa exposição nesse caminho, sem sanar o modo nativo. O relatório também distingue avisos de módulos sem chamada identificada; ausência de chamada reportada não equivale a ausência de risco. [Aviso oficial](https://pkg.go.dev/vuln/GO-2024-3218).

Auditoria de candidatos Git e gitleaks verificam fontes antes de commit/pacote. Não incluímos perfil, chaves, bancos, logs privados, node_modules ou credenciais no ZIP fonte. Wails conserva a validação padrão de origem; CSP não usa script remoto ou wildcard de origem. O nonce do helper de testes não é incorporado ao app de produção. As APIs de comunidade usam a chave do processo, sem aceitar identidade/papel escolhidos pelo chamador.

## Windows e publicação

Aplicativo e instalador locais estão **NotSigned**. Não há credencial de assinatura fornecida. A consulta ao Defender ficou indisponível neste ambiente: não comprova arquivo limpo e não permite classificar o aviso relatado como falso positivo. SmartScreen, ausência de assinatura e detecção real são diagnósticos diferentes. [Procedimento e preparação Authenticode](ASSINATURA-WINDOWS.md).

Não alteramos firewall, Defender, SmartScreen, VPNs ou exceções. Não houve publicação GitHub, tag ou release. Caso haja detecção real, interromper distribuição e investigar arquivo/hash/ameaça antes de nova tentativa. Hash consistente comprova integridade em relação à cópia de referência, não inocuidade.
