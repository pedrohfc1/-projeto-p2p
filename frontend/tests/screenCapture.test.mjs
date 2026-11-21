import {test} from 'node:test';import assert from 'node:assert/strict';import {ScreenCapture,screenFailure,videoBitrate} from '../src/screenCapture.ts';
function source(){let stops=0;const track={contentHint:'',onended:null,applyConstraints:async()=>{},stop(){stops++}};return {stream:{getTracks:()=>[track],getVideoTracks:()=>[track]},track,get stops(){return stops}}}
function media(fn){Object.defineProperty(globalThis,'navigator',{configurable:true,value:{mediaDevices:{getDisplayMedia:fn}}})}
test('screen selection is explicit, video only, limited, local preview and cancel closes capture',async()=>{const src=source();let calls=0,opts;media(async o=>{calls++;opts=o;return src.stream});const c=new ScreenCapture(()=>{});assert.equal(calls,0);assert.equal(await c.select(),src.stream);assert.equal(opts.audio,false);assert.equal(opts.surfaceSwitching,'exclude');assert.equal(opts.systemAudio,'exclude');assert.equal(src.track.contentHint,'detail');c.stop();c.stop();assert.equal(src.stops,1)});
test('cancel while native selection is pending closes late source and never reopens it',async()=>{const src=source();let resolve;media(()=>new Promise(r=>resolve=r));const c=new ScreenCapture(()=>{});const pending=c.select();c.stop();resolve(src.stream);assert.equal(await pending,null);assert.equal(src.stops,1);assert.equal(c.stream,null)});
test('source ended releases capture once; denial does not retry another source',async()=>{const src=source();let ended=0;media(async()=>src.stream);const c=new ScreenCapture(()=>ended++);await c.select();src.track.onended();assert.equal(ended,1);assert.equal(src.stops,1);let calls=0;media(async()=>{calls++;throw {name:'NotAllowedError'}});await assert.rejects(c.select());assert.equal(calls,1);assert.match(screenFailure({name:'NotAllowedError'}),/negada/)});
test('adaptive video bitrate respects total fanout ceiling and reduces on congestion',()=>{assert.equal(videoBitrate(1500000,5,0,1500000),300000);assert.equal(videoBitrate(1500000,1,3,1000000),750000);assert.ok(videoBitrate(256000,5,5,64000)<=64000)});

test('1080p60 requests real capture constraints without cloning or manufacturing frames',async()=>{const src=source();let options,constraints;src.track.applyConstraints=async c=>{constraints=c};media(async o=>{options=o;return src.stream});const c=new ScreenCapture(()=>{});await c.select('1080p60');assert.equal(options.video.frameRate.ideal,60);assert.equal(constraints.width.max,1920);assert.equal(constraints.height.max,1080);assert.equal(constraints.frameRate.max,60);assert.equal(src.track.contentHint,'motion');c.stop()});

test('high-resolution constraint rejection lowers the existing capture without a second native chooser',async()=>{
 const src=source();
 let selections=0,attempts=[];
 src.track.applyConstraints=async constraints=>{attempts.push(constraints);if(constraints.width.max===1920)throw new Error('rejected')};
 media(async()=>{selections++;return src.stream});
 const capture=new ScreenCapture(()=>{});
 assert.equal(await capture.select('1080p60'),src.stream);
 assert.equal(capture.appliedProfile,'720p30');
 assert.equal(attempts[1].frameRate.max,30);
 assert.equal(selections,1);
 capture.stop();
});
