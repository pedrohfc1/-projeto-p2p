// Run from repository root with P2P_PLAYWRIGHT_PATH set to existing QA tooling.
// P2P_VISUAL_DIST can point to a prebuilt frontend to test without Vite hot reload.
const {chromium}=require(process.env.P2P_PLAYWRIGHT_PATH||'playwright');const {spawn}=require('node:child_process');const path=require('node:path');const fs=require('node:fs/promises');const assert=require('node:assert/strict');let server,browser,debugPage;const output=path.resolve('../work/frontend-visual');
(async()=>{await fs.mkdir(output,{recursive:true});const visualDist=process.env.P2P_VISUAL_DIST;server=spawn(process.execPath,['node_modules/vite/bin/vite.js',...(visualDist?['preview','--outDir',path.resolve(visualDist)]:[]),'--host','127.0.0.1','--port','5193','--strictPort'],{cwd:path.resolve('frontend'),windowsHide:true,stdio:'ignore'});for(let i=0;i<80;i++){try{if((await fetch('http://127.0.0.1:5193')).ok)break}catch{}await new Promise(r=>setTimeout(r,100))}
browser=await chromium.launch({executablePath:process.env.P2P_BROWSER_EXECUTABLE||'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',headless:true});const page=await browser.newPage();debugPage=page;const errors=[];page.on('pageerror',e=>{errors.push(String(e));console.error('PAGE_ERROR',String(e))});
await page.addInitScript(()=>{window.__fixtureMode='empty';const state=()=>({unread:{'d:fixture-ana':3},communities:Array.from({length:12},(_,i)=>({id:'fixture-community-'+i,name:'Fixture · Comunidade '+i+' com nome extenso',icon:i%2?'🌿':'QA',members:[],channels:[]})),settings:{name:'Fixture local',mode:'integrated',bootstrap:[],relays:[],assisted:true},network:{id:'fixture-self',addresses:[],peers:[],reachability:'desconhecida',relayAddresses:0},connectivity:{mode:'integrated',detectedProviders:{},detected:false,ready:false,selected:null,ip:'',message:'Fixture isolada',adapters:[]},networkError:'',contacts:window.__fixtureMode==='empty'?[]:[{id:'fixture-ana',name:'Ângela com um nome bastante longo para testar o recorte',path:'offline'},{id:'fixture-bruno',name:'Bruno · fixture',path:'integrada: caminho não verificado'}],messages:window.__fixtureMode==='empty'?[]:[{id:'one',from:'fixture-ana',peer:'fixture-ana',text:'Mensagem longa de fixture. '.repeat(35),time:1794830400,state:'recebida'},{id:'two',from:'fixture-ana',peer:'fixture-ana',text:'Outra mensagem do mesmo autor.',time:1794830420,state:'recebida'},{id:'three',from:'fixture-self',peer:'fixture-ana',text:'Resposta local de fixture.',time:1794830450,state:'confirmada'}],notice:'Dados fictícios somente neste teste',event:''});window.go={main:{App:{State:async()=>state(),VoiceState:async()=>({joined:false,participants:[],media:{state:'idle',packetsIn:0,packetsOut:0}}),AudioPreferences:async()=>({input:'',output:'',monitorVolume:.3,closeToTray:false,listeners:{}}),SaveAudioPreferences:async()=>{},SetChatForeground:async()=>{},NotificationPreferences:async()=>({enabled:false,preview:false,muted:{}}),DiagnosticEvents:async()=>({events:[],last:0,dropped:0}),ActivityView:async()=>({overlays:{},reactions:{},typing:[]}),ChatAction:async()=>({}),Copy:async()=>{},ClearDiagnostics:async()=>0}}};navigator.mediaDevices.enumerateDevices=async()=>[]});
await page.addInitScript(()=>{window.__communityActions=[];window.__communityRecords={};const get=id=>window.__communityRecords[id]||(window.__communityRecords[id]={channels:[{id:'geral',name:'geral',kind:'text'},{id:'long',name:'canal-com-um-nome-muito-longo-para-validar',kind:'text'},{id:'voz',name:'Sala de voz de fixture',kind:'voice',limit:6,role:'member'}],messages:[{id:'fixture-one',sequence:100,author:'fixture-ana',name:'Ângela · fixture',text:'Mensagem de comunidade de fixture. '.repeat(40),time:1794830400,status:'recebida'},{id:'fixture-two',sequence:101,author:'fixture-ana',name:'Ângela · fixture',text:'Mensagem consecutiva de fixture.',time:1794830420,status:'recebida'}]});window.go.main.App.CommunityState=async(id,channel,before)=>{const v=get(id);return {community:{id,name:'Comunidade de fixture com nome extenso '+id,description:'Dados isolados para validar a interface',icon:'🌿',owner:id.endsWith('0')?'fixture-self':'fixture-owner',role:id.endsWith('0')?'owner':'member',sequence:101,members:[{id:'fixture-self',name:'Fixture local',role:id.endsWith('0')?'owner':'member',removed:false},{id:'fixture-ana',name:'Ângela · fixture',role:'member',removed:false}],channels:v.channels,archived:false,denied:!!window.__communityDenied,error:window.__communityFailure?'Sincronização indisponível nesta fixture':''},messages:window.__communityEmpty?[]:before?[{id:'older',sequence:1,author:'fixture-ana',name:'Ângela · fixture',text:'Histórico de fixture. '.repeat(300),time:1794744000,status:'recebida'}]:v.messages,hasMore:!before}};window.go.main.App.CommunityAction=async(id,kind,body)=>{window.__communityActions.push({id,kind,body});const v=get(id);if(kind==='channel.create')v.channels.push({id:'created',name:body.name,kind:body.channelKind});if(kind==='message.text')v.messages.push({id:'sent',sequence:102,author:'fixture-self',name:'Fixture local',text:body.text,time:1794830470,status:'confirmada'});return {id:'fixture-op',status:'confirmada',message:''}}});

await page.addInitScript(()=>{window.RTCPeerConnection=class{constructor(){this.iceGatheringState='complete';this.connectionState='connected';this.localDescription=null}addTransceiver(){return {sender:{track:null,replaceTrack:async()=>{},getParameters:()=>({encodings:[{}]}),setParameters:async()=>{}},setCodecPreferences(){}}}addTrack(){return {}}createOffer(){return Promise.resolve({type:'offer',sdp:'fixture only'})}setLocalDescription(d){this.localDescription=d;return Promise.resolve()}setRemoteDescription(){return Promise.resolve()}getStats(){return Promise.resolve(new Map())}close(){this.connectionState='closed'}};window.__mediaCaptureCalls=0;window.__voiceJoined=true;window.__voiceMuted=false;window.__voiceDeaf=false;window.__watch='';window.__screenEnded=false;window.__uiCalls=[];navigator.mediaDevices.getUserMedia=async()=>{window.__mediaCaptureCalls++;throw Error('No physical media in UI fixture')};Object.assign(window.go.main.App,{
 VoiceState:async()=>({joined:window.__voiceJoined,community:'fixture-community-0',channel:'voz',session:'isolated-ui-session',channelName:'Sala de voz de fixture',muted:window.__voiceMuted,deaf:window.__voiceDeaf,participants:[{id:'fixture-self',name:'Fixture local',self:true,state:'waiting'},{id:'fixture-ana',name:'Ângela · fixture',self:false,state:'connected',muted:true,deaf:true,media:{state:'connected',packetsIn:12,packetsOut:16}}],media:{state:'connected',transport:'fixture only',packetsIn:12,packetsOut:16,bytesIn:120,bytesOut:160}}),
 VoiceControls:async(m,d)=>{window.__voiceMuted=m;window.__voiceDeaf=d;window.__uiCalls.push('controls')},VoiceControlsForSession:async(s,m,d)=>{window.__voiceMuted=m;window.__voiceDeaf=d;window.__uiCalls.push('controls')},VoiceLeave:async()=>{window.__voiceJoined=false;window.__uiCalls.push('leave')},ScreenBackgroundAllowed:async()=>false,ScreenConnect:async()=>({type:'answer',sdp:'fixture only'}),ScreenHeartbeat:async()=>{},ScreenStop:async()=>{},ScreenPublish:async()=>{},ScreenWatch:async id=>{window.__watch=id;window.__uiCalls.push('watch:'+id)},ScreenState:async()=>({active:true,publishing:false,watching:window.__watch,playing:'',participants:window.__screenEnded?[]:['fixture-ana'],viewers:0,maxViewers:3,framesIn:0,framesOut:0,bytesIn:0,bytesOut:0,dropped:0,roundTripMs:0,bitrate:0,peers:window.__screenEnded?[]:[{id:'fixture-ana',name:'Ângela · fixture',sharing:true,connected:true,accepted:true}]}),ScreenAudioSources:async()=>[],ScreenAudioState:async()=>({capturing:false,connected:false,encoded:0,source:'',peers:{}}),ScreenAudioDisconnect:async()=>{},ScreenAudioStopCapture:async()=>{},ScreenAudioConnect:async()=>({type:'answer',sdp:'fixture only'})
})});
await page.addInitScript(()=>{
 const app=window.go.main.App;
 const state=app.ScreenState;
 window.__selectedVideo=null;
 window.__pickerCalls=0;
 window.__publications=[];
 window.__publishing=false;
 window.__screenBridgeConnections=0;
 window.__audioPreferences=null;
 window.__logsCopied='';
 window.__logsCleared=0;
 window.__endedPeers=[];
 Object.defineProperty(navigator,'clipboard',{value:{writeText:async value=>{window.__logsCopied=value}},configurable:true});
 app.SaveAudioPreferences=async prefs=>{window.__audioPreferences=prefs};
 app.CopyDiagnostics=async value=>{window.__logsCopied=value};
 app.DiagnosticEvents=async after=>({last:3,dropped:0,events:after?[]:[
  {id:1,time:Date.now(),level:'INFO',category:'network',component:'EasyTier',message:'Rota de teste estabelecida'},
  {id:2,time:Date.now(),level:'WARN',category:'voice',component:'Autorização',message:'Renovação de teste pendente'},
  {id:3,time:Date.now(),level:'INFO',category:'video',component:'Tela',message:'Vídeo sintético recebido'},
 ]});
 app.ClearDiagnostics=async()=>{window.__logsCleared++;return 3};
 const voiceState=app.VoiceState;
 app.VoiceState=async()=>{
  const view=await voiceState();
  return {...view,authorization:window.__voiceAuthorization||'valid',expires:Date.now()/1000+90,
   media:{...view.media,state:window.__voiceAuthorization==='paused'?'authorization-paused':view.media.state},
   participants:view.participants.concat({id:'fixture-bruno',name:'Bruno · fixture',self:false,state:'waiting'},{id:'fixture-davi',name:'Davi · fixture',self:false,state:'connected'})};
 };
 const remoteSource=document.createElement('canvas');
 remoteSource.width=1280;
 remoteSource.height=720;
 const ctx=remoteSource.getContext('2d');
 let step=0;
 setInterval(()=>{
  const color=ctx.createLinearGradient(0,0,1280,720);
  color.addColorStop(0,'#163756');
  color.addColorStop(1,'#122e3b');
  ctx.fillStyle=color;
  ctx.fillRect(0,0,1280,720);
  ctx.fillStyle='#54aca4';
  ctx.fillRect((step++*5)%1100,460,180,80);
  ctx.fillStyle='#d5ecef';
  ctx.font='36px sans-serif';
  ctx.fillText('Vídeo sintético para teste da interface',72,100);
 },33);
 const remoteStream=remoteSource.captureStream(30);
 window.__fixtureRemoteStream=remoteStream;
 const setRemoteDescription=window.RTCPeerConnection.prototype.setRemoteDescription;
 window.RTCPeerConnection.prototype.setRemoteDescription=async function(description){
  await setRemoteDescription.call(this,description);
  if(this===window.__fixtureVideoBridge) this.ontrack({track:remoteStream.getVideoTracks()[0],streams:[remoteStream]});
 };
 navigator.mediaDevices.getDisplayMedia=async()=>{
  window.__pickerCalls++;
  const canvas=document.createElement('canvas');
  canvas.width=640;canvas.height=360;
  canvas.getContext('2d').fillRect(0,0,640,360);
  const stream=canvas.captureStream(10);
  window.__selectedVideo=stream.getVideoTracks()[0];
  return stream;
 };
 const addTransceiver=window.RTCPeerConnection.prototype.addTransceiver;
 window.RTCPeerConnection.prototype.addTransceiver=function(kind,options){
  const t=addTransceiver.call(this,kind,options);
  if(kind==='video'){
   window.__fixtureVideoBridge=this;
   t.sender.replaceTrack=async track=>{t.sender.track=track};
  }
  return t;
 };
 app.ScreenConnect=async()=>{
  window.__screenBridgeConnections++;
  window.__publishing=false;
  return {type:'answer',sdp:'fixture only'};
 };
 app.ScreenPublish=async publishing=>{window.__publications.push(publishing);window.__publishing=publishing};
 app.ScreenState=async()=>{
  const screen=await state();
  const peers=(window.__screenEnded?[]:screen.peers.concat({id:'fixture-davi',name:'Davi · fixture',sharing:true,connected:true,accepted:true}))
   .filter(peer=>!window.__endedPeers.includes(peer.id));
  return {...screen,publishing:window.__publishing,playing:peers.some(peer=>peer.id===window.__watch)?window.__watch:'',peers};
 };
});
await page.goto('http://127.0.0.1:5193');
await page.getByRole('heading',{name:'Vamos começar uma conversa'}).waitFor();
await page.getByRole('button',{name:'Fechar aviso de microfone'}).click().catch(()=>{});
await page.getByRole('button',{name:'Abrir comunidade Fixture · Comunidade 0 com nome extenso'}).click();
await page.locator('.voice-roster .voice-person').first().waitFor();
const channel=page.locator('.voice-channel-heading>button').first();
await channel.click();
const room=page.locator('.voice-room-stage');
await room.waitFor();
const bar=room.locator('.voice-stage-controls');
const sidebar=room.locator('.voice-participant-sidebar');
const strip=room.getByRole('region',{name:'Miniaturas da sala'});
await strip.waitFor();
await bar.waitFor();
assert.equal(await room.locator('.voice-participant-tile').count(),4);
assert.equal(await page.locator('[data-settings-entry=true]').count(),1);
assert.equal(await page.getByRole('button',{name:/encerrar peetwopee/i}).count(),0);
assert.equal(await page.evaluate(()=>window.__watch),'');
assert.match(await room.locator('.voice-room-header').innerText(),/Sala de voz de fixture/);
assert.match(await room.locator('.voice-room-header').innerText(),/Comunidade de fixture/);
assert.equal(await channel.getAttribute('aria-current'),'true');

const records=[];
async function measure({screenshot=false,playing=false}={}){
 if(screenshot)await page.screenshot({path:path.join(output,`voice-${playing?'watching':'room'}-${await page.evaluate(()=>innerWidth)}.png`)});
 const boxes=await page.evaluate(()=>{
  const metrics=(selector)=>{
   const node=document.querySelector(selector);
   if(!node||!node.getClientRects().length)return null;
   const {left,right,top,bottom,width,height}=node.getBoundingClientRect();
   return {left,right,top,bottom,width,height};
  };
  const blockedActions=Array.from(document.querySelectorAll('.voice-stage-controls button:not(:disabled)')).filter(button=>{
   const {left,top,width,height}=button.getBoundingClientRect();
   return !button.contains(document.elementFromPoint(left+width/2,top+height/2));
  }).map(button=>button.getAttribute('aria-label'));
  return {width:innerWidth,height:innerHeight,scroll:document.documentElement.scrollWidth,blockedActions,
   workspace:metrics('.voice-workspace'), media:metrics('.voice-media'),
   bar:metrics('.voice-stage-controls'), dock:metrics('.profile-dock'),
   player:metrics('.screen-stage:not(.local-screen-stage)')};
 });
 assert.ok(boxes.scroll<=boxes.width,JSON.stringify(boxes));
 assert.deepEqual(boxes.blockedActions,[],JSON.stringify(boxes));
 for(const key of ['workspace','media','bar','dock']){
  const box=boxes[key];
  assert.ok(box?.width>0&&box.height>0&&box.left>=-1&&box.right<=boxes.width+1&&box.bottom<=boxes.height+1,JSON.stringify({key,boxes}));
 }
 assert.ok(boxes.bar.left>=boxes.media.left-1&&boxes.bar.right<=boxes.media.right+1,JSON.stringify(boxes));
 if(playing){
  const box=boxes.player;
  assert.ok(box?.width>boxes.media.width*.65&&box.left>=boxes.media.left-1&&box.right<=boxes.media.right+1&&box.top>=boxes.media.top-1&&box.bottom<=boxes.media.bottom+1,JSON.stringify(boxes));
  const fits=await page.locator('.screen-stage>video').evaluate(node=>getComputedStyle(node).objectFit);
  assert.equal(fits,'contain');
 }
 records.push({playing,...boxes});
}
for(const [width,height] of [[960,720],[1280,720],[1600,900],[1920,1080]]){
 await page.setViewportSize({width,height});
 await measure({screenshot:true});
}
await sidebar.waitFor();
assert.equal(await sidebar.locator('.voice-participant-row').count(),4);
assert.match(await sidebar.locator('.voice-participant-owner').innerText(),/Proprietário/);
assert.match(await sidebar.getByRole('region',{name:'Aguardando',exact:true}).innerText(),/Bruno · fixture/);
assert.equal(await sidebar.getByRole('region',{name:'Assistindo',exact:true}).count(),0);
const toggleSidebar=room.getByRole('button',{name:'Participantes da sala',exact:true});
await toggleSidebar.click();
await sidebar.waitFor({state:'hidden'});
await toggleSidebar.click();
await sidebar.waitFor();
const anaRow=sidebar.locator('.voice-participant-row').filter({hasText:'Ângela · fixture'});
await anaRow.click({button:'right'});
const userVolume=page.getByRole('slider',{name:'Volume do usuário'});
await userVolume.fill('1.5');
assert.equal(await userVolume.inputValue(),'1.5');
await page.getByRole('menuitemcheckbox',{name:'Silenciar localmente'}).click();
await page.waitForFunction(()=>window.__audioPreferences?.listeners['fixture-ana']?.muted===true);
assert.equal(await page.evaluate(()=>window.__audioPreferences.listeners['fixture-ana'].volume),1.5);
await room.getByRole('region',{name:'Miniaturas da sala'}).getByRole('button',{name:'Ver perfil de Ângela · fixture'}).click();
await page.getByRole('dialog').waitFor();
await page.keyboard.press('Escape');
await page.getByRole('dialog').waitFor({state:'hidden'});

await bar.getByRole('button',{name:'Silenciar microfone',exact:true}).click();
await bar.getByRole('button',{name:'Ativar microfone',exact:true}).waitFor();
await bar.getByRole('button',{name:'Ensurdecer áudio',exact:true}).click();
await bar.getByRole('button',{name:'Reativar áudio',exact:true}).waitFor();
assert.equal(await bar.getByRole('button',{name:'Câmera em breve'}).isDisabled(),true);
await bar.getByRole('button',{name:'Configurações rápidas de voz e vídeo',exact:true}).click();
const settings=page.getByRole('dialog',{name:'Central de configurações'});
await settings.getByRole('tab',{name:'Voz',exact:true,selected:true}).waitFor();
await settings.getByRole('button',{name:'Fechar configurações',exact:true}).click();
assert.equal(await page.evaluate(()=>window.__voiceJoined),true);

const share=bar.getByRole('button',{name:'Compartilhar tela',exact:true});
await share.click();
const preview=page.getByRole('dialog',{name:'Prévia antes de transmitir'});
await preview.waitFor();
const publish=preview.getByRole('button',{name:'Iniciar compartilhamento',exact:true});
await page.waitForFunction(()=>window.__pickerCalls===1&&window.__selectedVideo?.readyState==='live');
assert.equal(await page.evaluate(()=>window.__publications.filter(Boolean).length),0);
await publish.click();
await page.waitForFunction(()=>window.__publications.filter(Boolean).length===1);
await page.evaluate(()=>{window.__fixtureVideoBridge.connectionState='failed'});
await page.waitForFunction(()=>window.__screenBridgeConnections===2&&window.__publications.filter(Boolean).length===2);
assert.deepEqual(await page.evaluate(()=>[window.__pickerCalls,window.__selectedVideo.readyState,window.__voiceJoined]),[1,'live',true]);
await bar.getByRole('button',{name:/Parar compartilhamento/,exact:false}).click();
await page.waitForFunction(()=>!window.__publishing&&window.__selectedVideo.readyState==='ended');

const chooser=page.getByRole('dialog',{name:'Telas dos participantes'});
await bar.getByRole('button',{name:'Telas dos participantes',exact:true}).click();
await chooser.waitFor();
assert.match(await chooser.innerText(),/Ângela · fixture/);
assert.match(await chooser.innerText(),/Davi · fixture/);
assert.equal(await page.evaluate(()=>window.__uiCalls.some(call=>call.startsWith('watch:'))),false);
await page.keyboard.press('Escape');
await chooser.waitFor({state:'hidden'});
await anaRow.getByRole('button',{name:'Ver transmissão de Ângela · fixture'}).click();
await chooser.waitFor();
assert.equal(await page.evaluate(()=>window.__watch),'');
await chooser.getByRole('listitem').filter({hasText:'Ângela · fixture'}).getByRole('button',{name:'Assistir',exact:true}).click();
await page.waitForFunction(()=>window.__watch==='fixture-ana');
const player=page.locator('.screen-stage:not(.local-screen-stage)');
await player.waitFor();
const video=player.locator(':scope > video');
await page.waitForFunction(()=>document.querySelector('.screen-stage>video')?.videoWidth===1280);
const videoHandle=await video.elementHandle();
const samePlayer=()=>video.evaluate((node,previous)=>node===previous&&node.srcObject===window.__fixtureRemoteStream,videoHandle);
assert.ok(await samePlayer());
assert.match(await player.innerText(),/Ângela · fixture/);
assert.match(await player.innerText(),/AO VIVO/);
await player.getByText(/1280\s*[×x]\s*720|720p/).waitFor();
assert.match(await sidebar.getByRole('region',{name:'Assistindo',exact:true}).innerText(),/Fixture local/);
for(const [width,height] of [[1280,720],[1600,900],[1920,1080]]){
 await page.setViewportSize({width,height});
 await measure({screenshot:true,playing:true});
}

await player.hover();
const streamVolume=player.getByRole('slider',{name:'Volume do áudio da transmissão'});
await streamVolume.fill('38');
assert.equal(await streamVolume.inputValue(),'38');
await player.getByRole('button',{name:'Silenciar áudio da transmissão',exact:true}).click();
await player.getByRole('button',{name:'Reativar áudio da transmissão',exact:true}).click();
assert.equal(await streamVolume.inputValue(),'38');
assert.deepEqual(await page.evaluate(()=>[window.__audioPreferences.listeners['fixture-ana'].volume,window.__audioPreferences.listeners['fixture-ana'].muted,window.__voiceDeaf]),[1.5,true,true]);
await player.getByRole('button',{name:'Modo teatro',exact:true}).click();
await player.getByRole('button',{name:'Sair do modo teatro',exact:true}).waitFor();
assert.ok(await samePlayer());
await player.getByRole('button',{name:'Sair do modo teatro',exact:true}).click();
await player.getByRole('button',{name:'Tela cheia',exact:true}).click();
await page.waitForFunction(()=>document.fullscreenElement?.contains(document.querySelector('.screen-stage>video')));
assert.ok(await samePlayer());
await player.getByRole('button',{name:'Sair da tela cheia',exact:true}).click();
await page.waitForFunction(()=>!document.fullscreenElement);
await page.getByRole('button',{name:'geral',exact:true}).click();
await room.waitFor({state:'hidden'});
await player.waitFor();
assert.ok(await samePlayer());
const playbackTime=await video.evaluate(node=>node.currentTime);
await page.waitForFunction(time=>document.querySelector('.screen-stage>video')?.currentTime>time,playbackTime);
await channel.click();
await room.waitFor();
assert.ok(await samePlayer());
assert.deepEqual(await page.evaluate(()=>window.__uiCalls.filter(call=>call.startsWith('watch:'))),['watch:fixture-ana']);

const technical=room.locator('.voice-technical');
const details=technical.locator('summary[aria-label="Detalhes técnicos"]');
await details.click();
assert.match(await technical.innerText(),/fixture only/);
await details.click();
await room.getByRole('button',{name:'Abrir logs da chamada',exact:true}).click();
const logs=page.getByRole('region',{name:'Console de diagnóstico'});
await logs.waitFor();
for(const line of ['Rota de teste estabelecida','Renovação de teste pendente','Vídeo sintético recebido'])await logs.getByText(line,{exact:false}).waitFor();
await logs.getByRole('button',{name:'Copiar tudo',exact:true}).click();
await logs.getByRole('button',{name:'Confirmar cópia',exact:true}).click();
await page.waitForFunction(()=>window.__logsCopied.includes('Vídeo sintético recebido'));
await logs.getByRole('button',{name:'Limpar',exact:true}).click();
await page.waitForFunction(()=>window.__logsCleared===1);
await logs.getByText('Renovação de teste pendente',{exact:false}).waitFor({state:'hidden'});
await logs.getByRole('button',{name:'Fechar console',exact:true}).click();

await player.hover();
await player.getByRole('button',{name:'Parar de assistir',exact:true}).click();
await page.waitForFunction(()=>window.__watch==='');
await player.waitFor({state:'hidden'});
await room.locator('.voice-participant-tile').filter({hasText:'Davi · fixture'}).getByRole('button',{name:'Ver transmissão de Davi · fixture'}).click();
await chooser.waitFor();
await chooser.getByRole('listitem').filter({hasText:'Davi · fixture'}).getByRole('button',{name:'Assistir',exact:true}).click();
await page.waitForFunction(()=>window.__watch==='fixture-davi');
await player.waitFor();
await page.evaluate(()=>window.__endedPeers=['fixture-davi']);
await page.getByText('Transmissão encerrada ou acesso indisponível.',{exact:true}).waitFor();
await player.waitFor({state:'hidden'});
await page.evaluate(()=>{window.__screenEnded=true});
await room.getByText('Nenhuma transmissão ativa',{exact:true}).waitFor();
await page.evaluate(()=>{window.__voiceAuthorization='paused'});
await room.getByText(/aguardando autorização/i).first().waitFor();
assert.equal(await page.evaluate(()=>window.__voiceJoined),true);
await bar.getByRole('button',{name:'Desconectar chamada',exact:true}).click();
await bar.waitFor({state:'hidden'});
assert.equal(await page.evaluate(()=>window.__mediaCaptureCalls),0);
assert.deepEqual(errors,[]);
await fs.writeFile(path.join(output,'voice-results.json'),JSON.stringify({
 scope:'isolated frontend fixtures with local synthetic canvas tracks; no physical capture, remote hardware or external network',
 records,errors,calls:await page.evaluate(()=>window.__uiCalls),
},null,2));
console.log('PASS: voice stage and filmstrip layouts, participant sidebar and local preferences, call controls, explicit sharing/watching, video-only recovery, observed quality, stream volume isolation, theater/fullscreen/navigation with the same player, logs copy/clear, stream end, authorization and leave');

})().catch(async e=>{if(debugPage){await debugPage.screenshot({path:path.join(output,'failure.png')});await fs.writeFile(path.join(output,'failure.txt'),await debugPage.locator('body').innerText())};console.error(e);process.exitCode=1}).finally(async()=>{await browser?.close();server?.kill()});

