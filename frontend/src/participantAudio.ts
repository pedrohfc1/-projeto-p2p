import {RemotePlayout} from './remotePlayout.ts';
// Stream IDs are optional in Unified Plan/WebView2. Never route every streamless track to slot zero.
export function audioSlot(event:{streams?:readonly {id:string}[];transceiver:unknown},transceivers:unknown[]){
 const named=event.streams?.[0]?.id.match(/(?:^|-)slot-(\d+)$/);const slot=named?Number(named[1]):transceivers.indexOf(event.transceiver);
 return Number.isInteger(slot)&&slot>=0&&slot<5?slot:null;
}
export type ListenerPreference={volume:number;muted:boolean};
export const preference=(p?:Partial<ListenerPreference>):ListenerPreference=>({volume:Number.isFinite(p?.volume)?Math.max(0,Math.min(2,p!.volume!)):1,muted:p?.muted===true});
export function limiterCurve(){const curve=new Float32Array(4097);for(let i=0;i<curve.length;i++){const x=2*i/(curve.length-1)-1;const a=Math.abs(x);curve[i]=a<=.8?x:Math.sign(x)*(.8+.18*Math.tanh((a-.8)/.18))}return curve}
// Five independent decoded tracks are mixed locally; no changes are sent to peers.
export class ParticipantMixer{
 readonly context:AudioContext;readonly output:MediaStreamAudioDestinationNode;private compressor:DynamicsCompressorNode;private limiter:WaveShaperNode;
 private closed=false;private report:(state:string)=>void;private meterNode:AudioWorkletNode|null=null;private outputSamples:{samples:number;nonSilentSamples:number;quanta:number}|null=null;
 private slots=new Map<number,{source:MediaStreamAudioSourceNode;gain:GainNode;analyser:AnalyserNode;peer:string;playout:RemotePlayout|null}>();private preferences:Record<string,ListenerPreference>={};private roster=new Map<number,string>();
 constructor(report:(state:string)=>void=()=>{}){this.report=report;this.context=new AudioContext({latencyHint:'interactive'});this.context.onstatechange=()=>{this.report('context:'+this.context.state);if(!this.closed&&this.context.state==='suspended')void this.context.resume().catch(()=>{})};this.output=this.context.createMediaStreamDestination();this.compressor=this.context.createDynamicsCompressor();this.compressor.threshold.value=-6;this.compressor.knee.value=6;this.compressor.ratio.value=20;this.compressor.attack.value=.003;this.compressor.release.value=.15;this.limiter=this.context.createWaveShaper();this.limiter.curve=limiterCurve();this.limiter.oversample='2x';this.compressor.connect(this.limiter).connect(this.output);void this.context.resume().catch(()=>{});void this.installMeter()}
 private async installMeter(){if(!this.context.audioWorklet||typeof AudioWorkletNode==='undefined')return;let node:AudioWorkletNode|null=null;let rewired=false;try{await this.context.audioWorklet.addModule('/voice-output-meter.js');if(this.closed)return;node=new AudioWorkletNode(this.context,'peetwopee-voice-output-meter');node.port.onmessage=e=>{if(!this.closed)this.outputSamples=e.data};node.connect(this.output);this.limiter.connect(node);this.limiter.disconnect(this.output);rewired=true;this.meterNode=node}catch{if(node){node.port.onmessage=null;node.disconnect();try{this.limiter.disconnect(node)}catch{}};if(rewired)this.limiter.connect(this.output);this.report('output-samples-unavailable')}}
 add(slot:number,stream:MediaStream){if(this.closed||slot<0||slot>4)return;this.remove(slot);
  const playout=typeof document!=='undefined'?new RemotePlayout(stream,state=>this.report('slot='+slot+' '+state)):null;
  let source:MediaStreamAudioSourceNode|null=null,gain:GainNode|null=null,analyser:AnalyserNode|null=null;try{source=this.context.createMediaStreamSource(stream);gain=this.context.createGain();analyser=this.context.createAnalyser();analyser.fftSize=256;source.connect(analyser).connect(gain).connect(this.compressor);this.slots.set(slot,{source,gain,analyser,peer:this.roster.get(slot)||'',playout});this.apply()}catch(e){playout?.close();source?.disconnect();gain?.disconnect();analyser?.disconnect();this.slots.delete(slot);throw e}}
 blocked(){return [...this.slots.values()].some(s=>s.playout?.status().paused)}
 async resumeInputs(){await Promise.all([...this.slots.values()].map(s=>s.playout?.resume()));}
 diagnostics(){return {context:this.context.state,contextSeconds:this.context.currentTime,outputGraph:this.outputSamples,physicalPlayback:'not measurable by WebView',tracks:[...this.slots.entries()].map(([slot,s])=>({slot,peerKnown:!!s.peer,gain:s.gain.gain.value,...s.playout?.status()}))}}
 resetInputs(){for(const slot of [...this.slots.keys()])this.remove(slot)}
 rosterChanged(people:{id:string;slot:number;self:boolean}[],prefs:Record<string,ListenerPreference>){this.preferences=prefs||{};this.roster.clear();for(const p of people)if(!p.self&&p.slot>=0)this.roster.set(p.slot,p.id);for(const [slot,item]of this.slots)item.peer=this.roster.get(slot)||'';this.apply()}
 private apply(){for(const item of this.slots.values()){const p=preference(this.preferences[item.peer]);item.gain.gain.setTargetAtTime(item.peer&&!p.muted?p.volume:0,this.context.currentTime,.015)}}
 levels(){const out:Record<string,number>={};const values=new Float32Array(256);for(const item of this.slots.values()){if(!item.peer)continue;item.analyser.getFloatTimeDomainData(values);out[item.peer]=Math.min(1,Math.sqrt(values.reduce((sum,x)=>sum+x*x,0)/values.length)*8)}return out}
 private remove(slot:number){const item=this.slots.get(slot);if(item){item.playout?.close();item.source.disconnect();item.analyser.disconnect();item.gain.disconnect();this.slots.delete(slot)}}
 async close(){if(this.closed)return;this.closed=true;this.context.onstatechange=null;for(const slot of [...this.slots.keys()])this.remove(slot);this.compressor.disconnect();this.meterNode?.disconnect();if(this.meterNode)this.meterNode.port.onmessage=null;this.meterNode=null;this.limiter.disconnect();this.output.disconnect();await this.context.close()}
}
