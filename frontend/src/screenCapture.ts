import {diagnostic} from './diagnostics.ts';
export const screenProfiles={auto:{width:1280,height:720,fps:30},'720p30':{width:1280,height:720,fps:30},'720p60':{width:1280,height:720,fps:60},'1080p30':{width:1920,height:1080,fps:30},'1080p60':{width:1920,height:1080,fps:60}};
export type ScreenProfile=keyof typeof screenProfiles;
export class ScreenCapture {
 private generation=0;stream:MediaStream|null=null;appliedProfile:ScreenProfile='720p30';
 private ended:()=>void; constructor(ended:()=>void){this.ended=ended}
 async select(profile:ScreenProfile='auto'){
  let quality=screenProfiles[profile];
  this.stop();const turn=this.generation;
  if(!navigator.mediaDevices?.getDisplayMedia)throw Error('Compartilhamento indisponível neste WebView. Atualize o WebView2; nenhuma fonte será escolhida automaticamente.');
  const options={video:{width:{ideal:quality.width},height:{ideal:quality.height},frameRate:{ideal:quality.fps}},audio:false,surfaceSwitching:'exclude',systemAudio:'exclude',selfBrowserSurface:'exclude'} as DisplayMediaStreamOptions;
  const stream=await navigator.mediaDevices.getDisplayMedia(options);
  if(turn!==this.generation){stream.getTracks().forEach(t=>t.stop());return null}
  for(const audio of stream.getAudioTracks?.()||[])audio.stop();
  const track=stream.getVideoTracks()[0];if(!track){stream.getTracks().forEach(t=>t.stop());throw Error('A fonte selecionada não forneceu vídeo.')}
  let effective=profile;
  const apply=()=>track.applyConstraints({width:{max:quality.width},height:{max:quality.height},frameRate:{max:quality.fps}});
  try {
   try { await apply(); } catch (error) {
    if (profile==='auto'||profile==='720p30'||turn!==this.generation) throw error;
    effective='720p30';
    quality=screenProfiles[effective];
    await apply();
   }
  } catch (error) {
   stream.getTracks().forEach(t=>t.stop());
   if (turn!==this.generation) return null;
   throw Error('A fonte recusou os limites de captura. Nenhuma transmissão foi iniciada.',{cause:error});
  }
  if(turn!==this.generation){stream.getTracks().forEach(t=>t.stop());return null};track.contentHint=quality.fps===60?'motion':'detail';this.appliedProfile=effective;this.stream=stream;const settings=track.getSettings?.();diagnostic('INFO','video','frontend/captura',`perfil=${profile} efetivo=${effective} limites=${quality.width}x${quality.height}@${quality.fps} real=${settings?.width??'n/d'}x${settings?.height??'n/d'} fps_configurado=${settings?.frameRate??'n/d'} (FPS real nas métricas)`);track.onended=()=>{if(this.stream===stream){this.stop();this.ended()}};return stream;
 }
 stop(){if(this.stream)diagnostic('INFO','video','frontend/captura','Captura encerrada; faixas liberadas');this.generation++;const stream=this.stream;this.stream=null;for(const track of stream?.getTracks()||[]){track.onended=null;track.stop()}}
}
export function screenFailure(e:unknown){const name=(e as {name?:string})?.name;if(name==='NotAllowedError')return 'Compartilhamento cancelado ou permissão negada. Nenhuma fonte foi transmitida.';if(name==='NotReadableError')return 'A fonte não pôde ser capturada. Verifique se a janela ainda existe e tente selecioná-la novamente.';return String(e)}
export function videoBitrate(total:number,viewers:number,drops:number,current:number){const ceiling=Math.max(64000,Math.floor(total/Math.max(1,viewers)));return drops>0?Math.max(64000,Math.min(ceiling,Math.floor(current*.75))):Math.min(ceiling,Math.max(64000,Math.floor(current*1.05)))}

