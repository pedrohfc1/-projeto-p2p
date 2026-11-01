# Beta privado 0.6.0 — comunidades e voz para duas pessoas

Data: 1 de novembro de 2026. Não publicar automaticamente. A 0.6.1 (malha 3–6 e volume individual) ainda não foi iniciada.

## O que mudou

O terceiro ingresso falhava por reutilização de um convite de uso único, não por um limite de dois membros. O convite legado conserva essa restrição. Novos convites permitem uso único, múltiplos usos (2–100 no backend; opções 5/10/25/50/100 na interface) ou permanente. Validades: 1h/24h/7d/30d/sem expiração. Um convite permanente da interface não expira, mas pode ser revogado. A política inteira é assinada Ed25519 e incluída no compacto/QR. Status mostra consumo/restante consultados na autoridade; não é uma autorização baseada no contador da interface.

Admissões são serializadas pelo proprietário em um registro assinado. Exatamente cinco ingressos vencem uma corrida de vinte solicitações contra um convite com cinco usos. Sem o proprietário não há ingresso offline nem promessa de cota global descentralizada exata. Revogação individual é um evento assinado; mudanças de papel/remoção incrementam a época e invalidam convites anteriores. Convite utilizado, esgotado, expirado, revogado, comunidade cheia, falha de sincronização e proprietário/rede indisponível têm diagnósticos distintos. Uma falha de conexão preserva a operação para tentativa posterior; não significa aprovação.

Novas comunidades: 100 membros registrados (incluindo proprietário), 50 canais de texto e 20 de voz. Isso não significa cem conexões simultâneas nem cem participantes em áudio. Os limites libp2p de conexões/streams permanecem; uma chamada aceita no máximo duas identidades nesta primeira entrega. Salas podem ser renomeadas, excluídas, limitadas a uma ou duas pessoas e restritas a proprietário/moderadores. A arquitetura reserva capacidade maior, sem ativar grupos antes da validação básica.

## Texto e sincronização

O registro de controle continua ordenado/assinado pelo proprietário: identidades, canais, papéis, ingressos e revogações. Para comunidades criadas na 0.6, texto usa um diário separado, assinado pelo autor e vinculado à raiz, época e prefixo conhecido desse registro. Membros trocam mensagens diretamente e recuperam entradas por cursores locais com deduplicação transacional. Não há uma ordem global consensual de mensagens; a sequência mostrada é a ordem de chegada local. Uma confirmação de réplica não comprova leitura por todos.

O proprietário fornece apresentações de endpoints assinados somente a membros atuais. As rotas ficam em memória por até dez minutos e as credenciais EasyTier ficam no armazenamento protegido já existente. Não são inseridas no registro público da comunidade. Com apresentações já recebidas, dois membros continuam trocando texto enquanto o proprietário está desligado. Reiniciar todos os nós ou expirar as apresentações pode exigir o proprietário novamente. O diário tem limite de 10.000 mensagens por comunidade e divide cotas de 16 MiB/comunidade e 64 MiB/perfil com o registro de controle.

Comunidades legadas preservam o formato original e o caminho de texto ordenado pelo proprietário. Não são reinterpretadas ou convertidas silenciosamente. Para testar o caminho novo crie uma comunidade na 0.6. O histórico antigo permanece acessível. Sob partição, revogações ainda desconhecidas não podem ser descobertas instantaneamente; revogações conhecidas bloqueiam novas aceitações. Entradas de época antiga não são admitidas como texto novo após a mudança de permissões; cópias históricas já recebidas são preservadas. Não há E2EE de grupo.

## Arquitetura de voz

Captura: WebView2 getUserMedia após clicar em Entrar. Opus mono com AEC/redução de ruído/ganho automático solicitados ao sistema, limite de envio de 32 kbit/s no navegador. O navegador negocia uma ligação WebRTC local com Pion no processo Go. Go encaminha RTP Opus, sem codec próprio, decodificação ou gravação. Outra ligação Pion DTLS-SRTP chega ao processo remoto, que entrega o áudio ao navegador remoto. Somente áudio/Opus, sem vídeo ou datachannel.

A sinalização usa `/projeto-p2p/voice/0.6.0` em streams libp2p autenticados. Identidade esperada, sessão, destino, nonce, autorização e fingerprint DTLS são verificados. Apenas o iniciador determinístico pode oferecer; repetição idêntica é idempotente e renegociações conflitantes são recusadas. Autoridade do proprietário assina presenças com validade máxima de 45 segundos, limite de duas identidades e permissões do canal. Não retransmite o áudio. Revogação conhecida fecha mídia; sem renovação, a chamada encerra em até 45 segundos. Reingresso exige ação explícita; a reconexão de transporte de uma chamada ainda ativa não pede novo microfone.

O SOCKS5 do etcat fixado aceita CONNECT TCP, não UDP ASSOCIATE. A mídia usa um adaptador distinto: encaminhamento UDP nativo do EasyTier em espaço de usuário. O auxiliar recompilado permite apenas a porta UDP de mídia daquele processo, mais o gateway TCP existente. Cada par de mídia tem um auxiliar dedicado, independente do chat. Candidatos ICE são reescritos para o encaminhador local autorizado; o fingerprint autenticado é preservado. O transporte externo EasyTier pode escolher relay TCP: não afirmamos ausência de head-of-line blocking nem caminho direto sem evidência. RTT medido não é latência boca-ouvido.

Não cria TUN, adaptador virtual, serviço, rotas, DNS ou regras de firewall. Não exige Radmin/Hamachi/EasyTier externo nem administrador por design. Sem microfone, a rede pode iniciar, mas não captura. Sair, falhar ou perder autorização encerra tracks e conexões. A barra de chamada fica fora da navegação de canais; minimizar não deve encerrar a chamada. Teste local de microfone dura quatro segundos, apenas medidor, sem gravação ou envio. Trocar entrada exige sair/entrar; saída depende de setSinkId no WebView. Sem suporte, usa o dispositivo padrão do Windows.

## Teste real entre dois Windows, em casas diferentes

1. Feche a versão anterior. Faça uma cópia da pasta `%APPDATA%/ProjetoP2P` com o aplicativo fechado. Use o mesmo instalador privado nos dois PCs. O setup preserva o perfil, identidade, contatos e histórico. A migração para schema 3 recusa downgrade destrutivo; para voltar à 0.5.2 utilize o backup anterior, nunca apague a identidade atual.
2. Em Configurações, selecione EasyTier integrado e salve nos dois PCs. Não instale VPN separada e não altere o roteador. Se a rede corporativa bloquear tráfego, registre o erro; o aplicativo não muda firewall automaticamente. Relays comunitários não têm garantia de disponibilidade.
3. Na pessoa A, crie uma comunidade nova. Clique em Convidar, escolha múltiplos usos/5/24h e gere o convite. Envie o compacto ou QR por canal de confiança. Não publique convites: carregam autorização/credenciais de conexão.
4. Na pessoa B, entre pelo convite e confirme o nome/Peer ID por um canal independente. Aguarde a participação aparecer como membro. Testem mensagens em #geral nos dois sentidos.
5. Mantenha o proprietário online durante a chamada. Com fones para evitar eco, abra Configurar e testar áudio, escolha entrada e saída e clique no teste local. Cada pessoa deve autorizar a permissão de microfone do Windows/WebView quando solicitada. A barra deve responder à fala no teste local. Nada deve captar antes dessa ação.
6. Ambos clicam em Entrar na mesma sala de voz. Confiram nomes autorizados e estado connected; falem alternadamente e simultaneamente. Se reprodução automática for bloqueada, clique em Ativar reprodução. Registre se a saída escolhida funciona; setSinkId não é garantido em todo WebView.
7. Silencie A e confirme que B deixa de ouvi-lo. Ative ensurdecer em B e confirme que ele não recebe nem transmite enquanto ensurdecido. Restaure os controles. Navegue para outro canal, abra o chat privado e minimize/restaure: a barra deve permanecer ativa e a voz continuar.
8. Interrompa temporariamente a conexão de rede de um PC por poucos segundos e restaure. Acompanhe reconexão/erro/expiração. Se passar de 45 segundos sem renovar autorização, o microfone deve ser desconectado e será preciso Entrar novamente. Não desative proteções do sistema.
9. Teste remover um membro ou restringir a sala enquanto há chamada. Ao conhecer a revogação, a mídia deve parar. Um convite antigo não pode readmitir a identidade removida. Para testar reingresso, gere autorização nova.
10. Clique em Desconectar e confira o indicador do microfone no Windows. Feche o aplicativo e verifique que seus auxiliares próprios encerraram. Registre consumo dos processos Go, WebView2 e etcat separadamente no Gerenciador de Tarefas, além de duração, jitter/perdas e erros. Não envie gravações, tokens, bancos ou chaves nos relatos.

A voz entre casas, CGNAT real, dispositivos físicos, troca real de caminho e instalação/atualização em Windows limpo ainda precisam dessa execução. Não foram declarados concluídos. Para testar Radmin como alternativa, ambos escolhem explicitamente a mesma VPN em Configurações e geram novos convites desse modo; não alternem o transporte de uma chamada em curso.

## Reproduzir testes disponíveis

Compile o frontend antes do Go. Veja AGENTS.md para toolchains e comandos de build. `npm ci`, `npm test`, `npm run build`; `go test ./...`; `go vet ./...`; `python scripts/check-version.py`; `scripts/build.ps1`. O desktop usa `wails build -s -skipbindings -trimpath` após compilar os assets.

Integrações reais locais opcionais: defina `P2P_ETCAT_TEST_DIR` para a pasta contendo o auxiliar com hash do manifesto. Compile `go build -o CAMINHO/webui-v060.exe ./tests/webui` e defina `P2P_WEBUI_TEST_BIN` para o executável. Execute `go test ./internal/desktop ./internal/voice ./internal/overlay -run 'TestReal|TestConcurrentInviteQuota' -count=1 -v`. O QA escuta só em loopback e exige token/Origin; não é servidor de produção nem vem no instalador. Os testes usam perfis temporários isolados e Opus DTX sintético; não ativam o microfone do usuário.

Execute ainda `python tests/community_desktop_smoke.py CAMINHO/webui-v060.exe --integrated --metrics CAMINHO/metrics.json`, `python tests/cli_smoke.py dist/projetop2p-windows-amd64.exe` e `python scripts/audit-pe.py build/bin/easytier/etcat-p2p.exe build/bin/ProjetoP2P.exe`. O procedimento de empacotamento exige fontes correspondentes LGPL completas; consulte third_party/etcat/BUILD.md.

## Fontes técnicas

[Etcat upstream](https://github.com/EasyTier/etcat), [fonte fixada do adaptador SOCKS/cliente](https://github.com/EasyTier/etcat/blob/ccb45cd84c20b2aa1e6615a3f62a83827e961695/src/app.rs), [Pion WebRTC](https://github.com/pion/webrtc). A comprovação do caminho UDP desta entrega vem dos testes do binário modificado, não da suposição de suporte UDP em SOCKS.
