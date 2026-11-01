# Auditoria de segurança — versão 0.5 (em desenvolvimento)

Base analisada: 0.4.1. Banco atual é bbolt, não SQLite. Escopo inicial: identidade, mensagens individuais, convites, streams, Wails, persistência e distribuição. Este documento será completado com a validação da entrega.

## Achados iniciais e correções

- **Integridade do histórico (média):** AddMessage reutilizava a chave (remetente, ID), mas substituía conteúdo se um remetente assinasse outro texto com o mesmo ID após o cache de replay expirar/reiniciar. Agora texto, remetente, conversa e timestamp são imutáveis por ID; atualizações de estado de envio permanecem possíveis. Duplicatas idênticas continuam idempotentes. Teste cobre reinício e tentativa de substituição.
- **Leitura local sem limite da chave (baixa, acesso ao perfil necessário):** leitura passa a ter teto de 64 KiB e rejeita link simbólico no arquivo final. Chaves inválidas/grandes nunca são substituídas. A identidade persistente deve continuar Ed25519. Não é uma proteção contra processos do mesmo usuário/administradores, nem valida todos os componentes ancestrais de um caminho.
- **Proteção existente:** Ed25519 persistente; Windows DPAPI, Linux 0700/0600; Noise; Peer ID vinculado à assinatura; frames/texto/deadlines/replay/frequência limitados. Replay em memória de mensagens individuais não é uma garantia de entrega exatamente uma vez. A deduplicação persistente cobre o histórico retido.
- **Fronteira da interface:** dados de peers são renderizados como texto React; sem HTML arbitrário nem carregamento de ícones remotos. Métodos Wails assinantes usam a chave local, nunca papel/autor declarado como prova de autoridade. Não há servidor HTTP de administração no desktop; tests/webui é uma ponte QA local autenticada.
- **Dependência conhecida:** GO-2024-3218 na DHT sem correção conhecida; bloqueio de publicação preservado. Modo VPN não inicia DHT, mas o modo nativo mantém o risco documentado.
- **Distribuição:** aplicativo/setup sem certificado Authenticode próprio. SmartScreen/reputação e detecção real do Defender são situações distintas. Sem evidência de ameaça específica não classificamos o aviso como falso positivo. Nenhuma proteção será desativada.

Histórico e futuras réplicas de comunidades não têm criptografia de conteúdo em repouso ou E2EE de grupo nesta fase. Noise protege cada conexão de transporte; participantes autorizados e o nó do proprietário veem o conteúdo.
