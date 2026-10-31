# Instalação 0.4 — Windows e Linux

## Windows: instalador real

Arquivo de entrega: **ProjetoP2P-Setup-v0.4.exe**. É um instalador por usuário, não um portátil renomeado. Feche versões anteriores, execute e confirme o diretório apresentado. A segunda pergunta oferece atalho de área de trabalho. O atalho do menu Iniciar é criado na instalação.

Ele instala em `%LOCALAPPDATA%\Programs\ProjetoP2P`, grava o desinstalador e registra a entrada em Aplicativos Instalados do Windows (HKCU). Verifica o WebView2 e, se ausente, executa o bootstrapper Microsoft embutido (internet necessária). O bootstrapper tem assinatura Microsoft conferida no build. **Hamachi e Radmin VPN não estão incluídos e nunca é instalado/autenticado automaticamente.**

Perfil: `%APPDATA%\ProjetoP2P`. O instalador e desinstalador não removem esse diretório. Atualizações mantêm identidade, contatos, mensagens e configurações anteriores. Feche o aplicativo para substituir arquivos. Se ocorrer falha no meio da instalação, o instalador informa que não foi concluída; pode haver arquivos parcialmente instalados, sem rollback transacional completo. Corrija a causa e repita a instalação. Não apague o perfil para tentar corrigir problemas de rede.

O instalador 0.4 foi escrito em Go 64 bits para não depender da execução do NSIS 32 bits neste ambiente. Usa diálogos Win32, atalhos COM e manifest asInvoker. O aplicativo e o setup não têm certificado de assinatura de distribuição. Não desative proteções do Windows. O desinstalador usa um helper temporário para remover seu próprio executável após sair; esse pequeno helper pode permanecer no diretório temporário para limpeza normal do sistema.

**O arquivo foi gerado e seu payload verificado. Não foi executada uma instalação completa sobre seu perfil**, nem foi confirmada a janela WebView2 neste ambiente isolado. Instalação, atualização, atalhos e desinstalação devem passar por teste de aceitação num Windows comum antes de distribuição ampla. O sucesso de verificação do payload não equivale a sucesso de instalação.

## Reproduzir o build

Ferramentas: Go >=1.26 (usado 1.27.3), Node 24 e Wails v2.17.0. Na raiz:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
cd frontend
npm ci
npm run build
cd ..
./scripts/desktop.ps1 -Installer
```

A saída é `build/bin/ProjetoP2P-Setup-v0.4.exe`. O script primeiro compila o desktop e depois chama `scripts/package-windows.ps1`. Esse segundo script exige um build 0.4, confere a assinatura Microsoft do runtime, gera o desinstalador, incorpora app comprimido e grava manifest de execução por usuário. O executável desktop exato é conferido por SHA-256 dentro do setup.

Para conferir o arquivo sem instalá-lo:

```powershell
./tests/installer-smoke.ps1
```

O teste executa somente `--verify-payload`, sem criar atalhos, registro de instalação ou perfil. `build/bin/setup-payload.json` contém versão, tamanho e hash do aplicativo e indica que Hamachi não foi incluído. As fontes NSIS da 0.2 foram preservadas, mas não são o empacotamento canônico da 0.4.

O workflow `.github/workflows/desktop.yml` foi atualizado para **Desktop v0.4** e gera os mesmos artefatos em runner Windows. Não foi executado remotamente nesta entrega. Ele verifica o payload, sem declarar teste de instalação completo.

## Linux Mint

O CLI Linux AMD64 foi cross-compilado. O desktop usa build nativo com GTK3 e WebKit2GTK 4.1. Em Mint 22, instale `build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev`, Go e Node, compile frontend e instale Wails. Execute:

```sh
go test -tags webkit2_41 ./... -timeout=180s
sh scripts/desktop-linux.sh
```

O modo automático e a seleção manual são mantidos no Linux; a detecção por metadados do driver Hamachi foi priorizada e implementada no Windows. Não houve execução nativa em Mint nesta entrega. O workflow Linux prepara build e testes no Ubuntu 24.04.
