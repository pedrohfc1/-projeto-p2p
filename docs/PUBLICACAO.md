# Publicação no GitHub

## Estado atual

Código local preparado em Git/main, licença MIT confirmada, versão mantida em 0.4.0. Não foi criado repositório remoto, feito push, enviada tag ou publicada release. Destino informado: https://github.com/pedrohfc1/-projeto-p2p.git ; remoto local origin configurado, sem envio. Não é necessário token pessoal nos workflows.

A URL real já foi informada; antes de publicar, ainda é necessário confirmar o envio. A versão 0.4.0 não foi retrocedida para 0.3.0: ela já inclui as funcionalidades Radmin da 0.4.

**Bloqueio de segurança conhecido:** govulncheck 1.9.0 identifica GO-2024-3218 no Kademlia DHT, sem versão corrigida conhecida. A CI mantém o achado visível e o job de segurança falha; o job publicador não executa enquanto esse gate falhar. Não há exceção silenciosa nem suppressão. Resolver a dependência/mitigação ou decidir uma política de exceção explícita, fundamentada e revisada pelo mantenedor requer trabalho separado antes da publicação. Em VPN a DHT está desativada, mas a distribuição conserva o modo nativo. Consulte SECURITY.md.

## Preparar o destino (manual, somente após autorização)

1. Confirme nome/visibilidade e crie um repositório vazio na conta pretendida. Não inicialize outro README/licença remotos se for enviar este histórico local.
2. Configure proteção da main para exigir revisão e CI. Habilite Private vulnerability reporting, secret scanning/push protection quando disponíveis e Dependabot.
3. Em Settings → Environments crie `github-release` e configure **Required reviewers**. O workflow referencia esse ambiente, mas o YAML sozinho não cria a exigência de aprovação: sem configuração no GitHub essa proteção não existe. Limite as tags permitidas conforme a política do projeto. Não avance para tags sem essa revisão.
4. Use a permissão padrão de Actions somente leitura. O YAML dá contents: write exclusivamente ao job publish; não adicione PAT em código, variáveis de frontend ou arquivos .env.
5. Confira o link `/releases/latest` já atualizado no README e confirme o envio. Links relativos internos já funcionam no GitHub.

Fontes oficiais: https://docs.github.com/en/actions/reference/security/secure-use e https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments . Actions oficiais foram verificadas na API GitHub e fixadas por SHA; Dependabot propõe atualizações para revisão.

## Git local e primeiro envio

Confira a identidade de autoria desejada com `git config user.name` e `git config user.email` antes de criar commits públicos. Use um endereço noreply GitHub se quiser evitar exposição do e-mail. Não alteramos a configuração global do Git.

```sh
python scripts/audit-public.py
python scripts/scan-secrets.py
git status --short
git diff --cached --stat
```

A auditoria considera tracked, staged e untracked não ignorados, bloqueia caminhos privados/artefatos e não imprime conteúdo de possíveis segredos. Logs locais em docs/evidence e dados privados ficam fora do Git. Revise os fontes e staged diff; scanners não garantem ausência de todo segredo. Se um segredo tiver sido exposto, remova-o e revogue/rotacione, não apenas acrescente ao .gitignore.

O repositório foi inicializado em main. Se ainda não houver commits, prepare um commit inicial único com código, testes, licença e infraestrutura auditados; em seguida use commits por mudança. Nunca inclua instaladores/binários como arquivos do repositório: eles pertencem a Releases/Actions artifacts.

Com o destino real autorizado, os comandos são:

```sh
git add .
git diff --cached --check
git commit -m "Preparar Projeto P2P 0.4.0 para distribuição"
git remote get-url origin
git remote -v
git push -u origin main
```

origin já foi configurado localmente. Confira seu destino antes de executar o push. Nenhum desses envios foi executado nesta preparação.

## Testar workflow sem publicar

Depois do envio autorizado, abra Actions → Release → Run workflow na main. workflow_dispatch executa validação e builds, gera artefatos e nunca publica, inclusive se iniciado sobre uma tag. O job Windows gera instalador real e ZIP portátil; o Linux gera desktop tar.gz nativo. O job assemble agrega arquivos, fonte auditada, inventários e SHA256SUMS. Artefatos ficam disponíveis por 14 dias.

O gate de vulnerabilidades é separado dos builds: com o aviso DHT atual, a execução fica com falha de segurança, mas os builds podem completar e disponibilizar artefatos para teste. Assemble tenta agregar os resultados mesmo com gate de segurança falhando e exige instalador Windows e desktop Linux presentes; não publica um conjunto incompleto. Se algum build falhar, corrija antes de prosseguir. Este workflow ainda não foi executado no GitHub nesta preparação.

Faça a aceitação Windows 10/11: instalar com usuário comum, verificar menu Iniciar/atalho opcional, atualizar preservando Peer ID/contatos/mensagens, desinstalar preservando perfil; WebView2 presente/ausente. Teste também desktop Linux Mint e dois PCs Radmin/Hamachi conforme os guias. A verificação de payload não substitui essas etapas.

## Publicar uma versão

Só após autorização, segurança resolvida/revisada e validação nativa:

1. Atualize VERSION e as ocorrências verificadas por scripts/check-version.py: main.go, internal/installwin, recursos Windows, Wails, package.json/package-lock e nomes do setup/arquivo de licenças nos scripts/teste. A interface lê a versão de package.json.
2. Atualize CHANGELOG, documentação e testes. `python scripts/check-version.py v0.4.1` deve passar apenas se todos os campos forem 0.4.1.
3. Faça commit e envie main após revisão. Crie uma tag anotada correspondente ao commit validado. Exemplo para a versão atual, somente quando aprovada:

```sh
git tag -a v0.4.0 -m "Projeto P2P 0.4.0"
git push origin v0.4.0
```

4. Release valida tag exata SemVer contra VERSION, chama CI, compila nativamente, verifica payload/segredos/dependências, gera pacotes e checksums. Só o job publish, após sucesso de CI e assemble e aprovação do ambiente se configurada, usa GITHUB_TOKEN para `gh release create --verify-tag --generate-notes`. Notas são geradas pelo GitHub e arquivos associados à tag exata. Release existente não é sobrescrita.
5. Confira notas e downloads. O fluxo não publica a cada commit na main. Não mova tags já publicadas; correções devem receber nova versão (0.4.1), novas funcionalidades 0.5.0.

## Arquivos produzidos

- ProjetoP2P-Setup-v0.4.0.exe: instalador Windows AMD64 real.
- ProjetoP2P-v0.4.0-windows-amd64-portatil.zip: alternativa portátil identificada.
- ProjetoP2P-v0.4.0-linux-amd64.tar.gz: desktop Linux nativo, gerado no runner Linux.
- ProjetoP2P-v0.4.0-licencas-windows.zip / -linux.zip: licenças e inventários de dependências.
- projeto-p2p-v0.4.0-fonte.zip: somente fontes auditados tracked.
- SHA256SUMS.txt: SHA-256 de cada arquivo da release.

O pacote local adicional `-linux-amd64-cli.tar.gz` contém apenas terminal e não comprova build/execução do desktop Linux. Scripts de distribuição usam listas explícitas e não incluem perfis. O runtime WebView2 pertence à Microsoft; VPNs não são redistribuídas.
