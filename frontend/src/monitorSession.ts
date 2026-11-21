import {AudioMonitor} from './audioMonitor.ts';
export type MonitorOptions={input:string;output:string;volume:number;level:(n:number)=>void;noiseSuppression?:boolean;borrowed?:MediaStream};
// Desired monitoring survives device failures; active means playback actually started.
export class MonitorSession {
 desired=false;private epoch=0;private monitor:AudioMonitor;
 private state:(active:boolean,pending:boolean)=>void;private failure:(e:unknown)=>void;
 constructor(state:(active:boolean,pending:boolean)=>void,failure:(e:unknown)=>void,monitor=new AudioMonitor()){this.state=state;this.failure=failure;this.monitor=monitor}
 pause(){this.epoch++;this.monitor.stop();this.state(false,false)}
 stop(){this.desired=false;this.pause()}
 setVolume(value:number){this.monitor.setVolume(value)}
 async restart(options:MonitorOptions){this.pause();if(!this.desired)return false;const epoch=this.epoch;this.state(false,true);try{const ok=await this.monitor.start(options.input,options.output,options.volume,options.level,options.borrowed,()=>{if(epoch===this.epoch&&this.desired){this.state(false,false);options.level(0);this.failure(Error('Microfone desconectado. O retorno permanece solicitado; reconecte ou selecione outro dispositivo.'))}},options.noiseSuppression);if(epoch!==this.epoch||!this.desired)return false;this.state(ok,false);return ok}catch(e){if(epoch===this.epoch){this.state(false,false);options.level(0);this.failure(e)}return false}}
}
