# Métricas iniciais 0.5

Medição local em 2026-11-01. Três processos do backend real, perfis temporários, adaptador Radmin real **no mesmo PC Windows**. Não é teste entre casas, benchmark de longa duração ou medição do aplicativo completo. O helper chama as mesmas APIs de produção, sem WebView2.

| Medida | Valor observado |
|---|---:|
| RSS Go do dono antes das comunidades | 27.14 MiB |
| Heap Go inicial | 2.77 MiB |
| CPU amostrada inicial | 1.45% de um núcleo |
| Goroutines iniciais | 28 |
| Cenário completo | 13.36 s |
| RPC de mensagem até commit do dono | 24.50 ms |
| Ping manual local registrado | 0.000 ms |

O ping desta execução retornou zero na resolução disponível; não significa latência física zero. A aplicação guarda microssegundos quando medidos e preserva milissegundos para compatibilidade. Repetir em dois PCs antes de tirar conclusões sobre latência. O tempo do RPC inclui chamada do helper, assinatura, rede e confirmação do dono; não é latência de renderização nem leitura pelo destinatário.

| Após cenário | RSS Go (MiB) | CPU amostrada (%) | Upload acumulado (bytes) | Download acumulado (bytes) | Conexões / streams |
|---|---:|---:|---:|---:|---:|
| owner | 27.60 | 1.41 | 4065 | 3737 | 2 / 0 |
| moderator | 28.29 | 0.39 | 6735 | 13396 | 1 / 0 |
| member | 33.35 | 67.17 | 6665 | 13642 | 1 / 0 |

O dono foi encerrado e reiniciado durante o cenário. Seus contadores finais e RSS pertencem a outro processo: não calcular crescimento antes/depois como se fosse a mesma sessão. CPU é uma amostra curta e pode incluir picos do teste, inclusive no membro; 100% representa um núcleo lógico e pode ser ultrapassado. Não há média estável ou percentil de desempenho comprovado.

Tráfego corresponde ao reporter libp2p, não a toda a placa/VPN. Estado expõe totais e taxas, conexões, streams, último tempo de conexão e ping manual. Perda de pacotes é `null`, com motivo: TCP não fornece a perda UDP necessária para avaliar voz. Não apresentar indisponível como 0%.

Tempo de commit React está instrumentado com performance.now/useLayoutEffect no frontend, mas **não foi medido na janela nativa**. RSS/CPU dos processos WebView2 não estão incluídos. A tentativa de inspeção visual nativa não produziu janela de testes controlável; não usamos a sessão pessoal já aberta. Portanto consumo total, fluidez visual, hardware modesto e vazamento após horas permanecem pendentes.

## Mudanças fundamentadas

Paginação de 100 mensagens limita renderização; índices por sequência evitam carregar toda a comunidade; outbox mantém ordem e deduplica; sincronização usa lotes de até 16 e trabalho limitado por ciclo; contadores globais evitam mapa de métricas por peer sem limite. Não há alegação de ganho percentual sem comparação controlada com o mesmo cenário anterior. Limites de fila e registro são explícitos.

## Repetição

```powershell
go build -buildvcs=false -o work-webui.exe ./tests/webui
python tests/community_desktop_smoke.py ./work-webui.exe --radmin --metrics metrics-radmin.json
```

O JSON produzido não contém IPs ou IDs pessoais. Para baseline de voz, medir também árvore inteira de processos em repouso por 5 minutos, histórico grande paginado, 30 minutos de conversa e retorno ao repouso. Usar duas máquinas/redes, repetir amostras e registrar versões do Windows, CPU/RAM e caminho de rede sem publicar endereços privados.
