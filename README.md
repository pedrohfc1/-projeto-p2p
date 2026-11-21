Beta privada atual: **0.10.3** — [voz, teste único, PTT, sensibilidade e validação](docs/VOZ-PTT-v0.10.3.md).

Versão anterior de instrumentação parcial: **0.10.2** — [escopo medido e pendências](docs/DIAGNOSTICO-REDE-v0.10.2.md). Sem causa de instabilidade comprovada.

Versão anterior: **0.10.1** — [microfone, supressão nativa, menu e limites de validação](docs/MICROFONE-MENU-v0.10.1.md).

Versão anterior: **0.9.7** — [voz e áudio da tela: uso e limitações](docs/AUDIO-TELA-v0.9.7.md).

Versão anterior: **0.10.0** — [central de configurações e validação](docs/CENTRAL-CONFIGURACOES-v0.10.0.md). A engrenagem do perfil reúne as preferências; voz e vídeo mantêm seus componentes e transportes existentes.

Versão anterior: **0.9.6** — [hotfix e limites de validação](docs/HOTFIX-v0.9.6.md).

# PeeTwoPee 0.9.5 — atualização segura, beta privado

Instalação com inventário, encerramento consentido, recuperação e rollback; perfil e mídia preservados. [Guia/testes/limites](docs/PEETWOPEE-v0.9.2.md), [validação](docs/PEETWOPEE-v0.9.2-VALIDACAO.json). Instalação gráfica completa, Registro e dispositivos físicos pendentes. Sem publicação pública.

# PeeTwoPee 0.9.1 — hotfix privado de áudio/tela

Roteamento de áudio, player persistente e limites de vídeo corrigidos. [Guia e limitações](docs/PEETWOPEE-v0.9.1.md), [validação](docs/PEETWOPEE-v0.9.1-VALIDACAO.json). H.264/hardware e1080p60 físico não estão entregues/validados; sem publicação pública.

# PeeTwoPee 0.9.0 — experiência de uso e transmissões, beta privado

AO VIVO com acesso explícito, chat com ações assinadas e notificações opcionais. [Guia e limites](docs/PEETWOPEE-v0.9.0.md), [validação](docs/PEETWOPEE-v0.9.0-VALIDACAO.json). Atualize todos os participantes; dados preservados e publicação pública bloqueada.

# PeeTwoPee 0.8.2 — painel de chamada, beta privado

Painel de voz no canto inferior esquerdo, ações rápidas, duração, destino e estado da tela. Câmera apenas como botão desabilitado. [Guia](docs/PEETWOPEE-v0.8.2.md) e [validação](docs/PEETWOPEE-v0.8.2-VALIDACAO.json). Não publicado.

# PeeTwoPee 0.8.1 — beta privado

Configurações de áudio durante chamadas, seleção de rede salva imediatamente e integrada como padrão para novos perfis. Veja [guia](docs/PEETWOPEE-v0.8.1.md) e [validação](docs/PEETWOPEE-v0.8.1-VALIDACAO.json). Perfis, identidades e métodos anteriores preservados; sem publicação pública.

# PeeTwoPee 0.8.0 — beta privado

Interface reorganizada, retorno local que acompanha a troca de microfone, volumes persistentes e convites autossuficientes menores. Consulte [o guia](docs/PEETWOPEE-v0.8.0.md) e [a validação](docs/PEETWOPEE-v0.8.0-VALIDACAO.json). Não há publicação pública.

# PeeTwoPee

![PeeTwoPee](assets/brand/logo.png)

Beta privado **0.7.0**: correção de microfone/dispositivos, consentimento WebView2 e compartilhamento de tela P2P em salas de voz. [Guia, testes e limites](docs/PEETWOPEE-v0.7.0.md). Atualize os participantes; use o instalador privado entregue localmente. Os perfis antigos continuam no mesmo caminho. No Windows, desenvolvedores devem usar `scripts/desktop.ps1` ou `scripts/build-desktop.ps1`; o build verifica a política de consentimento antes de empacotar.

Beta privado **0.6.2**: recuperação de comunidades via rota EasyTier autenticada, endpoints estáveis por identidade, réplica local e voz temporariamente offline. Consulte [guia de conectividade](docs/PEETWOPEE-v0.6.2.md). Histórico 0.6.1: voz mesh para até seis pessoas, volume local por participante, nova marca e bandeja. Perfis existentes continuam em `%APPDATA%/ProjetoP2P`; não renomeie essa pasta. Consulte [guia 0.6.1](docs/PEETWOPEE-v0.6.1.md). O instalador atual é `PeeTwoPee-Setup-v0.6.2-beta-privado.exe`, entregue localmente; não há release pública desta versão. Windows 10/11 AMD64 com WebView2. Não é necessário instalar Go, Node.js ou EasyTier separadamente. Atualize todos os participantes; confira os hashes fornecidos. A validação física da bandeja e da atualização ainda está pendente. As notas abaixo são o histórico das versões anteriores.

> **Áudio beta 2 (2026-11-04):** Opus em streams libp2p, retorno local do microfone e reparação de mídia sem sair da sala. [Guia e limitações](docs/AUDIO-BETA2.md). Atualize os dois PCs.

> **Beta privado 0.6.0:** convites com políticas assinadas, novas comunidades com texto entre membros e voz Opus para duas pessoas via EasyTier userspace. [Guia e teste entre PCs](docs/BETA-v0.6.0.md) · [Resultados e segurança](docs/VALIDACAO-v0.6.0.md). Release público bloqueado; nenhum upload automático. A documentação 0.5 abaixo descreve o histórico e as comunidades legadas.

> Beta privado 0.5.2: integração real EasyTier sem TUN em validação. Consulte [arquitetura e teste entre PCs](docs/EASYTIER-INTEGRADO-BETA-v0.5.2.md) e [quarentena DHT](docs/SEGURANCA-DHT-BETA.md). As descrições antigas de laboratório abaixo são históricas. Release público bloqueado.

## Histórico: Projeto P2P 0.5.1

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


Beta privada 0.9.8: [áudio, interface e validação](docs/AUDIO-INTERFACE-v0.9.8.md). Áudio automático de janela depende de identificação inequívoca; monitor exige saída física distinta para voz. Vídeo permanece ativo em falha de áudio.


Beta privada 0.9.9: [reprodução de voz e limites da validação](docs/REPRODUCAO-VOZ-v0.9.9.md). A chamada agora inicializa seus próprios consumidores RTC, independentemente do áudio da tela.
