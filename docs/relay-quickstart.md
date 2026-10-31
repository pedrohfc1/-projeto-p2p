# Bootstrap + Circuit Relay v2 — iniciar e diagnosticar

## Resultado da análise

O modo já estava implementado no CLI `cmd/projetop2p`, inclusive no binário Linux AMD64 da 0.2. `--relay-server` habilita `EnableRelayService`, com quotas, e a DHT usa `ModeServer` e o protocolo do Projeto P2P. Portanto, a mesma instância serve como bootstrap e Circuit Relay v2. O executável gráfico `ProjetoP2P.exe` não é o operador de relay.

Foram acrescentados somente diagnóstico/infraestrutura: `--announce`, `--check-relay`, `--status-every`, `/status`, contadores por instância e um exemplo systemd. Não houve alteração no código da interface gráfica.

O encaminhamento real e a recuperação após falha de um relay foram testados em loopback com caminhos diretos entre participantes bloqueados. O novo teste CLI verifica daemon sem stdin, serviço ativo, reserva real e voucher assinado. Isso não comprova alcance público de uma máquina ou CGNAT real.

## 1. Requisito da máquina pública

Use Linux AMD64 em uma máquina com IPv4 público alcançável (ou configuração de DNS/IPv6 apropriada) e entrada TCP 4001 permitida no firewall do sistema e da hospedagem. Seus dois usuários domésticos precisam somente conseguir sair até ela. **Nenhum dos dois precisa encaminhar portas no roteador de casa.**

`--public` não cria IP público nem abre firewall. Um computador doméstico inacessível atrás de CGNAT não se torna público com essa flag. Nenhuma máquina pública foi provisionada nesta entrega; é necessário obter uma de um operador/voluntário ou hospedagem independente.

## 2. Executar no Linux

Descompacte o pacote atualizado e entre em `projeto-p2p`. O CLI já está compilado para Linux AMD64; não exige Go, Wails, Node ou WebView2 para rodar:

```sh
chmod +x dist/projetop2p-linux-amd64
```

Substitua `IP_PUBLICO_REAL` pelo endereço real da máquina, sem colchetes ou nomes de exemplo, e execute:

```sh
./dist/projetop2p-linux-amd64 \
  --data ./data/relay \
  --listen /ip4/0.0.0.0/tcp/4001 \
  --announce /ip4/IP_PUBLICO_REAL/tcp/4001 \
  --relay-server --public --daemon --status-every 30s
```

`--announce` recebe o transporte público **sem** `/p2p/ID`. Ele o acrescenta aos endereços do host; o ID persistente é inserido automaticamente na saída completa. Também aceita `/dns4/SEU_DNS_REAL/tcp/4001`, se o DNS resolver para essa máquina. Endereços locais, portas efêmeras e prefixos de documentação não servem como anúncio público. O operador é responsável por conferir que o endereço informado realmente chega ao listener.

Alternativa com wrapper:

```sh
P2P_PUBLIC_ADDR='/ip4/IP_PUBLICO_REAL/tcp/4001' sh scripts/run-relay.sh
```

O diretório `data/relay` guarda a identidade e deve persistir. Não o apague nas atualizações nem o execute simultaneamente em dois operadores. Para um segundo relay, use máquina e identidade próprias. Opcionalmente conecte operadores com `--bootstrap MULTIADDR_COMPLETO_DO_OUTRO`.

Se desejar compilar somente a infraestrutura a partir do código, com Go >=1.26:

```sh
CGO_ENABLED=0 go build -buildvcs=false -trimpath -o dist/projetop2p-linux-amd64 ./cmd/projetop2p
```

## 3. Obter o multiaddr completo

A saída inclui `Peer ID: ...` e `Endereço: ...`. Copie a linha que começa com o IP público ou DNS anunciado e termina no ID real. O formato é:

```text
/ip4/IP_PUBLICO_REAL/tcp/4001/p2p/ID_REAL_DO_RELAY
```

Não use `0.0.0.0` (escuta), `127.0.0.1` (loopback), IP privado ou um ID inventado. A presença da linha não comprova acesso externo; execute o teste remoto abaixo. Sem `--daemon`, `/addresses` repete os endereços e `/status` mostra o diagnóstico. Com daemon, o JSON periódico também contém `id` e `addresses`.

Uma DHT vazia (`dhtPeers: 0` e anúncio sem pares) pode ocorrer enquanto esse primeiro bootstrap está sozinho. Isso não impede o serviço de aceitar o primeiro cliente. Depois da entrada de participantes compatíveis, a tabela deve começar a receber pares.

## 4/5. Configurar os dois aplicativos Windows

Se essa máquina oferece as duas funções, coloque **o mesmo multiaddr completo** nos dois campos:

| Campo | Valor |
|---|---|
| Bootstraps | `/ip4/IP_PUBLICO_REAL/tcp/4001/p2p/ID_REAL_DO_RELAY` |
| Relays independentes | `/ip4/IP_PUBLICO_REAL/tcp/4001/p2p/ID_REAL_DO_RELAY` |

Substitua os valores pelo endereço copiado do seu operador. Um endereço por linha. Com dois operadores, inclua ambos nos dois campos. Mantenha o modo assistido NAT/CGNAT habilitado e salve as configurações.

**Não coloque `/p2p-circuit` nesses campos de infraestrutura.** Esse componente aparece no endereço do usuário que reservou o relay:

```text
/ip4/IP_PUBLICO_REAL/tcp/4001/p2p/ID_REAL_DO_RELAY/p2p-circuit/p2p/ID_REAL_DO_USUARIO
```

Gere os convites depois de a reserva aparecer. Cada usuário gera sua própria identidade e convite; nunca compartilhe pastas de perfil ou chaves.

## 6. Verificar as reservas

### De um Windows em outra rede

Use o CLI atualizado do pacote, não o executável gráfico:

```powershell
Test-NetConnection -ComputerName IP_PUBLICO_REAL -Port 4001

.\dist\projetop2p-windows-amd64.exe --check-relay /ip4/IP_PUBLICO_REAL/tcp/4001/p2p/ID_REAL_DO_RELAY
```

No Linux remoto:

```sh
./dist/projetop2p-linux-amd64 --check-relay /ip4/IP_PUBLICO_REAL/tcp/4001/p2p/ID_REAL_DO_RELAY
```

O primeiro comando testa TCP; o segundo faz uma reserva **real** Circuit Relay v2 e valida o voucher assinado, incluindo identidade do relay, cliente e expiração. Sucesso retorna exit code zero e JSON contendo:

```json
{
  "reserved": true,
  "voucherVerified": true,
  "expires": "data real retornada pelo relay",
  "limitSeconds": 120,
  "limitBytes": 1048576,
  "addresses": ["multiaddr público real do relay"]
}
```

O JSON acima é ilustrativo; os campos `relayId` e `clientId` também aparecem na saída real. Em erro, o comando retorna exit code não zero e a causa (porta inacessível, ID incorreto, protocolo ausente, recusa de quota ou timeout). O prazo é de 15 segundos.

O probe usa uma identidade temporária e **fecha a própria conexão ao terminar**, liberando sua reserva. Não grava chaves, não substitui a identidade do usuário e não mantém uma reserva para o aplicativo. Se executado só dentro do servidor ou em loopback, não comprova alcance pela internet. `addresses` vazio significa que a reserva foi aceita, mas o relay não devolveu um endereço público utilizável: confira `--announce` e a rede antes de distribuir convites.

### No operador

`--status-every 30s` imprime uma linha JSON de diagnóstico a cada 30 segundos; no modo interativo, use `/status`. Exemplo dos campos relevantes:

```text
relayService.enabled              serviço realmente ativado pelo libp2p
relayService.activeReservations   reservas observadas como ativas no processo
relayService.reservationsAccepted novas reservas aceitas desde a inicialização
relayService.reservationsRenewed  renovações observadas
relayService.reservationsRejected pedidos não aceitos (inclui erros)
relayService.activeCircuits       circuitos observados em andamento
relayService.circuitsOpened       circuitos abertos desde a inicialização
relayService.bytesForwarded       bytes de transporte encaminhados nas duas direções
dhtPeers                         pares da tabela Kademlia local
```

Os contadores vêm dos callbacks do serviço de relay, não da quantidade de peers conectados. São telemetria por processo, não um registro persistente por cliente. Reiniciar zera os contadores. Após um probe, `reservationsAccepted` deve aumentar; `activeReservations` pode voltar a zero, pois o probe encerra. Com os aplicativos abertos e usando esse relay, espere reservas ativas. Bytes/circuitos podem incluir o handshake criptográfico; a prova de chat é a mensagem recebida no outro aplicativo.

### Nos aplicativos

Em **Estado da rede**, aguarde endereços com reserva; os endereços devem incluir `/p2p-circuit`. Esse número conta endereços, não operadores distintos. Depois gere/troque convites e envie mensagens nos dois sentidos. No contato, confira `relay` ou `direta` (se ocorreu conexão direta/upgrade). Reserva aceita não garante que todos os caminhos ou todas as redes estejam funcionando. Continuidade de renovação por mais de 20 minutos e testes em duas casas ainda precisam de validação externa.

## Serviço Linux persistente

Há um unit pronto em `configs/projetop2p-relay.service`. Para instalar numa máquina pública sob seu controle (ajuste os caminhos e o IP):

```sh
sudo useradd --system --user-group --home-dir /var/lib/projetop2p-relay --shell /usr/sbin/nologin projetop2p
sudo install -d -m 0755 /opt/projeto-p2p/dist /etc/projetop2p
sudo install -m 0755 dist/projetop2p-linux-amd64 /opt/projeto-p2p/dist/
printf '%s\n' 'P2P_PUBLIC_ADDR=/ip4/IP_PUBLICO_REAL/tcp/4001' | sudo tee /etc/projetop2p/relay.env
sudo install -m 0644 configs/projetop2p-relay.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now projetop2p-relay
sudo systemctl status projetop2p-relay
sudo journalctl -u projetop2p-relay -f
```

Crie o usuário apenas se ele ainda não existir. O systemd cria `/var/lib/projetop2p-relay` para a identidade. Pare o serviço antes de substituir o binário e preserve esse diretório. O unit utiliza SIGTERM para encerramento limpo. Se usa UFW, uma regra de entrada TCP 4001 pode ser necessária; libere também a porta no firewall da hospedagem. Não é necessário encaminhar portas nas casas dos usuários. Não alterei seu sistema, firewall ou infraestrutura externa.

## Limites e evidências

Quotas preservadas: 64 reservas, 8 reservas por IP, 32 por ASN, 8 circuitos por peer; reserva de 20 minutos; circuito de até 2 minutos/1 MiB por direção. Há limites adicionais do resource manager. Não é um teto agregado de Mbps nem proteção completa contra Sybil/DoS; monitore antes de disponibilizar amplamente. Consulte `relay-operations.md`.

Resultados desta atualização em `relay-validation.md` e logs em `docs/evidence/relay-ops`. Builds Linux são cross-compilados no Windows; não foram executados em um Linux público. Nem o unit systemd nem conexão entre operadoras foi executado aqui. Um relay basta para o teste inicial; dois operadores independentes reduzem o ponto único de falha.

Referências: [Circuit Relay v2](https://github.com/libp2p/specs/blob/master/relay/circuit-v2.md), [API Reserve](https://pkg.go.dev/github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client). Nenhum serviço público arbitrário foi presumido disponível.
