export type SoundSource={pid:number;name:string;title?:string};
// Fail closed: an opaque browser label is not a process identity. Never guess by substring.
export function automaticSoundSource(surface:string|undefined,label:string,sources:SoundSource[],separateOutput:boolean):{mode:string;pid:number}{
 if(surface==='monitor'){
  if(!separateOutput)throw Error('Áudio da saída padrão inclui a voz recebida. Para evitar eco, selecione outra saída física para a chamada ou uma fonte nos detalhes técnicos. O vídeo continua.');
  return {mode:'device',pid:0};
 }
 if(surface!=='window')throw Error('O Windows não identificou a fonte de áudio. Escolha o aplicativo nos detalhes técnicos.');
 const exact=sources.filter(s=>s.pid>0&&s.title&&s.title===label);
 if(exact.length!==1)throw Error('Não foi possível associar com segurança a janela ao aplicativo. Escolha a fonte nos detalhes técnicos; nenhum áudio foi capturado.');
 return {mode:'process',pid:exact[0].pid};
}
export function separatePhysicalOutput(output:string,devices:readonly Pick<MediaDeviceInfo,'kind'|'deviceId'|'groupId'>[]){
 const selected=devices.find(d=>d.kind==='audiooutput'&&d.deviceId===output);
 const standard=devices.find(d=>d.kind==='audiooutput'&&d.deviceId==='default');
 return !!(output&&output!=='default'&&selected?.groupId&&standard?.groupId&&selected.groupId!==standard.groupId);
}
