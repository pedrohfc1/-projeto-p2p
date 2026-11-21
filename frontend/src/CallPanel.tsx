import React from 'react';
import {Activity, HeadphoneOff, Headphones, Mic, MicOff, Monitor, MonitorPlay, PhoneOff, Radio, SlidersHorizontal, Terminal, Video} from 'lucide-react';
import {callDuration, callStatus} from './callPresentation';

export type CallActions = {
 muted: boolean;
 deaf: boolean;
 sharing: boolean;
 screenReady: boolean;
 screenBusy: boolean;
 onMute: () => void;
 onDeaf: () => void;
 onScreen: () => void;
 onLeave: () => void;
 onStreams?: () => void;
 streamsOpen?: boolean;
 onLogs?: () => void;
 onSettings?: () => void;
 onDiagnostics?: () => void;
};

export type CallPanelProps = CallActions & {
 channel: string;
 community: string;
 participants: number;
 seconds: number;
 mediaState: string;
 packetsIn?: number;
 packetsOut?: number;
 bridgePackets?: number;
 condensed?: boolean;
};

export function CallControlsBar(p: CallActions & {floating?: boolean}) {
 const micLabel = p.muted ? 'Ativar microfone' : 'Silenciar microfone';
 const audioLabel = p.deaf ? 'Reativar áudio' : 'Ensurdecer áudio';
 const screenLabel = p.sharing ? 'Parar compartilhamento' : 'Compartilhar tela';

 return <div className={'call-actions'+(p.floating ? ' voice-stage-controls' : '')} role="group" aria-label="Ações da chamada">
  <button type="button" className={'call-action '+(p.muted||p.deaf?'muted':'')} aria-label={micLabel} title={p.deaf?'Microfone silenciado enquanto o áudio está ensurdecido':micLabel} aria-pressed={p.muted||p.deaf} onClick={p.onMute}>{p.muted||p.deaf?<MicOff size={19}/>:<Mic size={19}/>}</button>
  <button type="button" className={'call-action '+(p.deaf?'muted':'')} aria-label={audioLabel} title={audioLabel} aria-pressed={p.deaf} onClick={p.onDeaf}>{p.deaf?<HeadphoneOff size={19}/>:<Headphones size={19}/>}</button>
  <span className="future-camera" title="Câmera em breve"><button type="button" className="call-action" aria-label="Câmera em breve" title="Câmera em breve" disabled><Video size={18}/></button></span>
  <button type="button" className={'call-action call-share '+(p.sharing?'sharing':'')} aria-label={screenLabel} title={screenLabel} aria-pressed={p.sharing} disabled={!p.sharing&&(!p.screenReady||p.screenBusy)} onClick={p.onScreen}><Monitor size={19}/></button>
  {p.onStreams&&<button type="button" className="call-action" title="Telas dos participantes" aria-label="Telas dos participantes" aria-expanded={!!p.streamsOpen} onClick={p.onStreams}><MonitorPlay size={19}/></button>}
  {p.onLogs&&<button type="button" className="call-action" title="Abrir logs da chamada" aria-label="Abrir logs da chamada" onClick={p.onLogs}><Terminal size={19}/></button>}
  {p.onSettings&&<button type="button" className="call-action" title="Configurações rápidas de voz e vídeo" aria-label="Configurações rápidas de voz e vídeo" onClick={p.onSettings}><SlidersHorizontal size={19}/></button>}
  {p.onDiagnostics&&<button type="button" className="call-action" title="Diagnóstico em tempo real" aria-label="Diagnóstico" onClick={p.onDiagnostics}><Activity size={18}/></button>}
  <button type="button" className="call-action disconnect" aria-label="Desconectar chamada" title="Desconectar da chamada" onClick={p.onLeave}><PhoneOff size={19}/></button>
 </div>;
}

export function CallPanel(p: CallPanelProps) {
 const status = callStatus(p.mediaState, p.participants, p.packetsOut, p.packetsIn);
 return <section className={'call-panel'+(p.condensed ? ' call-panel-condensed' : '')} aria-label="Painel da chamada de voz">
  <div className={'call-connection '+(status==='Voz conectada'?'connected':'reconnecting')} role="status"><Radio size={16}/><b title={status}>{status}</b><time title="Duração desta sessão no aplicativo">{callDuration(p.seconds)}</time></div>
  <div className="call-destination"><b title={p.channel}>{p.channel}</b>{!p.condensed&&<small title={p.community}>{p.community}</small>}</div>
  {!p.condensed&&<>
   <div className="call-meta"><span>{p.participants} participante(s)</span><span>{p.muted||p.deaf?'Microfone silenciado':'Microfone habilitado'} · {p.deaf?'Saída silenciada':'Saída ativa'}</span></div>
   <details className="call-technical"><summary>Detalhes técnicos</summary><small className="call-wire-metrics">Ponte local {p.bridgePackets??0} · Rede TX {p.packetsOut??0} / RX {p.packetsIn??0}</small></details>
   <CallControlsBar {...p}/>
  </>}
 </section>;
}
