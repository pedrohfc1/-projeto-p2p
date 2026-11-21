// Invoke synchronously from Join/Resume, before any permission or bridge await.
// Playback failure is recoverable and must not tear down room membership.
export function activateVoice(contexts: Pick<AudioContext,'resume'|'state'>[], player: Pick<HTMLAudioElement,'play'|'paused'>, timeout=2000):Promise<void>{
 const attempts:Promise<unknown>[]=[];
 for(const context of contexts){try{attempts.push(context.resume())}catch(e){attempts.push(Promise.reject(e))}}
 try{attempts.push(player.play())}catch(e){attempts.push(Promise.reject(e))}
 let timer:ReturnType<typeof setTimeout>|undefined;
 return Promise.race([Promise.all(attempts).then(()=>{if(contexts.some(c=>c.state!=='running')||player.paused)throw Error('Áudio suspenso. Clique em Ativar reprodução de áudio.')}),new Promise<never>((_,reject)=>{timer=setTimeout(()=>reject(Error('Áudio aguardando ativação pelo usuário.')),timeout)})]).finally(()=>clearTimeout(timer));
}
