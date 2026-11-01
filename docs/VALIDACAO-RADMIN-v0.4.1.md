# Correção de conexão Radmin — 0.4.1 — 31/10/2026

## Causa e correção

Na 0.4.0 o modo Automático significava libp2p nativo: detectar Hamachi/Radmin não selecionava nenhuma VPN. O host nativo usava porta dinâmica e podia gerar convite com endereços LAN inadequados para outra residência. Salvar configurações não informava a falha ao iniciar o host. A seleção Radmin explícita já existia, mas o fluxo permitia continuar no modo errado sem uma escolha clara.

Na 0.4.1, a geração de convite em modo nativo com VPN detectada pede escolher Radmin/Hamachi ou continuar explicitamente no nativo. A tela de rede oferece ativação direta e separa interface detectada, interface selecionada e nó em escuta. Receber convite Radmin em outro modo pede confirmação para ativar a VPN antes de salvar/conectar. Não há mudança silenciosa de rede quando ambas estão instaladas.

A escolha fica persistida pelo GUID com porta fixa (4001 por padrão), mantendo o perfil e a chave. IP/índice/estado são novamente lidos do adaptador selecionado. Seleção ignora APIPA e aliases de LAN; se a interface cair ou perder IPv4 válido, o nó não muda para outra VPN/rede pública. SaveSettings informa quando a escolha foi salva, mas o host não iniciou.

No Radmin: escuta e origem TCP vinculadas ao IPv4 selecionado, anúncio restrito a esse IPv4 e porta real, convites assinados com Peer ID real e escopo Radmin; destinos limitados ao TCP IPv4 26.x. Hamachi, LAN, APIPA, DNS/relay e porta zero não entram nesse escopo. Gater também barra endereços incompatíveis aprendidos pelo peerstore. Convites novos substituem o cache de reconexão da interface; endereços antigos do mesmo peer não são acumulados nesse host. O nativo conserva DHT/bootstrap/relay e LAN útil; APIPA é removido dos anúncios nativos. A faixa 26.x é a pista do perfil, não prova de associação ou conectividade.

O diagnóstico exibe modo, GUID/nome/índice, IPv4 local, porta de escuta, endereços fornecidos à discagem, caminho local/remoto autenticado, resultado do ping e detalhe da falha. Não atribui timeout automaticamente ao firewall. Porta de origem dinâmica de saída é normal e distinta da porta de escuta anunciada.

## Executado

- **28 testes Go**: regressões anteriores e novos casos de aliases APIPA/LAN, duas VPNs, seleção/persistência, convite incompatível, porta zero/observada incorreta, filtragem de destino e diagnóstico de conexão recusada sem salvar identidade não autenticada.
- **5 testes frontend**: escolha explícita de VPN, filtro de adaptadores por provedor, rótulos e limites UTF-8/envio. TypeScript/Vite compilados.
- **4 testes de infraestrutura de distribuição**, go vet e go mod verify.
- **Adaptadores reais Radmin e Hamachi** enumerados e bind TCP testado individualmente, sem alterar sua configuração.
- **API desktop com Radmin real**: seleção do adaptador Radmin com as duas VPNs disponíveis, porta não zero escolhida e persistida, anúncio/convite com IPv4/porta/Peer ID reais e reinício preservando o perfil. Recepção de convite no modo nativo exigiu confirmação de modo.
- **Dois perfis locais pelo adaptador Radmin real**: handshake autenticado, ping libp2p, texto persistido e histórico/identidade após reinício. Não é teste entre dois computadores.
- **Regressão API desktop**: contatos, convites, mensagem durável, reinício/crash e controle de acesso da ponte QA.
- **CLI de três nós**: DHT, texto assinado, bootstrap/fallback, crash, identidade, mudança de endereço e recuperação.
- **Relay CLI**: daemon, status, reservas assinadas, quotas e contadores.
- Build desktop Windows AMD64 e CLI Windows/Linux AMD64. Instalador real ProjetoP2P-Setup-v0.4.1.exe gerado; payload executado em modo de verificação com hash do app, MIT e licenças transitivas conferidos. Bootstrapper Microsoft validado; nenhuma VPN incluída. App/setup sem certificado próprio de assinatura.
- npm audit sem vulnerabilidades informadas; auditoria de fontes e gitleaks sem segredos detectados. Logs locais em docs/evidence/v0.4.1 são ignorados no Git; relatório público não inclui IPs pessoais nem chaves.

## Limites e teste restante

O teste entre os dois PCs Windows em residências diferentes ainda precisa ser repetido. Atualize ambos para 0.4.1, escolha Radmin, confirme IP virtual/porta/nó em escuta e gere convites novos. Use o roteiro CONEXAO-RADMIN-VPN.md; antigos convites nativos contendo só LAN não ganham o IP Radmin por atualização automática.

A inspeção visual da janela nativa e o fluxo completo de instalar/atualizar/desinstalar em Windows 10/11 de aceitação não foram executados; backend/assets/payload não equivalem a essas verificações. O perfil em %APPDATA%/ProjetoP2P mantém o mesmo caminho/formato e não é acessado pelo instalador. Desktop Linux 0.4.1 não foi compilado nativamente neste ambiente Windows; o workflow Linux permanece preparado. CLI Linux foi cross-compilado.

Nenhuma regra de firewall, rota, configuração de VPN ou roteador foi alterada. O teste local não comprova acesso remoto em qualquer política de rede. O aviso DHT GO-2024-3218, sem versão corrigida conhecida, continua aplicável ao modo nativo e bloqueia a publicação pelo workflow de segurança; a DHT fica desativada no modo Radmin. Não foram publicados código, tags ou releases desta correção durante esta tarefa.
