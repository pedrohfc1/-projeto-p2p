# Operar relays comunitários

Para o CLI atualizado com anúncio público, teste de reserva assinada e diagnóstico do serviço, siga [o guia de início e diagnóstico](relay-quickstart.md). A infraestrutura abaixo já existia na 0.2; a atualização mantém o protocolo e as quotas.

## Infraestrutura mínima

Uma máquina independente, Linux ou Windows AMD64, continuamente online, com endereço publicamente alcançável e TCP de entrada liberado na porta escolhida (exemplo: 4001). Pode ser uma máquina hospedada por um voluntário. O usuário de chat não precisa de IP público nem de alterar seu roteador; o operador do relay precisa oferecer alcance público real. `--public` apenas declara essa condição ao libp2p, não a cria.

Não há requisito de tamanho de VPS medido. Reserve memória, descritores e banda conforme a carga e monitore antes de aumentar usuários. Os limites do resource manager não são limite de RSS total do processo. Relays devem ser operados com consentimento e custos conhecidos. Para redundância, use pelo menos dois operadores em redes/provedores distintos; isso é recomendação arquitetural, não disponibilidade garantida.

## Inicialização

Compile o CLI com os scripts de build. Em máquina Linux pública:

```sh
./dist/projetop2p-linux-amd64 --data ./data/relay-a --listen /ip4/0.0.0.0/tcp/4001 --relay-server --public --daemon
```

Em Windows público:

```powershell
./dist/projetop2p-windows-amd64.exe --data ./data/relay-a --listen /ip4/0.0.0.0/tcp/4001 --relay-server --public --daemon
```

Confira o Peer ID. Divulgue somente o multiendereço público completo, usando o IP público ou DNS verdadeiro do operador e o ID real. Não distribua `0.0.0.0`, `127.0.0.1`, chaves privadas ou nomes fictícios. Nenhum endereço pronto está incluído nesta documentação.

Repita em um segundo operador com perfil e identidade próprios. Configure `--bootstrap ENDERECO_DO_OUTRO_RELAY` para ligar as DHTs. Clientes podem informar ambos em bootstraps e relays; um relay configurado também entra como semente inicial de conexão. Outros relays podem ser descobertos pelo namespace DHT `projeto-p2p/relays/v0.2`; o anúncio não é confiável por si só, e o AutoRelay verifica suporte e reserva.

Em Linux, mantenha o processo sob um supervisor de sua escolha. Exemplo systemd (ajuste usuário, diretórios e endereço reais):

```ini
[Unit]
Description=Projeto P2P relay comunitario
After=network-online.target
Wants=network-online.target
[Service]
User=projetop2p
WorkingDirectory=/opt/projeto-p2p
ExecStart=/opt/projeto-p2p/dist/projetop2p-linux-amd64 --data /var/lib/projetop2p-relay --listen /ip4/0.0.0.0/tcp/4001 --relay-server --public --daemon
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/var/lib/projetop2p-relay
PrivateTmp=true
[Install]
WantedBy=multi-user.target
```

Crie o usuário sem privilégios, o diretório gravável de dados e as regras necessárias antes de ativar o serviço. Não configurei máquinas externas, firewall ou serviços do seu sistema.

## Quotas e manutenção

Configuração da 0.2, definida no núcleo:

| Recurso | Teto |
|---|---|
| Reservas ativas | 64 |
| Reservas por IP / ASN | 8 / 32 |
| Circuitos por participante | 8 |
| Vida de uma reserva | 20 minutos |
| Circuito | 2 minutos, até 1 MiB por direção |
| Host | 128 conexões, 256 streams, 128 FDs contabilizados, 64 MiB contabilizados |

Os limites de bytes/duração restringem consumo **por circuito**, não constituem um limitador global de Mbps. Para um teto de banda total no operador, aplique quotas/shaping de rede no ambiente de hospedagem. Os valores são conservadores iniciais e não representam capacidade medida. Uma revisão operacional é necessária antes de tornar um relay amplamente público. Sybil e abuso distribuído não são resolvidos por quotas por IP.

AutoRelay mantém e renova reservas usando as APIs do libp2p e tenta até dois operadores. Quando um operador cai, endereços antigos podem permanecer em convites/peerstores por um período; contatos tentam caminhos alternativos conhecidos e a descoberta atualiza referências. Se todos os relays/bootstrap conhecidos ficarem inacessíveis, é necessário fornecer infraestrutura atualizada. Não há garantia de disponibilidade global ou descoberta sem algum caminho inicial.

O relay encaminha transporte criptografado; não armazena histórico nem decide identidades/permissões. O operador observa IPs, volumes, horários e metadados de conexão. Não publicar dados privados ou logs identificáveis sem uma política apropriada.
