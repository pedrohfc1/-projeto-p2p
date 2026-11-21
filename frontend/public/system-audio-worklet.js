// Bounded 100 ms local PCM bridge. It never connects to the physical output.
class SystemPCM extends AudioWorkletProcessor {
 constructor(){super();this.ring=new Float32Array(4800);this.read=0;this.write=0;this.port.onmessage=e=>{const p=e.data;if(!(p instanceof Float32Array)||p.length>1920)return;for(const v of p){this.ring[this.write++%4800]=Number.isFinite(v)?Math.max(-1,Math.min(1,v)):0;if(this.write-this.read>4800)this.read=this.write-4800}}}
 process(inputs,outputs){const out=outputs[0]?.[0];if(out)for(let i=0;i<out.length;i++)out[i]=this.read<this.write?this.ring[this.read++%4800]:0;return true}
}
registerProcessor('peetwopee-system-pcm',SystemPCM);
