import {test} from 'node:test';import assert from 'node:assert/strict';import {VideoTelemetry} from '../src/videoTelemetry.ts';
test('reports counter deltas, not desired encoder FPS or bitrate; resets cannot become negative',()=>{const t=new VideoTelemetry(),network={framesOut:5,framesIn:7,bytesOut:1000,bytesIn:500};const rows=[{id:'e',type:'outbound-rtp',kind:'video',framesEncoded:10},{id:'d',type:'inbound-rtp',kind:'video',framesDecoded:8}];let s=t.sample(rows,network,1000);assert.equal(s.capture,null);assert.equal(s.encode,null);s=t.sample([{...rows[0],framesEncoded:70},{...rows[1],framesDecoded:38}],{framesOut:65,framesIn:37,bytesOut:1001000,bytesIn:500500},2000);assert.equal(s.encode,60);assert.equal(s.sent,60);assert.equal(s.received,30);assert.equal(s.decode,30);assert.equal(s.uploadMbps,8);assert.equal(s.capture,null);assert.equal(s.efficientEncoder,null);s=t.sample(rows,network,3000);assert.equal(s.sent,null);assert.equal(s.encode,null)});

test('viewer resolution uses decoder dimensions, not requested profile',()=>{const t=new VideoTelemetry();const net={framesOut:0,framesIn:0,bytesOut:0,bytesIn:0};assert.equal(t.sample([],net,1000).width,null);const measured=t.sample([{id:'in',type:'inbound-rtp',kind:'video',frameWidth:960,frameHeight:540}],net,2000);assert.equal(measured.width,960);assert.equal(measured.height,540)});

test('codec processing times are measured counter deltas per frame, missing times stay unavailable',()=>{const t=new VideoTelemetry();const n={framesOut:0,framesIn:0,bytesOut:0,bytesIn:0};t.sample([{id:'out',type:'outbound-rtp',kind:'video',framesEncoded:20,totalEncodeTime:.2},{id:'in',type:'inbound-rtp',kind:'video',framesDecoded:20,totalDecodeTime:.1}],n,1000);const r=t.sample([{id:'out',type:'outbound-rtp',kind:'video',framesEncoded:40,totalEncodeTime:.4},{id:'in',type:'inbound-rtp',kind:'video',framesDecoded:40,totalDecodeTime:.2}],n,2000);assert.equal(r.encodeMs,10);assert.equal(r.decodeMs,5);assert.equal(new VideoTelemetry().sample([],n,1).encodeMs,null)});

test('reports actual stage dimensions, negotiated codec and hardware hint independently',()=>{
 const telemetry=new VideoTelemetry();
 const counters={framesOut:0,framesIn:0,bytesOut:0,bytesIn:0};
 const rows=[
  {id:'capture',type:'media-source',kind:'video',width:1600,height:900,frames:20},
  {id:'encode',type:'outbound-rtp',kind:'video',frameWidth:1280,frameHeight:720,framesEncoded:11,codecId:'vp8',encoderImplementation:'libvpx',powerEfficientEncoder:false},
  {id:'vp8',type:'codec',mimeType:'video/VP8'},
  {id:'decode',type:'inbound-rtp',kind:'video',frameWidth:854,frameHeight:480,framesReceived:6,framesDecoded:4},
 ];
 telemetry.sample(rows,counters,1000);
 rows[0].frames+=24;rows[1].framesEncoded+=15;rows[3].framesReceived+=12;
 const actual=telemetry.sample(rows,counters,2000);
 assert.deepEqual([actual.captureWidth,actual.captureHeight,actual.encodeWidth,actual.encodeHeight,actual.width,actual.height],[1600,900,1280,720,854,480]);
 assert.deepEqual([actual.capture,actual.encode,actual.localReceived,actual.decode],[24,15,12,0]);
 assert.deepEqual([actual.codec,actual.encoder,actual.efficientEncoder,actual.decoder,actual.efficientDecoder],['video/VP8','libvpx',false,null,null]);
});
