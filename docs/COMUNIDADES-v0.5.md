# Comunidades 0.5 — guia de teste

Build experimental, sem release pública nesta entrega. Feche a versão anterior, faça backup privado do perfil com o app fechado e atualize usando `ProjetoP2P-Setup-v0.5.0.exe`. O instalador preserva `%APPDATA%/ProjetoP2P`. Não envie o perfil a outra pessoa: envie somente o instalador. Cada participante mantém sua própria identidade.

## Dois computadores Windows com Radmin

1. Instale o mesmo build nos dois PCs. Entre na mesma rede pelo cliente Radmin, sem alterar a configuração da VPN pelo Projeto P2P.
2. Em Configurações, ambos escolhem **Radmin VPN**, selecionam o adaptador ativo e salvam. Confira IPv4 virtual, porta TCP real e escuta. Não preencha Bootstrap/Relay para esse caminho. Hamachi instalado simultaneamente não deve substituir a escolha Radmin. Adaptador detectado não comprova conexão.
3. O dono usa o botão **+** na barra de comunidades para criar uma comunidade. Escolha nome, descrição e ícone textual/emoji. São criados `geral` (texto) e `voz` (somente definição).
4. Em configurações da comunidade, gere o convite e envie por um canal confiável. O outro PC usa **+ / Entrar por convite**. Confirme a rede indicada pelo convite. O convite de comunidade começa com `projetop2p-community:`; não é o convite individual.
5. Confira o dono e o Peer ID por outro canal. O ticket é assinado pelo dono, fixa a comunidade e o endereço do dono, vale 24 horas e permite uma entrada. Gere outro para cada participante. Não publique tickets em locais abertos.
6. Envie texto nos dois sentidos, crie outro canal e reinicie os dois aplicativos. Confira participantes, histórico, identidade e os contatos individuais antigos. A confirmação exibida significa inclusão no registro pelo dono; não é recibo de leitura nem prova de entrega a todos.
7. Com um terceiro PC, promova um membro a moderador. Ele pode criar/remover canais, solicitar convite ao dono e remover membros comuns. Somente o dono altera metadados, concede papéis e arquiva a comunidade. Tente operações indevidas: o backend deve recusá-las mesmo se a interface for contornada.
8. Feche o dono, envie texto no membro e confira a fila pendente. Reabra o dono: a fila deve sincronizar sem duplicar IDs. Reinicie também o membro antes de reabrir o dono para comprovar persistência. Nunca apague o banco para recuperar a conexão.
9. Remova um membro. A sincronização deve negar acesso e novos envios. Papéis, saídas e remoções invalidam tickets anteriores. Para readmitir, gere um convite novo. Cópias que o ex-membro já recebeu não podem ser apagadas remotamente.
10. Repita mensagens individuais e **Testar conectividade**. Se falhar, registre modo, interface, porta e motivo exibidos; não altere firewall automaticamente. Se necessário, peça autorização ao responsável pelo PC para analisar regras.

## Funcionamento e limites

A rede é entre usuários, sem servidor global. **Nesta versão, o dono é o sequenciador da sua comunidade e precisa estar online para confirmar entrada, mensagens e mudanças.** Moderadores não substituem o dono quando ele está offline. Os membros guardam réplicas locais verificadas e filas persistentes; não há eleição, consenso, transferência de dono nem recuperação automática da perda da chave do dono.

O registro contém assinaturas do autor e do dono, sequência, hash anterior e ID estável. A sincronização aceita somente uma extensão verificável do histórico local; divergências são recusadas, sem sobrescrever o histórico. Isso detecta alterações e conflitos, mas não impede um dono malicioso de oferecer histórias diferentes a membros isolados. Não é blockchain.

Controle concorrente exige o cabeçalho atual; atualize e repita se houver conflito. Texto pode aguardar offline, mas será revalidado contra as permissões vigentes quando o dono recebê-lo. Não há promessa de entrega exatamente uma vez em toda a rede: há deduplicação persistente por operação e confirmação verificável local.

O histórico é paginado em blocos de até 100 mensagens. Remover canal e arquivar comunidade são operações lógicas: preservam o histórico local. Sair não apaga as cópias dos outros usuários. Não existe upload de ícones, arquivos, áudio, vídeo ou captura de tela. Salas de voz apenas reservam estrutura para 0.6.

Limites atuais: 32 comunidades por perfil; 256 membros e canais por comunidade, incluindo removidos; 50 mil eventos e 16 MiB de registro por comunidade; 64 MiB de registros por perfil; 64 operações pendentes por comunidade e 2 MiB de fila no perfil. São limites lógicos, não limites exatos de arquivo bbolt/RAM. Atingi-los requer intervenção planejada, sem descartar registros silenciosamente.

## Migração e retorno

O banco bbolt passa do esquema legado 1 para 2, criando buckets separados sem reescrever mensagens individuais. Há testes de preservação, repetição idempotente, versão futura recusada e reversão 2→1 somente com buckets de comunidades vazios. O app não faz downgrade automático. Com comunidades já usadas, a reversão é recusada para evitar perda; restaure um backup privado anterior com o app fechado se realmente precisar retornar. A chave Windows DPAPI não é portável para outro usuário/máquina.

## Teste automatizado local

Compile o frontend antes do backend. Use três perfis temporários, nunca o perfil pessoal:

```powershell
go build -buildvcs=false -o work-webui.exe ./tests/webui
python tests/community_desktop_smoke.py ./work-webui.exe --metrics metrics-local.json
python tests/community_desktop_smoke.py ./work-webui.exe --radmin --metrics metrics-radmin.json
```

O segundo comando opcional exige adaptador Radmin real. Usa três processos no mesmo PC e não substitui o roteiro entre casas. O helper é exclusivo de testes, com token/origem local; não o distribua como aplicativo ou serviço público.
