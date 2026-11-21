# PeeTwoPee 0.6.1 — beta privado

A identidade visual foi alterada; o perfil continua em `%APPDATA%/ProjetoP2P` (Linux: caminho legado). Não apague o perfil para atualizar. Identidade Ed25519, histórico, contatos, convites e assinaturas permanecem compatíveis. O instalador substitui o atalho antigo, mantendo a pasta de instalação e o registro de desinstalação existentes.

## Voz

Até seis participantes por sala, respeitando limites menores definidos pelo proprietário. Cada par usa seu próprio stream libp2p autenticado e criptografado, separado do chat, reaproveitando EasyTier. Opus mono/20 ms; cinco trilhas locais de reprodução são misturadas no Web Audio. Filas de três frames, descarte de áudio atrasado e prazos de escrita limitam acúmulo. TCP pode causar bloqueio por retransmissão e latência variável; não há garantia de latência entre casas.

Clique direito no participante para volume local de 0–200% ou silenciamento individual. Preferências são guardadas por Peer ID, junto aos dispositivos, volume do monitor e configuração da bandeja. Compressor e saturação suave limitam picos; ganho acima de 100% não recupera áudio já distorcido. Indicadores de fala usam nível local, sem registrar conteúdo.

Saída de outro participante não encerra a sala. Reconexão preserva a captura e a identidade. Trilhas reutilizadas mantêm sequência RTP para evitar rejeição antirreplay. A autorização do proprietário é renovada e expira em 45 segundos; autorização expirada ou mudança no registro pausa a mídia até nova autorização, sem expulsar os demais. Revogação do próprio membro encerra sua participação. O proprietário precisa estar disponível; não existe E2EE de grupo além da criptografia dos transportes entre pares.

## Bandeja e segundo plano

X oculta a janela quando a bandeja está disponível e a opção está ativa. Duplo clique restaura; menu permite abrir, alternar microfone de chamada já iniciada, sair da chamada e encerrar. Configurações → Áudio permite desativar o comportamento e encerrar explicitamente. Linux usa StatusNotifierItem/DBus quando o ambiente fornece watcher; caso contrário fechar encerra normalmente.

A interface oculta interrompe polling visual, animações e leitura de níveis, liberando listas temporárias. Mensagens persistem no banco. Captura, reprodução, autorização e conectividade continuam; o WebView não é suspenso. Flags locais do WebView2 evitam throttling de áudio em segundo plano. Não se força GC. A redução de RAM da janela oculta ainda exige medição em desktop real.

“Ouvir minha voz” continua exclusivamente local, sem gravação ou envio; seleção de saída e volume próprios, limpeza ao fechar configurações/trocar dispositivo. Use fones para evitar realimentação.

## Convites e permissões

Causa comprovada do aviso indevido: sincronização de identidade ainda não admitida retornava `forbidden`, persistido como negação do proprietário. Agora ausência de matrícula é ingresso pendente; convite consumido mantém o erro autenticado do ingresso; membro removido recebe revogação. Timeout, conectividade e proprietário indisponível são distintos. Convite antigo ou prefixo de registro não apagam revogação conhecida. Não se autoriza ingresso sem assinatura válida.

## Validação e limites

Testes automatizados cobrem preferências, volume/mute/limite, suspensão/restauração do polling, autorização, convites consumidos e multiuso, revogação, mídia entre processos independentes 2/3/4/6, saída/reentrada e chat concorrente. O teste de seis processos transmitiu por cinco minutos. Uma execução curta sob carga concorrente reprovou o critério de entrega de 80% em dois fluxos; o transporte descarta dados atrasados e se reconecta em vez de acumular latência. A capacidade sob carga elevada não está garantida. Houve também uma corrida identificada entre a validação de uma autorização antiga e sua renovação: o guard agora só invalida a mesma geração/seqüência/validade que verificou. Áudio sintético Opus em processos não comprova microfones, caixas, seis casas, CGNAT ou janela oculta. Veja PEETWOPEE-v0.6.1-METRICAS.json para medições disponíveis e seu escopo.

Pendentes: reprodução audível e volume em dispositivos físicos; captura/reprodução em seis PCs; bandeja/restauração/encerramento no desktop e instalação/atualização completas (controle da interface interrompido por Escape); comparação RAM/CPU visível/oculto/voz incluindo WebView2 e helper. Compilação Linux CLI e bandeja não comprova desktop Linux.

## Teste manual

1. Instale o beta em perfis existentes sem apagar dados; confirme Peer ID e histórico anteriores.
2. Dois a seis PCs entram por convites multiuso em comunidade com dono online. Use EasyTier integrado ou Radmin já configurado. Não é necessário instalar ambas VPNs.
3. Selecione microfone/saída, teste retorno local com fones, interrompa o monitor e entre na voz explicitamente.
4. Fale simultaneamente; ajuste volume de uma pessoa e confirme que outros ouvintes não mudam. Saia/reentre e interrompa/reabra a conexão sem perder o chat.
5. Feche X por cinco minutos em chamada; confirme áudio bidirecional, ícone mute, duplo clique e ausência de processos após Encerrar. Compare CPU/RAM da árvore completa de processos nas três situações, sem misturar o teste sintético com áudio real.

## Distribuição

Somente beta privado, sem publicação automática. DHT legado vulnerável continua em quarentena. Alertas Rust não resolvidos continuam bloqueando release pública. O auxiliar permanece userspace/no-TUN, sem Packet.dll/Npcap ou drivers; não há alterações de firewall, rotas ou DNS. Instalador inclui licenças e fontes correspondentes LGPL do auxiliar. Binários não assinados por certificado comercial.

## Reproduzir testes automatizados (PowerShell)

```powershell
npm --prefix frontend ci
npm --prefix frontend test
npm --prefix frontend run build
go test ./...
go vet ./...
go build -o dist/webui-qa.exe ./tests/webui
$env:P2P_WEBUI_TEST_BIN=(Resolve-Path dist/webui-qa.exe).Path
$env:P2P_ETCAT_TEST_DIR=(Resolve-Path build/bin/easytier).Path
$env:P2P_GROUP_SOAK_SECONDS='300'
go test ./internal/desktop -run TestRealGroupVoiceIndependentProcesses -count=1 -timeout 20m -v
```

O auxiliar precisa corresponder ao hash de `internal/overlay/manifest.json`, com sua proveniência/licenças. O teste usa perfis descartáveis, não o perfil real. Execute o ensaio de capacidade isoladamente; compilações ou carga intensa podem acionar descarte/reconexão pelos limites de latência. Teste sintético não substitui validação física.
