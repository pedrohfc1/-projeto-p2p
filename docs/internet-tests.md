# Roteiro de testes 0.2

## A — Dois perfis no mesmo computador

Abra duas instâncias desktop com `--data ./data/desktop-a` e `--data ./data/desktop-b`, troque convites e envie mensagens nos dois sentidos. Reinicie e confira histórico, contatos, nome e identidade. O teste automatizado `internal/desktop` executa os mesmos casos de uso sem janela nativa; não substitui o teste visual.

## B — Dois computadores Windows na LAN

Copie somente o executável/instalador e documentação, nunca perfis. Abra cada aplicativo, autorize somente o acesso necessário no firewall conforme sua rede e gere os dois convites. Confira se incluem endereços LAN alcançáveis. Adicione contatos e envie mensagens. O caminho deve mostrar direto quando a LAN permitir. Registre resultado e regras aplicadas; esse cenário físico não foi executado aqui.

## C/D — Redes distintas, NAT/CGNAT, sem encaminhar portas nos clientes

1. Obtenha dois endereços reais de operadores independentes que executem o relay deste projeto. Veja relay-operations.md. Não use serviços arbitrários IPFS esperando permissão de relay.
2. Em ambos os desktops, configure esses endereços em relays e bootstraps. Mantenha o modo assistido ligado. Salve.
3. Em Estado da rede, aguarde os endereços com reserva. O libp2p filtra endereços não públicos de relays no anúncio AutoRelay: um teste só com relay em loopback não produzirá um convite de internet válido.
4. Gere convites **depois** da reserva, troque-os e adicione os contatos. Confira `/p2p-circuit` nos endereços do participante quando necessário. Um convite sem endereços alcançáveis não garante contato.
5. Envie mensagens nos dois sentidos. Confira se o cabeçalho mostra `relay` ou `direta`. A promoção para conexão direta pode ocorrer por DCUtR, mas não é obrigatória para aprovação do caminho relay.
6. Registre redes/operadoras, versão, acessibilidade, número de endereços com reserva, caminho observado e tempos de recuperação. Não publique IPs privados dos testadores sem consentimento.

Se não houver reserva, verifique se o relay está online, publicamente alcançável, oferece Circuit Relay v2 e não atingiu quotas. Se não houver bootstrap alcançável, a DHT não consegue juntar as redes. Se tudo estiver bloqueado pelo firewall de saída ou pela rede, o programa não consegue garantir conexão. Não há fallback para servidor central oculto.

## E — Desconexão e recuperação

Com dois relays ativos, encerre o primeiro operador e mantenha o segundo. Aguarde reconexão e envie nova mensagem sem editar o contato. Verifique o caminho usado. Depois interrompa temporariamente a rede de um cliente e restaure-a. A ausência de confirmação pode significar entrega incerta: consulte o destinatário antes de reenviar conteúdo importante. O teste automatizado de relay bloqueia conexões diretas e verifica failover automático por rotas alternativas, em loopback; não simula uma operadora real.

Deixe o sistema ativo além de 20 minutos para verificar renovação real das reservas e refaça convites/conexões. Esse teste de longa duração ainda não foi executado.

## F — Reinício e atualização

Feche as duas janelas, reabra, confira IDs e contatos e leia o histórico. Para atualização, instale por cima da versão anterior em uma máquina de teste e confirme preservação do perfil. Desinstale e confirme que o perfil permaneceu. Nenhum teste de instalador foi declarado aprovado nesta entrega.

Consulte validation.md para separar o que foi executado dos procedimentos ainda pendentes.
