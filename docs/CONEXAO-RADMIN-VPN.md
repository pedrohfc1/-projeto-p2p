# Radmin VPN — teste real entre dois PCs Windows

## Preparação

Use o instalador ProjetoP2P-Setup-v0.4.1.exe nos dois PCs, fechando a versão anterior. O perfil existente em %APPDATA%/ProjetoP2P é preservado. Não copie identity.key de um computador para outro: cada participante precisa de identidade própria.

Instale somente a VPN que pretende usar. Para este roteiro, instale Radmin VPN pelo site oficial https://www.radmin-vpn.com/ e coloque os dois PCs na mesma rede Radmin usando o próprio Radmin. O aplicativo não instala, autentica, cria redes nem modifica configurações do Radmin. Não precisa instalar Hamachi. Não desative o firewall ou antivírus.

## Conectar

1. Nos dois PCs, abra Configurações → Modo de conectividade → Radmin VPN. Escolha detecção automática do adaptador Radmin ou o adaptador confirmado. Use TCP 4001 e salve. PCs distintos podem usar a mesma porta.
2. Em Estado da rede, confirme VPN selecionada, nome, índice, IPv4 virtual, interface ativa e host iniciado. O perfil Radmin espera IPv4 26.x. Nome amigável/IP isoladamente não comprovam o fornecedor. Se não reconhecido, use interface manual e confirme o adaptador correto.
3. Não preencha Bootstraps ou Relays independentes para este modo. Configurações nativas antigas continuam salvas, mas ficam inativas no host vinculado à VPN. Automático significa libp2p nativo; para este teste escolha explicitamente Radmin VPN.
4. No PC A copie o convite. No B use Conectar por convite e cole. O convite assinado contém Peer ID e multiaddr do host real, por exemplo /ip4/IP_REAL/tcp/4001/p2p/PEER_ID_REAL. Não use os placeholders literalmente. Troque também o convite do B para o A para nomes assinados recíprocos.
5. Execute Testar conectividade. Exija conexão autenticada e ping libp2p bem-sucedidos. Interface ativa ou contato salvo não bastam. A indicação de conexão direta significa ausência de Circuit Relay libp2p; a VPN ainda pode retransmitir tráfego pelo fornecedor.
6. Envie mensagens nos dois sentidos. Reinicie o aplicativo e confira Peer ID, contatos e histórico preservados.
7. Desconecte a VPN em um PC. Verifique estado indisponível e falha de conectividade; não deve haver migração silenciosa para a rede pública. Reconecte a VPN, aguarde monitoramento/reconexão e repita o ping. Se ambos mudaram IP e não têm caminho válido, troque novos convites.

## Diagnóstico e aceitação

Se falhar, confirme a mesma rede no próprio Radmin, IPv4/porta do convite e se o aplicativo está ouvindo. No PC B: `Test-NetConnection -ComputerName IP_VIRTUAL_A -Port 4001`. Esse teste confirma TCP, não identidade. Depois repita o teste libp2p no aplicativo. O diagnóstico de firewall é consultivo e somente leitura, não comprova a política efetiva. Autorize o aplicativo segundo a política do Windows se necessário, sem desativar proteções.

CLI opcional: `projetop2p-windows-amd64.exe --interfaces`. Para diagnóstico explícito: `projetop2p-windows-amd64.exe --data data/radmin --interface-ip IP_VIRTUAL_REAL --listen /ip4/IP_VIRTUAL_REAL/tcp/4001`. Não execute ao mesmo tempo que o desktop na mesma porta. `/status` deve mostrar localOnly true.

Registre em cada PC: versão do Windows/Radmin/app, modo, interface/índice, horário, resultado TCP, ping libp2p, mensagens nos dois sentidos e recuperação após desconexão. Não publique convites/IPs sem necessidade nem chaves privadas. Aprovar apenas após essas etapas em computadores distintos. Este teste ainda não foi executado pela equipe neste ambiente.

Não há descoberta automática de todos os membros Radmin nem varredura de IPs. Convites e contatos conhecidos permitem conexão sem bootstrap/relay externos quando a VPN permite TCP entre participantes.

## Correção 0.4.1 — atualizar os dois PCs e refazer convites

A detecção das duas VPNs não seleciona nenhuma no modo Automático. Na 0.4.1, a tela de rede oferece Usar Radmin VPN, e gerar convite com VPN detectada exige uma escolha explícita. Não use Continuar com libp2p nativo para este roteiro. Ao receber convite Radmin em outro modo, a aplicação pede confirmação para ativar Radmin antes de conectar; não troca de rede silenciosamente.

Feche o aplicativo antigo, instale ProjetoP2P-Setup-v0.4.1.exe nos dois PCs e escolha Radmin VPN. Confirme IP 26.x, adaptador Radmin (mesmo com Hamachi instalado), porta TCP de escuta 4001 e nó Em escuta. A interface é persistida pelo GUID, sem recriar a identidade. Se a porta estiver ocupada, escolha outra porta fixa livre; o convite conterá essa porta real.

Gere novos convites após ativar Radmin. Convites antigos gerados no modo nativo podem conter apenas 192.168.x/10.x e não servem como convite Radmin; eles não podem ser corrigidos sem obter o endereço virtual atual do outro PC. O perfil, contatos e mensagens continuam preservados; adicionar novo convite do mesmo Peer ID atualiza o endereço do contato sem apagar seu histórico.

Endereços APIPA 169.254.x.x são descartados. No modo Radmin, anúncios e destinos exigem TCP IPv4 26.x, com a porta real do listener; endereços 25.x do Hamachi, LAN, DNS/relays e portas zero são rejeitados nesse escopo. IP e índice são lidos do adaptador confirmado, sem inferir conectividade apenas pela presença do driver. A faixa 26.x é uma pista deste perfil, não prova de associação à rede VPN.

Estado da rede mostra o último diagnóstico: modo, interface/GUID/índice, IP local, porta de escuta, destinos fornecidos à discagem, caminho efetivo quando autenticado e motivo/detalhe da falha. Timeout não identifica sozinho o firewall como causa. Uma porta de origem dinâmica numa conexão de saída é normal e não significa que a porta de escuta anunciada mudou.

Teste local real executado: seleção Radmin, convite assinado com IP/Peer ID/porta reais, persistência após reinício, ping e mensagem entre dois perfis locais. Isso não substitui o roteiro entre dois PCs. Nenhuma regra de firewall, configuração da VPN ou roteador foi alterada.

Diagnóstico opcional para desenvolvedores: compile `go build -buildvcs=false -o dist/webui.exe ./tests/webui` e execute `python tests/radmin_desktop_smoke.py dist/webui.exe --messages`. Exige adaptador Radmin ativo e usa perfis temporários, sem tocar no perfil pessoal. Não é requisito dos runners normais de CI.
