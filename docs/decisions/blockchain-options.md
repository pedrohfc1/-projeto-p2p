# Registro verificável futuro — avaliação preliminar

Nenhuma blockchain foi selecionada ou implementada. Valores financeiros e benchmarks não foram estimados sem medições. Comparação de requisitos, a rever na fase de integração:

Referência para o modelo clássico BFT e seus limites de falhas: [Castro e Liskov, Practical Byzantine Fault Tolerance](https://www.usenix.org/legacy/publications/library/proceedings/osdi99/full_papers/castro/castro_html/castro.html). A comparação abaixo é uma análise de engenharia preliminar, não um benchmark de produtos.

| Alternativa | Custo operacional | Escala e disponibilidade | Consenso e confiança |
|---|---|---|---|
| Log assinado por comunidade, com testemunhas | Armazenamento e distribuição, sem taxas de rede intrínsecas | Escala por comunidade; testemunhas precisam reter histórico | Detecta adulteração; precisa estratégia para detectar e resolver bifurcações |
| Ledger permissionado com consenso BFT | Validadores independentes permanentemente disponíveis e governança | Throughput depende do conjunto e do protocolo; partições podem suspender finalização | Protocolos clássicos costumam exigir 3f+1 validadores para tolerar f bizantinos; adesão não é livre |
| Blockchain pública existente | Taxas variáveis, RPC/indexação ou execução de nós próprios | Disponibilidade herdada; throughput, latência e retenção limitados pela rede escolhida | Segurança e finalização dependem do protocolo; metadados públicos e dependência externa |
| Nova blockchain aberta | Desenvolvimento, auditoria e manutenção substanciais | Rede pequena e churn dificultam segurança e disponibilidade | Precisa resistência a Sybil sem presumir que identidades gratuitas são independentes |

Antes de escolher: definir quem pode escrever, modelo de adversário, resolução de propriedade concorrente, necessidade de ordem global, tolerância a partições, finalização, revogação, privacidade e manutenção de validadores. Medir operações/s, tamanho de registro, latência p95, recuperação de partições, custo por comunidade e custo por nó em protótipos equivalentes. Exigir revisão técnica e decisão própria.

Somente compromissos criptográficos e registros mínimos poderão entrar nessa camada. Mensagens, arquivos, voz, chaves e dados sensíveis permanecem fora. Hashes também podem revelar correlações e permitir ataques de dicionário; não são anonimização. Os contratos atuais serão refinados após definir semântica, provas e consenso.
