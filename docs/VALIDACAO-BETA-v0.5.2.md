# Validação privada 0.5.2 — 2026-11-01

Plataforma de execução: Windows x64 no computador de desenvolvimento. Não foram executados dois computadores em casas diferentes, CGNAT controlado, VM limpa, interface gráfica nativa completa ou Linux Mint neste ciclo. Nenhuma mudança de firewall, driver, adaptador, rota ou DNS foi solicitada pelos testes. Os perfis de QA são separados do perfil pessoal.

## Resultados disponíveis

| Verificação | Resultado e alcance |
|---|---|
| Frontend | 10 testes, TypeScript e Vite aprovados; interação visual nativa não aceita por esses testes |
| Go | go test ./... e go vet ./... aprovados |
| DHT legado | regressão com tag legacydht; vulnerabilidade upstream permanece documentada |
| Quarentena DHT | grafo do desktop padrão sem kad-dht e sem laboratório antigo; govulncheck: 0 vulnerabilidades alcançáveis, 8 avisos apenas em módulos requeridos |
| CLI | compilação Windows e Linux amd64; smoke de três processos Windows, bootstrap, falha, troca de endereço e identidade persistente |
| Convites | rota integrada assinada, endereço ligado ao Peer ID, adulteração e mistura de modos recusadas; compatibilidade dos convites anteriores mantida |
| Integração real | TestRealIntegratedChatAndCommunity aprovado: três perfis, DM/resposta, dois ingressos com convites próprios, texto em canal, reinício e rota protegida persistente |
| Idempotência | mesma mensagem assinada reenviada após fechar conexão: uma linha gravada e ACK repetido; teste separado confirma ausência de ACK em falha de gravação |
| Três processos independentes | community_desktop_smoke.py --integrated aprovado (58,14 s): caminho padrão do helper incluído, cargos, canais, dono encerrado abruptamente/reiniciado, outbox, deduplicação, revogação e DM |
| Supervisor | auxiliar encerrado forçadamente, reiniciado com mesma credencial; encerramento normal sem processo vivo nem arquivo JSON privado remanescente |
| Packet.dll | imports PE normais/atrasados + referências proibidas + execução real do helper isolado; aplicativo e auxiliar sem Packet.dll/Npcap, Wintun ou WinDivert |
| Dependências Rust | 493 pacotes no grafo normal, 540 com ferramentas de build; sem capture/TUN/UPnP/zstd no grafo normal. Dez pacotes sinalizados no conjunto normal/build, inventário sem omissões em third_party/etcat/security-inventory.json |
| Empacotamento | setup --verify-payload compara SHA do desktop, auxiliar, licença e fontes; manifesto asInvoker; nenhuma instalação no perfil pessoal |
| Fontes LGPL | 30.812 arquivos de fontes/dependências + notices, ZIP validado por CRC; nenhum executável/driver no ZIP correspondente |
| Autenticode/Defender | binários do projeto e helper NotSigned; consulta Defender indisponível. Não houve exclusão, desativação nem afirmação de varredura limpa |

Os testes de mensagens não usam um protocolo experimental substituto: exercitam Core/Node e a camada comunitária de produção. Os ACKs repetidos são limitados por frequência e só reconhecem mensagens assinadas cujo conteúdo coincide com recibo de gravação concluída; falhas podem ser tentadas novamente sem criar outro ID. Após reinício, o banco mantém idempotência por remetente/ID e compara conteúdo imutável.

## Relay público observado

Endpoint oficial do etcat: 38.76.179.190:11010, TCP e UDP. TCP respondeu e o handshake EasyTier e os testes de chat funcionaram em 2026-11-01. Não há chave pública independente fixada, SLA ou segundo operador verificado. O aviso aparece no estado integrado. A chave Ed25519 esperada do contato ainda é verificada por libp2p Noise. Disponibilidade hoje não comprova travessia de qualquer CGNAT.

O auxiliar final é SHA-256 `472df99fa0d9d365957b84c548a8228a655de6984d67e971654fe893c3626d10`. Ele foi recompilado das revisões registradas em third_party/etcat/provenance.json. A ausência da feature zstd no runtime não elimina a ferramenta zip/zstd de build; PROTOC foi fornecido explicitamente por compilador de descritores Go. O fallback upstream que baixa protoc não foi exercitado e não deve ser usado sem verificação.

## Consumo observado e limites

No teste de recuperação do servidor, um auxiliar apresentou aproximadamente 22,1 MB de working set e 0,234 s acumulado de CPU na amostra após inicialização. Isso não é média de CPU nem consumo total do app: não inclui outros auxiliares, WebView2 e tráfego de controle do EasyTier. O painel Go mede o processo principal e streams do app; não mede toda a malha. Job Objects limitam cada auxiliar a 256 MiB e 25% de CPU; até oito clientes por perfil, mais servidor. Esses tetos não são estimativa de consumo normal.

No cenário de três processos, o RPC de commit de texto levou 105,1 ms e o ping final 0,506 ms no mesmo PC; esses números não representam latência entre casas nem garantem que o caminho permaneceu via relay durante todo o teste. A amostra do processo Go antes das comunidades tinha 26,8 MB RSS; métricas numéricas completas em METRICAS-BETA-v0.5.2.json, sem auxiliares/WebView2.

Medição prolongada CPU/RAM/rede total, perdas, latência entre casas e comparação direto/relay continuam pendentes. Não foram coletados pacotes nem alteradas permissões para obter números.

## Pendências que bloqueiam publicação pública

- Auditoria/atualização das APIs maiores Rust com avisos hickory/russh e dependências sem manutenção. Os caminhos SSH e DNS externos não são invocados pelo gerenciador do app, mas isso não permite declarar o helper universalmente livre de riscos.
- Validar identidade/alternativas autorizadas de relay, capacidade, queda e recuperação entre operadores. Há somente um operador padrão.
- Executar o roteiro de dois PCs, instalação/upgrade/desinstalação em Windows limpo, comportamento do antivírus/SmartScreen e aceitação visual nativa. Não fornecemos instruções para desativar proteções.
- Rever revogação/rotação e credenciais individualizadas de transporte antes de comunidades abertas. Convites beta são segredos reutilizáveis e não credenciais por pessoa.
- Validar Linux Mint em ambiente nativo; modo integrado deste beta exige DPAPI Windows, mantendo os modos libp2p/VPN existentes no código.

PRIVATE_BETA_ONLY mantém o publicador GitHub bloqueado. Nenhum push, tag ou Release foi feito nesta etapa. O instalador é para teste privado controlado, não uma distribuição pública aprovada.

## Repetir testes

Depois de compilar o frontend:

```powershell
go test ./...
go vet ./...
go test -tags legacydht ./internal/network ./internal/desktop
govulncheck ./...
$env:P2P_ETCAT_TEST_DIR = (Resolve-Path build/bin/easytier).Path
go test ./internal/desktop ./internal/overlay -run TestReal -v -count=1 -timeout=250s
python scripts/audit-pe.py build/bin/ProjetoP2P.exe build/bin/easytier/etcat-p2p.exe
./tests/installer-smoke.ps1
```

O teste opcional TestReal usa conexão pública e exige helper com o hash do manifesto. Sem a variável ele é marcado SKIP; isso não conta como aprovação da integração. Para o procedimento entre casas consulte EASYTIER-INTEGRADO-BETA-v0.5.2.md.

A CI normal compila e testa o desktop sem buscar automaticamente o helper privado. Enquanto PRIVATE_BETA_ONLY existir, não monta/publica um instalador incompleto; o beta foi montado localmente com auxiliar auditado e fontes correspondentes. Retirar o marcador por si só não disponibiliza os insumos do instalador. actionlint foi executado localmente; não afirmamos que este commit rodou no GitHub.
