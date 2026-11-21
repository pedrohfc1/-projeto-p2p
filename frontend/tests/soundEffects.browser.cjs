// Run from the repository root with P2P_PLAYWRIGHT_PATH set to the QA tooling.
const assert = require('node:assert/strict');
const {spawn} = require('node:child_process');
const {once} = require('node:events');
const {chromium} = require(process.env.P2P_PLAYWRIGHT_PATH || 'playwright');
const origin = 'http://127.0.0.1:5201';
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
let browser;
let server;

async function run() {
 server = spawn(process.execPath, ['node_modules/vite/bin/vite.js','--host','127.0.0.1','--port','5201','--strictPort'], {
  cwd:'frontend',windowsHide:true,stdio:'ignore',
 });
 for (let attempt=0; attempt<100; attempt++) {
  if (server.exitCode!==null) throw Error('Vite de teste encerrou.');
  try { if ((await fetch(origin)).ok) break; } catch {}
  await sleep(100);
 }
 browser = await chromium.launch({
  executablePath:process.env.P2P_BROWSER_EXECUTABLE || 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
  headless:true,
 });
 const page = await browser.newPage();
 const pageErrors = [];
 page.on('pageerror',error=>pageErrors.push(String(error)));
 await page.route(origin+'/sound-fixture', route=>route.fulfill({
  contentType:'text/html',
  body:`<button>Executar efeitos locais</button><script type="module">
   import {callSounds} from '/src/soundEffects.ts';
   import {diagnosticBuffer} from '/src/diagnostics.ts';
   window.events = () => diagnosticBuffer.snapshot().filter(event=>event.message.startsWith('sound_event_played')).map(event=>event.message.split(' ').at(-1));
   window.outputElements = [];
   window.sources = [];
   const LocalAudio = window.Audio;
   window.Audio = function() {
    const element = new LocalAudio();
    window.outputElements.push(element);
    return element;
   };
   const start = AudioBufferSourceNode.prototype.start;
   AudioBufferSourceNode.prototype.start = function(time) {
    window.sources.push({duration:this.buffer.duration, time});
    return start.call(this,time);
   };
   window.voice = {joined:true,community:'room-group',channel:'room-channel',session:'room-session',authorization:'valid',media:{state:'connected'},participants:[{id:'self',self:true},...Array.from({length:8},(_,n)=>({id:'remote-'+n,self:false}))]};
   window.screen = {session:'room-session',ready:true,sharing:false,busy:false,peers:[{id:'remote-0',sharing:false,connected:true,statusKnown:true}]};
   window.sample = () => callSounds.observe(window.voice,window.screen);
   window.setSound = patch => callSounds.update(patch);
   document.querySelector('button').onclick = () => {
    callSounds.update({volume:27});
    callSounds.prepare();
    window.sample();
   };
  </script>`,
 }));
 await page.goto(origin+'/sound-fixture');
 await page.waitForFunction(()=>typeof window.sample==='function');
 await page.getByRole('button',{name:'Executar efeitos locais'}).click();
 await page.waitForFunction(()=>window.events().length===1);
 assert.deepEqual(await page.evaluate(()=>window.events()),['call-connect']);
 assert.equal(await page.evaluate(()=>window.outputElements.length),1);
 assert.equal(await page.evaluate(()=>window.outputElements[0].volume),.27);
 assert.equal(await page.evaluate(()=>window.outputElements[0].paused),false);

 await page.evaluate(()=>{
  window.voice.participants.push({id:'new',self:false});
  window.sample();
  window.sample();
 });
 await page.waitForFunction(()=>window.events().includes('voice-user-join'));
 await page.evaluate(()=>{
  window.voice.participants.pop();
  window.sample();
  window.sample();
 });
 await page.waitForFunction(()=>window.events().includes('voice-user-leave'));

 await page.evaluate(()=>{
  window.screen.peers[0].sharing=true;
  window.sample();
  window.screen.peers[0].connected=false;
  window.sample();
  window.screen.peers[0].connected=true;
  window.sample();
  window.screen.peers[0].sharing=false;
  window.sample();
  window.sample();
  window.screen.sharing=true;
  window.sample();
  window.screen.sharing=false;
  window.sample();
 });
 await page.waitForFunction(()=>window.events().length===7);
 const expected=['call-connect','voice-user-join','voice-user-leave','stream-start','stream-stop','local-share-start','local-share-stop'];
 assert.deepEqual(await page.evaluate(()=>window.events()),expected);
 assert.equal(await page.evaluate(()=>window.sources.length),expected.length);

 await page.evaluate(()=>{
  window.setSound({enabled:false,volume:62});
  window.screen.sharing=true;
  window.sample();
  window.voice.joined=false;
  window.sample();
 });
 await page.waitForTimeout(450);
 assert.deepEqual(await page.evaluate(()=>window.events()),expected);
 assert.deepEqual(await page.evaluate(()=>JSON.parse(localStorage.getItem('peetwopee.sound-effects.v1'))),{enabled:false,volume:62,people:true,sharing:true});
 assert.equal(await page.evaluate(()=>window.outputElements[0].volume),.62);
 assert.deepEqual(pageErrors,[]);
 console.log('PASS: original WAVs decoded in Edge, one local playback route, verified events, deduplicated reconnection, disabled effects and applied volume.');
}

run().catch(error => {
 console.error(error);
 process.exitCode=1;
}).finally(async()=>{
 if (browser) await browser.close();
 if (server && server.exitCode===null) {
  server.kill();
  await once(server,'exit').catch(()=>{});
 }
});
