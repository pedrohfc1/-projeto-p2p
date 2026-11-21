import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {CallSoundTracker, soundPreferences, soundStorageSlot} from '../src/callSounds.ts';
import {SoundEffects} from '../src/soundEffects.ts';

const screen = (patch={}) => ({session:'session',ready:true,sharing:false,busy:false,peers:[],...patch});
const voice = (peers=[],patch={}) => ({
 joined:true,community:'group',channel:'voice',session:'session',authorization:'valid',
 media:{state:'connected'},participants:[{id:'self',self:true},...peers.map(id=>({id,self:false}))],
 ...patch,
});
const screenPeer = (sharing,connected=true,statusKnown=true) => ({id:'remote',name:'QA',sharing,connected,statusKnown});

test('a baseline with eight peers is silent apart from one local connection',()=>{
 const tracker=new CallSoundTracker();
 const initial=voice(Array.from({length:8},(_,i)=>'peer'+i));
 assert.deepEqual(tracker.observe(null,screen()),[]);
 assert.deepEqual(tracker.observe(initial,screen()),['call-connect']);
 assert.deepEqual(tracker.observe({...initial},screen()),[]);
 const later=voice([...initial.participants.slice(1).map(p=>p.id),'new']);
 assert.deepEqual(tracker.observe(later,screen()),['voice-user-join']);
 assert.deepEqual(tracker.observe(later,screen()),[]);
 assert.deepEqual(tracker.observe(initial,screen()),['voice-user-leave']);
 assert.deepEqual(tracker.observe(initial,screen()),[]);
 assert.deepEqual(tracker.observe(null,screen()),['call-disconnect']);
 assert.deepEqual(tracker.observe(null,screen()),[]);
});

test('reconnects, paused authorization and fresh snapshots do not create duplicate events',()=>{
 const tracker=new CallSoundTracker();
 const a=voice(['remote']);
 tracker.observe(a,screen({peers:[screenPeer(true)]}));
 assert.deepEqual(tracker.observe({...a,media:{state:'reconnecting'}},screen({peers:[screenPeer(false,false,false)]})),[]);
 assert.deepEqual(tracker.observe({...a},screen({peers:[screenPeer(true)]})),[]);
 assert.deepEqual(tracker.observe(voice([], {authorization:'paused'}),screen()),[]);
 assert.deepEqual(tracker.observe(voice(['new']),screen()),[]);
 assert.deepEqual(tracker.observe(voice(['new'],{session:'replacement'}),screen()),[]);
 assert.deepEqual(tracker.observe(voice(['new','next'],{session:'replacement'}),screen()),['voice-user-join']);
});

test('remote share transitions require verified status and announcements remain quiet during recovery',()=>{
 const tracker=new CallSoundTracker(), a=voice(['remote']);
 tracker.observe(a,screen({peers:[screenPeer(false)]}));
 assert.deepEqual(tracker.observe(a,screen({peers:[screenPeer(true,true,false)]})),[]);
 assert.deepEqual(tracker.observe(a,screen({peers:[screenPeer(true)]})),['stream-start']);
 assert.deepEqual(tracker.observe(a,screen({peers:[screenPeer(true)]})),[]);
 assert.deepEqual(tracker.observe(a,screen({peers:[screenPeer(false,false,false)]})),[]);
 assert.deepEqual(tracker.observe(a,screen({peers:[screenPeer(true)]})),[]);
 assert.deepEqual(tracker.observe(a,screen({peers:[screenPeer(false)]})),['stream-stop']);
 assert.deepEqual(tracker.observe(a,screen({peers:[screenPeer(false)]})),[]);
});

test('a share announced before entering is a baseline and roster departure ends access once',()=>{
 const tracker=new CallSoundTracker();
 assert.deepEqual(tracker.observe(voice(['remote']),screen({peers:[screenPeer(true)]})),['call-connect']);
 assert.deepEqual(tracker.observe(voice([]),screen({peers:[screenPeer(true)]})),['stream-stop','voice-user-leave']);
 assert.deepEqual(tracker.observe(voice([]),screen({peers:[screenPeer(true)]})),[]);
});

test('local share follows confirmed publication, ignores stale sessions and ends once when leaving',()=>{
 const tracker=new CallSoundTracker();
 tracker.observe(voice(),screen());
 assert.deepEqual(tracker.observe(voice(),screen({busy:true})),[]);
 assert.deepEqual(tracker.observe(voice(),screen({sharing:true,session:'earlier'})),[]);
 assert.deepEqual(tracker.observe(voice(),screen({sharing:true})),['local-share-start']);
 assert.deepEqual(tracker.observe(voice(),screen({sharing:true})),[]);
 assert.deepEqual(tracker.observe(voice(),screen()),['local-share-stop']);
 assert.deepEqual(tracker.observe(voice(),screen()),[]);
 tracker.observe(voice(),screen({sharing:true}));
 assert.deepEqual(tracker.observe(null,screen()),['local-share-stop','call-disconnect']);
 assert.deepEqual(tracker.observe(null,screen()),[]);
});

function fixture(raw=null) {
 const heard=[], volumes=[], output=[], cancelled=[], prefs=new Map([[soundStorageSlot,raw]]);
 const storage={getItem:key=>prefs.get(key),setItem:(key,value)=>prefs.set(key,value)};
 const player={
  play:async(cue,allowed)=>{if(!allowed())return false;heard.push(cue);return true},
  prepare:async()=>{}, volume:value=>volumes.push(value),
  output:value=>output.push(value), cancel:()=>cancelled.push(true),
 };
 return {heard,volumes,output,cancelled,prefs,storage,player,service:new SoundEffects(player,storage)};
}
const flush = () => new Promise(resolve=>setImmediate(resolve));

test('master/group preferences, 0–100 volume, selected local output and persistence',async()=>{
 const f=fixture('unparseable');
 assert.deepEqual(soundPreferences({volume:-5,people:'bad'}),{enabled:true,volume:0,people:true,sharing:true});
 assert.equal(soundPreferences({volume:Infinity}).volume,40);
 assert.equal(soundPreferences({volume:999}).volume,100);
 f.service.output('headset');
 f.service.update({enabled:false,volume:27});
 f.service.observe(voice(),screen());
 assert.equal(await f.service.preview(),false);
 assert.deepEqual(f.heard,[]);
 f.service.update({enabled:true,people:false,sharing:true});
 f.service.observe(voice(['remote']),screen());
 f.service.observe(voice(['remote']),screen({sharing:true}));
 await flush();
 assert.deepEqual(f.heard,['local-share-start']);
 assert.deepEqual(f.volumes,[.4,.27,.27]);
 assert.deepEqual(f.output,['headset']);
 assert.ok(f.cancelled.length);
 const reloaded=new SoundEffects(f.player,f.storage);
 assert.deepEqual(reloaded.get(),{enabled:true,volume:27,people:false,sharing:true});
 f.service.update({volume:0});
 f.service.observe(voice(['remote']),screen());
 await flush();
 assert.deepEqual(f.heard,['local-share-start']);
});

test('effects queued before a mute do not replay after re-enabling',async()=>{
 const f=fixture();
 let settle;
 f.player.play=async(cue,allowed)=>{
  await new Promise(resolve=>{settle=resolve});
  if(!allowed())return false;
  f.heard.push(cue);
  return true;
 };
 f.service.observe(voice(),screen());
 await flush();
 f.service.observe(voice(['remote']),screen());
 f.service.update({enabled:false});
 f.service.update({enabled:true});
 settle();
 await flush();
 assert.deepEqual(f.heard,[]);
});

test('all eight distinct packaged WAV effects are small, mono and short',async()=>{
 const names=['call-connect','call-disconnect','local-share-start','local-share-stop','stream-start','stream-stop','voice-user-join','voice-user-leave'];
 const hashes=new Set();
 for(const cue of names) {
  const bytes=await readFile(new URL('../src/assets/sounds/'+cue+'.wav',import.meta.url));
  assert.equal(bytes.subarray(0,4).toString(),'RIFF');
  assert.equal(bytes.subarray(8,12).toString(),'WAVE');
  assert.equal(bytes.readUInt16LE(22),1);
  assert.equal(bytes.readUInt32LE(24),16000);
  assert.ok(bytes.length>3000&&bytes.length<9000);
  hashes.add(createHash('sha256').update(bytes).digest('hex'));
 }
 assert.equal(hashes.size,8);
});
