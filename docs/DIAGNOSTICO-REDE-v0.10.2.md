# Diagnóstico de rede 0.10.2 — beta privada, escopo parcial

O painel em Configurações > Conectividade mede RTT autenticado de aplicação, não latência física de voz. Duração 10/30/60 segundos limita o agendamento; uma sonda em andamento pode terminar depois desse período usando o timeout existente do teste de conectividade. Cancelar descarta resultados, mas não cancela um RPC já em andamento. Não inicia carga nem muda transporte, interface ou Radmin. Repita manualmente o mesmo peer físico e condições para comparar modos.

Contadores por participante medem bytes de streams libp2p (payload, não overhead físico). Tempo é a abertura da conexão atual e não o ingresso na sala. O contador é limitado a 256 identidades por execução; após atingir o limite, identidades novas não têm medição completa. A ausência de medição é serializada como null e exibida como não disponível. Taxas globais incluem todas as conexões. Caminho interno do EasyTier, capacidade disponível, perdas UDP/reordenação, retransmissões TCP, RTT do relay e latência sob carga não foram instrumentados e não devem ser inferidos.

Histórico local: até 12 testes de 60 sondagens, apenas tempos, RTT e modo. JSON validado por allowlist não guarda IDs, IPs, convites, áudio/vídeo ou mensagens. Copiar/exportar exige confirmação. Gráficos mostram RTT observado e a tabela estatísticas de testes anteriores. Lacunas/falhas não são perdas de datagramas. Variação sucessiva de RTT não equivale a jitter RTP. Quantis usam nearest-rank e poucas amostras não sustentam conclusões fortes.

Não foi introduzido protocolo de throughput nem nova autorização remota de carga. Diagnósticos existentes de mídia permanecem no console; integração de séries de FPS, bitrate e amostras de reprodução, correlação, hipóteses automatizadas e comparação controlada ainda estão pendentes. Nenhum gargalo EasyTier vs Radmin foi comprovado por esta etapa.

Não instalar Radmin, alterar codecs, encoder, firewall/DNS/rotas nem publicar Release. Testes físicos entre casas, Radmin real e desempenho do WebView2 permanecem pendentes. O instalador é beta parcial de instrumentação, não uma solução confirmada de estabilidade.
