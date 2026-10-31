export const vpnName=(mode:string)=>mode==='hamachi'?'Hamachi':mode==='radmin'?'Radmin VPN':mode==='interface'?'Interface manual':'Automatico (libp2p)';
export const messageByteLength=(text:string)=>new TextEncoder().encode(text.trim()).length;
export const canSendMessage=(text:string,peer:string,busy:boolean)=>Boolean(peer)&&!busy&&text.trim().length>0&&messageByteLength(text)<=4096;
