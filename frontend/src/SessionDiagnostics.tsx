import {RoomProbeControls,type RoomProbeAPI} from './RoomProbeControls';
import React,{useEffect,useRef,useState} from 'react';
import {X,Pause,Play,Copy,Download,Activity} from 'lucide-react';
import {archiveSession,sessionArchive,importDiagnosticReport,evidence,average,percentile,historyWindowMs,type SessionSeries,type HistoryPoint,type Point} from './mediaDiagnosticModel';
import {copyDiagnosticText} from './diagnostics';
type API=RoomProbeAPI & {CopyDiagnostics?:(s:string)=>Promise<void>;ExportVoiceDiagnostic?:(s:string,text:boolean)=>Promise<void>};
const n=(v:number|null|undefined,suffix='')=>typeof v==='number'&&Number.isFinite(v)?v.toFixed(1)+suffix:'n/d';
type GraphField = 'rtt'|'jitter'|'bitrate'|'fps'|'interruptions';
type PlotPoint = Pick<Point, 'time'|GraphField>;

function Graph({points, field, label, windowMs}:{points:PlotPoint[];field:GraphField;label:string;windowMs:number}){
 const values = points.map(point => point[field]);
 const max = Math.max(1, ...values.filter((value): value is number => value !== null));
 const end = points.at(-1)?.time ?? 0;
 const segments: string[][] = [];
 let segment: string[] = [];
 values.forEach((value, index) => {
  const point = points[index];
  if (value === null || (index && point.time - points[index-1].time > 11000)) {
   if (segment.length) segments.push(segment);
   segment = [];
  }
  if (value !== null) segment.push(`${(point.time-end+windowMs)*300/windowMs},${48-value*44/max}`);
 });
 if (segment.length) segments.push(segment);
 return <figure className="diagnostic-graph">
  <figcaption>{label} · {n(values.at(-1))}</figcaption>
  <svg viewBox="0 0 300 52" role="img" aria-label={label+(windowMs===historyWindowMs?' nos últimos 30 minutos':' nos últimos 60 segundos')}>
   {segments.map((row,index) => row.length===1
    ? <circle key={index} cx={row[0].split(',')[0]} cy={row[0].split(',')[1]} r={2} fill="currentColor"/>
    : <polyline key={index} points={row.join(' ')} fill="none" stroke="currentColor" strokeWidth="2"/>)}
  </svg>
 </figure>;
}

const dimensions = (width?:number|null, height?:number|null) => width && height ? `${width} × ${height}` : 'n/d';
const efficient = (value?:boolean|null) => value === true ? 'eficiente segundo o WebView' : value === false ? 'sem eficiência indicada' : 'n/d';

export function SessionDiagnostics({api,onClose,series,joined}:{api:()=>API;onClose:()=>void;series:SessionSeries;joined:boolean}){
 const [,setVersion]=useState(0);
 const [paused,setPaused]=useState(false),pause=useRef(false);
 const [error,setError]=useState(''),[compare,setCompare]=useState(false);
 const [selection,setSelection]=useState<number[]>([]),[duration,setDuration]=useState(10);
 const [recording,setRecording]=useState<number|null>(null),[longPlot,setLongPlot]=useState(false);
 const observation=useRef<{from:number;until:number;session:string}|null>(null);
 pause.current=paused;

 useEffect(()=>{
  const key=(e:KeyboardEvent)=>{if(e.key==='Escape'&&!document.querySelector('.modal-overlay,.context-menu'))onClose()};
  document.addEventListener('keydown',key);
  return()=>document.removeEventListener('keydown',key);
 },[onClose]);

 useEffect(()=>{
  const unsubscribe = series.subscribe(() => {
   const current = observation.current;
   const last = series.points.at(-1);
   if (current && last && (last.time >= current.until || last.snapshot.session !== current.session)) {
    if (last.snapshot.session === current.session) archiveSession(series.summary(current.from));
    observation.current = null;
    setRecording(null);
    setSelection([]);
   }
   if (!pause.current) setVersion(value=>value+1);
  });
  return () => {
   unsubscribe();
   if (series.points.length) archiveSession(series.summary());
  };
 },[series]);

 useEffect(() => {
  if (!joined && observation.current) {
   observation.current = null;
   setRecording(null);
  }
 },[joined]);

 function startObservation() {
  const point = series.points.at(-1);
  if (!point || !joined) return;
  const from = Date.now();
  observation.current = {from, until:from+duration*1000, session:point.snapshot.session};
  setRecording(duration);
 }

 const report=()=>JSON.stringify(series.report(),null,2);
 const confirm=()=>window.confirm('O diagnóstico contém horários e identificadores pseudonimizados de sessões/peers. Não inclui conteúdo de mensagens, áudio ou vídeo. Revise antes de compartilhar. Continuar?');
 const copy=async()=>{if(!confirm())return;try{await copyDiagnosticText(report(),s=>api().CopyDiagnostics?.(s)??Promise.reject(Error('Cópia indisponível')),navigator.clipboard?.writeText?.bind(navigator.clipboard))}catch(e){setError(String(e))}};
 const save=async(text:boolean)=>{if(!confirm())return;try{const fn=api().ExportVoiceDiagnostic;if(!fn)throw Error('Exportação indisponível neste backend.');await fn(report(),text)}catch(e){setError(String(e))}};
 // Pause freezes rendered values while the bounded collector continues collecting.
 const rendered=useRef({points:[] as Point[],events:[] as {time:number;message:string}[],history:[] as HistoryPoint[]});if(!paused)rendered.current={points:[...series.points],events:[...series.events],history:[...series.history]};const points=rendered.current.points,last=points.at(-1),audio=last?.audio,video=last?.video;
 const selectedSummaries = selection.map(index => sessionArchive[index]).filter(Boolean);
 const comparable = selectedSummaries.length === 2
  && selectedSummaries[0].testType === selectedSummaries[1].testType
  && Math.abs(selectedSummaries[0].seconds - selectedSummaries[1].seconds) <= 2;
 return <aside className="session-diagnostics" aria-label="Diagnóstico da chamada"><header><Activity size={17}/><h2>Diagnóstico</h2><button className="icon" title="Fechar diagnóstico" aria-label="Fechar diagnóstico" onClick={onClose}><X size={18}/></button></header><div className="diagnostic-actions"><button onClick={()=>setPaused(!paused)}>{paused?<Play size={13}/>:<Pause size={13}/>} {paused?'Retomar':'Pausar visualização'}</button><button onClick={()=>void copy()}><Copy size={13}/> Copiar</button><button onClick={()=>void save(false)}><Download size={13}/> JSON</button><button onClick={()=>void save(true)}>TXT</button></div><p className="diagnostic-note">Sondas pequenas de RTT apenas em peers conectados da sala. Teste de carga opcional em Comparar conexões. n/d significa indisponível; silêncio e DTX não comprovam perda.</p>{(error||series.error)&&<p role="alert">{error||series.error}</p>}<div className="diagnostic-body"><p role="status">Autorização: {last?.snapshot.authorization||'aguardando'} · transporte: {last?.snapshot.mode||'n/d'} · atualização ~1 s{!joined?' · sessão encerrada':''}{paused?' · visualização pausada':''}</p><div className="diagnostic-actions" aria-label="Janela do gráfico">
 <button aria-pressed={!longPlot} onClick={()=>setLongPlot(false)}>60 segundos</button>
 <button aria-pressed={longPlot} onClick={()=>setLongPlot(true)}>30 minutos</button>
 </div><div className="diagnostic-charts">{(['rtt','jitter','bitrate','fps','interruptions'] as const).map((field,i)=><Graph key={field} points={longPlot?rendered.current.history:points} windowMs={longPlot?historyWindowMs:60_000} field={field} label={['RTT sonda (ms)','Jitter RTP (ms)','Voz TX+RX (kbps)','FPS renderizados','Transições de mídia'][i]}/>)}</div><p>Processo Go: CPU {n(last?.snapshot.cpuPercent,'% de um núcleo')} · RAM {n(last?.snapshot.rssBytes?last.snapshot.rssBytes/1048576:null,' MiB')}. Heap Go {n(last?.snapshot.heapBytes?last.snapshot.heapBytes/1048576:null,' MiB')}. CPU e RAM são do processo Go; utilização da GPU: n/d.</p><h3>Participantes · rede observada</h3>{last?.snapshot.peers.map(p=>{const history=points.flatMap(s=>s.snapshot.peers.filter(q=>q.alias===p.alias).map(q=>q.rttMs)).filter((v):v is number=>v!==null);const prev=points.at(-2)?.snapshot.peers.find(q=>q.alias===p.alias),seconds=points.at(-2)?(last.time-points.at(-2)!.time)/1000:0;const rate=(key:'bytesIn'|'bytesOut')=>seconds>0&&prev&&p.media[key]>=prev.media[key]?(p.media[key]-prev.media[key])*8/seconds/1000:null;return <section className="diagnostic-peer" key={p.alias}><b>Peer {p.alias}</b><p>{p.transport} · {p.path}</p><dl><dt>RTT atual / média / P95</dt><dd>{n(p.rttMs)} / {n(average(history))} / {n(percentile(history))} ms</dd><dt>Variação sucessiva de RTT (não jitter RTP)</dt><dd>{n(history.length>1?average(history.slice(1).map((r,i)=>Math.abs(r-history[i]))):null,' ms')}</dd><dt>Origem RTT</dt><dd>{p.rttSource}</dd><dt>Jitter RTP / perda RTP</dt><dd>{n(p.media.jitterMs,' ms')} / {n(p.media.packetsLost)} (n/d em stream TCP)</dd><dt>Upload / download de voz</dt><dd>{n(rate('bytesOut'))} / {n(rate('bytesIn'))} kbps</dd><dt>Último RX de mídia</dt><dd>{p.media.lastReceiveMs?n(Math.max(0,last.time-p.media.lastReceiveMs)/1000,' s atrás'):'nenhum nesta stream / n/d'}</dd><dt>Conexões / streams</dt><dd>{p.connections} / {p.streams}</dd><dt>Abertura / tentativas / falhas / timeout</dt><dd>{n(p.openMs,' ms')} / {p.openAttempts} / {p.openFailures} / {p.timeouts}</dd><dt>Resets locais / relatados / falhas transporte</dt><dd>{p.localResets??'n/d'} / {p.resetReports??'n/d'} / {p.transportErrors??'n/d'} · cumulativos do nó; relato não prova origem</dd><dt>Encerramentos EOF reportados</dt><dd>{p.eofReports??'n/d'} · não são automaticamente falhas</dd><dt>TX / RX rede / fila</dt><dd>{p.media.packetsOut} / {p.media.packetsIn} · {p.media.queueDepth??'n/d'} · {n(p.media.queueDelayMs,' ms')}</dd><dt>RX comprovado no outro computador</dt><dd>n/d · depende de relatório remoto</dd></dl></section>})}{!last?.snapshot.peers.length&&<p>Nenhum peer remoto na sala atual.</p>}<h3>Pipeline de voz</h3><p className="diagnostic-pipeline">Microfone → Processamento → Opus → Ponte local → Rede → Peer remoto → Decodificação → Reprodução</p><dl><dt>Microfone / captura</dt><dd>{audio?`${audio.available?'disponível':'ausente'} / ${audio.capture?'ativa':'interrompida'}`:'n/d'}</dd><dt>Processamento / modo / gate</dt><dd>{audio?.processing||'n/d'} / {audio?.mode||'n/d'} / {audio?.gate||'n/d'}</dd><dt>Mute / ensurdecer</dt><dd>{audio?`${audio.mute} / ${audio.deaf}`:'n/d'}</dd><dt>Blocos/amostras capturados no worklet</dt><dd>{n(audio?.captureBlocks)} / {n(audio?.captureSamples)} (blocos de processamento; não frames Opus de 20 ms)</dd><dt>Frames capturados de 20 ms / duração real</dt><dd>n/d / {n(audio?.captureSeconds,' s')} (não convertida artificialmente em frames)</dd><dt>Frames Opus / pacotes ponte local</dt><dd>{n(audio?.encodedFrames)} / {n(audio?.bridgePackets)}</dd><dt>Pacotes recebidos no backend local</dt><dd>{last?.snapshot.bridgePackets??'n/d'} (não comprova TX remoto)</dd><dt>Frames decodificados / amostras / ocultadas</dt><dd>{n(audio?.decodedFrames)} / {n(audio?.decodedSamples)} / {n(audio?.concealedSamples)}</dd><dt>Grafo de saída / reprodução</dt><dd>{n(audio?.outputSamples)} amostras · {audio?.outputContext||'n/d'} · {audio?audio.playbackPaused?'pausada':'elemento reproduzindo':'n/d'}</dd><dt>Espera média no buffer WebRTC / eventos de ocultação</dt><dd>{n(audio?.jitterBufferMeanMs,' ms')} / {n(audio?.concealmentEvents)} · ponte local, desde o início de cada trilha</dd><dt>Saída física / underrun</dt><dd>O WebView não informa estes dados.</dd></dl><h3>Transmissão de tela</h3><dl>
 <dt>Codec / encoder</dt><dd>{video?.codec??'n/d'} / {video?.encoder??'n/d'} · {efficient(video?.powerEfficientEncoder)}</dd>
 <dt>Decoder</dt><dd>{video?.decoder??'n/d'} · {efficient(video?.powerEfficientDecoder)}</dd>
 <dt>Resolução capturada / codificada / exibida</dt><dd>{dimensions(video?.captureWidth,video?.captureHeight)} / {dimensions(video?.encodeWidth,video?.encodeHeight)} / {dimensions(video?.renderWidth,video?.renderHeight)}</dd>
 <dt>FPS captura / encoder / envio / recepção / render</dt><dd>{[video?.capture,video?.encode,video?.sent,video?.received,video?.render].map(v=>n(v)).join(' / ')}</dd><dt>Limite / TX / RX</dt><dd>{n(video?video.configured/1e6:null)} / {n(video?.uploadMbps)} / {n(video?.downloadMbps)} Mbps</dd><dt>Descartes / encode / decode</dt><dd>{video?.dropped??'n/d'} / {n(video?.encodeMs,' ms')} / {n(video?.decodeMs,' ms')}</dd><dt>Fila de envio de tela</dt><dd>{video?.queue??'n/d'} / {video?.queueCapacity??'n/d'} quadros · {n(video?.queueWaitMs,' ms')} na última retirada (maior entre espectadores)</dd></dl><h3>Análise da conexão</h3>{last?evidence(last,points.at(-2)).map((e,i)=><p key={i}>{e}</p>):null}<small>RTT, FPS e filas podem ser correlacionados; não atribuem causalidade ao EasyTier/libp2p.</small><h3>Eventos observados</h3>{rendered.current.events.slice(-30).map((e,i)=><p className="diagnostic-event" key={i}>{new Date(e.time).toLocaleTimeString()} · {e.message}</p>)}<button className="secondary" onClick={()=>setCompare(!compare)}>Comparar conexões</button>{compare&&<section><p>Até 8 resumos nesta execução do app. Selecione duas sessões; compare somente condições equivalentes. Tailscale permanece no laboratório isolado.</p><label>Importar relatório ou resumo<input type="file" accept="application/json,.json" onChange={e=>{const file=e.target.files?.[0];if(!file)return;if(file.size>2*1024*1024){setError('Relatório excede 2 MiB.');return};void file.text().then(raw=>{const summary=importDiagnosticReport(JSON.parse(raw));if(!summary)throw Error('Relatório ou resumo incompatível.');archiveSession(summary);setSelection([]);setVersion(v=>v+1)}).catch(e=>setError(String(e)))}}/></label><label>Janela de observação<select value={duration} disabled={recording!==null} onChange={e=>setDuration(Number(e.target.value))}>{[10,30,60].map(n=><option key={n} value={n}>{n} s</option>)}</select></label><button disabled={recording!==null||!joined||!last} onClick={startObservation}>{recording?'Registrando janela…':'Registrar sessão sem carga'}</button><RoomProbeControls api={api} peers={last?.snapshot.peers ?? []} ready={joined&&last?.snapshot.authorization==='valid'} mode={last?.snapshot.mode??'desconhecido'} onResult={summary=>{sessionArchive.push(summary);sessionArchive.splice(0,Math.max(0,sessionArchive.length-8));setSelection([]);setVersion(v=>v+1)}}/>{sessionArchive.map((s,i)=><label key={i}><input type="checkbox" checked={selection.includes(i)} onChange={()=>setSelection(old=>old.includes(i)?old.filter(n=>n!==i):[...old.slice(-1),i])}/>{new Date(s.at).toLocaleTimeString()} · {s.mode} · {s.samples} amostras{s.testType ? ' · carga limitada' : ''}</label>)}{selectedSummaries.length===2&&<>
 <p>{comparable?'Confirme se participantes e condições de rede foram equivalentes.':'As janelas usam métodos ou durações diferentes. Use o mesmo teste e duração para comparar desempenho.'}</p>
 <div className="diagnostic-comparison">{selectedSummaries.map((summary,index)=><pre key={index}>{JSON.stringify(summary,null,2)}</pre>)}</div>
 </>}</section>}<small>Coletas dos estágios são assíncronas. Histórico local: até 30 minutos com uma amostra por faixa de cinco segundos. A sonda RTT funciona com o painel visível. Até 120 eventos e oito resumos em RAM. Sem envio automático.</small></div></aside>
}
