export type DiagnosticEvent={id:number;time:number;level:'INFO'|'WARN'|'ERROR';category:'voice'|'video'|'network';component:string;message:string};
export type DiagnosticFilter='all'|'voice'|'video'|'network'|'authorization'|'errors';
export function redact(value:unknown):string {
 let s=String(value).slice(0,4096);
 s=s.replace(/(?:peetwopee|projetop2p):\/\/\S+|Bearer\s+\S+|(?:token|secret|password|private.?key|invite|sdp|credential)["']?\s*[:=]\s*["']?[^\s,;]+|(?:12D3KooW|Qm)[A-Za-z0-9]+|[A-Za-z0-9_+/=-]{40,}/gi,'[oculto]');
 return s.replace(/(?:\d{1,3}\.){3}\d{1,3}|\[[0-9a-f:]+\]/gi,'[endereço]').replace(/[\r\n\x00-\x08\x0b-\x1f]/g,' ').slice(0,768);
}
export class DiagnosticBuffer {
 revision=0;private rows:DiagnosticEvent[]=[];private head=0;private sequence=0;
 add(event:Omit<DiagnosticEvent,'id'>){this.revision++;const row={...event,id:++this.sequence,component:redact(event.component),message:redact(event.message)};if(this.rows.length<2000)this.rows.push(row);else{this.rows[this.head]=row;this.head=(this.head+1)%2000}return row}
 snapshot(){return this.rows.slice(this.head).concat(this.rows.slice(0,this.head))}
 clear(){this.revision++;this.rows=[];this.head=0}
}
export const diagnosticBuffer=new DiagnosticBuffer();
export function diagnostic(level:DiagnosticEvent['level'],category:DiagnosticEvent['category'],component:string,message:unknown){try{diagnosticBuffer.add({time:Date.now(),level,category,component,message:redact(message)})}catch{/* Diagnostics must never interrupt media. */}}
export function filtered(rows:DiagnosticEvent[],filter:DiagnosticFilter) {
 if (filter==='all') return rows;
 if (filter==='errors') return rows.filter(row=>row.level==='ERROR');
 if (filter==='authorization') return rows.filter(row=>row.category==='voice'&&/(voice_authorization|autoriza[cç][aã]o)/i.test(row.component+' '+row.message));
 return rows.filter(row=>row.category===filter);
}
export function formatEvent(r:DiagnosticEvent){const d=new Date(r.time);return `${d.toLocaleTimeString('pt-BR',{hour12:false})}.${String(d.getMilliseconds()).padStart(3,'0')} ${r.level.padEnd(5)} [${r.component}] ${r.message}`}
// Existing structured codec diagnostics, with an explicit field allowlist.
let installed=false;
export function captureMediaConsole(){if(installed)return;installed=true;for(const method of ['info','warn','error'] as const){const original=console[method].bind(console);console[method]=(...args:unknown[])=>{original(...args);const name=args[0];if(typeof name!=='string'||!/^voice_[a-z_]+$/.test(name))return;const fields=args[1];const allowed=new Set(['captureSeconds','capturedFrames','codec','direction','packets','bytes','decodedSamples','concealedSamples','playbackPaused','slot','state','captureReused','reason']);const values=fields&&typeof fields==='object'?Object.entries(fields).filter(([k])=>allowed.has(k)).map(([k,v])=>`${k}=${Array.isArray(v)?v.map(String).join(','):typeof v==='object'?'n/d':String(v)}`).join(' '):'';diagnostic(method==='error'?'ERROR':method==='warn'?'WARN':'INFO','voice','frontend/Opus',name+' '+values)}}}

// Called only after the user's copy confirmation. Browser clipboard keeps the
// user gesture; native IPC remains the fallback when WebView lacks the API.
export async function copyDiagnosticText(text:string,nativeWrite:(text:string)=>Promise<void>,browserWrite?:((text:string)=>Promise<void>)){
 if(browserWrite){try{await browserWrite(text);return}catch{/* Native fallback below. */}}
 await nativeWrite(text);
}
