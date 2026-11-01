# Validação 0.5.0

Execução local Windows em 2026-10-31/2026-11-01, branch `feature/v0.5-communities-security`. Não houve push, tag ou release. Ferramentas: Go 1.27.3, Wails 2.17.0, Node/npm locais e Python 3. Frontend compilado antes dos testes Go. Os artefatos desta entrega são para testes.

| Verificação | Resultado |
|---|---|
| `go test ./... -count=1 -timeout=180s` | PASS, 42 funções de teste; inclui integração de comunidades e regressões anteriores |
| `go vet ./...` e `go mod verify` | PASS |
| `npm --prefix frontend test` | PASS, 8 testes, incluindo permissões e escape de conteúdo remoto |
| `npm --prefix frontend run build` | PASS, TypeScript e Vite |
| `python tests/distribution_test.py` | PASS, 4 testes |
| `python scripts/check-version.py v0.5.0` | PASS |
| `tests/webui_smoke.py` | PASS, API real, assets, token/origem, configurações, convites, DM durável e reinício |
| `tests/radmin_desktop_smoke.py --messages` | PASS, adaptador Radmin real, escuta/convite corretos, ping e DM persistida; dois perfis no mesmo PC |
| `tests/community_desktop_smoke.py` | PASS, três serviços reais em loopback |
| Mesmo teste com `--radmin` | PASS, três serviços reais pelo adaptador Radmin no mesmo PC |
| `tests/cli_smoke.py` | PASS, três processos, DHT, fallback, crash, identidade e recuperação |
| `tests/relay_cli_smoke.py` | PASS, daemon, diagnóstico, reserva assinada e limites do relay |
| CLI Windows AMD64 e Linux AMD64 | Build PASS; ELF Linux não executado neste Windows |
| Desktop Wails Windows AMD64 | Build PASS, sem captura de mídia |
| Instalador e licenças | Payload corresponde ao app, desinstalador, MIT e inventário; bootstrapper Microsoft validado |
| `npm audit` | Zero vulnerabilidades reportadas |
| Auditoria de candidatos Git e gitleaks | PASS, sem padrões de credenciais ou caminhos privados encontrados |
| `actionlint` e análise sintática dos scripts PowerShell | PASS; caminho Authenticode não executado sem certificado |
| `govulncheck ./...` | **FAIL conhecido:** GO-2024-3218 alcançável no DHT; sem supressão |

## Casos de segurança e persistência

Testes verificam assinatura/autoria, papel forjado, membro externo, moderador sem poder sobre dono/moderador, permissão revogada, ticket expirado/usado, controle concorrente antigo, cadeia divergente, IDs repetidos, origem de stream, frames malformados e limites. A integração autentica peers reais com Noise/libp2p, não apenas mocks.

O dono é encerrado e reaberto com a mesma chave; o membro reinicia com outbox pendente e depois sincroniza sem duplicar. Há paginação de 103 mensagens, reentrada com ticket novo após revogação ainda não conhecida pela réplica, migração do histórico legado e recusa de downgrade que apagaria comunidades. Replay de ID não reescreve DM após reinício. Limites de contatos não solicitados e de chave são testados.

O teste de Radmin comprova o uso do adaptador e comunicação real **local**, não duas casas. O relato prévio do usuário comprova DM 0.4.1 em dois PCs; não é apresentado como teste de comunidades 0.5. O roteiro de aceitação é [COMUNIDADES-v0.5.md](COMUNIDADES-v0.5.md).

## Reprodutibilidade e integridade

Duas compilações consecutivas do desktop final, com os mesmos fontes, frontend, toolchain e `GOFLAGS=-buildvcs=false`, produziram SHA-256 idêntico:

`25ec906e31d88170d37e206fd58f6fe6819ab22ca98210e4b18c589e38b26707`

Esse resultado comprova repetição local unsigned, não reprodutibilidade independente em outra máquina. O inventário ZIP de licenças tem ordem e timestamps determinísticos. Assinatura futura com timestamp modifica os bytes, exigindo hashes posteriores. O setup verifica o app embutido, mas não foi instalado no perfil pessoal para esta validação.

App e setup: `NotSigned`. Consulta Defender: `unavailable`. Isso não é varredura completa nem resultado de arquivo limpo. [Preparação Authenticode](ASSINATURA-WINDOWS.md).

## Pendências de aceitação

- Inspeção visual da nova interface na janela nativa. A tentativa com perfil isolado encontrou erro de recurso WebView2 em uma execução; após isolamento do cache a criação do ambiente foi registrada, mas não apareceu uma janela de QA controlável. Não interagimos com o app pessoal aberto e não consideramos o teste visual aprovado. SSR/TypeScript e API real passaram.
- Instalação, atualização 0.4.1→0.5 e desinstalação completas em Windows limpo/VM; conferir atalhos, esquema e preservação do perfil. A verificação de payload não substitui essas etapas.
- Comunidades entre dois PCs Windows por Radmin, depois Hamachi e caminho nativo conforme configuração real. Validar firewall com autorização específica se necessário.
- Desktop Linux Mint nativo, `-race` no núcleo e testes de mídia futuros. Não há compilador C local disponível para race Windows; CI Linux está preparada, mas não foi executada neste chat.
- CPU/RSS da árvore completa Go+WebView e tempo de renderização na UI; medições atuais são de backend. [Métricas](METRICAS-v0.5.md).
- Resolver risco DHT ou adotar decisão técnica revisada antes da publicação; não remover o gate para fazer a release passar.

Sem E2EE de grupo, eleição de dono, áudio, vídeo, tela, anexos ou blockchain. Nenhuma configuração de firewall, VPN ou proteção do sistema foi alterada.
