# Histórico de mudanças

O projeto usa versionamento semântico. As versões abaixo descrevem desenvolvimento local; não significam que existam releases públicas no GitHub.

## [Não lançado]

- Infraestrutura de distribuição: CI Windows/Linux, Release por tag e build manual sem publicação, Actions fixadas por SHA, checksums e inventários de licenças.
- Auditoria de candidatos Git, gitleaks, análise de vulnerabilidades e documentação pública em português.
- Licença MIT confirmada, testes do frontend e consistência de versão.

## [0.5.0] — 2026-11-01

- Comunidades com identidade única, metadados, membros, owner/moderator/member e convites assinados de uso único.
- Canais de texto, definições de voz sem áudio, histórico paginado e registro autenticado com cursor, hash anterior e deduplicação.
- Outbox durável, recuperação após reinício, revogação e controle de conflitos; dono é sequenciador online nesta etapa.
- Interface com barra de comunidades, canais, chat, participantes e configurações; DM e configurações VPN preservados.
- Migração bbolt versionada 1→2 com testes de preservação e downgrade seguro condicionado a buckets vazios.
- Correção de replay de ID no histórico individual, limites de leitura da chave e contatos não solicitados, CSP e file drop desativado.
- Telemetria inicial, testes de integração Radmin locais, preparação opcional de assinatura Windows e documentação E2EE/voz.
- Build Windows de testes sem assinatura; sem publicação. GO-2024-3218 mantém gate bloqueado. Comunidades ainda sem E2EE de grupo.

## [0.4.1] — 2026-10-31

- Escolha explícita da VPN ao gerar/receber convites; aviso de VPN detectada mas não selecionada no modo nativo.
- Persistência do GUID e porta da interface selecionada, erro claro quando a configuração foi salva mas o nó não iniciou.
- Filtros de endereço comuns entre convite, discagem e reconexão: rejeição de APIPA, separação Hamachi/Radmin e anúncio apenas da porta real de escuta.
- Diagnóstico estruturado com modo, interface, endereço, porta, destino, handshake e ping.
- Testes de regressão e smoke opcional com adaptador Radmin real, dois perfis locais, reinício e mensagem persistida.

## [0.4.0] — 2026-10-31

- Arquitetura genérica de adaptadores virtuais Windows com Hamachi e Radmin VPN independentes.
- Seleção automática do adaptador da VPN escolhida ou seleção manual; exibição de interface, índice, IPv4 e estado.
- Escuta, origem TCP e convites vinculados ao endereço selecionado, com escopo assinado e ping autenticado.
- Preservação de identidade, contatos, histórico e modo libp2p nativo.

## [0.3.0] — 2026-10-31

- Integração opcional com Hamachi, monitoramento da interface e instalador por usuário em Go 64 bits.

## [0.2.0] — 2026-10-31

- Desktop Wails/React, histórico e contatos bbolt, convites assinados e diagnóstico de Circuit Relay v2.

## [0.1.0] — 2026-10-31

- Núcleo Go/libp2p, identidade Ed25519 persistente, DHT, bootstraps, texto assinado, limites de recursos e testes com três nós.
