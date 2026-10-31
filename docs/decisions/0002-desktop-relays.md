# ADR 0002 — desktop, persistência e conectividade 0.2

## Decisão

Manter Go/libp2p e o protocolo assinado de streams 0.1. Usar Wails v2.17.0, React 19.2.0 e TypeScript 5.9.3. A versão v2 foi escolhida por manter o modelo Go/React, empacotamento NSIS documentado e integração pequena com o núcleo; a versão atual foi resolvida no proxy Go, baixada e compilada com Go 1.27.3. Wails v3 não foi escolhido automaticamente por ser mais novo: exigiria avaliar outra API/cadeia de build para este escopo. A decisão não afirma que v3 é instável.

Vite foi atualizado para 7.3.7 depois que `npm audit` apontou vulnerabilidades na versão inicialmente avaliada. O audit final não encontrou vulnerabilidades na árvore resolvida na data do teste. Isso não substitui auditoria de segurança. `package-lock.json` fixa a resolução; o programa não consulta npm em execução.

Persistência: bbolt v1.5.0, banco transacional embutido em Go, em vez de adicionar driver SQLite/CGO. As transações armazenam configurações, contatos, mensagens e um índice de deduplicação. Retenção inicial: 10 mil mensagens totais e 300 carregadas por conversa. O banco impede abertura concorrente do mesmo perfil. Perfil fora do diretório de instalação permite atualização sem apagar identidade. ACK desktop passa a ocorrer depois do commit local; timeout continua sujeito a entrega incerta.

Convites: envelope JSON assinado com chave Ed25519 da identidade, versão, validade, nome, chave pública, ID e multiendereços públicos (no sentido de não secretos; podem incluir endereços LAN). O convite vincula a chave ao Peer ID e rejeita adulteração/expiração. Não prova identidade civil. Adicionar um contato não altera silenciosamente relays ou bootstraps do usuário.

AutoRelay recebe uma fonte limitada de candidatos estáticos e descobertos por DHT. Servidores voluntários anunciam um namespace distinto. Identify e checagem do protocolo pelo libp2p verificam capacidade; dados DHT não são tratados como autoridade. O modo assistido declara alcance privado para evitar que redes pequenas aguardem indefinidamente observadores AutoNAT. O modo automático permanece disponível nas configurações. Clientes tentam manter duas reservas e usam a lógica de renovação/backoff do libp2p. Não há cadastro central de relays.

Streams de chat autorizam explicitamente conexões limitadas de relay através de `WithAllowLimitedConn`; a 0.1 não fazia isso. Endereços alternativos para um mesmo Peer ID são mesclados e limitados a 16, permitindo reconexão sem perder os caminhos restantes. DCUtR permanece habilitado para promoção direta quando possível. Não forçamos desligar segurança do transporte para obter conectividade.

## Limites de validação

Build desktop Windows comprovado. NSIS falhou ao iniciar neste sandbox; workflow Windows produz o setup real. WebView2 falhou com 0x8000ffff ao criar o controlador; não foi confirmado funcionamento visual nativo. O harness de QA usa o mesmo serviço Go em HTTP loopback com token por execução e validação de origem, sem estar incluído no binário desktop. Testes desse harness não comprovam a janela Wails. Linux desktop exige build nativo com GTK3 e WebKit2GTK 4.1; workflow preparado, não executado remotamente aqui.

Fontes oficiais consultadas: [Wails v2](https://github.com/wailsapp/wails/tree/v2.17.0), [instalação](https://wails.io/docs/gettingstarted/installation/), [NSIS](https://v2.wails.io/docs/v2.13.0/guides/windows-installer/), [AutoRelay](https://pkg.go.dev/github.com/libp2p/go-libp2p/p2p/host/autorelay), [libp2p hole punching](https://docs.libp2p.io/concepts/hole-punching), [bbolt](https://github.com/etcd-io/bbolt). Além da documentação, foram lidas as APIs das versões fixadas no cache dos módulos.
