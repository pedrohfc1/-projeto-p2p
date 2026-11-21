import {diagnostic} from './diagnostics.ts';
export function audioFailure(error:unknown):string{
 const name=(error as {name?:string})?.name;diagnostic('ERROR','voice','frontend/microfone',name||'Falha de inicialização');
 switch(name){
 case 'NotAllowedError':case 'PermissionDeniedError':return 'Permissão de microfone negada. Autorize o pedido do WebView e confira Configurações do Windows → Privacidade e segurança → Microfone (acesso para aplicativos desktop). O aplicativo não altera essas permissões.';
 case 'NotFoundError':case 'DevicesNotFoundError':case 'OverconstrainedError':return 'Nenhum microfone compatível encontrado. Reconecte o dispositivo, atualize a lista ou escolha Padrão do sistema.';
 case 'NotReadableError':case 'TrackStartError':return 'Dispositivo ocupado ou indisponível. Confira se outro aplicativo está usando o microfone e tente novamente.';
 case 'AbortError':return 'Falha de inicialização do microfone. Reconecte o dispositivo e tente novamente.';
 default:return 'Falha de inicialização de áudio: '+String(error);
 }
}
export async function requestMicrophone(input:string,onFallback:()=>void=()=>{},allowFallback=true,noiseSuppression=true){
 if(!globalThis.navigator?.mediaDevices?.getUserMedia)throw new Error('Captura indisponível no WebView. Atualize o Microsoft Edge WebView2 e reinicie o aplicativo; nenhuma permissão foi alterada.');
 try{diagnostic('INFO','voice','frontend/captura','Solicitando microfone '+(input?'selecionado':'padrão'));const stream=await navigator.mediaDevices.getUserMedia({audio:{deviceId:input?{exact:input}:undefined,channelCount:1,echoCancellation:true,noiseSuppression,autoGainControl:true},video:false});diagnostic('INFO','voice','frontend/captura','Microfone ativo; faixas='+(stream.getAudioTracks?.().length??'n/d'));return stream}
 catch(error){ // A removed saved device must not make the system default unusable.
  if(allowFallback&&input&&['NotFoundError','OverconstrainedError','DevicesNotFoundError'].includes((error as DOMException).name)){const stream=await navigator.mediaDevices.getUserMedia({audio:{channelCount:1,echoCancellation:true,noiseSuppression},video:false});onFallback();return stream;}
  throw error;
 }
}
let enumeration:Promise<{inputs:MediaDeviceInfo[];outputs:MediaDeviceInfo[]}>|null=null;
export function listAudioDevices(timeoutMs=4000){
 if(enumeration)return enumeration;
 if(!globalThis.navigator?.mediaDevices?.enumerateDevices)return Promise.reject(Error('Listagem de dispositivos indisponível no WebView. Atualize o WebView2.'));
 let timer:ReturnType<typeof setTimeout>;
 const query=Promise.resolve().then(()=>navigator.mediaDevices.enumerateDevices());
 const deadline=new Promise<MediaDeviceInfo[]>((_,reject)=>{timer=setTimeout(()=>reject(Error('A lista de dispositivos não respondeu. A chamada continua; tente Atualizar dispositivos novamente.')),timeoutMs)});
 const pending=Promise.race([query,deadline]).then(devices=>({inputs:devices.filter(d=>d.kind==='audioinput'),outputs:devices.filter(d=>d.kind==='audiooutput')})).finally(()=>{clearTimeout(timer);if(enumeration===pending)enumeration=null});
 enumeration=pending;return pending;
}
// Commit a device handover only while the same call/PC is still current.
export async function replaceMicrophoneTrack(sender:RTCRtpSender,stream:MediaStream,current:()=>boolean){
 const stop=()=>stream.getTracks().forEach(t=>t.stop());
 try{const track=stream.getAudioTracks()[0];if(!track)throw Error('A captura não forneceu uma faixa de microfone.');await sender.replaceTrack(track);if(!current()){stop();return false};return true}catch(e){stop();throw e}
}

export async function selectAudioOutput(audio:HTMLMediaElement,id:string){
 const sink=audio as HTMLMediaElement&{setSinkId?:(id:string)=>Promise<void>};
 if(!id&&!sink.setSinkId)return;
 if(!sink.setSinkId)throw Error('Seleção de saída indisponível neste WebView; escolha Padrão do sistema.');
 try{await sink.setSinkId(id)}catch{throw Error('Saída de áudio selecionada indisponível ou sem permissão. Escolha outra saída ou Padrão do sistema; o aplicativo não altera permissões automaticamente.')}
}

// The replacement cannot emit live microphone samples until identity/session and
// the latest mute intent have been rechecked after the asynchronous sender swap.
export async function replaceCallMicrophone(sender:RTCRtpSender,raw:MediaStream,processed:MediaStream,current:()=>boolean,muted:()=>boolean){
 const track=processed.getAudioTracks()[0];if(!track)throw Error('Emissor de microfone indisponível.');track.enabled=false;
 if(!await replaceMicrophoneTrack(sender,processed,current))return false;
 for(const input of raw.getAudioTracks())input.enabled=!muted();return true; // Caller activates only after committing its capture reference.
}
