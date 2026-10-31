# Arquitetura e próximas etapas

CLI → network (host/streams/ciclo de conexão) → discovery (rendezvous DHT).
identity fornece a chave; messaging assina e valida; security limita frequência e replay.
registry define contratos futuros de leitura de registros assinados; storage define a fronteira de ciphertext. `pkg/` será criado apenas quando houver API pública estável.

Próximas etapas propostas: persistir observações de participantes com expiração; tornar políticas configuráveis; adicionar filas duráveis e criptografia ponta a ponta; implementar armazenamento criptografado e verificação de hashes; depois replicação e reparação, com consentimento e quotas. A DHT localizará provedores de conteúdo, sem guardar arquivos. Chaves de conteúdo precisam de distribuição separada e autorizada. Reter ciphertext implica responsabilidades de disponibilidade, exclusão e recuperação que ainda não estão implementadas.

Comunidades e permissões exigirão autorização explícita, rotação/revogação e proteção contra Sybil. Não converter descoberta ou conexão em confiança. WebRTC/Pion e encaminhamento cooperativo de voz exigirão estudo de topologia, banda e limites por voluntário antes da implementação.
