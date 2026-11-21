# PeeTwoPee 0.10.4 — etapa 3 (beta privado)

## Interface

Salas de voz possuem grade responsiva por identidade, perfil ao clicar, menu contextual e estados reais de fala, mute, áudio e rede. O canal lateral continua usando os participantes da sessão existente. A grade só existe na navegação: não cria AudioContext, captura, decoder ou transporte. Entrar usa a função de entrada existente; clicar na sala já conectada abre sua visualização. A desconexão fica na barra persistente da chamada.

A barra preserva mute, ensurdecer, compartilhamento e desconexão; câmera permanece desabilitada. A única engrenagem principal continua ao lado do perfil. O atalho textual Voz e vídeo abre a mesma central. Métricas da barra ficam em Detalhes técnicos. O microfone habilitado não é apresentado como prova de áudio transmitido.

A transmissão preserva o elemento de vídeo, a seleção nativa, encoder, bitrate, FPS e transporte existentes. A apresentação usa os tokens globais; consentimento fica recolhível, os diagnósticos e WASAPI continuam acessíveis na central. O player mantém tela cheia, modo compacto, volume existente, controles sobrepostos, indicação de reconexão e aviso de término. A navegação entre canais não remonta o player.

## Instalador

Assistente WinForms em português, logo próprio, pasta editável/Procurar, versão instalada, resumo de confirmação, atalhos, inicialização opcional com Windows, progresso e Abrir PeeTwoPee. A pasta padrão continua por usuário, sem exigir administrador. Pastas sem permissão falham com erro do Windows; o assistente não eleva privilégios nem muda firewall, DNS ou rotas.

A instalação continua usando a transação existente: inventário, hashes, confirmação antes de encerrar instâncias, guardas de processo/perfil, recuperação de falha e rollback. Não há migração de dados nesta versão. Identidade e banco ficam fora da pasta de executáveis. A localização registrada passa a ser reutilizada nas atualizações e no desinstalador; limpeza do desinstalador é vinculada ao processo da pasta correta. Atalhos e inicialização opcional fazem parte do snapshot/rollback. Mover uma instalação existente para outra pasta pode ser recusado quando seus atalhos ainda apontam à pasta anterior; mantenha a pasta registrada ou desinstale preservando o perfil antes de mudar.

Radmin é uma opção desmarcada, identificada claramente como abertura do download oficial após concluir. Instalação existente é detectada pelo serviço RvControlSvc. Não se embute Radmin, não se executa instalação silenciosa, não se aceita licença pelo usuário e não se remove o produto na desinstalação. Sem integração automatizada de pacote verificada nesta etapa, a opção apenas abre https://www.radmin-vpn.com/; os termos permanecem em https://www.radmin-vpn.com/eula/. O usuário baixa e aceita os termos no instalador oficial. A EULA consultada em 16/11/2026 distingue produto 1.x de novas versões principais; não foi assumida autorização de redistribuição de Radmin 2.x. EasyTier continua padrão.

## Validação

- 112 testes de frontend, TypeScript/Vite e build Wails Windows.
- Browser Edge com fixtures isoladas em 1280×720, 1600×900 e 1920×1080: cartões, menu/volume, perfil, mute/deaf, câmera desabilitada, AO VIVO sob ação explícita, player persistente, tela cheia, término e saída. Nenhuma captura física foi iniciada nesses testes.
- Regressões de mensagens diretas e comunidades em browser passaram.
- go test ./... e go vet ./... passaram. Testes adicionais de opções do assistente, BOM, JSON inválido, isolamento do perfil, atalhos reais em pastas temporárias e rollback passaram.
- Atualização com bytes reais do desktop anterior e atual, auxiliar, fontes/licenças e desinstalador em pasta temporária: hashes/versão/política de permissão verificados e identidade isolada preservada. Esse teste injeta metadados de instalação; não equivale a instalação completa no Windows de uso diário.
- Renderização WinForms em fixture local para revisar o layout do assistente; não é teste completo de interação humana ou instalação.
- Payload do instalador verificado contra desktop/auxiliar/fontes/licenças; sem Radmin/Npcap embutido. Consulte VALIDACAO.json e SHA256SUMS.txt no pacote privado.

Pendentes: teste humano completo do assistente (incluindo cancelamentos, pasta protegida, opções Radmin e desinstalação), Windows 10 e Windows 11 limpos, bandeja/minimização no aplicativo real e chamada/transmissão física entre computadores. Fixtures não comprovam áudio audível, FPS, conectividade entre casas nem instalação de Radmin. O beta não possui assinatura Authenticode de editor; não foi publicado.

Rollback de instalação disponível por PeeTwoPee-Setup-v0.10.4-beta-privado.exe --rollback. Ponto de rollback do desenvolvimento: work/stage3-rollback.zip, fora do repositório. O rollback de código não substitui o rollback transacional da instalação.
