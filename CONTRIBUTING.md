# Contribuir

Use uma branch por mudança e explique problema, comportamento resultante e validação no pull request. Preserve perfis existentes e não inclua identidades, histórico, convites pessoais, tokens, configurações locais ou arquivos de build. Leia AGENTS.md e SECURITY.md.

## Ferramentas

Go 1.27.3, Node.js 24, Python 3 e Wails 2.17.0. Windows AMD64: WebView2 para executar o desktop. Linux Mint 22/Ubuntu 24.04: `sudo apt-get install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev`. O runtime Linux exige GTK3/WebKitGTK 4.1. Instale Wails com `go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0` e inclua o binário do Go no PATH.

## Verificar

```text
npm --prefix frontend ci
npm --prefix frontend test
npm --prefix frontend run build
python scripts/check-version.py
go mod verify
go test ./... -count=1 -timeout=180s
go vet ./...
python scripts/audit-public.py
```

Linux: use `GOFLAGS=-tags=webkit2_41` nas verificações Go e execute também `go test -race ./internal/... -count=1 -timeout=180s`. No Windows compile com `./scripts/desktop.ps1 -Installer` e `./scripts/build.ps1`; no Linux use `sh scripts/desktop-linux.sh` e `sh scripts/build.sh`. Gere pacotes com `python scripts/package-distribution.py windows` ou `linux` e `python scripts/collect-licenses.py`. A CI usa Python 3 nos dois sistemas.

Os testes de rede normais usam peers locais e não exigem participantes externos, VPN instalada ou infraestrutura pública. O roteiro com dois PCs é um teste manual separado; não marque como aprovado apenas por existir adaptador.

Antes do primeiro commit, instale `go install github.com/zricethezav/gitleaks/v8@v8.30.1` e rode `python scripts/scan-secrets.py`. A auditoria lê somente candidatos Git, inclusive arquivos staged; nenhuma ferramenta garante ausência de todo segredo. Revise `git diff --cached` e a lista de arquivos. Não inclua o diretório docs/evidence local, que contém caminhos da máquina. Commits devem descrever o resultado; não é obrigatório usar um prefixo específico.

Testes de instalação/atualização devem usar um usuário/VM de teste. Registre Peer ID, contato e mensagem antes/depois, sem publicar a chave. Não teste sobre dados de terceiros. Não altere redes ou firewall automaticamente.

Versões e publicação: docs/PUBLICACAO.md. Mudanças em dependências exigem lockfiles atualizados, verificação de vulnerabilidades e revisão das licenças. Não adicione voz ou blockchain sem novo escopo aprovado.
