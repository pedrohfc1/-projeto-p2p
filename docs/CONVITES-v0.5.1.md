# Convites 0.5.1

## Formatos e autorização

Convite de contato apresenta identidade, nome, endereços, relays opcionais e validade; não concede papel em comunidade. Convite de comunidade é ticket assinado pelo dono para solicitar ingresso, de uso único e até 24 horas. Localizar um ticket não é prova de ingresso: o dono verifica assinatura, raiz, época, validade, uso e autorização antes do commit. O dono continua necessário online. Noise não equivale a E2EE de grupo.

Novos prefixos: `projetop2p:c2:` e `projetop2p-community:c2:`. Base64URL sem padding contém envelope **Protobuf** definido em internal/compactinvite/invite.proto. Avaliamos CBOR (mapa/array eficiente, mas exigiria codec adicional) e Protobuf (biblioteca já fixada no projeto, schema interoperável). Não usamos compressão genérica nem descompressão de entrada remota.

Chave Ed25519 bruta substitui serialização textual; Peer ID é derivado da chave. Multiaddr usa representação binária e omite somente o sufixo final /p2p/ID do próprio autor, reconstruído na leitura. IDs/raiz/nonce de comunidade usam bytes. Domínio, versão e flags de listas reconstruem **exatamente o payload JSON v1 assinado**; a assinatura original permanece e é verificada pelo validador existente. Assim não inventamos uma nova primitiva criptográfica nem descartamos escopo, expiração ou prova de autoridade. O wire prefix c2 versiona o envelope, não uma nova autorização.

Limites: 24 mil caracteres contato e 16 mil comunidade antes da decodificação, no máximo 48 campos, 4096 bytes por campo, tipos/duplicatas/campos desconhecidos recusados. Os validadores existentes ainda limitam endereços, escopo, identidade e tempo. Assinatura Ed25519 de 64 bytes e chave correta obrigatórias. Não há chamada a servidor para validar o convite.

## Compatibilidade

O leitor aceita JSON/Base64URL 0.5.0 e o envelope c2. A 0.5.0 não entende c2: no modal use **Formato antigo (0.5.0)** antes de copiar para essa pessoa. O backend valida o conteúdo e devolve o mesmo convite assinado no envelope antigo; não muda identidade, prazo ou permissões. Operações de ingresso no registro replicado usam o ticket legado reconstruído, permitindo que reducers 0.5.0 continuem a verificar o histórico.

Teste real com executável helper 0.5.0 preservado e helper novo: convite antigo aceito, convite novo convertido para antigo, DM bidirecional, dono antigo/membro novo e dono novo/membro antigo. Executar `python tests/invite_backward_smoke.py CAMINHO_HELPER_ANTIGO CAMINHO_HELPER_NOVO`. Esses helpers não são builds para usuários e não abrem serviço público.

## Copiar, colar e QR

Os modais de contato e comunidade mantêm copiar/colar e agora oferecem QR Code, gerado **localmente**, com node-qrcode 1.5.4 (MIT), sem serviço de imagens, upload ou câmera. O QR contém o texto completo; não é um atalho de descoberta. Correção M, margem de quatro módulos, preto sobre branco. Se ultrapassar capacidade, mostra erro e mantém cópia textual, sem remover campos. Escanear com leitor externo e colar no app; não há scanner de câmera embutido. Evite compartilhar ticket/QR privado em tela pública.

Referência da biblioteca: https://github.com/soldair/node-qrcode . Teste verifica preservação dos bytes do segmento QR e recusa de entrada acima da capacidade; leitura óptica real e layout nativo ainda exigem aceitação em PCs.

## Medição reproduzível

`go test ./internal/invite ./internal/community -run Compact -v -count=1` compara o mesmo payload assinado antes/depois, um endereço TCP, nome Pessoa/Comunidade e chaves geradas só para o teste. Os valores incluem prefixo e Base64URL; caracteres ASCII equivalem a bytes.

| Caso | Legado | Compacto | Redução |
|---|---:|---:|---:|
| Contato Radmin | 553 | 193 | 65,10% |
| Contato Hamachi | 554 | 194 | 64,98% |
| Contato nativo | 531 | 185 | 65,16% |
| Comunidade (interface/loopback) | 869 | 327 | 62,37% |

São exemplos medidos, não promessa de tamanho fixo. Mais endereços e relays aumentam o convite. Endereço relay conserva o Peer ID do relay; somente o sufixo do autor é omitido. Listas nil/vazias são preservadas porque participam da assinatura antiga.

## Códigos curtos: somente estrutura local

`internal/invitediscovery` implementa registro assinado, código P2P-C1 com 160 bits derivados do hash do registro e nonce aleatório de 128 bits, expiração configurável limitada pelo ticket, mirrors locais autorizados e revogação assinada. O código é maior que seis caracteres por segurança; exemplos curtos do pedido não oferecem entropia adequada para comunidades privadas. Hash vincula código ao registro e impede substituição prática; não há garantia matemática de ausência de colisões.

Mirrors aceitam somente a partir do estado local reconstruído do registro assinado, para membro ativo, raiz/dono/época iguais e comunidade não arquivada. Capacidade 64 registros por mirror. Testes cobrem replicação entre dois stores locais, tentativa de publicação por estranho, alteração, expiração, revogação e replay. O ticket continua de uso único no dono: resolver código não incrementa uso nem promove membro. Revogação precisa chegar a cada mirror; réplica offline pode estar desatualizada, e o dono ainda revalida o ticket.

**Não há resolução distribuída global, UI de código curto nem publicação na DHT.** A DHT atual usa DisableValues; não fornece namespace/validador/replicação autorizada necessários. Os mirrors são voláteis e locais, não serviço de produção. Antes de ativar: implementar transporte/resolução real, persistência e tombstones, validação de estado recente, limites e antienumeração, cotas/expurgo, privacidade dos tickets e testes entre redes. Não usar blockchain.

## Validação e instalação

Frontend: 9 testes, TypeScript/Vite. Go: suíte completa, permissões e outbox, convites antigos/novos, adulteração, expiração, envelope truncado/duplicado e seed de fuzz. Radmin real local preservou DM e comunidades com três processos; não comprova duas casas. Vet, consistência de versão e payload do instalador verificados. Código separado em branch feature/v0.5.1-compact-invites.

Feche o app antigo antes de instalar ProjetoP2P-Setup-v0.5.1.exe; faça backup privado com app fechado. Não enviar perfil/chave/banco ao amigo. Sem migração nova do banco; preserva esquema 2, identidade e configurações. Instalador não assinado; instalação/atualização completa e inspeção visual nativa ainda precisam de teste. Risco DHT GO-2024-3218 permanece, sem suprimir gate. Nenhuma release publicada e nenhuma alteração de firewall/VPN/Defender.
