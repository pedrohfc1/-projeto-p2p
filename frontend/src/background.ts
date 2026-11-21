let hidden=false;
export const isBackground=()=>hidden||document.hidden;
if(typeof window!=='undefined'){window.addEventListener('peetwopee:visibility',((e:CustomEvent)=>{hidden=e.detail.hidden;document.documentElement.classList.toggle('background-mode',isBackground())}) as EventListener)}
export function visualPolling(fn:()=>void,ms:number){let timer:ReturnType<typeof setInterval>|undefined;const update=()=>{if(timer)clearInterval(timer);timer=undefined;if(!isBackground()){fn();timer=setInterval(fn,ms)}};document.addEventListener('visibilitychange',update);window.addEventListener('peetwopee:visibility',update);update();return()=>{if(timer)clearInterval(timer);document.removeEventListener('visibilitychange',update);window.removeEventListener('peetwopee:visibility',update)}}
