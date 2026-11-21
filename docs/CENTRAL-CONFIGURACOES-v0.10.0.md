# PeeTwoPee 0.10.0 — central de configurações e frontend

Beta privada, sem publicação pública. Rollback do código 0.9.9 salvo antes das alterações.

## Uso
A engrenagem ao lado do perfil é a entrada principal e permanece disponível durante chamadas. Os atalhos duplicados na barra de comunidades, no rodapé antigo e nas ações da chamada foram removidos. O clique no avatar abre um cartão de perfil; a identidade completa fica em detalhes avançados.

A central tem 15 seções: Conta, Perfil, Dados e privacidade, Permissões de mensagens, Notificações, Voz, Transmissão, Sons, Painel de efeitos sonoros, Avançado, Aparência, Comunidades, Convites, Conectividade e Sobre / versão. Voz e vídeo usam subnavegação própria. Setas, Home/End, Tab e Escape são suportados. O foco retorna à interface ao fechar. A central inicia fechada, mesmo lembrando a última seção.

Voz mantém entrada/saída lado a lado, volumes separados, teste local, medidor, retorno e seleção persistente de dispositivos. Ajuda de dispositivos fica recolhida. Ao sair da aba Voz, o teste local termina; a captura da chamada permanece com o mesmo proprietário. O perfil de entrada atual é Padrão; novos perfis e DSP são indicados como futuros, sem processamento novo ou controles falsamente funcionais.

Transmissão reúne os perfis e limites já suportados. Os padrões locais podem ser escolhidos antes de entrar na sala. Durante uma transmissão, os limites preservam os bloqueios existentes. Fontes manuais WASAPI ficam em Voz e vídeo → Avançado durante uma chamada. A seleção automática de áudio e suas proteções anteriores permanecem. Volume da transmissão e controles de assistir/parar continuam no player; métricas somente em detalhes recolhíveis e console.

Comunidades e edição de salas usam a mesma central para os formulários de configuração. Criação, ingresso, convites e confirmações de operações continuam fluxos explícitos. As verificações de proprietário/moderador permanecem no backend assinado. Salvar perfil ou rede pode reconectar o nó, como no serviço existente; há aviso antes da ação. Não foram adicionadas permissões de mensagens sem suporte real.

Categorias de canais recolhem sem desmontar seus filhos, lembrando a escolha local. O chat preserva respostas, reações, edição e exclusão existentes; ganha separadores de data, destaque do canal, composição mais clara e os contadores reais de não lidos. Perfis e membros mantêm menus de contexto, menção e volume individual quando aplicável.

Aparência oferece densidade, texto ampliado e redução de animações locais. O tema permanece escuro. Sons personalizados, soundboard e novos processamentos estão claramente marcados como indisponíveis/em breve.

## Preservação da mídia e dos dados
Os componentes donos de voz, captura de tela e áudio WASAPI continuam montados. Portais levam somente formulários para a central; não movem o player nem recriam sessões. Foram comparados com 0.9.9: 68 arquivos de backend/mídia e bibliotecas de captura/reprodução inalterados, além dos corpos de entrada/rebind/troca de microfone e captura/publicação/assinatura/player de tela. Nenhuma migração de identidade/histórico ou mudança em VPN, firewall, rotas e DNS.

## Validação
85 testes frontend e build; Go completo e vet; CLI Windows/Linux e smoke; 2/3 processos locais de voz; mídia gerada simultânea sobre EasyTier; recuperação do proprietário/streams; três serviços locais de comunidades, convites, permissões, persistência e chat. O primeiro ensaio combinado teve timeout na conexão inicial SOCKS, antes de executar frontend; a repetição isolada passou sem alterar backend ou timeouts. O resultado inicial não foi omitido.

A interface Wails/WebView2 foi inspecionada em perfil sintético isolado: uma engrenagem, central fechada ao iniciar, Voz/Transmissão/Avançado/Comunidades, navegação por seta e Escape, formulários autorizados, cartão de perfil, categorias recolhíveis, janela normal/maximizada e minimizar/restaurar preservando a interface. O conflito inicial de largura/altura herdadas foi corrigido e o build final reinspecionado. Nenhuma permissão de microfone, notificações ou tela foi concedida automaticamente. Identidade e mensagem local do perfil QA permaneceram após reiniciar. O Edge headless não iniciou neste ambiente; isso não é contado como teste de navegador aprovado.

## Teste físico pendente
1. Atualizar dois PCs, conferir identidade, contatos, histórico e comunidade existente.
2. Entrar em chamada e abrir a engrenagem. Navegar entre Voz, Transmissão e demais seções; verificar áudio bidirecional sem interromper streams.
3. Autorizar o microfone pessoalmente, testar retorno, trocar entrada/saída e conferir volumes/mute/ensurdecer. Fechar a central deve encerrar somente o teste local.
4. Compartilhar tela com consentimento, assistir por AO VIVO, navegar, minimizar/restaurar e conferir voz/vídeo/áudio do computador. Repetir na rede integrada e Radmin já configurado.
5. Recolher categorias, reabrir canais, usar perfis, mensagens, reações e convites. Confirmar permissões após reconexão.

Mídia gerada não substitui áudio/vídeo físico. Janela mínima de 760px, Linux desktop, instalação/atualização gráfica completa e reboot continuam pendentes. Fontes correspondentes e licenças EasyTier acompanham o instalador privado. Bloqueios existentes de publicação pública preservados.
