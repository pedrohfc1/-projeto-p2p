# Configuração 0.1

Configuração por flags, sem arquivo JSON redundante. `--help` lista as opções.

- `--data`: diretório exclusivo de uma identidade; padrão `data`.
- `--listen`: repetível; padrão TCP IPv4/IPv6 em porta efêmera.
- `--bootstrap`: repetível; nenhum nó público embutido.
- `--relay`: repetível; candidatos estáticos de Circuit Relay v2.
- `--relay-server`: disponibiliza serviço voluntário de relay quando o libp2p determina alcance público.

O peer ID ao final do endereço é obrigatório. Para disponibilidade externa use uma porta fixa e anuncie um endereço alcançável. Não use o mesmo diretório de identidade em duas instâncias simultâneas.

## Operação Bootstrap + Relay (CLI atualizado)

- `--public --relay-server --daemon`: serviço público sem depender de stdin; o operador precisa oferecer acesso externo real.
- `--announce`: endereço TCP público fixo sem `/p2p/ID`; o ID persistente será acrescentado pelo host.
- `--status-every 30s`: diagnóstico JSON nos logs, com estado real do serviço, reservas, circuitos e bytes.
- `/status`: mesmo diagnóstico no modo interativo.
- `--check-relay MULTIADDR_COMPLETO`: teste pontual de reserva assinada com identidade temporária, saída JSON e exit code não zero em falha. Não grava nem substitui identidade do usuário.

Serviço systemd de exemplo: `configs/projetop2p-relay.service`. Guia operacional completo em `docs/relay-quickstart.md`. Nenhuma alteração na interface gráfica acompanha esta atualização de infraestrutura.
