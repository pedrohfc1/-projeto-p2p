# Laboratório isolado Tailscale / tsnet — PeeTwoPee 0.10.6

Este executável é um protótipo separado. **Não transporta o chat, comunidades, voz ou vídeo do aplicativo principal** e não modifica sua seleção de rede. A opção Tailscale na central principal permanece desabilitada com essa explicação.

O módulo oficial `tailscale.com/tsnet v1.104.7` exige Go 1.27.4; o app continua em Go 1.26. O módulo Go independente evita atualizar silenciosamente as dependências do transporte que já funciona. Compile com `./build.ps1`, que usa `-tags ts_omit_logtail`, executa testes e gera os avisos de licença. A biblioteca Tailscale usa BSD-3-Clause; outras dependências e licenças constam em Tailscale-Lab-licencas.zip. Não há Tailscale/Wintun/Radmin/Npcap externo redistribuído pelo protótipo.

## Arquitetura verificada

- `tsnet.Server.Start`, `LocalClient`, `Status` e `StartLoginInteractive` implementam login interativo; não há chave compartilhada ou senha global.
- `Server.Listen` é um listener userspace, não um listener TCP do sistema. Foi implementado um adaptador limitado a 8 conexões que transfere bytes para um listener libp2p em loopback; O destino precisa constar na lista de peers atuais do LocalAPI. A conexão de saída usa a pilha userspace do SDK fixado, vinculada ao endereço Tailscale local. Isso evita uma conexão pelo sistema se o destino sair da tailnet. O adaptador depende de uma API interna do SDK e precisa ser validado ao atualizar a versão fixada.
- Noise autentica o Peer ID Ed25519 esperado. O connection gater aceita somente a identidade explicitamente autorizada por `-allow` ou `-peer`. Endereço IP e autenticação Tailscale não substituem essa validação.
- A identidade do laboratório usa a mesma biblioteca de identidade do projeto, mas um perfil separado. Não abra seu perfil principal no laboratório.
- O estado tsnet fica em uma subpasta privada; diretórios desconhecidos não vazios ou links são recusados. No Windows, a ACL permite apenas o usuário atual e SYSTEM; no Unix, modo 0700. Identidade Ed25519 usa a proteção existente (DPAPI no Windows).
- O SDK recebe `Tun=nil`, usa o netstack e configuradores de rede/DNS no-op. Não configura serviço, roteamento global ou DNS do Windows. O portmapper é desativado no processo pelo knob oficial TS_DISABLE_PORTMAPPER da versão fixada, sem UPnP/NAT-PMP/PCP automáticos. Nenhum driver é incluído. Os diagnósticos do laboratório são locais e o envio de logs do SDK é desativado pelo opt-out oficial `envknob.SetNoLogsNoSupport` e pela tag de build.

## Auditoria Wintun: distinção necessária

A auditoria estrita do app **não passa** para o protótipo, porque o SDK compartilha código Windows com referência textual a `wintun.dll` (o verificador do laboratório também consulta o nome desse módulo). Não se removeu nem relaxou a auditoria principal. A referência não é importação estática ou prova de dependência obrigatória. O comando `-check-userspace` iniciou o SDK real sem autenticar e retornou `tun=false`, `wintunLoaded=false`, `state=NeedsLogin`. Portanto, a inicialização userspace foi testada sem carregar essa DLL; transferência real pela tailnet ainda não foi validada. O relatório PE separado lista a referência explicitamente. Não declarar ausência da string ou teste completo de tráfego sem driver.

## Testar dois computadores Windows (mesma tailnet autorizada)

1. Em cada PC, extraia o laboratório e abra PowerShell na pasta. Gere seu Peer ID sem iniciar Tailscale:
   ```powershell
   .\PeeTwoPee-Tailscale-Lab.exe -identity-only
   ```
   Troque os **Peer IDs públicos** por um canal de confiança. Nunca envie a pasta de estado, chaves, URL de login ou credenciais.
2. No PC B, autorize o Peer ID de A e inicie o nó:
   ```powershell
   .\PeeTwoPee-Tailscale-Lab.exe -name p2p-lab-b -allow PEER_ID_A -open-auth
   ```
   O navegador abrirá a URL HTTPS oficial somente se for necessário autenticar. A própria pessoa deve fazer login e aprovar o dispositivo; administradores podem exigir aprovação adicional. A sessão fica persistida para próximas execuções. Ctrl+C desconecta o nó do processo sem apagar o estado.
3. Leia o multiaddr real que B imprime após a autorização. No PC A:
   ```powershell
   .\PeeTwoPee-Tailscale-Lab.exe -name p2p-lab-a -peer /ip4/IP_TAILSCALE_B/tcp/7447/p2p/PEER_ID_B -open-auth -duration 10 -report tailscale-resumo.json
   ```
   Também são aceitos 30 e 60. Sem `-duration`, somente escuta, sem teste ativo. A sonda usa libp2p ping de 32 bytes em ~1 Hz; não é teste de carga/throughput. O resultado contém RTT médio/P95 e condições, sem credenciais ou conteúdo. A telemetria de `LocalClient.Ping` disco confirma direto, DERP ou peer relay quando esses campos estão presentes; falta de confirmação permanece desconhecida.
4. Importe `tailscale-resumo.json` em Diagnóstico → Comparar conexões no PeeTwoPee. Compare somente condições equivalentes. O resumo do laboratório não tem FPS/voz nem throughput sob carga; não significa que o chat do app utilizou Tailscale.

Os dois usuários precisam ter acesso à mesma tailnet, ou compartilhamento e ACL explicitamente concedidos no Tailscale, incluindo a porta do laboratório. Um convite PeeTwoPee não concede esse acesso. Visibilidade de peer no status também não comprova que a ACL permite o protocolo. Erros de conexão podem indicar tailnet/ACL, Peer ID ou alcance; sem execução real não se atribui uma causa específica.

Estado padrão: `%APPDATA%\PeeTwoPee-tsnet-lab` (separado de `%APPDATA%\ProjetoP2P`). Use `-state-dir` com outra pasta exclusiva para perfis isolados. Variáveis de autenticação Tailscale no ambiente são recusadas para evitar uso acidental de chaves. Sem auth key no binário. A URL de login, quando exibida no terminal, é sensível e não faz parte do JSON exportado.

## Testes e limites

Passaram: build Windows, testes de política de URL, diretório privado, restrição de endereços, encerramento do adaptador; libp2p/Noise real sobre um underlay loopback simulado com aceitação do Peer ID autorizado e rejeição de outro. O SDK real iniciou sem TUN, sem DLL Wintun carregada e aguardando login. Identidades locais foram testadas quanto a persistência em perfis isolados.

**Pendentes:** login humano no serviço hospedado, aprovação da tailnet, conexões em PCs de redes diferentes, peer relay, redescoberta remota após reinício e integração de mídia/chat do aplicativo. Os testes de loopback não comprovam NAT traversal, CGNAT ou autenticação bem-sucedida. Não há troca automática de transporte nem publicação pública.

Referências oficiais verificadas: [tsnet](https://tailscale.com/docs/features/tsnet), [API tsnet](https://pkg.go.dev/tailscale.com/tsnet), [LocalAPI da versão fixada](https://github.com/tailscale/tailscale/blob/v1.104.7/client/local/local.go), [campos de ping](https://github.com/tailscale/tailscale/blob/v1.104.7/ipn/ipnstate/ipnstate.go), [licença](https://github.com/tailscale/tailscale/blob/v1.104.7/LICENSE).

## Teste de carga limitada

Primeiro autentique os dois nós conforme os passos acima. Depois reinicie o receptor com a permissão explícita. A janela de 90 segundos começa quando o receptor está pronto e aceita um único teste do Peer ID autorizado.

```powershell
.\PeeTwoPee-Tailscale-Lab.exe -load -name p2p-lab-b -allow PEER_ID_A
```

No emissor, acrescente `-load` e informe o endereço e a duração:

```powershell
.\PeeTwoPee-Tailscale-Lab.exe -load -name p2p-lab-a -peer /ip4/IP_TAILSCALE_B/tcp/7447/p2p/PEER_ID_B -duration 10 -report tailscale-carga.json
```

O teste usa a mesma sonda do aplicativo, com blocos de 4 kB e limite de 128 kbit/s de payload por direção. O JSON apresenta RTT médio e P95 dos ecos, variação sucessiva e vazão observada. Importe em **Diagnóstico > Comparar conexões** e compare com uma janela de carga limitada de duração equivalente. Ctrl+C encerra a permissão e a conexão do laboratório.

## Reproduzir o teste local do SDK

No módulo do laboratório, execute `go test -tags ts_omit_logtail -run TestTailnetUserspaceNoise -count=1 -v`. A fixture usa controle, DERP e STUN locais da infraestrutura de teste do próprio SDK. Ela valida dados libp2p/Noise sobre tsnet, separação de identidade, persistência do cadastro e ausência de Wintun carregada no Windows. O serviço hospedado e a rede entre casas não participam do teste. [Fonte do SDK](https://github.com/tailscale/tailscale/tree/v1.104.7/tsnet)
