# Validação 0.4 — 31/10/2026

## Executado

- 22 testes Go passaram (`go test ./... -v -count=1 -timeout=180s`); `go vet ./...` sem achados; `go mod verify` confirmou módulos.
- Fixtures Radmin sem Hamachi: driver/nome renomeado, IPv4, índice, estado, ausência/erro, escolha independente, rejeição de falso positivo; convite assinado e escopo; preservação do perfil e falha de bind sem fallback público.
- Compilação TypeScript/Vite, Wails desktop Windows e CLI Windows/Linux amd64.
- Smoke CLI: três processos reais, DHT, texto assinado, fallback/falha bootstrap, crash, identidade e reconexão.
- Smoke relay: daemon após EOF, status real, reserva assinada e limites/contadores.
- Smoke API desktop com assets de produção: configurações, convites, contatos, mensagens persistentes e reinício. Não constitui inspeção visual da janela nativa.
- Adaptador Hamachi real: enumeração e bind TCP vinculado à interface; não testou mensagens via VPN.
- Instalador 0.4 gerado; payload executado em modo de verificação, hash do app conferido; desinstalador e bootstrapper WebView2 Microsoft incluídos. Nenhuma VPN incluída.

Logs em `docs/evidence/v0.4/`.

## Não executado / limitações

Radmin real: SKIP por ausência de adaptador Radmin ativo. Conexão Radmin entre dois PCs Windows em casas distintas não foi executada. Aprovação de conectividade depende do roteiro `radmin-guide.md`; fixtures não comprovam funcionamento de uma rede real.

Instalação completa, atualização, atalhos/desinstalação e inspeção visual da janela WebView2 não foram executadas neste ambiente. O teste da API valida backend e assets, não interação visual. Desktop Linux Mint não foi executado; CLI Linux cross-compilado e workflow nativo preparado. Aplicativo/setup sem certificado de assinatura de distribuição.

Nenhuma configuração de VPN, firewall ou roteador foi alterada. Interface ativa não é peer conectado; o diagnóstico realiza conexão autenticada e ping libp2p. Regras de firewall são lidas apenas para orientação. Não há garantia de conexão sob qualquer política de rede.

Identidade, contatos e histórico mantêm os mesmos caminhos e formatos compatíveis. Não foram implementadas voz, blockchain ou armazenamento distribuído de conteúdo.
