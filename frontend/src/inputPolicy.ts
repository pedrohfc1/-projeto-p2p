export type InputPolicy={mode:'voice'|'ptt';automatic:boolean;thresholdDb:number;key:string;releaseMs:number};
const defaults:InputPolicy={mode:'voice',automatic:true,thresholdDb:-45,key:'F8',releaseMs:100};let current:InputPolicy|undefined;
export function policy(value:Partial<InputPolicy>={}):InputPolicy{return {mode:value.mode==='ptt'?'ptt':'voice',automatic:value.automatic!==false,thresholdDb:Number.isFinite(value.thresholdDb)?Math.max(-60,Math.min(-10,value.thresholdDb!)):-45,key:typeof value.key==='string'&&/^(F[1-9]|F1[0-2]|Key[A-Z]|Digit[0-9]|Space|CapsLock|ShiftLeft|ShiftRight)$/.test(value.key)?value.key:'F8',releaseMs:Number.isFinite(value.releaseMs)?Math.max(0,Math.min(500,value.releaseMs!)):100}}
export function inputPolicy(){if(!current){try{current=policy(JSON.parse(localStorage.getItem('peetwopee.voice-input')||'{}'))}catch{current={...defaults}}}return current}
export function saveInputPolicy(value:InputPolicy){current=policy(value);try{localStorage.setItem('peetwopee.voice-input',JSON.stringify(current))}catch{};window.dispatchEvent(new Event('peetwopee:input-policy'))}
export function textTarget(target:EventTarget|null){return target instanceof HTMLElement&&(target.isContentEditable||!!target.closest('input,textarea,select,[role=textbox]'))}
export class InputGate {held=false;private released=-Infinity;private lastVoice=-Infinity;private floor=.005;private at:number|undefined;threshold=.01;
 press(){this.held=true} release(now:number){if(this.held)this.released=now;this.held=false} reset(){this.held=false;this.released=-Infinity;this.lastVoice=-Infinity;this.at=undefined}
 sample(rms:number,now:number,p:InputPolicy){if(p.mode==='ptt')return this.held||now-this.released<p.releaseMs;const level=Number.isFinite(rms)?Math.max(0,rms):0;const elapsed=this.at===undefined?20:Math.max(0,Math.min(1000,now-this.at));this.at=now;const alpha=1-Math.exp(-elapsed/1000);if(p.automatic&&level<this.floor*2)this.floor=(1-alpha)*this.floor+alpha*level;this.threshold=p.automatic?Math.min(.06,Math.max(.003,this.floor*2.5)):10**(p.thresholdDb/20);if(level>=this.threshold)this.lastVoice=now;return now-this.lastVoice<150}
}

export function bindingCode(event:{code?:string;key?:string;location?:number}){const code=event.code||'';if(code&&policy({key:code}).key===code)return code;const key=event.key||'';if(/^[a-zA-Z]$/.test(key))return 'Key'+key.toUpperCase();if(/^[0-9]$/.test(key))return 'Digit'+key;if(/^F([1-9]|1[0-2])$/.test(key))return key;if(key===' ')return 'Space';if(key==='Shift')return event.location===2?'ShiftRight':'ShiftLeft';if(key==='CapsLock')return key;return ''}

export function meterFraction(rms:number){return !Number.isFinite(rms)||rms<=0?0:Math.max(0,Math.min(1,(20*Math.log10(rms)+60)/60))}
