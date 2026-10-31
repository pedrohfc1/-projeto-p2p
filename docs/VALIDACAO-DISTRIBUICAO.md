# Validação da preparação pública — 31/10/2026

Versão 0.4.0, licença MIT confirmada. Nenhum código, token, instalador ou release foi enviado ao GitHub. O destino remoto ainda não foi informado.

## Executado localmente

- Consistência SemVer entre VERSION, aplicativo, interface, Wails, instalador e tag v0.4.0; tags divergentes rejeitadas.
- 22 testes Go e go vet; verificação de todos os módulos.
- 4 testes da infraestrutura: rejeição de tag divergente, bloqueio de .env forçado no staged, detecção de credencial sob nome inocente sem imprimir seu valor e aceitação de fonte normal.
- 3 testes frontend reais: rótulos de VPN, limite UTF-8 de mensagens e bloqueio de envio vazio/sem contato/ocupado; TypeScript/Vite compilados.
- Compilação Wails desktop Windows AMD64 e CLIs Windows/Linux.
- Instalador Windows real recompilado com nome completo v0.4.0; execução --verify-payload e comparação SHA-256 do app incorporado. Não é portátil renomeado. Nenhuma VPN incluída; bootstrapper WebView2 Microsoft validado.
- Auditoria de candidatos Git sem caminhos privados/artefatos proibidos e gitleaks sem segredos detectados. Logs brutos da máquina excluídos do Git, fontes/pacotes com seleção explícita.
- npm audit: zero vulnerabilidades informadas no momento da execução.
- actionlint 1.7.12: workflows válidos; Actions oficiais fixadas por SHA consultado na API GitHub.
- CLI Linux empacotado em tar.gz com ELF AMD64 e permissão de execução; não é desktop.

Logs locais em docs/evidence/distribution (ignorados); relatórios revisados em docs. Regressão executada com sucesso: três processos reais com DHT/mensagens/reconexão, daemon relay/reservas/limites e API desktop com convites/contatos/histórico após crash/reinício.

## Achado de segurança que bloqueia publicação

Govulncheck 1.9.0 com análise de símbolos/caminhos encontrou **GO-2024-3218**, CVE-2023-26248, na dependência go-libp2p-kad-dht 0.43.0. A base Go informa todas as versões afetadas e nenhuma corrigida conhecida: https://pkg.go.dev/vuln/GO-2024-3218 . O aviso descreve censura/disponibilidade da descoberta DHT; não deve ser apresentado como ausente por funcionar no modo VPN. A DHT é desativada nesse modo, mas o modo nativo continua disponível.

O gate security permanece com falha por esse achado e impede publicação automática; builds para testes continuam separados. Não foi adicionada supressão. É necessário resolver/mitigar ou obter uma decisão explícita e documentada do mantenedor sobre a política antes de uma release pública.

A primeira tentativa com govulncheck 1.1.4 encontrou incompatibilidade interna com Go 1.27. Foi substituída pela versão 1.9.0, que executou a análise completa. Outros 8 avisos foram classificados pela ferramenta como presentes em módulos mas sem chamadas afetadas aparentes; isso não equivale a auditoria independente.

## Não executado

- Workflows no GitHub: sem repositório/destino e sem publicação autorizada.
- Build/execução desktop Linux Mint: ambiente Windows sem WSL/distribuição Linux disponível. O workflow usa runner Ubuntu 24.04 e GTK3/WebKitGTK 4.1 para gerar o tar.gz desktop nativo.
- Instalação/atualização/desinstalação completas, atalhos e janela WebView2 em Windows 10/11 de aceitação. Payload verificado não comprova essas ações; testes do backend comprovam persistência do perfil, não o fluxo completo do instalador.
- Conexão Radmin real entre dois PCs Windows; não há adaptador Radmin ativo aqui. Os testes por fixtures e o roteiro manual permanecem distintos.
- Assinatura digital de distribuição: não existe certificado próprio. Não desative proteções.

## Aceitação antes da publicação

Executar workflow_dispatch, resolver o gate de segurança e conferir ambos os builds. Instalar/atualizar com usuário de teste preservando Peer ID, contato e mensagem; desinstalar preservando o perfil; testar Linux Mint e ping/mensagens bidirecionais entre PCs da mesma VPN. Depois fornecer URL real, revisar arquivos/autoria Git e autorizar o push/tag/publicação. Configurar Required reviewers no ambiente github-release; o YAML sozinho não ativa aprovação.

O instalador também incorpora e instala LICENSE.txt (MIT) e THIRD_PARTY_LICENSES.zip com avisos/inventários de dependências. A verificação de payload confere os hashes desses arquivos. O desinstalador remove esses arquivos do programa, sem acessar o perfil. ZIP portátil e tar.gz também carregam as licenças.
