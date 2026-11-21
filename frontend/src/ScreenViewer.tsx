import React, {useLayoutEffect, useRef, useState, type ReactNode} from 'react';
import {createPortal} from 'react-dom';
import {Fullscreen, Maximize2, Minimize2, Monitor, PanelsTopLeft, X} from 'lucide-react';

import {IdentityAvatar,IdentityName} from './ProfileIdentity';
import type {ScreenQuality} from './callPresentation';

export function ScreenViewerOverlay({name, peerId, connected, width, height, fps, mbps, children}: ScreenQuality & {
 name: string;
 peerId?: string;
 connected: boolean;
 mbps?: number|null;
 children: ReactNode;
}) {
 const dimensions = width && height ? `${width}×${height}` : 'Resolução n/d';
 return <>
  <div className="screen-viewer-heading">
   <span className="screen-broadcaster" title={name}><IdentityAvatar id={peerId||''} name={name} className="stream-avatar"/><strong><IdentityName id={peerId||''} name={name}/></strong></span>
   <span className="screen-viewer-quality" title={`${dimensions} · ${fps?.toFixed(1) ?? 'n/d'} FPS · ${mbps?.toFixed(2) ?? 'n/d'} Mbps`}>
    {dimensions} · {fps?.toFixed(1) ?? 'n/d'} FPS
   </span>
   <span className="live-badge">AO VIVO</span>
  </div>
  {!connected && <div className="screen-buffering" role="status"><i aria-hidden="true"/>Reconectando transmissão…</div>}
  <div className="screen-overlay" role="group" aria-label="Controles da transmissão">{children}</div>
 </>;
}

// Moving a single portal host keeps the video and audio controls mounted when
// the user leaves the room page or opens it again.
export function ScreenStagePortal({target, children}: {target?: HTMLElement|null; children: ReactNode}) {
 const dock = useRef<HTMLDivElement>(null);
 const [host] = useState(() => {
  const element = document.createElement('div');
  element.className = 'screen-share-host';
  return element;
 });
 useLayoutEffect(() => {
  const destination = target ?? dock.current;
  if (!destination || host.parentElement === destination) return;
  const movable = destination as HTMLElement & {moveBefore?: (node: Node, before: Node|null) => void};
  if (host.isConnected && destination.isConnected && movable.moveBefore) {
   movable.moveBefore(host, null);
  } else {
   destination.append(host);
   // Older WebView2 versions can interrupt playback when a running element is moved.
   const video = host.querySelector('video');
   if (video?.srcObject) void video.play().catch(() => {});
  }
 }, [host, target]);
 useLayoutEffect(() => () => host.remove(), [host]);
 return <>
  <div className="screen-share-dock" ref={dock} hidden={!!target}/>
  {createPortal(children, host)}
 </>;
}

export function ScreenViewerModeControls({inRoom, theater, compact, fullscreen, onFocus, onFullscreen}: {
 inRoom: boolean;
 theater: boolean;
 compact: boolean;
 fullscreen: boolean;
 onFocus: () => void;
 onFullscreen: () => void;
}) {
 const focusLabel = inRoom ? theater ? 'Sair do modo teatro' : 'Modo teatro' : compact ? 'Ampliar transmissão' : 'Modo compacto';
 const fullLabel = fullscreen ? 'Sair da tela cheia' : 'Tela cheia';
 return <>
  <button type="button" className="screen-control-icon" title={focusLabel} aria-label={focusLabel} aria-pressed={inRoom ? theater : !compact} onClick={onFocus}>
   {inRoom ? <PanelsTopLeft size={17}/> : compact ? <Maximize2 size={17}/> : <Minimize2 size={17}/>}
  </button>
  <button type="button" className="screen-control-icon" title={fullLabel} aria-label={fullLabel} aria-pressed={fullscreen} onClick={onFullscreen}>
   {fullscreen ? <Minimize2 size={17}/> : <Fullscreen size={17}/>}
  </button>
 </>;
}

export type StreamOption = {id: string; name: string; sharing: boolean; connected: boolean};
export function ScreenShareDirectory({streams, selected, watching, ready, onClose, onWatch}: {
 streams: StreamOption[];
 selected: string;
 watching: string;
 ready: boolean;
 onClose: () => void;
 onWatch: (id: string) => void;
}) {
 const dialog = useRef<HTMLDialogElement>(null);
 useLayoutEffect(() => {
  const element = dialog.current;
  if (!element) return;
  element.showModal();
  return () => element.close();
 }, []);
 const chosen = streams.find(stream => stream.id === selected);
 const ordered = chosen ? [chosen, ...streams.filter(stream => stream !== chosen)] : streams;
 return <dialog className="screen-directory" ref={dialog} aria-label="Telas dos participantes" onCancel={e => { e.preventDefault(); onClose(); }} onClick={e => { if (e.target === e.currentTarget) onClose(); }}>
  <div className="screen-directory-content">
   <div className="screen-directory-heading"><h2>Telas dos participantes</h2><button type="button" aria-label="Fechar telas dos participantes" className="screen-control-icon" onClick={onClose}><X size={18}/></button></div>
   {ordered.length === 0 ? <p>Nenhuma transmissão de outro participante está disponível agora.</p> : <ul>
    {ordered.map(stream => <li key={stream.id} className={stream.id === selected ? 'is-selected' : undefined}>
     <span className="screen-directory-icon"><Monitor size={20}/></span>
     <span className="screen-directory-name"><strong title={stream.name}><IdentityName id={stream.id} name={stream.name||'Participante'}/></strong><span className="live-badge">AO VIVO</span></span>
     {watching === stream.id ? <button type="button" className="secondary" onClick={onClose}>Voltar à tela</button> :
      <button type="button" className="primary" disabled={!ready} onClick={() => onWatch(stream.id)}>Assistir</button>}
    </li>)}
   </ul>}
  </div>
 </dialog>;
}

export function ScreenShareEmpty({ready, busy, remoteStreams, onShare, onBrowse}: {
 ready: boolean;
 busy: boolean;
 remoteStreams: number;
 onShare: () => void;
 onBrowse: () => void;
}) {
 return <section className="screen-empty" aria-label="Área de transmissão">
  <span className="screen-empty-icon"><Monitor size={31} strokeWidth={1.55}/></span>
  <h3>{remoteStreams ? 'Há telas ao vivo na sala' : 'Nenhuma transmissão ativa'}</h3>
  <div className="screen-empty-actions">
   {remoteStreams > 0 && <button type="button" className="primary" onClick={onBrowse}><PanelsTopLeft size={16}/> Telas dos participantes</button>}
   <button type="button" className={remoteStreams ? 'secondary' : 'primary'} disabled={!ready || busy} onClick={onShare}><Monitor size={16}/> Compartilhar tela</button>
  </div>
 </section>;
}
