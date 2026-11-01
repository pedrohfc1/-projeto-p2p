# Conectar dois Windows usando Hamachi — versão 0.4.1

## Antes de abrir o Projeto P2P

Cada pessoa instala e autentica o Hamachi por conta própria e entra na **mesma rede virtual**, com autorização do administrador dessa rede. Confiram no cliente Hamachi se os dois aparecem disponíveis. O aplicativo não instala Hamachi, não lê senhas, não autentica e não presume acesso a uma lista de membros.

Não é necessário contratar VPS, configurar bootstraps ou encaminhar portas dos roteadores de casa para esse modo. Ainda é necessário que a rede virtual e o firewall permitam TCP entre os endereços virtuais. O Hamachi depende da infraestrutura de seu fornecedor e pode encaminhar tráfego por retransmissão própria.

## Instalação 0.4.1

Feche a versão antiga e execute `ProjetoP2P-Setup-v0.4.1.exe` nos dois PCs. Confirme a instalação por usuário e escolha se deseja atalho de área de trabalho. O instalador registra desinstalador e verifica/instala WebView2 se necessário, com internet. Não inclui Hamachi. Não copie sua pasta de perfil para seu amigo.

O diretório de programa é `%LOCALAPPDATA%\Programs\ProjetoP2P`; o perfil permanece em `%APPDATA%\ProjetoP2P`. A identidade, os contatos, as mensagens e configurações da versão anterior são preservados. Chave protegida por DPAPI é vinculada ao usuário/máquina, não é um arquivo portável para compartilhar.

## Configurar A e B

1. Abra o Projeto P2P pelo menu Iniciar.
2. Em **Configurações**, escolha **Modo de conectividade → Hamachi**.
3. Use **Detectar automaticamente** ou selecione o adaptador confirmado. Mantenha porta TCP **4001**, salvo se estiver em uso. Nos dois computadores diferentes, a porta pode ser igual.
4. Salve. Abra **Estado da rede → Conectividade — Hamachi**.
5. Confira: Hamachi detectado, adaptador ativo, IPv4 virtual válido, Peer ID e endereço anunciado. O endereço tem formato `/ip4/25.x.x.x/tcp/4001/p2p/ID`; o aplicativo preenche os valores reais.
6. Se não houver detecção confiável, volte a Configurações e use **Seleção manual de interface de rede**, escolhendo o adaptador virtual Hamachi. Não selecione a placa de internet por engano.

**Não preencha Bootstraps/Relays para conectar pela VPN.** Valores antigos ficam salvos, mas inativos nesse modo. Para voltar à rede nativa, selecione Automático e salve; isso é uma ação explícita, não um fallback automático após erro.

## Convite e conversa

No A, clique **Copiar convite**. Envie o texto ao B por um canal de sua confiança. No B, clique **Conectar por convite**, cole e confirme. O programa interpreta o endereço contido no convite e verifica assinatura, prazo e identidade. Faz a conexão autenticada e abre a conversa, sem exigir digitar IP/ID.

Também é aceito um multiaddr completo copiado do painel. Ele não tem o nome assinado do envelope, portanto o Peer ID só é autenticado no handshake, e um contato novo recebe nome provisório. Não aceite endereços/IDs de desconhecidos.

Para que A e B vejam os nomes escolhidos por ambos, troquem os dois convites. Um contato que recebe mensagem antes do convite recíproco aparece inicialmente identificado pelo ID. As mensagens continuam assinadas e o transporte libp2p continua criptografado, inclusive dentro da VPN. O histórico em disco não recebe criptografia de conteúdo nesta versão.

O convite é válido por sete dias e carrega apenas chave pública, ID, nome e endereço necessário. Nunca contém chave privada. O escopo Hamachi/interface faz parte da assinatura. Se o IP mudar e a reconexão por contatos conhecidos não bastar, gere e troque um novo convite.

## Diagnóstico

Selecione o contato e, no painel de rede, clique **Testar conectividade**. Sucesso exige handshake autenticado e resposta ao ping libp2p. No Windows, também há leitura consultiva das regras de entrada potencialmente compatíveis com o programa e a porta. Não é uma prova completa da política de firewall: restrições de endereço, interface, políticas de grupo e regras de terceiros podem prevalecer.

O indicador **interface pronta** só confirma estado/IP local. O indicador **contato conectado** confirma uma conexão P2P. Um caminho mostrado como direto é direto para o libp2p, não necessariamente para a infraestrutura física do Hamachi.

Quando houver erro:

| Situação | Ação |
|---|---|
| Hamachi não detectado | Verifique instalação/conexão; detecção ausente não prova que o programa não está instalado. Use seleção manual se o driver não puder ser identificado. |
| Adaptador inativo | Reconecte pelo próprio Hamachi. O Projeto P2P não ativa adaptadores. |
| Sem IPv4 virtual | Aguarde o Hamachi atribuir endereço; no modo Hamachi, é esperado IPv4 25/8. |
| Porta em uso | Feche outra instância ou escolha outra porta e gere novo convite. |
| Peer não responde | Confirme mesma rede virtual, usuário disponível, aplicativo aberto, convite atual e firewall. Timeout não determina sozinho a causa. |
| Convite inválido | Copie o texto completo; assinatura, ID, validade e endereço precisam conferir. |
| Interface caiu | O nó desse modo é interrompido, sem passar à rede pública. Ele tenta voltar quando a interface/IP reaparecer. |

### Verificações manuais, somente leitura

Em PowerShell, adaptadores/evidências do Projeto P2P:

```powershell
.\dist\projetop2p-windows-amd64.exe --interfaces
```

Com o aplicativo aberto em A, o B pode testar o IPv4 virtual real de A, e depois inverter o teste:

```powershell
Test-NetConnection -ComputerName IP_HAMACHI_REAL_DO_OUTRO_PC -Port 4001
```

No receptor, confirme que o aplicativo escuta na porta escolhida:

```powershell
Get-NetTCPConnection -State Listen -LocalPort 4001
Get-NetFirewallProfile | Select-Object Name,Enabled,DefaultInboundAction
```

ICMP/ping tradicional pode estar bloqueado mesmo quando TCP funciona; o teste do aplicativo usa o protocolo libp2p sobre TCP, não ICMP.

### Firewall: não desativar

Nenhuma regra foi criada ou alterada pelo aplicativo ou durante esta entrega. Se o Windows pedir autorização para o aplicativo, a decisão é do usuário. Se o teste remoto apontar problema e uma regra for necessária, o proprietário do PC deve autorizá-la antes de executá-la. Exemplo **opcional**, em PowerShell elevado e somente após consentimento, substituindo caminho, IP e alias pelos reais:

```powershell
New-NetFirewallRule -DisplayName "Projeto P2P Hamachi TCP 4001" `
  -Direction Inbound -Action Allow -Protocol TCP -LocalPort 4001 `
  -Program "$env:LOCALAPPDATA\Programs\ProjetoP2P\ProjetoP2P.exe" `
  -LocalAddress IP_HAMACHI_LOCAL_REAL -RemoteAddress 25.0.0.0/8 `
  -InterfaceAlias "ALIAS_REAL_DO_ADAPTADOR_HAMACHI" -Profile Any
```

Essa regra restringe programa, endereço local, interface e porta. Se mudar a porta/IP/caminho, revise-a. Não a execute sem preencher os valores, sem autorização ou se não for necessária. Não desative Windows Defender Firewall nem proteções do navegador.

## Roteiro obrigatório entre casas — ainda não executado

Registre os resultados de cada item, sem divulgar senhas, convites ou IPs dos participantes publicamente:

1. Dois Windows em redes residenciais diferentes, ambos com versão 0.4.1.
2. Ambos na mesma rede Hamachi e disponíveis no cliente Hamachi.
3. Adaptador/IP corretos nos dois Projetos P2P.
4. TCP alcançável em ambos os sentidos ou mensagem clara de falha.
5. Convites trocados, Peer IDs conferidos e contatos conectados.
6. Mensagens nos dois sentidos, com confirmação e histórico após reinício.
7. Queda/reconexão do Hamachi, preservando identidade e contatos.
8. Nenhum encaminhamento de porta de roteador configurado para esse teste.

O sistema não verifica a lista de membros do Hamachi nem sua configuração de roteador. Esses itens precisam de confirmação dos testadores. Veja `VALIDACAO-DISTRIBUICAO.md` para distinguir testes automatizados, bind real local e o que permanece sem validação.
