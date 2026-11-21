import React from 'react';
// Presentation only. The persistent video and capture/transport remain in ScreenPanel.
export function ScreenViewerOverlay({name,connected,width,height,fps,mbps,children}:{name:string;connected:boolean;width?:number|null;height?:number|null;fps?:number|null;mbps?:number|null;children:React.ReactNode}){
 return <div className="screen-overlay"><span className="stream-avatar" aria-hidden="true">{name.slice(0,1).toUpperCase()||'?'}</span><span><strong>{name}</strong> <span className="live-badge">AO VIVO</span><small role="status">{connected?'Conexão autorizada':'Reconectando transmissão…'}</small><small>{width&&height?`${width}×${height}`:'Resolução n/d'} · {fps?.toFixed(1)??'n/d'} FPS render · ↓ {mbps?.toFixed(2)??'n/d'} Mbps</small></span>{children}</div>
}
