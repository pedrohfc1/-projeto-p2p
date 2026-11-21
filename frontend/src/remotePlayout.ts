// Chromium has a distinct RTC renderer. Bind the actual remote stream, not only
// a WebAudio-generated stream. This sink is permanently inaudible; the mixer is
// the only audible path and still enforces participant gain/deaf/output routing.
export class RemotePlayout {
 readonly element:HTMLAudioElement;private closed=false;private stream:MediaStream;private track:MediaStreamTrack;private changed:()=>void;private report:(state:string)=>void;
 constructor(stream:MediaStream,report:(state:string)=>void=()=>{}){
  this.stream=stream;this.track=stream.getAudioTracks()[0];this.report=report;
  this.element=document.createElement('audio');this.element.autoplay=true;this.element.muted=true;this.element.volume=0;this.element.hidden=true;this.element.dataset.voiceDecoder='true';this.element.srcObject=stream;document.body.appendChild(this.element);
  this.changed=()=>{if(!this.closed)void this.resume().catch(()=>{})};this.track?.addEventListener('unmute',this.changed);this.element.addEventListener('loadedmetadata',this.changed);
  void this.resume().catch(()=>{});
 }
 async resume(){if(this.closed)return;this.element.muted=true;this.element.volume=0;try{await this.element.play();if(!this.closed)this.report('rtc-renderer-playing')}catch(e){if(this.closed)return;this.report('rtc-renderer-blocked:'+((e as Error).name||'Error'));throw e}}
 status(){return {paused:this.element.paused,muted:this.element.muted,ready:this.element.readyState,trackState:this.track?.readyState,trackMuted:this.track?.muted,trackEnabled:this.track?.enabled}}
 close(){if(this.closed)return;this.closed=true;this.track?.removeEventListener('unmute',this.changed);this.element.removeEventListener('loadedmetadata',this.changed);this.element.pause();this.element.srcObject=null;this.element.remove();this.report('rtc-renderer-released')}
}
