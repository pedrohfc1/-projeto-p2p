# Ameaças e limites 0.1

Documento de base da 0.1. A 0.2 adiciona persistência, convites e relays; veja [ameaças desktop 0.2](desktop-v0.2.md) para as mudanças e limites atuais.

Pares desconhecidos podem fornecer endereços falsos, mensagens inválidas, dados repetidos e tráfego excessivo. Peer ID é identidade criptográfica, não identidade civil, autorização ou reputação. O usuário deve verificar por outro canal os IDs que pretende contatar.

Controles implementados: transporte autenticado; assinatura Ed25519 vinculada ao remetente do stream e ao destinatário; nonce de 128 bits; validade de ±2 minutos; replay em memória retido por 5 minutos; limite global de 4096 entradas de replay e 256 emissores; até 10 mensagens/s por emissor (intervalo mínimo de 100 ms); 4096 bytes UTF-8 de texto; frame de até 32768 bytes; deadline de 5 segundos por stream; fila de 64 mensagens. Recepção sem espaço ou além dos limites é rejeitada, não bloqueada indefinidamente.

O resource manager libp2p tem teto de 128 conexões, 256 streams, 128 descritores contabilizados e 64 MiB de memória contabilizada. Isso não limita o RSS total do processo, disco ou toda banda consumida. O connection manager busca manter entre 32 e 64 conexões após o período de graça. Cache local tem até 256 pares. Os limites são constantes nesta versão; flags de políticas serão uma evolução. Descarte da fila de eventos de diagnóstico é possível sob carga.

Limites conhecidos: reiniciar limpa o cache de replay; mensagens ainda temporalmente válidas podem ser reapresentadas após reinício. Não há entrega exatamente uma vez, persistência, revogação, autorização, moderação, resistência a Sybil/eclipse ou limites globais de banda. Verificação criptográfica consome CPU antes do controle de frequência da aplicação; limites de streams mitigam, mas não eliminam DoS. Registros DHT expiram e endereços podem estar desatualizados.

Identidade Linux: diretório 0700, arquivo 0600. Identidade Windows: DPAPI do usuário, com ACLs herdadas do diretório escolhido; processos do mesmo usuário e administradores estão fora da proteção. Não utilize diretório compartilhado com permissão de escrita por terceiros: há risco de exclusão/substituição, links e negação de serviço. Não execute duas instâncias com o mesmo diretório. Identidade não pode ser exportada de Windows para Linux simplesmente copiando o arquivo. Não há recuperação automática se o perfil DPAPI ou a identidade se perder.

Não existe criptografia de arquivos, histórico ou backup no aplicativo. Os terminais podem reter mensagens em logs externos. Relays veem metadados de transporte; DHT expõe presença e endereços. Endereços privados são aceitos: discagem automática de pares desconhecidos pode alcançar destinos internos, portanto use ambiente sem acesso privilegiado para testes públicos. Uma política de endereços/reputação é necessária antes de operação pública ampla.

Conexões podem estar silenciosamente partidas até que o transporte detecte a falha. A CLI observa mudanças de conexão a cada segundo; isso não é um detector perfeito de falhas nem consenso de disponibilidade. As tentativas de reconexão e a DHT não conseguem recuperar uma rede sem nenhum endereço atualmente alcançável.
