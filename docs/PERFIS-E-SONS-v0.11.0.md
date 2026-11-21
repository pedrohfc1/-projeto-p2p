# PeeTwoPee 0.11.0 — perfis, imagens e sons

## Perfil e cache

Abra **Configurações → Perfil** para editar o nome, escolher a fonte e recortar o avatar. O editor aceita PNG, JPEG e WebP de até 5 MB. Ele recorta em 1:1, desenha a imagem num novo canvas de 256×256 e prefere WebP. A imagem reencodificada fica livre dos metadados do arquivo original. O nome aceita de 1 a 32 pontos de código Unicode e remove controles e substituições explícitas de direção de texto. Convites antigos e registros de membros continuam recebendo uma versão compatível com seu limite de 80 bytes.

O perfil usa a identidade Ed25519 existente. Cada edição assinada vincula Peer ID, nome, identificador da fonte e hash/MIME do avatar à revisão e ao horário. O stream libp2p `/projeto-p2p/profile/1.0.0` consulta revisões de peers autenticados já conectados. O receptor busca a imagem em outro stream do mesmo protocolo se o hash ainda estiver ausente. A resposta de imagem tem cabeçalho com tamanho e MIME seguido de bytes. Chat, convites, presença e codecs usam seus fluxos existentes.

O receptor valida a assinatura e exige revisão crescente. Antes de aplicar uma revisão com avatar novo, valida SHA-256, tamanho, formato, dimensões e pixels da imagem. A anterior continua disponível se a transferência falhar. A aplicação busca alterações em tempo de execução e, após reiniciar, exibe o perfil em cache mesmo com o autor offline. Um perfil novo fica disponível quando o peer estiver alcançável. Clientes antigos continuam usando os nomes e ícones compatíveis com sua versão.

Os metadados ficam em `profile-metadata/` no diretório de dados existente (por padrão `%APPDATA%/ProjetoP2P` no Windows). Avatares e ícones compartilham o cache `profile-assets/`, com nomes internos derivados do SHA-256. Cada imagem recebe no máximo 256 KiB. O cache tem teto de 128 MiB e 2.048 arquivos. A limpeza considera referências atuais e remove arquivos órfãos com pelo menos 14 dias. Identidade, histórico e preferências continuam no perfil original.

## Imagens de comunidades e salas

O proprietário pode salvar ou remover a imagem na configuração da comunidade ou na edição da sala de voz. A revisão de imagens fica no banco local da comunidade, em extensão separada da cadeia assinada principal. O protocolo de comunidade oferece o pedido opcional `artwork.get/1`. O manifesto é assinado pelo dono e vinculado à raiz e ao ID da comunidade. A autoridade verifica a participação atual antes de liberar metadados ou bytes. Os clientes aplicam a atualização inteira depois de verificar as imagens referenciadas. Receber imagens novas requer que o proprietário esteja alcançável.

## Fontes e sons

Inter, Poppins, Montserrat, Space Grotesk e JetBrains Mono estão empacotadas localmente sob OFL 1.1. Outros peers recebem somente o identificador da fonte. Um identificador desconhecido usa Inter e o navegador fornece o fallback de glifos fora do subconjunto.

Em **Configurações → Voz e vídeo → Sons**, é possível ligar ou desligar os efeitos, ajustar o volume e desativar as categorias de entrada/saída e transmissão. Os oito WAVs originais distinguem a própria conexão e saída, a entrada e saída de participantes, o início e fim de uma transmissão remota e o início e fim do próprio compartilhamento. A reprodução usa a saída local da chamada e respeita o volume configurado. A primeira observação da sala estabelece o estado inicial. Perda temporária do transporte ou autorização não é tratada como uma nova entrada.

## Validação

Os ensaios automatizados usam perfis temporários, imagens sintéticas e peers locais. Eles cobrem edição, persistência, sincronização no libp2p, cache offline, permissões de imagem, revisões, limite de entrada, fontes offline, volume dos efeitos e duplicação em reconexões. Os testes em Edge local não equivalem à validação física em duas redes.
