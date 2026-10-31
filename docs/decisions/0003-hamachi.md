# ADR 0003 — conectividade por interface e Hamachi 0.3

Status: implementado, validação entre casas pendente.

## Auditoria anterior

A 0.2 tinha host global em IPv4, DHT/AutoRelay, convites assinados e bbolt. Não havia enumeração enriquecida de adaptadores, modo isolado de VPN ou recuperação de adaptador. A inicialização retornava erro definitivo se o nó não pudesse subir; isso impediria o usuário de corrigir uma configuração Hamachi ausente pela interface. O ACK já dependia do commit local e a identidade persistente era compatível.

## Decisões

`internal/adapters` fornece uma função de enumeração substituível, sem API de membros/login da VPN. Windows combina net.Interfaces, GetAdaptersAddresses (GUID, índice, OperStatus, descrição) e chaves somente leitura da classe de rede (NetCfgInstanceId, MatchingDeviceId, ComponentId e DriverDesc). Identificação por hardware/driver é prioritária. Descrição de driver contendo Hamachi + IPv4 25/8 é um fallback; nome amigável sozinho não é confirmação. Seleção manual funciona quando os metadados não são suficientes. Linux mantém enumeração padrão e seleção manual.

No modo Hamachi/interface, o host usa somente TCP, escuta no IPv4 selecionado e define esse mesmo IPv4 como origem do dialer. O gater verifica o endereço local da conexão. A DHT não é criada, não há AutoRelay, mapeamento NAT ou DCUtR nesse host, e o anúncio contém apenas o endereço selecionado. Não se presume que 25/8 seja público só porque não é RFC1918. O modo nativo permanece e filtra 25/8 do anúncio; o desktop também filtra endereços privados das referências DHT públicas. Configurações de infraestrutura ficam preservadas e inativas no modo VPN.

O monitor de três segundos recria o host se IP/seleção mudarem e o interrompe quando a interface cair. Falha de bind não troca de modo. Identidade é carregada independentemente do host, portanto a janela pode exibir perfil e configurações mesmo sem VPN. O cache de contatos permanece em bbolt e recebe atualizações de endereços de pares autenticados. Sem caminho conhecido após mudanças simultâneas, novo convite é necessário.

Convites mantêm versão 1, acrescentando `scope` opcional assinado. O campo vazio é omitido para preservar a codificação assinada de convites anteriores. Convite Hamachi inclui somente o multiaddr virtual e não contém relay. O fluxo ConnectInvite conecta e abre a conversa; raw multiaddr também é aceito, com identidade conferida no transporte. Nenhuma lista de membros é inferida da faixa IP. A autorização da rede virtual é administrada no Hamachi; a identidade humana exige verificar o ID por outro canal.

## Descoberta

Não há varredura. Convites e contatos conhecidos são o mecanismo principal. mDNS depende de multicast link-local UDP 5353; não foi validado que as topologias Hamachi em uso o propaguem, nem foi verificada política que permita anúncios a todos os membros. Por isso não o habilitamos automaticamente. Uma implementação futura exigirá consentimento, escopo e testes de multicast próprios. Tailscale/ZeroTier poderão fornecer outros providers, sem alterar o serviço de chat; não foram implementados.

## Firewall e instalador

O teste de contato verifica handshake e ping libp2p; a leitura COM de Windows Defender Firewall é consultiva e somente leitura. Regras potencialmente compatíveis não garantem política efetiva, e um timeout não distingue falta de rede virtual de bloqueio. Nenhuma regra, adaptador ou configuração do Windows é modificada pelo backend.

O empacotamento 0.3 usa Go 64 bits e diálogos Win32, em vez do compilador NSIS 32 bits que não executava neste ambiente. Também foi avaliado o Inno Setup oficial assinado, cuja instalação de ferramentas foi revertida ao encontrar acesso negado fora do workspace; não foi necessário ampliar permissões do sistema. O instalador próprio grava aplicativo e desinstalador por usuário, cria atalhos via COM, registra desinstalação em HKCU e confere/instala WebView2 com bootstrapper Microsoft verificado no build. Nunca distribui Hamachi. Inclui manifest asInvoker e preserva o diretório de perfil. Não há assinatura de distribuição, rollback transacional completo de instalação ou atualização automática; instalação/upgrade/desinstalação precisam de teste de aceitação fora do sandbox. O helper de limpeza do desinstalador pode permanecer temporariamente no diretório de arquivos temporários do usuário.

## Fontes primárias

- [Microsoft GetAdaptersAddresses](https://learn.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getadaptersaddresses).
- [IP_ADAPTER_ADDRESSES](https://learn.microsoft.com/en-us/windows/win32/api/iptypes/ns-iptypes-ip_adapter_addresses_lh).
- [Documentação LogMeIn: IPv4 virtual 25/8](https://lmi-documentation-origin.azurewebsites.net/webhelp/EN/CentralUserGuide/LogMeIn/c_hamachi_Virtual_IP.html).
- [RFC 6762: mDNS](https://www.rfc-editor.org/info/rfc6762/).
- Código das APIs das versões libp2p e x/sys fixadas no go.mod.
