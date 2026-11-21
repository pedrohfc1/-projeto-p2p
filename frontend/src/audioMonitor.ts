import {diagnostic} from './diagnostics.ts';
import {requestMicrophone,selectAudioOutput} from './audioDevices.ts';
// Local-only microphone monitor. No peer/backend API or recorder is used here.

export class AudioMonitor {

 private stream:MediaStream|null=null; private audio:HTMLAudioElement|null=null; private context:AudioContext|null=null; private timer:ReturnType<typeof setInterval>|null=null; private generation=0; private ownsStream=true;

 async start(input:string,output:string,volume:number,onLevel:(n:number)=>void,borrowed?:MediaStream,onEnded?:()=>void,noiseSuppression=true){

  this.stop();const turn=this.generation;

  const cloned=borrowed?.clone?.();const stream=cloned||borrowed||await requestMicrophone(input,undefined,false,noiseSuppression);const owned=!borrowed||!!cloned;

  if(turn!==this.generation){if(owned)stream.getTracks().forEach(t=>t.stop());return false}

  this.ownsStream=owned;this.stream=stream;if(owned)for(const track of stream.getAudioTracks?.()||[])track.onended=()=>{if(this.stream===stream){this.stop();onEnded?.()}};if(owned)for(const track of stream.getAudioTracks?.()||[])track.enabled=true;

  try {const context=new AudioContext();this.context=context;await context.resume();if(turn!==this.generation)return false;

   const analyser=context.createAnalyser();analyser.fftSize=256;context.createMediaStreamSource(stream).connect(analyser);const values=new Float32Array(256);

   this.timer=setInterval(()=>{analyser.getFloatTimeDomainData(values);onLevel(Math.min(1,Math.sqrt(values.reduce((n,x)=>n+x*x,0)/values.length)))},100);

   const audio=new Audio();this.audio=audio;audio.srcObject=stream;audio.volume=Math.max(0,Math.min(1,volume));

   if(output)await selectAudioOutput(audio,output)

   if(turn!==this.generation)return false;await audio.play();diagnostic('INFO','voice','frontend/retorno','Captura e reprodução exclusivamente locais ativas');console.info('audio_monitor',{capture:'active',playback:'local',channels:1});return true;

  }catch(e){if(turn===this.generation)this.stop();throw e}

 }

 setVolume(n:number){if(this.audio)this.audio.volume=Math.max(0,Math.min(1,n))}

 stop(){if(this.stream)diagnostic('INFO','voice','frontend/retorno','Retorno local encerrado');this.generation++;if(this.timer)clearInterval(this.timer);this.timer=null;if(this.ownsStream)this.stream?.getTracks().forEach(t=>{t.onended=null;t.stop()});this.stream=null;if(this.audio){this.audio.pause();this.audio.srcObject=null};this.audio=null;void this.context?.close();this.context=null}

}


