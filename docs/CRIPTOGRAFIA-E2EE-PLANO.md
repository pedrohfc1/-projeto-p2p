# Plano de E2EE — posterior à 0.5

Status: proposta, sem implementação ou declaração de E2EE de comunidades. Noise e Ed25519 já presentes não constituem criptografia ponta a ponta de grupo.

Avaliar **Messaging Layer Security, RFC 9420** para grupos dinâmicos, inclusive grupos de dois dispositivos para DM. MLS define epochs, Welcome, commits e propriedades de segredo futuro/pós-comprometimento condicionadas ao protocolo e ao ciclo correto das chaves. [Padrão](https://www.rfc-editor.org/rfc/rfc9420.html).

Candidatos de biblioteca: [OpenMLS](https://github.com/openmls/openmls) e [mls-rs](https://github.com/awslabs/mls-rs). Não são dependências atuais. Antes de escolher: fixar release, verificar licença, manutenção, relatórios públicos de auditoria e seu escopo exato, vetores interoperáveis, FFI Go/Rust, zeroização e suporte Windows/Linux. Não presumir que todo release de uma biblioteca foi auditado. Não implementar primitivas próprias.

## Separação das chaves

Manter a identidade Ed25519 existente para autenticação e vinculação verificável de dispositivos. Usar chaves de conteúdo/sessão próprias da biblioteca MLS; não reutilizar a chave de identidade como chave simétrica. Vincular credencial de dispositivo ao Peer ID, mostrar mudanças verificáveis e exigir confirmação de novos dispositivos. DPAPI/permissões continuam necessários, mas não substituem proteção e recuperação das novas chaves.

Entrada gera Welcome autorizado para o dispositivo, com política explícita de acesso ao passado; membro novo não deve receber chaves históricas por padrão. Saída/remoção exige commit de mudança de época e impede decriptar mensagens futuras depois da adoção da nova época. Não promete apagar texto/chaves que o usuário já copiou. Rotação periódica e após suspeita, com atualização efetiva de todos os membros, deve ser testada.

## Persistência e sincronização

Persistir ciphertext, epoch, IDs de grupo/dispositivo e estado MLS transacionalmente com cursor/outbox. Proteger contra rollback/replay e commits concorrentes sem inventar resolução criptográfica. Revalidar autorização, vincular metadados autenticados ao canal/comunidade e evitar vazar conteúdo em logs/erros. Definir retenção de chaves antigas e tratamento de mensagens atrasadas sem retenção ilimitada.

O sequenciador/armazenamento futuro pode encaminhar ciphertext; se o dono também for membro autorizado ele continuará podendo ler legitimamente. MLS não torna um membro malicioso incapaz de copiar texto, nem resolve descoberta/NAT/disponibilidade. O registro assinado atual poderá ordenar envelopes, mas sua integridade não substitui o estado MLS. Alterações do protocolo exigirão versão e migração explícitas, nunca reinterpretar texto claro antigo como ciphertext.

Recuperação deve explicar perda de acesso sem backup e risco de backup de chaves; não prometer recuperar conteúdo sem segredo correspondente. Migração do histórico existente será optativa, privada e verificável, preservando os registros originais até confirmação. Sincronização multi-dispositivo, reentrada e restauração de backup devem detectar bifurcações de estado.

## Critérios de liberação

Vetores de interoperabilidade, teste de membro removido incapaz de ler mensagens novas, novo membro sem histórico indevido, crash entre persistência e envio, rotação, reorder, replay, rollback, membro malicioso e restauração. Revisão de especialista independente antes de anunciar E2EE; UI mostra claramente quais conversas ainda usam o formato legado. Não incluir certificados ou segredos de teste reais no Git.
