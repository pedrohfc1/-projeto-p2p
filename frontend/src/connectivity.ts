export const vpnName=(mode:string)=>mode==='integrated'?'Rede P2P integrada':mode==='hamachi'?'Hamachi':mode==='radmin'?'Radmin VPN':mode==='interface'?'Interface manual':'Automatico (libp2p)';
export const messageByteLength=(text:string)=>new TextEncoder().encode(text.trim()).length;
export const canSendMessage=(text:string,peer:string,busy:boolean)=>Boolean(peer)&&!busy&&text.trim().length>0&&messageByteLength(text)<=4096;

export const needsInviteChoice=(mode:string,detected:Record<string,boolean>)=>mode==='auto'&&Boolean(detected.radmin||detected.hamachi);
export const adaptersForMode=<T extends {provider:string}>(items:T[],mode:string)=>mode==='radmin'||mode==='hamachi'?items.filter(a=>a.provider===mode):items;
