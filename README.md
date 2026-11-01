> Beta privado 0.5.2: integração real EasyTier sem TUN em validação. Consulte [arquitetura e teste entre PCs](docs/EASYTIER-INTEGRADO-BETA-v0.5.2.md) e [quarentena DHT](docs/SEGURANCA-DHT-BETA.md). As descrições antigas de laboratório abaixo são históricas. Release público bloqueado.

# Projeto P2P

Mensageiro desktop experimental com identidade criptográfica persistente e comunicação autenticada entre usuários via libp2p. A versão **0.5.1** adiciona comunidades, canais de texto, convites assinados e papéis verificados no backend, preservando chat individual, contatos, histórico e Radmin/Hamachi. Salas de voz são somente definições; não há áudio, vídeo ou tela nesta fase.

## Baixar e instalar

**Usuários Windows não precisam instalar Go, Node.js ou ferramentas de desenvolvimento.** Baixe `ProjetoP2P-Setup-v0.5.1.exe`, feche versões anteriores e execute o instalador. Requisitos: Windows 10/11 AMD64 e Microsoft Edge WebView2; o instalador oferece o runtime Microsoft se ausente (internet necessária). Instala por usuário, oferece atalhos e desinstalador e preserva o perfil existente. O aplicativo e o setup ainda não têm assinatura digital de distribuição; não desative as proteções do Windows.

**[Baixar a versão mais recente](https://github.com/pedrohfc1/-projeto-p2p/releases/latest)** — endereço configurado para o repositório informado. A primeira publicação ainda está pendente; o link só disponibilizará downloads após uma release publicada. O aviso DHT [GO-2024-3218](https://pkg.go.dev/vuln/GO-2024-3218) continua bloqueando o gate de segurança.

Na release, escolha o instalador Windows; o ZIP portátil é uma alternativa identificada separadamente. `SHA256SUMS.txt` permite conferir os arquivos. No Linux Mint 22 AMD64, use o pacote desktop `ProjetoP2P-v0.5.1-linux-amd64.tar.gz`, com GTK3 e WebKitGTK 4.1 disponíveis no sistema. [Instalação e atualização](docs/INSTALACAO.md).

## Comunidades 0.5

Use **+** na barra lateral para criar uma comunidade ou entrar com convite. O dono e moderadores gerenciam canais e membros conforme seus papéis criptograficamente verificados. Histórico local, sincronização incremental e fila offline persistente estão implementados. **O dono precisa estar online para confirmar novas operações. Comunidades não têm E2EE de grupo nem consenso distribuído nesta versão.**

[Guia de comunidades e teste entre PCs](docs/COMUNIDADES-v0.5.md) · [Auditoria de segurança](docs/SEGURANCA-v0.5.md) · [Métricas iniciais](docs/METRICAS-v0.5.md) · [Plano 0.6: voz](docs/PLANO-v0.6-VOZ.md) · [Plano E2EE](docs/CRIPTOGRAFIA-E2EE-PLANO.md)

## Conectar duas pessoas

1. Para usar Radmin VPN ou Hamachi, cada pessoa instala **somente a VPN escolhida**, entra na mesma rede virtual pelo cliente do fornecedor e abre o Projeto P2P. Nenhuma VPN é instalada ou configurada pelo aplicativo.
2. Se houver VPN detectada no modo nativo, a geração de convite solicita escolher Radmin/Hamachi ou continuar explicitamente no modo nativo. Em Configurações, selecione **Radmin VPN** ou **Hamachi**, confirme adaptador/IPv4 e salve. Nesse modo não precisa preencher Bootstraps ou Relays independentes, nem abrir portas no roteador, desde que a VPN e o firewall permitam TCP entre participantes.
3. A pessoa A copia o convite e o envia por um canal de confiança. A pessoa B usa **Conectar por convite**. O convite contém o Peer ID e o multiaddr reais, com escopo assinado.
4. Use **Testar conectividade** e confira conexão autenticada e ping libp2p. Adaptador ativo não significa contato conectado. Troque mensagens nos dois sentidos; troquem os dois convites para obter os nomes assinados recíprocos.

[Roteiro Radmin VPN](docs/CONEXAO-RADMIN-VPN.md) · [Roteiro Hamachi](docs/CONEXAO-HAMACHI.md) · [Bootstrap e Circuit Relay nativos](docs/relay-quickstart.md)

**Automático** mantém o modo libp2p nativo com DHT, bootstrap e Circuit Relay; não escolhe silenciosamente uma VPN. Há também seleção manual de interface. O caminho pela VPN pode usar retransmissão do fornecedor; o projeto não controla essa infraestrutura.

## Privacidade e limites atuais

Identidade Ed25519 persistente, transporte Noise e mensagens assinadas autenticam os peers, não suas identidades civis. Verifique os convites por outro canal. Os dados ficam em `%APPDATA%/ProjetoP2P` no Windows e `$XDG_CONFIG_HOME/ProjetoP2P` (ou `~/.config/ProjetoP2P`) no Linux. Windows protege a chave com DPAPI; Linux usa permissões restritas. **O histórico local não é criptografado em repouso.** Não compartilhe sua pasta de perfil.

A rede não oferece entrega exatamente uma vez nem conectividade garantida sob qualquer NAT/firewall. A presença de um adaptador não comprova comunicação. Não há descoberta de todos os membros das VPNs: use convites e contatos conhecidos. Mídia de voz, blockchain e armazenamento distribuído criptografado não fazem parte desta versão.

Os testes automatizados e as verificações do instalador estão documentados em [validação da correção Radmin](docs/VALIDACAO-v0.5.md). O teste local com adaptador Radmin real passou (dois perfis, ping e mensagem persistida); o teste Radmin entre dois PCs reais, a inspeção visual nativa, a instalação/atualização completa e a execução desktop Linux permanecem pendentes de aceitação; não são apresentados como validados.

## Desenvolvimento e publicação

Backend Go, libp2p e bbolt; desktop Wails v2, React e TypeScript. Dependências fixadas nos lockfiles. Go 1.27.3, Node.js 24, Python 3 e Wails 2.17.0 são ferramentas de desenvolvimento, não requisitos para quem instala o app Windows.

```powershell
npm --prefix frontend ci
npm --prefix frontend test
npm --prefix frontend run build
go test ./... -count=1 -timeout=180s
go vet ./...
python scripts/check-version.py
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
./scripts/desktop.ps1 -Installer
./scripts/build.ps1
```

No Linux, instale antes as dependências de desenvolvimento de [CONTRIBUTING.md](CONTRIBUTING.md) e use `sh scripts/desktop-linux.sh`. O frontend deve ser compilado antes dos testes Go, pois seus assets são incorporados no binário.

- [Arquitetura](docs/ARQUITETURA.md)
- [Contribuir](CONTRIBUTING.md) e [segurança](SECURITY.md)
- [Publicar no GitHub e lançar versões](docs/PUBLICACAO.md)
- [Histórico de mudanças](CHANGELOG.md)

A CI verifica código e gera artefatos em Windows/Linux. Release publica somente para tags SemVer que coincidam com `VERSION`, após os testes. O disparo manual gera artefatos sem publicar. A permissão de escrita pertence apenas ao job publicador. O repositório já contém a preparação 0.4.0; a versão 0.5.1 foi preparada localmente, sem novo envio, tag ou release nesta tarefa.

Licença [MIT](LICENSE), confirmada pelo mantenedor. Dependências têm [licenças próprias](THIRD_PARTY_NOTICES.md).

## Convites compactos 0.5.1

Novos convites Protobuf/Base64URL são cerca de 62–65% menores nos exemplos medidos. Ed25519, identidade, escopo e prazo permanecem. Aceita convites antigos; o modal permite gerar envelope antigo para 0.5.0 e QR local. Códigos curtos são somente estrutura local, sem resolução global. [Formatos, comparação e compatibilidade](docs/CONVITES-v0.5.1.md).

## Branch experimental 0.5.2

Este checkout contém também o laboratório EasyTier sem TUN, separado do chat/comunidades. Para uso normal de convites compactos, use a entrega 0.5.1. O laboratório exige preparação explícita dos auxiliares verificados e nós conhecidos autorizados; não há relay público embutido nem validação entre casas/CGNAT. [Execução, segurança, desempenho, LGPL e plano de integração](docs/EASYTIER-v0.5.2.md).
