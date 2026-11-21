# PeeTwoPee 0.9.2 — atualização segura, beta privado

## Causas e escopo
O instalador anterior escrevia primeiro o auxiliar e só depois tentava substituir o aplicativo: EXE bloqueado podia deixar arquivos de versões diferentes. Não havia transação, inventário de integridade ou rollback. Fechar a janela podia manter a versão antiga na bandeja; faltava exclusão de instâncias antes do WebView. Mensagens de instalação ainda diziam0.5.2 mesmo nos pacotes recentes. Essas falhas foram verificadas no código; não são prova da causa física específica da falha de áudio do usuário.

A comparação com o pacote-fonte0.9.1 confirma preservação dos componentes de microfone, Opus/libp2p, mixer, sessões, vídeo VP8 e supervisor EasyTier. Não há novo codec, recurso de voz/vídeo nem otimização1080p60 nesta etapa. O objetivo é instalar/iniciar componentes coerentes antes de investigar qualquer falha física remanescente. O último commit local continua c4e90b2; versões recentes estão no diretório de trabalho sem commits intermediários para bisect.

## Atualização
Instalação mantém %LOCALAPPDATA%/Programs/ProjetoP2P. Perfil mantém %APPDATA%/ProjetoP2P, ou o caminho explicitamente configurado anteriormente. Não há migração de identidade, contatos, mensagens, comunidades ou preferências. Instalação e perfil não podem sobrepor seus diretórios.

- Instaladores concorrentes e novos inícios são bloqueados durante a transação. Instâncias normais usam bloqueio por perfil antes de criar WebView, nó ou auxiliar; perfis de teste distintos continuam independentes.
- Se processos da instalação estiverem ativos, há confirmação antes de encerrar. A nova versão recebe IPC local de encerramento, sai da bandeja e executa Core.Close/supervisor/Job Object. Versões antigas sem IPC oferecem cancelamento para encerrar manualmente pela bandeja ou uma SEGUNDA confirmação para forçar somente processos identificados desta instalação.
- Força nunca usa apenas o nome: valida caminho exato, usuário Windows, PID, criação do processo e handle aberto. Não encerra Radmin, Hamachi, EasyTier externo ou outro PeeTwoPee em outro caminho. Processos/locks desconhecidos causam erro seguro. Cópias portáteis antigas em outras pastas devem ser encerradas manualmente; perfil ainda ocupado bloqueia a atualização, sem apagar dados.
- Os seis arquivos distribuídos são copiados/staged e verificados por SHA256. A cópia anterior e metadados são guardados antes da primeira substituição. O inventário install-state.json só representa uma instalação completa. Frontend e backend permanecem embutidos no mesmo EXE.
- Falha de arquivo bloqueado, disco ou finalização reverte arquivos e metadados. Um diário com checksum permite recuperação no próximo instalador após interrupção. Enquanto houver diário pendente, a nova aplicação não inicia. Backups corrompidos são recusados antes de restaurar.
- Registro e atalhos participam da finalização, com snapshot para reversão. Os tipos dos valores do Registro são preservados. A versão exibida agora usa a versão compilada real. Atalhos legados só são removidos quando apontam para executáveis conhecidos desta instalação.
- O inventário remove somente ProjetoP2P.exe legado, nomes distribuídos e temporários identificados. Não há remoção por *.dll ou exclusão recursiva do perfil. Arquivos desconhecidos permanecem. Conserva até dois backups completos; a limpeza de backups antigos só remove nomes inventariados e diretórios vazios.

A aplicação verifica os executáveis instalados e o próprio EXE contra o inventário antes de abrir a interface. A verificação completa de todos os arquivos ocorre ao concluir a instalação. Remover documentação não é motivo para interromper uma chamada. Uma cópia de EXE incoerente ou antiga dentro da instalação é recusada com instrução de usar o atalho atualizado/reinstalar.

## Cache WebView2
O caminho de dados do WebView permanece no perfil anterior. A revisão de cache deriva do frontend EMBUTIDO e do hash do EXE, incluindo mudanças de backend. Só HTTP cache, Code Cache e GPU/shader caches conhecidos são movidos para backup, incluindo o layout EBWebView/Default. Preferences, Local State, cookies, permissões, Local Storage e IndexedDB permanecem. Não se apaga EBWebView inteiro nem se concedem permissões automaticamente. Arquivo de cache bloqueado impede a inicialização e informa o motivo. Respostas de assets usam no-store. Backups de cache são conservados, não substituem backup do perfil e podem ocupar disco.

Referência: [dados e permissões do WebView2](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/user-data-folder). A documentação distingue dados persistentes de cache; o produto não limpa o perfil inteiro.

## Rollback
Execute no PowerShell o instalador beta desta versão com --rollback:

    & '.\PeeTwoPee-Setup-v0.9.2-beta-privado.exe' --rollback

Confirme a restauração e, quando solicitado, o encerramento dos processos. Restaura arquivos e registros/atalhos anteriores; não reverte nem apaga o perfil. Um primeiro install sem aplicativo anterior não oferece rollback manual; use o desinstalador. Se uma atualização foi interrompida, executar novamente o instalador recupera o diário antes de iniciar nova transação. Não remova .p2p-backups/.p2p-update.json durante uma recuperação. Não se declara validação de queda de energia; interrupção de processo foi simulada.

## Testes executados
44 testes frontend, TypeScript/Vite, Go/vet e testes novos de transação/recuperação/cache. Foi testada também a substituição do EXE real0.9.1 pelo EXE real0.9.2 com o payload atual real em pasta isolada: hashes, versão e política de consentimento conferidos, identidade preservada; metadados injetados, sem abrir a GUI antiga. Windows real com perfis temporários: instalação de arquivos limpa e por cima, processo oculto ativo encerrado via IPC, recusa de consentimento, força legada com segunda confirmação, processo homônimo de outra pasta preservado, rejeição de duplicata, arquivo bloqueado, rollback e reinício do processo. Identidade Ed25519 e banco/histórico isolados foram preservados. O processo é uma fixture nativa que implementa o mesmo contrato, não a UI de bandeja real. Metadados foram injetados nesse teste; teste separado de .lnk real passou. HKCU foi bloqueado pelo sandbox, portanto registro/instalação gráfica completa não foram validados aqui.

Regressões locais com auxiliar EasyTier real: supervisor recupera falha e encerra filhos; duas instâncias de produção trocam Opus, reconectam ponte local e mantêm chat; VP8 pré-codificado é transmitido/decodificado com Opus/chat simultâneos, dois transmissores, reconexão, revogação e limpeza. Comunidades e atividades assinadas passam em três processos. Isso não é teste de microfone/monitor físico nem de duas casas.

Pendentes: instalador completo fora do sandbox com Registro real; atualização sobre o EXE antigo efetivamente aberto na bandeja; permissões/dispositivos de áudio, janela/monitor e reprodução físicos; dois PCs/redes distintas; reinicialização real do Windows; Linux desktop. Reinício do PROCESSO foi testado e não é reinício do SISTEMA. Não foi adicionada inicialização automática no login. Não se garante ainda que o defeito físico relatado de áudio/vídeo esteja resolvido.

## Procedimento manual obrigatório
1. Salve sua conversa e mantenha cópia do perfil antes do teste. Não desinstale nem apague AppData. Registre Peer ID e uma mensagem antiga.
2. Execute o instalador por cima. Confirme o encerramento da versão ativa; para versões antigas, prefira Encerrar aplicativo na bandeja quando solicitado. Se cancelar, a instalação anterior deve continuar intacta.
3. Abra pelo MENU INICIAR, não por um EXE antigo da pasta descompactada. Confira install-state.json em Programs/ProjetoP2P e a versão do EXE com --version.
4. Confirme mesmo Peer ID, contatos, comunidade, histórico e preferências. Use um perfil separado para teste limpo; nunca apague o perfil real.
5. Teste primeiro voz bidirecional, depois tela em720p30 e voz simultânea. Teste sair/reentrar, chat, reconexão, minimizar/restaurar e encerramento pela bandeja.
6. Reinicie o Windows manualmente, abra novamente pelo atalho e reconecte os dois PCs. Esta etapa não foi realizada automaticamente.
7. Se necessário execute --rollback; confirme que dados permanecem e apenas a instalação retorna. Guarde o relatório de erro e a versão exata, sem publicar perfis ou credenciais.

Distribuição beta privada, sem assinatura de editor e sem release pública. Auxiliar auditado, licenças e fontes correspondentes LGPL completas incluídos. Sem alterações de firewall, rotas, DNS ou permissões globais. Bloqueios Rust e quarentena DHT legado mantidos.
