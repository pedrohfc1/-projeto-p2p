# EasyTier integrado — beta privado 0.5.2

Este relatório substitui a descrição anterior de laboratório isolado. Release público bloqueado por PRIVATE_BETA_ONLY. Nenhuma publicação automática autorizada. Testes entre casas, CGNAT, instalação em Windows limpo e medição de consumo prolongado continuam pendentes.

## Bloqueio DHT

Ver SEGURANCA-DHT-BETA.md. GO-2024-3218/GHSA-mqr9-hjr8-2m9w é um risco real de censura/indisponibilidade por Sybil na resolução de provedores. Não representa execução remota nem extração da identidade. A mensagem isolada `failed to find any peer in table` normalmente significa tabela vazia; ela não demonstra exploração. Não foi encontrado um patch upstream que permita declarar a implementação segura.

O beta padrão não importa nem instancia Kademlia; isso elimina o caminho afetado do binário, sem suprimir o scanner. A implementação anterior continua no código com `-tags legacydht`, exclusiva para regressão e uso consciente do risco. Bootstrap libp2p, Circuit Relay, convites e conexões nativas continuam disponíveis. O legado não deve ser publicado como corrigido. O scanner do módulo pode continuar mencionar dependência indireta no go.mod; a verificação do binário e do grafo alcançável é necessária.

## Arquitetura

Chat/comunidades existentes → streams libp2p + Noise e Peer ID esperado → TCP dialer próprio → SOCKS5 em 127.0.0.1 → smoltcp/EasyTier no-TUN → conexão direta ou relay → porta TCP local do libp2p remoto.

Não há novo protocolo de mensagens nem banco substituto. As assinaturas, cadeia de operações comunitárias, ACKs e identificadores antirrepetição continuam na implementação existente. O endereço IPv6 fd… do convite é um identificador sintético derivado do Peer ID; somente o dialer o interpreta. Não é criado no Windows nem anunciado como rota do sistema. Não tente pingar esse endereço no sistema operacional.

O helper etcat derivado de 0.1.0 usa as bibliotecas EasyTier internamente. Não usa easytier-core.exe externo. Não instala EasyTier/Radmin, adaptador, driver, serviço, rotas ou DNS. UPnP foi removido das features e explicitamente desabilitado; a aplicação também não solicita mais NATPortMap no modo nativo. Hole punching e libp2p relay permanecem. Não há alteração automática de firewall. WebView2 pode ser necessário para a interface, como nas versões anteriores.

SOCKS e encaminhamento TCP terminam apenas na porta `--serve` autorizada pelo helper; ACL e porta local do gateway são distintas da porta do app. O upstream oferece caminhos de UDP/QUIC e encaminhamento, mas o chat deste beta usa TCP. Isso não implementa voz, captura, WebRTC nem encaminhamento arbitrário de serviços do usuário.

## Identidade, isolamento e credenciais

A chave Ed25519 do aplicativo nunca é entregue ao auxiliar. Cada perfil tem identidade/segredo de rede EasyTier próprios, gerados aleatoriamente, e credencial de acesso própria. Não há senha global. O proprietário da comunidade e suas permissões continuam verificados por registros assinados, mesmo quando o convite de transporte é compartilhado.

O convite compacto assinado inclui Peer ID, porta, endereço sintético e credencial etcat. Considere o convite um segredo: qualquer portador pode tentar alcançar a porta autorizada, mas ainda precisa autenticar a identidade libp2p e respeitar permissões da aplicação. O beta não oferece revogação individual de credencial de transporte nem rotação independente pelo GUI; não o use para comunidades abertas com adversários. A expiração do convite controla importação; não revoga contatos já importados. Isso não muda as permissões de conteúdo.

Credenciais locais e rotas ficam em integrated-transport.sealed usando DPAPI do usuário Windows; identidade/contatos/histórico existentes são preservados. Material transitório entregue ao Rust usa arquivo privado com DACL usuário/SYSTEM e DELETE_ON_CLOSE. Tokens passam por variável de ambiente do processo filho, nunca por linha de comando/diagnóstico. Um processo com acesso completo ao mesmo usuário continua capaz de inspecionar memória: DPAPI não protege contra comprometimento dessa sessão.

Clientes simultâneos não podem reutilizar o UUID/endereço fixos do etcat original. O fork aloca UUID aleatório e IPv4 interno independente por cliente e marca explicitamente a credencial de convite como reutilizável, mantendo autenticação e as ACLs sem permissão de anunciar rotas. A reutilização é intencional para múltiplos convidados; não equivale a autenticar pessoas, que continua sendo responsabilidade do Peer ID e da comunidade. As redes são privadas por destino, não uma única malha global de comunidades. Os canais e a autorização continuam na camada comunitária existente; não há E2EE de grupo e o dono online ainda é necessário.

## Relay e estado verificável

Pool inicial: endpoints oficiais etcat `tcp://38.76.179.190:11010` e `udp://38.76.179.190:11010`. Disponibilidade verificada em 2026-11-01 por TCP e handshake EasyTier real; também foi usado no teste de chat local. São dois transportes do mesmo operador, não dois relays independentes. Não há SLA nem garantia de capacidade/disponibilidade futura.

O upstream não publicou chave de identidade fixada para esse relay. O GUI mantém aviso explícito; a identidade final do peer continua protegida por Noise. Não confundir criptografia com identidade do relay fixada. O operador pode observar metadados ou negar serviço. Até 8 endpoints autorizados podem ser configurados no mesmo pool; chave pública opcional permite fixar identidade quando fornecida por canal confiável. Não configuramos máquinas de terceiros sem autorização.

A indicação direta/relay vem das rotas da mesma sessão SOCKS em uso, com validade curta (6 s). Sem evidência recente mostra caminho não verificado. Receber conexão sem sessão de saída observável não permite afirmar o caminho. Handshake do relay não comprova conexão do contato.

Um supervisor reinicia o servidor até três vezes com espera crescente e conserva a credencial. A camada libp2p reconecta contatos conhecidos. Clientes SOCKS mortos são recriados na próxima discagem; não há troca automática para outra VPN. Limites: até 8 processos de cliente por perfil e 256 rotas persistidas, cada processo até 256 MiB e teto de CPU de 25% por Job Object. O limite de sessões não é uma garantia de suporte a grandes comunidades; sem expulsão por ociosidade, reinicie o app para liberar sessões históricas. O encerramento do app fecha os processos de sua propriedade.

## Packet.dll e distribuição

O antigo pacote completo de easytier-core exigia Packet.dll/Npcap; ele não é usado neste beta. O helper novo remove features tun/faketcp, a dependência Windows obrigatória windivert foi tornada opcional dessas features, e UPnP está ausente. scripts/audit-pe.py examina imports normais e atrasados e referências a carregadores proibidos no executável, não apenas o instalador. Execução e chat foram testados com somente o helper na pasta de auxiliares. Isso não é afirmação de inventário completo de drivers já instalados no PC de desenvolvimento.

Nenhum Npcap é redistribuído. Etcat é Apache-2.0; EasyTier é LGPL-3.0. O instalador leva EasyTier-corresponding-source.zip com fontes modificadas, Cargo.lock, dependências vendorizadas, licenças e instruções para recompilar/religar. É possível recompilar o app com o hash do helper modificado; não exige assinatura privada. Consulte third_party/etcat/BUILD.md. O setup é privado e não tem assinatura Authenticode do Projeto P2P.

## Teste em dois PCs Windows, sem Radmin

1. Use o mesmo instalador beta nos dois PCs. Feche a versão anterior e mantenha cópia de segurança do perfil antes do teste. A instalação não deve apagar o perfil em %APPDATA%/ProjetoP2P. Não instale VPN ou Npcap.
2. Abra Configurações, escolha Rede P2P integrada (beta privado), mantenha a porta TCP do aplicativo e salve. Não preencha Bootstraps/Relays libp2p para esse modo. O pool integrado tem campos próprios; deixe o padrão para o teste comunitário.
3. Em Estado da rede confirme auxiliar iniciado. Se não iniciar, registre apenas o erro exibido, sem copiar tokens. Verifique internet/permissões locais; o programa não muda o firewall. Não crie regra antes de compreender a falha e autorizar sua criação.
4. Gere um convite compacto novo no PC A, envie por canal privado ao PC B e importe. Repita de B para A para permitir iniciação em ambos os sentidos. Convites novos do modo integrado exigem este beta em ambos os lados.
5. Troque mensagens individuais nos dois sentidos. Confirme conteúdo e estado de entrega, não apenas adaptador/processo ativo. Crie comunidade em A, gere convite e entre por B; envie no canal geral e confira em ambos.
6. Desligue temporariamente a conexão de B e restaure; confira reconexão e mensagens sem duplicação. Feche e reabra B; confira Peer ID, contatos e histórico. Anote tempos, caminho observado, CPU/RAM pelo Gerenciador de Tarefas (app/WebView e todos os helpers) e rede total.
7. Falha de relay com recuperação alternativa só pode ser validada tendo um segundo endpoint autorizado operacional. Não derrube nem sobrecarregue relay comunitário. Não há segundo operador configurado por padrão; esse item permanece pendente.
8. Fallback manual: ambos escolhem Radmin, salvam e trocam convites recém-gerados desse modo. A identidade/histórico permanecem; a VPN precisa estar operacional, não basta existir o adaptador. Não exige Hamachi simultâneo.

Registre versão Windows, redes distintas/CGNAT quando verificável, resultados de cada etapa e falhas sem incluir os convites. Não afirme teste entre casas a partir de loopback, três perfis ou acesso ao relay público.

## Referências oficiais

- https://github.com/EasyTier/etcat (fonte e relays.toml do commit fixado)
- https://github.com/EasyTier/EasyTier
- https://easytier.rs/en/guide/network/host-public-server.html
- https://pkg.go.dev/vuln/GO-2024-3218
- https://github.com/advisories/GHSA-mqr9-hjr8-2m9w

## Avisos de dependências adicionais

A consulta OSV/RustSec de 2026-11-01 encontrou avisos que o govulncheck Go não cobre. Atualizações compatíveis corrigiram versões de Tokio, rustls/webpki, crypto-bigint, ctutils, quinn e ringbuf. Não foi usada lista de exceções para fingir ausência de avisos.

A feature zstd foi retirada do helper por avisos de segurança nos contextos de compressão/decompressão; a variante final não deve conter esses pacotes no grafo normal de execução (a ferramenta zip de compilação ainda os inclui apenas no build). Permanecem avisos em hickory 0.25 (DNS/DNSSEC), russh 0.63 (SSH) e zip 4.6 (extração de arquivos de compilação), além de crates sem manutenção (encoding, paste, proc-macro-error, unsafe-libyaml). O aplicativo só invoca serve/SOCKS com argumentos fixos, relays por IP e host server.etcat interceptado, sem SSH ou extração de arquivos recebidos de peers. Isso reduz exposição desses caminhos, mas não substitui revisão completa nem atualização para APIs maiores. As consultas e versões ficam no relatório de validação; release público continua bloqueado. Não classifique o conjunto Rust inteiro como livre de vulnerabilidades.

O aviso libp2p sobre SetKeepAlive vem do wrapper net.Conn do transporte TCP customizado; o socket subjacente é criado pelo net.Dialer com keepalive padrão e o app usa ping/reconexão. Não é o aviso DHT, não foi suprimido, e testes de interrupção são necessários. Não é evidência de instalação incorreta de VPN.
