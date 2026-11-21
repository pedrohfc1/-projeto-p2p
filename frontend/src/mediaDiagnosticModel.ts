export type Pipeline={jitterBufferMeanMs?:number|null;concealmentEvents?:number|null;captureBlocks?:number|null;captureSamples?:number|null;at:number;available:boolean;capture:boolean;processing:string;mode:string;mute:boolean;deaf:boolean;captureSeconds:number|null;encodedFrames:number|null;bridgePackets:number|null;decodedFrames:number|null;decodedSamples:number|null;concealedSamples:number|null;outputSamples:number|null;outputContext:string;playbackPaused:boolean;gate:string};
export type VideoSample={
 codec?: string|null;encoder?: string|null;decoder?: string|null;
 powerEfficientEncoder?:boolean|null;powerEfficientDecoder?:boolean|null;
 captureWidth?:number|null;captureHeight?:number|null;
 encodeWidth?:number|null;encodeHeight?:number|null;
 renderWidth?:number|null;renderHeight?:number|null;
 recovery?:string|null;recoveryAt?:number|null;at:number;capture:number|null;encode:number|null;sent:number|null;received:number|null;decode:number|null;render:number|null;uploadMbps:number|null;downloadMbps:number|null;dropped:number;configured:number;queue:number|null;queueCapacity?:number|null;queueWaitMs?:number|null;encodeMs:number|null;decodeMs:number|null};
export const liveTelemetry:{audio:Pipeline|null;video:VideoSample|null}={audio:null,video:null};
export function numberMetric(value:unknown):number|null{return typeof value==='number'&&Number.isFinite(value)&&value>=0?value:null}
export function audioSample(rows:Record<string,unknown>[],extra:Omit<Pipeline,'captureSeconds'|'encodedFrames'|'bridgePackets'|'decodedFrames'|'decodedSamples'|'concealedSamples'|'jitterBufferMeanMs'|'concealmentEvents'>):Pipeline{
 const sum=(type:string,key:string)=>{const values=rows.filter(r=>r.type===type&&r.kind==='audio').map(r=>numberMetric(r[key])).filter((n):n is number=>n!==null);return values.length?values.reduce((a,b)=>a+b,0):null};
 const buffers = rows.filter(row => row.type === 'inbound-rtp' && row.kind === 'audio')
  .map(row => ({seconds: numberMetric(row.jitterBufferDelay), count: numberMetric(row.jitterBufferEmittedCount)}))
  .filter((row): row is {seconds: number; count: number} => row.seconds !== null && row.count !== null && row.count > 0);
 const bufferedSamples = buffers.reduce((total, row) => total + row.count, 0);
 const jitterBufferMeanMs = bufferedSamples > 0
  ? buffers.reduce((total, row) => total + row.seconds, 0) * 1000 / bufferedSamples
  : null;
 return {...extra,jitterBufferMeanMs,concealmentEvents:sum('inbound-rtp','concealmentEvents'),captureSeconds:sum('media-source','totalSamplesDuration'),encodedFrames:sum('outbound-rtp','framesEncoded'),bridgePackets:sum('outbound-rtp','packetsSent'),decodedFrames:sum('inbound-rtp','framesDecoded'),decodedSamples:sum('inbound-rtp','totalSamplesReceived'),concealedSamples:sum('inbound-rtp','concealedSamples')};
}
export type PeerSnapshot={alias:string;connections:number;streams:number;path:string;transport:string;rttMs:number|null;rttSource:string;openMs:number|null;eofReports?:number;localResets?:number;resetReports?:number;transportErrors?:number;openAttempts:number;openFailures:number;timeouts:number;media:{state:string;transport:string;packetsIn:number;packetsOut:number;bytesIn:number;bytesOut:number;lastReceiveMs?:number;queueDepth?:number;queueDelayMs:number;jitterMs:number|null;packetsLost:number|null}};
export type Snapshot={heapBytes?:number;cpuPercent?:number|null;rssBytes?:number|null;time:number;session:string;mode:string;authorization:string;expires:number;joined:boolean;bridgePackets:number;peers:PeerSnapshot[]};
export type Point={time:number;rtt:number|null;jitter:number|null;bitrate:number|null;fps:number|null;interruptions:number;audio:Pipeline|null;video:VideoSample|null;snapshot:Snapshot};
export const percentile=(values:number[],p=.95)=>{const sorted=values.filter(n=>Number.isFinite(n)&&n>=0).sort((a,b)=>a-b);return sorted.length?sorted[Math.max(0,Math.ceil(sorted.length*p)-1)]:null};
export const average=(values:number[])=>values.length?values.reduce((a,b)=>a+b,0)/values.length:null;
export function evidence(point:Point,previous?:Point){const out:string[]=[];if(point.snapshot.authorization!=='valid')out.push('Comprovado: autorização da sala não está válida; mídia deve permanecer bloqueada.');if(point.rtt!==null&&point.rtt>200)out.push('Observação: RTT da sonda acima de 200 ms; não identifica sozinho a causa.');if(point.snapshot.peers.some(p=>p.path.includes('relay')))out.push('Confirmado: caminho libp2p limitado/relay; underlay pode ser diferente.');if(point.audio?.playbackPaused&&point.snapshot.peers.some(p=>p.media.packetsIn>0))out.push('Comprovado: reprodução local pausada apesar de pacotes recebidos nesta sessão.');if(previous&&point.audio?.capture&&!point.audio.mute&&!point.audio.deaf&&point.audio.gate!=='closed'&&point.snapshot.bridgePackets>previous.snapshot.bridgePackets&&point.snapshot.peers.every((p,i)=>p.media.packetsOut<=(previous.snapshot.peers.find(old=>old.alias===p.alias)?.media.packetsOut??0)))out.push('Evidência: ponte local avançou sem avanço de TX observado. Hipótese: autorização, sessão ou stream; silêncio/DTX não é perda comprovada.');return out}
export type SessionSummary = {
 at: number;
 mode: string;
 seconds: number;
 samples: number;
 rttMean: number | null;
 rttP95: number | null;
 jitter: number | null;
 kbps: number | null;
 fps: number | null;
 interruptions: number | null;
 noRxSeconds: number | null;
 conditions: string;
 testType?: 'limited-load';
 variationMs?: number | null;
 videoTxMbps?: number | null;
 videoRxMbps?: number | null;
 videoSendFps?: number | null;
 videoRenderFps?: number | null;
};
type Event = {time: number; message: string};

export type HistoryPoint = Pick<Point, 'time'|'rtt'|'jitter'|'bitrate'|'fps'|'interruptions'> & {
 cpuPercent: number|null;
 rssBytes: number|null;
 heapBytes: number|null;
 voiceRx: number;
 voiceTx: number;
 resets: number;
 reconnectAttempts: number;
 authorization: string;
 video: VideoSample|null;
 audioPlaybackPaused: boolean|null;
};

export const historyWindowMs = 30 * 60_000;
export const historyIntervalMs = 5_000;
const detailWindowMs = 60_000;
const maxHistoryPoints = Math.ceil(historyWindowMs/historyIntervalMs);

export function archiveSession(summary: SessionSummary) {
 const previous = sessionArchive.at(-1);
 if (previous?.at === summary.at && previous.mode === summary.mode &&
     previous.samples === summary.samples && previous.testType === summary.testType) return;
 sessionArchive.push(summary);
 while (sessionArchive.length > 8) sessionArchive.shift();
}

export class SessionSeries {
 points: Point[] = [];
 events: Event[] = [];
 history: HistoryPoint[] = [];
 error = '';
 private listeners = new Set<() => void>();
 private interrupted = 0;
 private lastRxAt = 0;
 private lastRx = 0;
 private archived = false;
 private rxByPeer = new Map<string, number>();

 subscribe = (listener: () => void) => {
  this.listeners.add(listener);
  return () => { this.listeners.delete(listener); };
 };

 setError(error: string) {
  if (this.error === error) return;
  this.error = error;
  this.notify();
 }

 finish() {
  if (this.archived || this.points.length === 0) return;
  archiveSession(this.summary());
  this.archived = true;
 }

 append(snapshot: Snapshot, audio: Pipeline|null, video: VideoSample|null) {
  const prior = this.points.at(-1);
  if (prior && snapshot.time <= prior.time) return prior;
  if (prior && prior.snapshot.session !== snapshot.session) {
   this.finish();
   this.points = [];
   this.events = [];
   this.history = [];
   this.interrupted = 0;
   this.lastRxAt = 0;
   this.lastRx = 0;
   this.archived = false;
   this.rxByPeer.clear();
  }
  const before = this.points.at(-1);
  const elapsed = before ? (snapshot.time - before.time)/1000 : 0;
  const peers = snapshot.peers;
  const audioNow = audio && Math.abs(snapshot.time-audio.at) < historyIntervalMs ? audio : null;
  const videoNow = video && Math.abs(snapshot.time-video.at) < historyIntervalMs ? video : null;
  const totalRx = peers.reduce((count, peer) => count + peer.media.packetsIn, 0);
  if (!this.lastRxAt || totalRx > this.lastRx) this.lastRxAt = snapshot.time;
  for (const peer of peers) {
   const lastRx = this.rxByPeer.get(peer.alias) ?? 0;
   if (peer.media.packetsIn > lastRx) this.lastRxAt = snapshot.time;
  }
  this.rxByPeer = new Map(peers.map(peer => [peer.alias, peer.media.packetsIn]));
  this.lastRx = totalRx;

  const event = (message: string) => this.events.push({time:snapshot.time,message});
  if (before) {
   const previous = before.snapshot;
   if (previous.authorization !== snapshot.authorization) event('Autorização: '+snapshot.authorization);
   if (previous.expires !== snapshot.expires && snapshot.authorization === 'valid') event('Renovação da autorização observada');
   for (const peer of peers) {
    const old = previous.peers.find(candidate => candidate.alias === peer.alias);
    if (!old) continue;
    if (old.path !== peer.path) event('Caminho alterado · '+peer.path);
    if (old.media.state !== peer.media.state) {
     event('Mídia: '+peer.media.state);
     if (peer.media.state !== 'connected') this.interrupted++;
    }
    if ((peer.localResets ?? 0) > (old.localResets ?? 0) || (peer.resetReports ?? 0) > (old.resetReports ?? 0)) event('Reset de stream observado');
    if (peer.openAttempts > old.openAttempts) event('Tentativa de abertura de stream');
    if (peer.openFailures > old.openFailures) event('Falha ao abrir stream');
   }
   if (before.audio && audioNow && before.audio.playbackPaused !== audioNow.playbackPaused) {
    event(audioNow.playbackPaused ? 'Reprodução local pausada' : 'Reprodução local retomada');
   }
   if (videoNow?.render === 0 && (before.video?.render ?? 0) > 0) {
    event('Nenhum frame renderizado no intervalo observado');
   }
  } else {
   event('Coleta iniciada · '+snapshot.mode);
  }
  if (videoNow?.recovery && videoNow.recoveryAt !== before?.video?.recoveryAt) {
   event(videoNow.recovery);
  }
  this.events = this.events.filter(item => snapshot.time-item.time < historyWindowMs).slice(-120);

  // Avoid treating a new stream, departure, or counter reset as throughput.
  let bytes = 0;
  let comparable = !!before && before.snapshot.peers.length === peers.length && peers.length > 0;
  if (before && comparable) {
   for (const peer of peers) {
    const old = before.snapshot.peers.find(candidate => candidate.alias === peer.alias);
    if (!old || peer.media.bytesIn < old.media.bytesIn || peer.media.bytesOut < old.media.bytesOut) {
     comparable = false;
     break;
    }
    bytes += peer.media.bytesIn-old.media.bytesIn + peer.media.bytesOut-old.media.bytesOut;
   }
  }
  const point: Point = {
   time: snapshot.time,
   rtt: average(peers.flatMap(peer => numberMetric(peer.rttMs) ?? [])),
   jitter: average(peers.flatMap(peer => numberMetric(peer.media.jitterMs) ?? [])),
   bitrate: comparable && elapsed > 0 ? bytes*8/elapsed/1000 : null,
   fps: videoNow?.render ?? null,
   interruptions: this.interrupted,
   audio: audioNow,
   video: videoNow,
   snapshot,
  };
  this.points.push(point);
  while (this.points.length > 61 || snapshot.time-this.points[0].time > detailWindowMs) this.points.shift();
  this.addHistory(point);
  this.notify();
  return point;
 }

 private addHistory(point: Point) {
  const {time, rtt, jitter, bitrate, fps, interruptions, snapshot, video, audio} = point;
  const limit = Math.floor(time/historyIntervalMs);
  const summary: HistoryPoint = {
   time, rtt, jitter, bitrate, fps, interruptions, video,
   cpuPercent:snapshot.cpuPercent ?? null, rssBytes:snapshot.rssBytes ?? null,
   heapBytes:snapshot.heapBytes ?? null, authorization:snapshot.authorization,
   voiceRx:snapshot.peers.reduce((total, peer) => total+peer.media.packetsIn, 0),
   voiceTx:snapshot.peers.reduce((total, peer) => total+peer.media.packetsOut, 0),
   reconnectAttempts:snapshot.peers.reduce((total, peer) => total+peer.openAttempts, 0),
   resets:snapshot.peers.reduce((total, peer) => total+(peer.localResets ?? 0), 0),
   audioPlaybackPaused:audio?.playbackPaused ?? null,
  };
  const last=this.history.at(-1);
  if (last && Math.floor(last.time/historyIntervalMs) === limit) {
   // A passive snapshot carries no probe result. Keep a measured RTT in its own bucket.
   summary.rtt ??= last.rtt;
   this.history[this.history.length-1] = summary;
  } else {
   this.history.push(summary);
  }
  while (this.history.length > maxHistoryPoints || time-this.history[0].time > historyWindowMs) this.history.shift();
 }

 summary(since?: number): SessionSummary {
  const points = since === undefined ? this.points : this.points.filter(point => point.time >= since);
  return this.summarize(points, 'janela observada; carga/peers não controlados; sem RX também pode ser silêncio/DTX');
 }

 private summarize(points: Pick<Point,'time'|'rtt'|'jitter'|'bitrate'|'fps'|'interruptions'|'video'>[], conditions: string): SessionSummary {
  const last = points.at(-1);
  const values = (field: 'rtt'|'jitter'|'bitrate'|'fps') => points.flatMap(point => point[field] ?? []);
  const videoAverage = (field: 'uploadMbps'|'downloadMbps'|'sent'|'render') =>
   average(points.flatMap(point => point.video?.[field] ?? []));
  return {
   at:last?.time ?? Date.now(), mode:this.points.at(-1)?.snapshot.mode || 'desconhecido',
   seconds:last ? (last.time-points[0].time)/1000 : 0, samples:points.length,
   rttMean:average(values('rtt')), rttP95:percentile(values('rtt')), jitter:average(values('jitter')),
   kbps:average(values('bitrate')), fps:average(values('fps')),
   videoTxMbps: videoAverage('uploadMbps'), videoRxMbps: videoAverage('downloadMbps'),
   videoSendFps: videoAverage('sent'), videoRenderFps: videoAverage('render'),
   interruptions:points.length > 1 ? last!.interruptions-points[0].interruptions : 0,
   noRxSeconds:this.lastRxAt && last ? Math.max(0,(last.time-this.lastRxAt)/1000) : null,
   conditions
  };
 }

 report() {
  return {
   version: 2,
   scope:'Diagnóstico local. RTT de sonda libp2p; jitter RTP somente quando mensurado. IDs pseudonimizados.',
   summary:this.summary(),
   points:this.points,
   history30Minutes:{
    intervalMs:historyIntervalMs, points:this.history,
    summary:this.summarize(this.history, 'Uma amostra por faixa de cinco segundos. RTT só quando o painel ficou visível.')
   },
   events:this.events,
   sessionHistory:[...sessionArchive],
  };
 }

 private notify() {
  this.listeners.forEach(listener => listener());
 }
}
export const sessionArchive:SessionSummary[]=[];

export function importLabSummary(value:unknown):ReturnType<SessionSeries['summary']>|null{
 if(!value||typeof value!=='object')return null;const v=value as Record<string,unknown>;
 const num=(key:string)=>typeof v[key]==='number'&&Number.isFinite(v[key])&&(v[key] as number)>=0&&(v[key] as number)<=1e9?v[key] as number:null;
 if(v.mode!=='tailscale-lab'||typeof v.at!=='number'||!Number.isFinite(v.at)||v.at<0||v.at>Date.now()+3600000||num('seconds')===null||num('seconds')!>300||num('samples')===null||num('samples')!>600)return null;
 if (v.testType === 'limited-load') {
  const seconds = num('seconds'), samples = num('samples'), kbps = num('kbps');
  if (seconds === null || seconds < 9.5 || seconds > 65 || samples === null
      || !Number.isInteger(samples) || samples < 1 || samples > 240
      || kbps === null || kbps > 128 || num('rttMean') === null || num('rttP95') === null) return null;
  return {
   at: v.at, mode: 'tailscale-lab', seconds, samples, rttMean: num('rttMean'),
   rttP95: num('rttP95'), jitter: null, kbps, fps: null, interruptions: null,
   noRxSeconds: null, variationMs: num('variationMs'), testType: 'limited-load',
   conditions: 'Eco de 4 kB a cada 250 ms pela tailnet do laboratório. Vazão de payload por direção e RTT do eco sob carga limitada.'
  };
 }
 if (v.testType !== undefined) return null;
 return {at:v.at,mode:'tailscale-lab',seconds:num('seconds')!,samples:num('samples')!,rttMean:num('rttMean'),rttP95:num('rttP95'),jitter:null,kbps:null,fps:null,interruptions:num('interruptions')??0,noRxSeconds:null,conditions:'laboratório tsnet; sonda libp2p 32 bytes/1Hz; sem teste de mídia ou carga. Comparar somente condições equivalentes.'};
}

// Read only the numeric session summary. Do not retain another report's
// snapshots, identifiers, text, or file path in the local comparison archive.
export function importDiagnosticReport(value: unknown): SessionSummary|null {
 const lab = importLabSummary(value);
 if (lab) return lab;
 if (!value || typeof value !== 'object') return null;
 const report = value as {version?:number; summary?:unknown; history30Minutes?:{summary?:unknown}};
 if (report.version !== 1 && report.version !== 2) return null;
 const candidate = report.history30Minutes?.summary ?? report.summary;
 if (!candidate || typeof candidate !== 'object') return null;
 const summary = candidate as Record<string, unknown>;
 const mode = summary.mode;
 if (typeof mode !== 'string' || !['integrated','radmin','hamachi','auto','interface'].includes(mode)) return null;
 const metric = (name: string, max=1e9) => {
  const value = numberMetric(summary[name]);
  return value !== null && value <= max ? value : null;
 };
 const at = metric('at', Date.now()+60*60_000);
 const seconds = metric('seconds', 1800);
 const samples = metric('samples', 360);
 if (at === null || seconds === null || samples === null || samples < 1 || !Number.isInteger(samples)) return null;
 return {
  at,mode,seconds,samples,rttMean:metric('rttMean'),rttP95:metric('rttP95'),jitter:metric('jitter'),
  kbps:metric('kbps'),fps:metric('fps'),interruptions:metric('interruptions'),noRxSeconds:metric('noRxSeconds'),
  videoTxMbps:metric('videoTxMbps'),videoRxMbps:metric('videoRxMbps'),
  videoSendFps:metric('videoSendFps'),videoRenderFps:metric('videoRenderFps'),
  conditions:'Resumo importado de um diagnóstico PeeTwoPee. Compare a mesma fonte, qualidade e duração.'
 };
}
