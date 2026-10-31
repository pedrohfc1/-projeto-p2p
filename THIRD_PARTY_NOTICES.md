# Componentes de terceiros

A licença MIT do Projeto P2P aplica-se ao código original. Dependências preservam suas próprias licenças; não são relicenciadas pelo projeto.

Dependências diretas Go: go-ole (MIT), go-libp2p (MIT), go-libp2p-kad-dht (MIT), go-multiaddr (MIT), Wails v2 (MIT), bbolt (MIT), golang.org/x/sys (BSD-3-Clause). Frontend: React/React DOM, Lucide React, Vite, TypeScript e tipos React (MIT; consulte também avisos específicos nos pacotes). As versões e dependências transitivas estão em go.mod/go.sum e frontend/package-lock.json. O runtime Go inclui sua licença BSD e avisos em https://go.dev/LICENSE.

Os workflows geram um inventário e cópias das licenças Go/frontend nos artefatos de distribuição. Consulte os arquivos de licença originais para os termos completos, inclusive avisos transitivos. Ferramentas de build/testes não fazem parte do aplicativo final.

Microsoft Edge WebView2: runtime proprietário da Microsoft, sujeito aos termos Microsoft. O instalador inclui o bootstrapper Microsoft com assinatura verificada no build, não declara o aplicativo como assinado. Hamachi e Radmin VPN são produtos de seus fornecedores, instalados separadamente; não incluídos, modificados ou relicenciados.
