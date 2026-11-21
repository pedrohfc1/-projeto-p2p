import test from 'node:test';import assert from 'node:assert/strict';
import {activateVoice} from '../src/voiceActivation.ts';
test('voice primes independent capture/playback and HTML output synchronously, without system audio',async()=>{
 const calls=[];let allow;const permission=new Promise(r=>allow=r);
 const mic={state:'suspended',resume(){calls.push('mic');this.state='running';return Promise.resolve()}};
 const remote={state:'suspended',resume(){calls.push('remote');this.state='running';return Promise.resolve()}};
 const player={paused:true,play(){calls.push('speaker');this.paused=false;return Promise.resolve()}};
 const activation=activateVoice([mic,remote],player);assert.deepEqual(calls,['mic','remote','speaker']);
 allow();await permission;await activation;assert.equal(mic.state,'running');assert.equal(remote.state,'running');
});
test('blocked playback is recoverable by a fresh gesture resuming both contexts',async()=>{
 let allowed=false;const context={state:'suspended',resume(){if(allowed)this.state='running';return Promise.resolve()}};
 const player={paused:true,play(){if(!allowed)return Promise.reject(Error('NotAllowedError'));this.paused=false;return Promise.resolve()}};
 await assert.rejects(activateVoice([context],player));assert.equal(context.state,'suspended');allowed=true;await activateVoice([context],player);assert.equal(player.paused,false);
});
test('never-resolving autoplay and fulfilled-but-suspended resume do not report success',async()=>{
 const player={paused:false,play:()=>Promise.resolve()};await assert.rejects(activateVoice([{state:'suspended',resume:()=>Promise.resolve()}],player),/suspenso/);
 await assert.rejects(activateVoice([{state:'suspended',resume:()=>new Promise(()=>{})}],player,5),/ativação/);
});
