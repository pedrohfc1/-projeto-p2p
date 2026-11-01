# Validação das entregas 0.5.1 e 0.5.2

Data: 2026-11-01. Trabalho local, sem push/tag/release. Branches separadas: feature/v0.5.1-compact-invites e experimental/v0.5.2-easytier. Perfis de teste isolados; identidade, banco esquema 2, contatos e configurações de VPN preservados.

| Verificação | Resultado |
|---|---|
| Suíte Go completa e go vet | PASS no código final; integrações EasyTier opcionais executadas separadamente |
| Frontend | 9 testes na 0.5.1; 10 na 0.5.2; TypeScript e Vite PASS |
| Distribuição | 4 testes Python e consistência VERSION PASS |
| Codec binário | Legado/compacto, tamanhos, assinatura, expiração, adulteração, listas e relays PASS |
| Fuzz do decoder Protobuf limitado | PASS, 125.633 entradas em execução curta, dois workers; não é prova formal |
| Backends 0.5.0 e novo | PASS, DM bidirecional e ingresso em ambos os sentidos; ticket do registro preserva compatibilidade |
| Desktop API e Radmin | PASS na 0.5.1; 0.5.2 também confirmou três serviços/comunidades no adaptador real local |
| EasyTier no-TUN real | PASS com core/CLI e Packet.dll verificados, dois auxiliares no mesmo PC |
| Relay EasyTier | PASS, três auxiliares e custo relay(2); A/B sem P2P automático |
| Isolamento e identidade | PASS, canário TCP alheio bloqueado, Peer ID falso recusado e identidade do perfil reutilizada |
| Ciclo de vida | PASS, auxiliar caiu/reiniciou e stream voltou; parada normal e crash do hospedeiro sem PID auxiliar remanescente |
| Windows desktop e CLI Windows/Linux | Build PASS; desktop Mint não executado aqui |
| Setup 0.5.1/0.5.2 | Payload corresponde ao app, desinstalador, MIT e notices. Sem EasyTier/Npcap/driver no setup |
| npm audit / go mod verify / actionlint | PASS; npm sem vulnerabilidades reportadas |
| Auditoria Git + gitleaks | PASS; sem credenciais/perfis/binários de terceiros nas fontes |
| govulncheck | FAIL conhecido GO-2024-3218 alcançável no DHT; não suprimido |

Medições de tamanho e comandos: [CONVITES-v0.5.1.md](CONVITES-v0.5.1.md). Segurança, arquitetura, licenças, métricas e passos de execução do protótipo: [EASYTIER-v0.5.2.md](EASYTIER-v0.5.2.md). O relatório contém o primeiro baseline e a repetição final confirmou echo de 5,146 ms e relay de 5,121 ms no mesmo PC. Não comparar diretamente com internet ou assumir consumo completo do aplicativo: WebView2 não foi medido, o processo Go do teste hospeda dois hosts e as amostras CPU são curtas.

## Limites materiais

A 0.5.1 é a entrega recomendada para convites compactos; códigos curtos só têm estrutura e mirrors locais. Não há descoberta global desses códigos.

A 0.5.2 é **laboratório de stream**, ainda não transporte do chat/comunidades. O auxiliar oficial Windows precisa da biblioteca Npcap Packet.dll, além da LGPL do EasyTier. Os artefatos entregues não redistribuem esses executáveis/DLL; preparador explícito para desenvolvimento verifica os três arquivos antes do uso. Distribuição turnkey exige fechar licenças/fontes/notices e preferencialmente gerar build mínimo sem pcap. A consulta de drivers via CIM foi indisponível, portanto no-TUN não é apresentado como prova de que nenhum driver pré-existia no PC.

O usuário informou não possuir nó público. Não foram testados dois PCs/casas, CGNAT, fallback de dois relays públicos ou NAT traversal externo. Houve teste local com dois nós conhecidos (um indisponível) e recuperação por outro nó, não demonstração de disponibilidade pública. Nenhum serviço público foi presumido/autoconfigurado.

Instalação/upgrade/desinstalação completos em VM, inspeção visual nativa, leitura óptica do QR, desktop Mint, race com compilador C, longa duração, cargo-audit/Rust e auditoria independente continuam pendentes. Verificação de payload não substitui instalação. App/setup permanecem NotSigned; Defender indisponível neste ambiente não equivale a arquivo limpo. Não houve assinatura com certificado próprio, alteração de firewall/Defender/SmartScreen ou criação de serviço de sistema.

Sem voz, vídeo, tela ou blockchain. Comunidades ainda dependem do dono para confirmar operações e não têm E2EE de grupo. Não é uma release de produção nem uma declaração de conectividade universal.
