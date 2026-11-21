import test from 'node:test';import assert from 'node:assert/strict';
import {automaticSoundSource,separatePhysicalOutput} from '../src/automaticSound.ts';
import {memberSections} from '../src/memberPresentation.ts';
test('window audio requires exact unique title and never falls back to all output',()=>{
 const app={pid:12,name:'game',title:'Game - Match'};assert.deepEqual(automaticSoundSource('window',app.title,[app],false),{mode:'process',pid:12});
 for(const [surface,label,sources] of [['window','Game',[app]],['window',app.title,[app,{...app,pid:13}]],['browser',app.title,[app]],[undefined,app.title,[app]]])assert.throws(()=>automaticSoundSource(surface,label,sources,false));
});
test('default monitor loopback requires a proven distinct physical call output to prevent recaptured voice',()=>{
 const devices=[{kind:'audiooutput',deviceId:'default',groupId:'speaker'},{kind:'audiooutput',deviceId:'phones',groupId:'headset'},{kind:'audiooutput',deviceId:'alias',groupId:'speaker'}];
 assert.equal(separatePhysicalOutput('phones',devices),true);for(const id of ['', 'default','alias','missing'])assert.equal(separatePhysicalOutput(id,devices),false);
 assert.equal(separatePhysicalOutput('phones',devices.map(d=>({...d,groupId:''}))),false);
 assert.throws(()=>automaticSoundSource('monitor','Screen 1',[],false),/eco/);assert.deepEqual(automaticSoundSource('monitor','Screen 1',[],true),{mode:'device',pid:0});
});
test('members are grouped once by verified presence and ordered by role without mutation',()=>{
 const members=[{id:'a',role:'member'},{id:'b',role:'owner'},{id:'c',role:'moderator'},{id:'d',role:'member'},{id:'b',role:'owner'}];
 const result=memberSections(members,'a',['b','c'],['d']);assert.deepEqual(result.map(s=>[s.label,s.items.map(m=>m.id)]),[['Em voz',['d']],['Online',['b','c','a']]]);assert.equal(members.length,5);
});
