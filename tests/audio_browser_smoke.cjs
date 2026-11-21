// Browser PCM tone -> actual Opus encoder -> two production processes/EasyTier -> Opus decoder.
// No physical microphone or external audio is captured. Default soak is five minutes.
const esbuild=require('../frontend/node_modules/esbuild');const graphBundle=esbuild.buildSync({stdin:{contents:"export {ParticipantMixer,audioSlot} from './participantAudio';export {MicrophoneGain} from './microphoneGain';export {saveInputPolicy,inputPolicy} from './inputPolicy';",resolveDir:pathResolve()},bundle:true,format:'iife',globalName:'P2PAudioGraph',write:false}).outputFiles[0].text;function pathResolve(){return require('node:path').resolve('frontend/src')}
const {chromium}=require(process.env.P2P_PLAYWRIGHT_PATH||'playwright');
const {spawn}=require('node:child_process');const fs=require('node:fs/promises');const path=require('node:path');const net=require('node:net');const assert=require('node:assert/strict');
const sleep=ms=>new Promise(r=>setTimeout(r,ms));const children=[];let browser,folder;
async function port(){const s=net.createServer();await new Promise(r=>s.listen(0,'127.0.0.1',r));const n=s.address().port;await new Promise(r=>s.close(r));return n}
async function bridge(name){const p=await port(),url=`http://127.0.0.1:${p}`;const child=spawn(path.resolve(process.argv[2]),['--data',path.join(folder,name),'--port',String(p)],{cwd:process.cwd(),windowsHide:true,stdio:'ignore'});children.push(child);for(let i=0;i<150;i++){try{const html=await(await fetch(url)).text();const token=html.match(/X-QA-Token':'([a-f0-9]{64})'/)[1];return {url,token}}catch{};await sleep(100)}throw Error('QA backend startup failed')}
async function rpc(b,Method,args={}){const res=await fetch(b.url+'/rpc',{method:'POST',headers:{Origin:b.url,'X-QA-Token':b.token,'Content-Type':'application/json'},body:JSON.stringify({Method,...args})});const value=await res.json();if(!res.ok)throw Error(Method+': '+value.error);return value}
async function pageFor(b,frequency){const context=await browser.newContext();await context.addInitScript(({frequency})=>{
 window.__captures=0;window.__localPlays=0;window.__ended=0;window.__diagnostics=[];
 const original=HTMLMediaElement.prototype.play;HTMLMediaElement.prototype.play=function(){if(this.srcObject)window.__localPlays++;return original.call(this)};
 navigator.mediaDevices.getUserMedia=async()=>{window.__captures++;const c=new AudioContext({sampleRate:48000});await c.resume();const o=c.createOscillator();o.frequency.value=frequency;const gain=c.createGain();gain.gain.value=.15;const dest=c.createMediaStreamDestination();o.connect(gain).connect(dest);o.start();for(const track of dest.stream.getTracks()){const stop=track.stop.bind(track);track.stop=()=>{window.__ended++;stop();o.stop();void c.close()}}return dest.stream};
 },{frequency});await context.addInitScript({content:graphBundle+';globalThis.P2PAudioGraph=P2PAudioGraph;'});const page=await context.newPage();page.on('pageerror',e=>console.error('PAGE_ERROR',String(e)));await page.goto(b.url);await page.waitForFunction(()=>window.go?.main?.App&&window.P2PAudioGraph);return page}
async function startAudio(page,group){await page.evaluate(async group=>{
 const mixer=new P2PAudioGraph.ParticipantMixer();await mixer.context.resume();const stream=await navigator.mediaDevices.getUserMedia({audio:true});const gain=new P2PAudioGraph.MicrophoneGain(stream,1,mixer.context);await gain.ready();const pc=new RTCPeerConnection({iceServers:[],bundlePolicy:'max-bundle'});window.__call={stream,pc,level:0,mixer,gain};const audio=new Audio();audio.autoplay=true;audio.srcObject=mixer.output.stream;window.__call.audio=audio;await audio.play();
 const analyser=mixer.context.createAnalyser();mixer.context.createMediaStreamSource(mixer.output.stream).connect(analyser);const values=new Float32Array(512);analyser.fftSize=512;window.__call.timer=setInterval(async()=>{analyser.getFloatTimeDomainData(values);window.__call.level=Math.sqrt(values.reduce((s,n)=>s+n*n,0)/values.length);const v=await window.go.main.App.VoiceState();mixer.rosterChanged(v.participants,{})},100);
 pc.ontrack=e=>{const slot=P2PAudioGraph.audioSlot({streams:[],transceiver:e.transceiver},pc.getTransceivers());if(slot===null)throw Error('unknown audio slot');mixer.add(slot,e.streams[0]||new MediaStream([e.track]))};
 for(const t of gain.output.stream.getAudioTracks())pc.addTrack(t,gain.output.stream);for(let n=0;n<4;n++)pc.addTransceiver('audio',{direction:'recvonly'});const offer=await pc.createOffer();offer.sdp=offer.sdp.replace(/a=fmtp:(\d+) ([^\r\n]*useinbandfec[^\r\n]*)/g,'a=fmtp:$1 $2;stereo=0;sprop-stereo=0;maxptime=20');await pc.setLocalDescription(offer);await new Promise((resolve,reject)=>{if(pc.iceGatheringState==='complete')return resolve();const timer=setTimeout(()=>reject(Error('ICE gather timeout')),8000);pc.onicegatheringstatechange=()=>{if(pc.iceGatheringState==='complete'){clearTimeout(timer);resolve()}}});const answer=await window.go.main.App.VoiceJoin(group,'voz',pc.localDescription);await pc.setRemoteDescription(answer);
 },group)}
async function audible(page){return page.evaluate(()=>({level:window.__call?.level||0,state:window.__call?.pc.connectionState,playing:window.__call?.audio.paused===false}))}
(async()=>{await fs.mkdir('data',{recursive:true});folder=await fs.mkdtemp(path.resolve('data/audio-browser-'));const a=await bridge('a'),b=await bridge('b');browser=await chromium.launch({executablePath:process.env.P2P_EDGE_PATH||'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',chromiumSandbox:true,headless:true,args:['--autoplay-policy=no-user-gesture-required','--no-first-run']});const pa=await pageFor(a,440),pb=await pageFor(b,660);
 // Actual production monitor UI: stays local, one capture, closes on settings close.
 await pa.getByRole('button',{name:'Configurações',exact:true}).click();await pa.getByRole('tab',{name:'Voz',exact:true}).click();await pa.getByRole('button',{name:'Testar microfone',exact:true}).click();await pa.getByRole('button',{name:'Parar teste',exact:true}).waitFor();await sleep(500);assert.equal(await pa.evaluate(()=>window.__captures),1);assert.ok(await pa.evaluate(()=>window.__localPlays)>0);assert.ok(Number(await pa.getByRole('meter',{name:'Nível do microfone',exact:true}).getAttribute('value'))>.1);assert.equal((await rpc(a,'VoiceState')).joined,false);await pa.getByRole('button',{name:'Fechar configurações',exact:true}).click();assert.equal(await pa.evaluate(()=>window.__ended),1);
 for(const [n,x]of ['a','b'].map((n,i)=>[n,[a,b][i]]))await rpc(x,'SaveSettings',{Settings:{name:n,mode:'integrated',port:await port(),integratedBinaryDir:path.resolve(process.argv[3])}});
 const group=await rpc(a,'CreateCommunity',{Name:'Audio browser PCM QA'});const ticket=await rpc(a,'CommunityInvite',{ID:group});assert.equal((await rpc(b,'JoinCommunity',{Text:ticket})).status,'participante');await startAudio(pa,group);await startAudio(pb,group);
 let ready=false;for(let i=0;i<200;i++){const aa=await audible(pa),bb=await audible(pb);if(aa.level>.02&&bb.level>.02&&aa.playing&&bb.playing){ready=true;break};await sleep(200)}assert.ok(ready,'Opus decoded PCM not audible at both renderers');console.log('PASS: local monitor UI capture/playback/cleanup and bidirectional browser Opus PCM');
 for(const backend of [a,b]){const d=await rpc(backend,'VoiceDiagnosticSnapshot');assert.equal(d.authorization,'valid');assert.ok(d.peers.length>=1);assert.ok(d.peers[0].media.packetsIn>0);assert.ok(d.peers[0].media.lastReceiveMs>0);assert.ok(d.peers[0].rttMs!==null);assert.match(d.peers[0].alias,/^[a-f0-9]{12}$/)}console.log('PASS: actual authorized peer diagnostic RTT, media counters and last receive; no remote payload export');
 // A valid passive sample can precede the first published room roster for
 // up to one cache interval. Check every sample remains free of RTT probes.
 let passiveReady=false;
 for(let i=0;i<12;i++){
  const passive=await rpc(a,'VoiceDiagnosticSample',{Enabled:false});
  assert.ok(passive.peers.every(peer=>peer.rttMs===null));
  if(passive.joined&&passive.peers.some(peer=>peer.media.packetsIn>0)){passiveReady=true;break}
  await sleep(200);
 }
 assert.ok(passiveReady,'passive snapshot did not catch up to the active roster');
 console.log('PASS: background diagnostic sample records voice traffic without an RTT probe');

 if(process.env.P2P_QA_ROOM_PROBE==='1'){
  const [da,db]=await Promise.all([rpc(a,'VoiceDiagnosticSnapshot'),rpc(b,'VoiceDiagnosticSnapshot')]);
  const ra={ID:da.peers[0].alias,Text:'10'};
  await assert.rejects(rpc(a,'RunRoomProbe',ra),/precisa permitir/);
  await rpc(b,'ApproveRoomProbe',{ID:db.peers[0].alias});
  const probe=await rpc(a,'RunRoomProbe',ra);
  assert.ok(probe.totalBlocks>=22&&probe.totalBlocks<=40,JSON.stringify(probe));
  assert.equal(probe.payloadBytes,probe.totalBlocks*4000);
  assert.ok(probe.payloadKbps>0&&probe.payloadKbps<=128);
  assert.ok(probe.rttMeanMs>0&&probe.rttP95Ms>0);
  await assert.rejects(rpc(a,'RunRoomProbe',ra),/precisa permitir/);
  await rpc(b,'CancelRoomProbe');
  console.log('PASS: bounded 10s authorized in-call echo, confirmed payload/RTT, rejected missing and reused approval');
 }

 await pa.evaluate(()=>P2PAudioGraph.saveInputPolicy({...P2PAudioGraph.inputPolicy(),mode:'ptt',key:'KeyV',releaseMs:100}));await sleep(800);assert.ok((await audible(pb)).level<.005,'PTT closed still audible');await pa.keyboard.down('v');await sleep(800);assert.ok((await audible(pb)).level>.02,'PTT key did not transmit');await pa.evaluate(()=>{const field=document.createElement('input');field.id='qa-typing';document.body.append(field);field.focus()});await sleep(800);assert.ok((await audible(pb)).level<.005,'typing focus did not close PTT');await pa.keyboard.up('v');await pa.evaluate(()=>document.getElementById('qa-typing').remove());await pa.evaluate(()=>P2PAudioGraph.saveInputPolicy({...P2PAudioGraph.inputPolicy(),mode:'voice'}));await sleep(800);assert.ok((await audible(pb)).level>.02,'voice activation did not recover');console.log('PASS: actual browser audio-worklet PTT gate, key hold, typing focus reset and voice recovery');
 const before=await rpc(a,'VoiceState');await rpc(a,'VoiceControls',{Muted:true,Deaf:false});await sleep(700);const muted=await rpc(a,'VoiceState');assert.ok(muted.media.packetsOut-before.media.packetsOut<10);await rpc(a,'VoiceControls',{Muted:false,Deaf:false});
 const seconds=Number(process.argv[4]||300);const started=Date.now();let checkpoints=0;while(Date.now()-started<seconds*1000){await sleep(2000);const [va,vb]=await Promise.all([rpc(a,'VoiceState'),rpc(b,'VoiceState')]);assert.ok(va.joined&&vb.joined,'unexpected presence disconnection');const aa=await audible(pa),bb=await audible(pb);assert.ok(aa.level>.02&&bb.level>.02,'PCM playback stalled');if(checkpoints++%10===0){await rpc(b,'CommunityAction',{ID:group,Kind:'message.text',Body:{channelId:'geral',text:'chat during audio '+checkpoints}})}}
 const final={seconds,checkpoints,a:await rpc(a,'VoiceState'),b:await rpc(b,'VoiceState'),pcm:[await audible(pa),await audible(pb)]};await fs.writeFile(path.resolve('..','work','audio-browser-results.json'),JSON.stringify(final,null,2));
 for(const page of[pa,pb])await page.evaluate(async()=>{await window.go.main.App.VoiceLeave();const c=window.__call;clearInterval(c.timer);c.stream.getTracks().forEach(t=>t.stop());c.pc.close();c.audio.pause();c.audio.srcObject=null;c.gain?.close();await c.mixer?.close();await c.context?.close()});assert.equal((await rpc(a,'VoiceState')).joined,false);assert.equal((await rpc(b,'VoiceState')).joined,false);console.log('PASS: '+seconds+'s continuous bidirectional real Opus encode/decode, concurrent chat and cleanup; synthetic PCM, not physical devices/two houses');
})().catch(e=>{console.error(e);process.exitCode=1}).finally(async()=>{await browser?.close();for(const c of children){c.kill();await new Promise(r=>{if(c.exitCode!==null)return r();c.once('exit',r)})};if(folder){assert.equal(path.dirname(folder),path.resolve('data'));assert.ok(path.basename(folder).startsWith('audio-browser-'));await fs.rm(folder,{recursive:true,force:true})}});
