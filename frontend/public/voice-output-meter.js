// Transparent output-graph telemetry. Posts counts only; no PCM leaves this node.
class VoiceOutputMeter extends AudioWorkletProcessor {
 constructor(){super();this.samples=0;this.nonSilentSamples=0;this.quanta=0;this.last=0}
 process(inputs,outputs){const input=inputs[0],out=outputs[0];if(!out)return true;for(let c=0;c<out.length;c++){const src=input?.[c]||input?.[0];if(src)out[c].set(src);else out[c].fill(0)}const mono=out[0];if(mono){this.samples+=mono.length;for(const x of mono)if(Math.abs(x)>0.00001)this.nonSilentSamples++}this.quanta++;if(currentTime-this.last>=3){this.port.postMessage({samples:this.samples,nonSilentSamples:this.nonSilentSamples,quanta:this.quanta});this.last=currentTime}return true}
}
registerProcessor('peetwopee-voice-output-meter',VoiceOutputMeter);
