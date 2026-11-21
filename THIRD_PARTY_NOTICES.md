## Laboratório Tailscale separado — 0.10.5

`experiments/tsnet` usa tailscale.com/tsnet v1.104.7 (BSD-3-Clause) em módulo Go independente. Não é ligado ao executável principal nem ao seu instalador. O ZIP separado do laboratório contém avisos dos módulos efetivamente compilados em Tailscale-Lab-licencas.zip, inventário e licença do Go. Nenhuma DLL Wintun/Npcap ou cliente Tailscale é redistribuído. Consulte experiments/tsnet/README.md.

# Componentes de terceiros

A licença MIT do Projeto P2P aplica-se ao código original. Dependências preservam suas próprias licenças; não são relicenciadas pelo projeto.

Dependências diretas Go: Pion WebRTC/RTP/RTCP e interceptors (MIT), go-ole (MIT), go-libp2p (MIT), go-libp2p-kad-dht (MIT), go-multiaddr (MIT), Wails v2 (MIT), bbolt (MIT), golang.org/x/sys (BSD-3-Clause). Frontend: React/React DOM, Lucide React, Vite, TypeScript e tipos React (MIT; consulte também avisos específicos nos pacotes). As versões e dependências transitivas estão em go.mod/go.sum e frontend/package-lock.json. O runtime Go inclui sua licença BSD e avisos em https://go.dev/LICENSE.

Os workflows geram um inventário e cópias das licenças Go/frontend nos artefatos de distribuição. Consulte os arquivos de licença originais para os termos completos, inclusive avisos transitivos. Ferramentas de build/testes não fazem parte do aplicativo final.

Microsoft Edge WebView2: runtime proprietário da Microsoft, sujeito aos termos Microsoft. O instalador inclui o bootstrapper Microsoft com assinatura verificada no build, não declara o aplicativo como assinado. Hamachi e Radmin VPN são produtos de seus fornecedores, instalados separadamente; não incluídos, modificados ou relicenciados.


## Auxiliar integrado do beta 0.6.0

etcat (Apache-2.0) e EasyTier (LGPL-3.0) recompilados, sem drivers. Fontes correspondentes, dependências e licenças acompanham o setup em EasyTier-corresponding-source.zip e permanecem na pasta instalada. Ver third_party/etcat/BUILD.md. A licença MIT do Projeto P2P não substitui essas licenças. Nenhum componente Npcap é redistribuído.


## Wails — política de consentimento do build Windows 0.7

O build Windows usa Wails v2.17.0 sob sua licença MIT, preservada no pacote de licenças. `scripts/prepare-webview-policy.py` verifica a versão e o hash revisado de `internal/frontend/desktop/windows/frontend.go`, copia o SDK para um diretório de build e muda somente a chamada global de `CoreWebView2PermissionStateAllow` para `CoreWebView2PermissionStateDefault`. A cópia do cache de módulos não é modificada. `scripts/build-desktop.ps1` usa essa cópia temporariamente e restaura go.mod/go.sum ao terminar. O código e os scripts da modificação acompanham os fontes do PeeTwoPee. Não é mudança das permissões de privacidade do Windows.
