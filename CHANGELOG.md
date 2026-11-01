## 0.6.0 — beta privado de voz para duas pessoas (2026-11-01)

- Convites compactos com políticas assinadas, cotas, expiração, status e revogação; legados continuam de uso único.
- Novas comunidades com 100 membros registrados, 50 canais de texto e 20 de voz; controle de salas e permissões no backend.
- Diário de texto assinado pelos autores e replicado entre membros nas novas comunidades, separado do registro de controle do proprietário.
- Captura explícita de microfone, seleção de dispositivos, Opus, Pion/DTLS-SRTP e sinalização autenticada libp2p; barra de chamada persistente, mute/deaf e saída.
- Encaminhador UDP EasyTier userspace dedicado à mídia, sem TUN/Npcap; reconexão e encerramento por revogação/expiração.
- Testes locais com dois processos e mídia sintética; testes de dispositivos físicos, instalação limpa e voz entre casas ainda pendentes. Malha de 3–6 e volume individual ficam para 0.6.1.
- Migração schema 3 preserva dados e impede downgrade silencioso. Release público continua bloqueado por avisos Rust e validações pendentes.

## 0.5.2 — beta privado integrado (2026-11-01)

- Chat e comunidades sobre libp2p/Noise via SOCKS/EasyTier sem TUN, com helper embutido, credenciais protegidas e convites assinados.
- Colisões de clientes etcat corrigidas com instâncias independentes e credencial de convite explicitamente reutilizável; permissões de comunidades preservadas.
- Reconexão e ACK idempotente após gravação; espera inicial ajustada à inicialização da malha.
- DHT provider afetada em quarentena por padrão; legado preservado com tag explícita, sem suprimir aviso de segurança.
- Sem Packet.dll/Npcap, drivers ou UPnP; fontes correspondentes e licenças no instalador. Publicação bloqueada e validação entre casas pendente.

# Histórico de mudanças

O projeto usa versionamento semântico. As versões abaixo descrevem desenvolvimento local; não significam que existam releases públicas no GitHub.

## [Não lançado]

- Infraestrutura de distribuição: CI Windows/Linux, Release por tag e build manual sem publicação, Actions fixadas por SHA, checksums e inventários de licenças.
- Auditoria de candidatos Git, gitleaks, análise de vulnerabilidades e documentação pública em português.
- Licença MIT confirmada, testes do frontend e consistência de versão.

## [0.5.2] — 2026-11-01 — experimental, não lançado

- Laboratório EasyTier 2.6.4 no-TUN/SOCKS5, com stream libp2p Noise isolado e identidade do perfil preservada.
- UI experimental, diagnóstico de peers/custos, métricas Go/auxiliar e credenciais somente na sessão.
- Processos com hashes fixos, Job Object Windows, limites, reinício limitado e encerramento após crash do hospedeiro.
- Integração local real: echo autenticado, canário de porta alheia bloqueado, Peer ID incorreto recusado e relay(2) com três auxiliares.
- Auxiliares/driver não redistribuídos no pacote; preparador explícito verificado para desenvolvimento. LGPL/notices/fontes correspondentes ainda precisam de fechamento para distribuição turnkey.
- Chat/comunidades continuam na rede anterior; sem alegação de teste entre casas, CGNAT, Linux nativo ou produto integrado pronto.

## [0.5.1] — 2026-11-01

- Envelope compacto Protobuf/Base64URL para contatos e comunidades, com assinatura e validação preservadas.
- Leitor legado, conversão para 0.5.0, QR Code local e testes entre backends antigo/novo.
- Estrutura local de códigos imprevisíveis, registros assinados, mirrors autorizados e revogação; sem resolução DHT/global.
- Exemplos medidos: contato Radmin 553→193 caracteres; comunidade 869→327. Perfis e esquema 2 preservados.

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
