# DHT: causa, risco e quarentena do beta 0.5.2

Investigação em 2026-11-01. GO-2024-3218/CVE-2023-26248 é ataque Sybil contra resolução de providers Kademlia, não um erro de firewall nem simples log operacional. Nós com IDs próximos à chave de rendezvous podem dominar a descoberta e ocultar participantes. Não demonstra execução de código ou recuperação da chave Ed25519. O aviso Go cobre todas as versões sem versão corrigida conhecida; o advisory GitHub descreve versões antigas e nenhuma versão patched. Não presumimos que atualizar a dependência resolva o desenho do protocolo. [Go](https://pkg.go.dev/vuln/GO-2024-3218), [advisory](https://github.com/advisories/GHSA-mqr9-hjr8-2m9w), [pesquisa](https://arxiv.org/abs/2307.12212).

No app, internal/discovery.Find usa Advertise/FindPeers para uma chave de rendezvous fixa. DisableValues bloqueia registros arbitrários de valores, **não providers**, portanto não eliminava a exposição. Prefixo de protocolo próprio separa a rede IPFS, mas não impede Sybil entre participantes do nosso protocolo. Noise e assinaturas impedem aceitar identidade forjada, porém não garantem disponibilidade da descoberta. O erro announce/failed to find any peer in table é outro assunto: nó sozinho/sem tabela, não prova de ataque nem causa do gate govulncheck.

## Medida aplicada antes de integrar o novo transporte

O build beta padrão não importa nem instancia go-libp2p-kad-dht. Interface/factory isolam o recurso; implementação antiga permanece em dht_legacy.go com tag explícita legacydht para regressão/pesquisa. Não existe fallback automático que ative a DHT afetada. Bootstrap explícito, convites, conexões libp2p, Noise, Circuit Relay e modos VPN continuam disponíveis. O beta troca descoberta provider por endpoints autenticados explícitos; o transporte integrado usa descoberta EasyTier/etcat, sujeita às limitações próprias documentadas.

Não é uma correção upstream do algoritmo. É redução da superfície do **binário beta**: código afetado não é linkado, em vez de ignorar o aviso. Não há filtro de ID de CVE ou allowlist no scanner. govulncheck no build padrão retornou zero vulnerabilidades alcançáveis; a dependência permanece em go.mod para o build legado e pode aparecer no inventário. O build legacydht continua afetado e proibido para distribuição deste beta. Não anunciar que Kademlia ficou imune a Sybil.

## Regressão

Teste beta exige DHT nil, estado desativado e entrega de texto autenticado por conexão explícita. Teste de relay/failover padrão usa relays configurados; com legacydht também testa descoberta de relay por DHT. O teste específico de descoberta de três nós roda somente com a tag e é executado na CI/regressão legada. O smoke CLI padrão declara conexão explícita; --legacy-dht preserva o roteiro antigo. Histórico, identidade, contatos e protocolos de mensagens não foram migrados.

Publicação pública permanece impedida por PRIVATE_BETA_ONLY e condição do job publish, independentemente do scanner padrão limpo. Não houve publicação. Retomar DHT por padrão exige revisão de mitigação contra Sybil/eclipses, diversidade e disponibilidade independente, sem esconder a exposição em build tags de release.
