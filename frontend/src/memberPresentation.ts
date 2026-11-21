export function memberSections<T extends {id:string;role:string}>(members:readonly T[],self:string,online:readonly string[],voice:readonly string[]){
 const connected=new Set([...online,self]),speaking=new Set(voice),seen=new Set<string>();
 const role=(m:T)=>m.role==='owner'?0:m.role==='moderator'?1:2;
 const unique=members.filter(m=>{if(seen.has(m.id))return false;seen.add(m.id);return true});
 return [['Em voz',unique.filter(m=>speaking.has(m.id))],['Online',unique.filter(m=>!speaking.has(m.id)&&connected.has(m.id))],['Offline',unique.filter(m=>!speaking.has(m.id)&&!connected.has(m.id))]].map(([label,items])=>({label:label as string,items:(items as T[]).sort((a,b)=>role(a)-role(b))})).filter(s=>s.items.length);
}
