# Arquitetura 0.4.0

O desktop usa Wails v2 para ligar React/TypeScript ao serviço Go. O frontend de produção não inicia um nó sozinho nem expõe um servidor HTTP de administração; tests/webui é uma ponte de QA exclusivamente local, autenticada, que não entra no executável desktop.

## Componentes

- `internal/identity`: criação/persistência Ed25519; DPAPI Windows e permissões Linux.
- `internal/network`: host libp2p, Noise, streams de mensagens, DHT Kademlia e Circuit Relay v2; gestão de conexões, reconexão e limites.
- `internal/desktop`: ciclo de vida, configurações, contatos, mensagens e diagnóstico de conectividade.
- `internal/adapters`: enumeração de interfaces, metadados Windows e registro genérico de provedores VPN (Hamachi/Radmin), sem configuração do fornecedor.
- `internal/invite`: convites assinados com Peer ID, multiaddrs e escopo opcional compatível com convites anteriores.
- `internal/localdb`: bbolt para configurações, contatos e histórico. Não é SQLite; ambos os formatos de banco ficam excluídos do Git.
- `internal/firewall`: diagnóstico Windows somente leitura; não altera regras.
- `internal/installwin` e `cmd/p2p-setup`: instalador real por usuário, desinstalador e bootstrapper WebView2 Microsoft.
- `cmd/projetop2p`: CLI, inclusive bootstrap/relay público e diagnóstico. `main.go`: desktop.

## Caminhos de rede

Automático conserva a rede libp2p nativa com DHT, bootstrap e Circuit Relay. Peers precisam de algum caminho alcançável: NAT traversal/relay não garantem conexão universal. Endereços das VPNs registradas são excluídos do anúncio nativo do desktop.

Hamachi, Radmin e interface manual criam host TCP vinculado ao IPv4 selecionado para escuta, anúncio e origem de discagem. Nesse host DHT público, AutoRelay, NAT mapping e DCUtR ficam desativados; configurações nativas permanecem salvas. A descoberta inicial usa convite e os contatos conhecidos são reconectados. Não há scanner IP nem enumeração de todos os membros da VPN. A conexão exige handshake autenticado e ping; o estado do adaptador é apenas informação local.

A enumeração Windows combina GUID, índice, estado, IPv4 e evidências de driver. Faixas 25/8 e 26/8 são pistas dos perfis, não prova de fornecedor/associação. O monitoramento a cada três segundos interrompe o host se a interface cair e tenta recriá-lo quando voltar, mantendo a identidade. Não muda silenciosamente para a rede pública. A seleção manual é alternativa quando a identificação específica falha.

## Dados e evolução

Perfil Windows em %APPDATA%/ProjetoP2P; Linux em XDG_CONFIG_HOME/ProjetoP2P ou ~/.config/ProjetoP2P. Instalação separada do perfil e novos campos compatíveis preservam identidade, contatos e histórico. O histórico não é criptografado em repouso. Não use o mesmo perfil em duas instâncias; use portas distintas no mesmo PC.

Comunidades, arquivos distribuídos criptografados, voz cooperativa e registros verificáveis são arquitetura futura; não estão implementados. Não há blockchain nesta versão. Consulte decisões/0003 e 0004 e os documentos em security para detalhes históricos, sempre observando sua versão.
