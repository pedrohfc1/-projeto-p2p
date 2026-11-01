# Instalação e atualização 0.4.1

## Windows 10/11 AMD64

Baixe o instalador real `ProjetoP2P-Setup-v0.4.1.exe` na release oficial que o mantenedor publicar. Não precisa instalar Go, Node.js ou Python. Feche o aplicativo antigo e execute o setup com seu usuário habitual; ele usa asInvoker e não solicita elevação própria. Instala em `%LOCALAPPDATA%/Programs/ProjetoP2P`, cria atalho no menu Iniciar, oferece atalho na área de trabalho e registra desinstalação em Aplicativos Instalados (HKCU).

Microsoft Edge WebView2 é necessário. Se ausente, o instalador executa o bootstrapper Microsoft com assinatura validada durante o build; precisa de internet e está sujeito às políticas do sistema. O setup/app não têm certificado próprio de assinatura de distribuição. Não desative o SmartScreen, antivírus ou firewall para instalar.

O perfil continua em `%APPDATA%/ProjetoP2P`. Atualizações e desinstalação preservam identidade, contatos e mensagens. Não copie a identidade para o amigo e não apague o perfil para resolver conexão. Faça backup privado com o app fechado se necessário; a chave Windows é protegida por DPAPI e não constitui backup portável entre usuários/máquinas. O histórico não é criptografado pelo app em repouso.

Se a instalação falhar, não considere o produto instalado: ela pode deixar arquivos parciais e não tem rollback transacional completo. Corrija a causa e repita com o aplicativo fechado. O desinstalador pode deixar um pequeno helper temporário para limpeza normal do sistema. Instalação, atualização e desinstalação completas precisam da aceitação descrita em VALIDACAO-DISTRIBUICAO.md.

A alternativa `windows-amd64-portatil.zip` contém o executável, não substitui o instalador nem cria atalhos. Exige WebView2 e usa o mesmo diretório de perfil do usuário; não cria um perfil portável.

## Linux Mint 22 AMD64

O workflow Linux gera `ProjetoP2P-v0.4.1-linux-amd64.tar.gz` com desktop nativo. Este ambiente Windows não gerou nem executou esse desktop. Requisitos de runtime no Mint 22/Ubuntu 24.04:

```sh
sudo apt-get update
sudo apt-get install libgtk-3-0t64 libwebkit2gtk-4.1-0
sha256sum --check SHA256SUMS.txt
mkdir -p "$HOME/.local/opt"
tar -xzf ProjetoP2P-v0.4.1-linux-amd64.tar.gz -C "$HOME/.local/opt"
"$HOME/.local/opt/ProjetoP2P-v0.4.1/ProjetoP2P"
```

Use o pacote para sua distribuição; nomes de dependências podem variar fora do Mint 22. Não é necessário Go/Node para executar o binário já compilado. Perfil em `$XDG_CONFIG_HOME/ProjetoP2P` ou `~/.config/ProjetoP2P`. Para atualizar, feche o app e extraia a nova versão; para remover, apague somente o diretório de programa escolhido, preservando o perfil. Radmin VPN é integração específica do Windows; modo nativo/interface manual continuam disponíveis no Linux.

O pacote separado com sufixo `-cli.tar.gz` contém somente o núcleo de terminal `projetop2p`, não a interface gráfica. Foi cross-compilado no Windows. Ele não deve ser confundido com o desktop Linux produzido pelo runner nativo.

## Conferir SHA-256

Windows: `Get-FileHash .\ProjetoP2P-Setup-v0.4.1.exe -Algorithm SHA256` e compare com a linha do arquivo na release. Linux: `sha256sum --check SHA256SUMS.txt` na pasta com os artefatos. Se baixou apenas alguns arquivos, compare suas linhas ou hashes individuais; a checagem completa acusa os arquivos ausentes. O hash verifica integridade, não substitui autenticação do publicador.

## Primeiro contato

Veja CONEXAO-RADMIN-VPN.md ou CONEXAO-HAMACHI.md. Instale apenas a VPN escolhida, configure a mesma rede nela e troque convites no aplicativo. Não distribua sua pasta de perfil junto ao executável. Modo Automático usa libp2p nativo; para VPN escolha o modo explícito.

O instalador também incorpora e instala LICENSE.txt (MIT) e THIRD_PARTY_LICENSES.zip com avisos/inventários de dependências. A verificação de payload confere os hashes desses arquivos. O desinstalador remove esses arquivos do programa, sem acessar o perfil. ZIP portátil e tar.gz também carregam as licenças.
