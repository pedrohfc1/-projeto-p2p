# Instalação e empacotamento 0.2

## Windows 10/11 AMD64

O executável desktop está em `build/bin/ProjetoP2P.exe`. Pode ser aberto com duplo clique sem terminal. WebView2 Evergreen é necessário; o build usa bootstrapper Wails embutido. A instalação do runtime pode precisar de internet. Não execute como administrador sem necessidade. O aplicativo e o instalador não estão assinados com certificado de distribuição; não contorne avisos do Windows automaticamente. Uma distribuição pública deve adicionar assinatura e processo de atualização verificável.

O **setup real** deve ser `ProjetoP2P-Setup-v0.2.exe`. O pacote atual não o contém: NSIS 3.11 baixado do projeto oficial falhou ao iniciar neste sandbox com `Illegal System DLL Relocation`. A abertura do aplicativo nativo também não foi confirmada: WebView2 criou o ambiente mas falhou ao criar o controlador (`0x8000ffff`). Isso é uma limitação de validação neste ambiente, não evidência de sucesso em Windows comum.

### Gerar o instalador pelo GitHub Actions

1. Crie um repositório seu no GitHub e envie **o conteúdo de `projeto-p2p` como raiz**. Não inclua `data`, identidades, bancos, `node_modules` ou caches. Não publiquei nada em seu nome.
2. Abra **Actions → Desktop v0.2 → Run workflow**. O evento `workflow_dispatch` permite executar sem criar uma release.
3. Confira os jobs Windows e Linux. Se algum falhar, consulte o log; não considere o instalador gerado sem o job terminar com sucesso.
4. Em **Artifacts**, baixe `ProjetoP2P-Windows-v0.2`. Ele deve conter `ProjetoP2P.exe` e **`ProjetoP2P-Setup-v0.2.exe`**, além dos CLIs. O workflow falha se o setup estiver ausente ou muito pequeno.
5. Teste o setup em Windows de teste: instalação, abertura, encerramento, atualização por cima e desinstalação. Confira ID, contatos e histórico antes e depois. Esses testes ainda não foram executados aqui.

O runner instala Wails v2.17.0 e NSIS 3.11, compila e testa Go/React, invoca `wails build -s -trimpath -nsis -installscope user -webview2 embed` e publica o artefato. O projeto NSIS está em `build/windows/installer/project.nsi`; `wails_tools.nsh` é regenerado pelo Wails, não editado manualmente.

O setup instala por usuário em `%LOCALAPPDATA%\Programs\ProjetoP2P`, cria atalho no menu Iniciar, oferece atalho opcional na área de trabalho, instala WebView2 se necessário e registra desinstalador. O perfil `%APPDATA%\ProjetoP2P` é separado e preservado em atualizações/desinstalação. Não há atualização automática na 0.2.

### Compilar localmente

Instale Go >=1.26 (validado com 1.27.3), Node 24 e NSIS 3.11. Adicione `makensis` e o diretório de binários Go ao PATH. Na raiz: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0`; depois `./scripts/desktop.ps1 -Installer`. A saída esperada fica em `build/bin`. O script sem `-Installer` gera somente o executável desktop.

## Linux Mint

Alvo inicial: Mint 22 AMD64 / base Ubuntu 24.04. Dependências de build: `build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev`, Go e Node. Instale Wails v2.17.0, rode `npm ci && npm run build` dentro de `frontend` e `sh scripts/desktop-linux.sh` na raiz. A saída desktop é `build/bin/ProjetoP2P`. Execute `chmod +x` se necessário. As bibliotecas GTK3/WebKit2GTK 4.1 também precisam existir no sistema de destino. Não foi criado pacote .deb.

O workflow Linux faz build nativo, testes e upload de `ProjetoP2P-Linux-v0.2`. A compilação local no Windows gera apenas o CLI Linux, não um desktop Linux validado. O workflow não foi executado remotamente nesta entrega.

Referências oficiais: [Wails v2, instalação](https://wails.io/docs/gettingstarted/installation/), [NSIS no Wails](https://v2.wails.io/docs/v2.13.0/guides/windows-installer/), [WebView2](https://learn.microsoft.com/microsoft-edge/webview2/concepts/distribution).
