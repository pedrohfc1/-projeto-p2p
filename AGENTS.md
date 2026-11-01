# Instruções de desenvolvimento

- Preserve identidade Ed25519, contatos, histórico, caminho do perfil e compatibilidade Windows/Linux. Não altere configurações de VPN ou firewall sem autorização.
- Não adicione chaves, tokens, .env privados, perfis, bancos ou logs pessoais ao Git. Antes de staged/commit: scripts/audit-public.py, gitleaks via scripts/scan-secrets.py e revisão do diff.
- Compile frontend antes dos testes Go: npm ci, npm test, npm run build em frontend. Execute scripts/check-version.py, go test ./... e go vet ./.... No Linux use tags webkit2_41 e teste race no núcleo. Compile CLI Windows/Linux e desktop na plataforma nativa quando disponível; declare limites quando não disponível.
- VERSION, package.json/lock, main.go, installwin, recursos Windows, Wails e nome do setup devem coincidir. Tags são vMAJOR.MINOR.PATCH. Não retroceda a versão existente.
- Instalações preservam %APPDATA%/ProjetoP2P. Não use perfis reais em testes destrutivos. A verificação de payload não comprova instalação/atualização completa.
- Adaptador ativo não é peer conectado. Não afirme teste Radmin/Hamachi real a partir de fixtures. Preserve modo nativo, DHT, bootstrap e Circuit Relay.
- Não publique, crie repositórios, envie commits ou tags sem confirmação e destino explícito. O disparo manual de Release deve apenas gerar artefatos. Permissão contents: write apenas no publicador; nenhuma credencial pessoal no código.
- A v0.6 autoriza voz real após ação explícita do usuário; não implemente blockchain ou tela. Use somente mudanças relacionadas à tarefa.

- Comunidades: nunca confiar em snapshot/papel remoto sem registro assinado; manter chave de autor ligada ao Peer ID, raiz/owner fixados e cadeia contígua. Não confirmar outbox só por ACK sem commit verificado. Preservar migrações e histórico legado.
- Documentar dependência do dono online e ausência de E2EE de grupo; definições voice não autorizam captura de mídia. Testar testes/community_desktop_smoke.py sem exigir VPN na CI normal; --radmin é integração opcional real.

- EasyTier 0.5.2 integra chat/comunidades por SOCKS userspace; preservar implementação de laboratório somente com tag legacyeasytierlab. Não ativar TUN, UPnP, rotas/DNS/firewall ou serviços. Auxiliar somente com hash revisado; nunca tokens/configuração privada em diagnóstico. Redistribuição exige fontes correspondentes completas e notices LGPL no instalador. Integração real opcional via P2P_ETCAT_TEST_DIR; não apresentar testes locais como CGNAT/duas casas. DHT legado afetado permanece somente sob tag legacydht; beta padrão em quarentena. PRIVATE_BETA_ONLY bloqueia publicação.
