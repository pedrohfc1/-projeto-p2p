// Only raw LOCAL microphone tracks may enter this function; never remote/WASAPI.
const constraintsQueue=new WeakMap<MediaStreamTrack,Promise<void>>();
export async function applyNoiseSuppression(streams:(MediaStream|null|undefined)[],enabled:boolean){
 const tracks=[...new Set(streams.flatMap(s=>s?.getAudioTracks()||[]))].filter(t=>t.readyState==='live');
 if(!navigator.mediaDevices?.getSupportedConstraints?.().noiseSuppression)return 'Supressão não suportada neste WebView/dispositivo.';
 if(!tracks.length)return 'Preferência salva; será verificada na próxima captura.';
 let verified=true;
 for(const track of tracks){const pending=(constraintsQueue.get(track)||Promise.resolve()).catch(()=>{}).then(async()=>{if(track.readyState==='live')await track.applyConstraints({noiseSuppression:enabled})});constraintsQueue.set(track,pending);await pending;if(track.getSettings().noiseSuppression!==enabled)verified=false}
 return verified?(enabled?'Supressão ativa no microfone.':'Supressão desativada no microfone.'):'Solicitada; o dispositivo não confirmou a aplicação.';
}
export class MicrophoneNotices {
 captureFailure='';private key='';private dismissed=false;
 deviceStatus(hasInputs:boolean,selectedPresent:boolean,ended:boolean){return !hasInputs?'Nenhum microfone listado. Conecte um microfone ou autorize a detecção nas configurações de voz.':!selectedPresent?'O microfone selecionado está indisponível. Selecione outro dispositivo.':ended?'Microfone desconectado. Tente reconectar para transmitir; a reprodução continua.':this.captureFailure}
 update(key:string){if(key!==this.key){this.key=key;this.dismissed=false}return !!key&&!this.dismissed}
 dismiss(){this.dismissed=true}
}
// One sendrecv slot, even without a microphone: attach a real track later without
// adding another m-line or altering the five authenticated inbound audio slots.
export function configureVoiceTracks(pc:RTCPeerConnection,stream?:MediaStream|null){
 if(stream?.getAudioTracks().length)for(const track of stream.getAudioTracks())pc.addTrack(track,stream);
 else pc.addTransceiver('audio',{direction:'sendrecv'});
 for(let n=0;n<4;n++)pc.addTransceiver('audio',{direction:'recvonly'});
}
