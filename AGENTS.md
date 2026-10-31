# Instruções de desenvolvimento

- Preserve identidade Ed25519, contatos, histórico, caminho do perfil e compatibilidade Windows/Linux. Não altere configurações de VPN ou firewall sem autorização.
- Não adicione chaves, tokens, .env privados, perfis, bancos ou logs pessoais ao Git. Antes de staged/commit: scripts/audit-public.py, gitleaks via scripts/scan-secrets.py e revisão do diff.
- Compile frontend antes dos testes Go: npm ci, npm test, npm run build em frontend. Execute scripts/check-version.py, go test ./... e go vet ./.... No Linux use tags webkit2_41 e teste race no núcleo. Compile CLI Windows/Linux e desktop na plataforma nativa quando disponível; declare limites quando não disponível.
- VERSION, package.json/lock, main.go, installwin, recursos Windows, Wails e nome do setup devem coincidir. Tags são vMAJOR.MINOR.PATCH. Não retroceda a versão existente.
- Instalações preservam %APPDATA%/ProjetoP2P. Não use perfis reais em testes destrutivos. A verificação de payload não comprova instalação/atualização completa.
- Adaptador ativo não é peer conectado. Não afirme teste Radmin/Hamachi real a partir de fixtures. Preserve modo nativo, DHT, bootstrap e Circuit Relay.
- Não publique, crie repositórios, envie commits ou tags sem confirmação e destino explícito. O disparo manual de Release deve apenas gerar artefatos. Permissão contents: write apenas no publicador; nenhuma credencial pessoal no código.
- Não implemente voz ou blockchain nesta fase. Use somente mudanças relacionadas à tarefa.
