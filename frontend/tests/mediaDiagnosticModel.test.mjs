import test from 'node:test';import assert from 'node:assert/strict';import {SessionSeries,audioSample,numberMetric,percentile,evidence,sessionArchive,importLabSummary,importDiagnosticReport} from '../src/mediaDiagnosticModel.ts';
const snapshot=(time=1000,tx=0,rx=0)=>({time,session:'pseudonymous',mode:'integrated',authorization:'valid',expires:10,joined:true,bridgePackets:tx,peers:[{alias:'abcdef012345',connections:1,streams:1,path:'desconhecido',transport:'libp2p',rttMs:20,rttSource:'sonda',openMs:12,openAttempts:1,openFailures:0,timeouts:0,media:{state:'connected',transport:'libp2p-stream',packetsIn:rx,packetsOut:tx,bytesIn:rx*10,bytesOut:tx*10,queueDelayMs:0,jitterMs:null,packetsLost:null}}]});
test('bounded 60-second series, no false UDP loss/DTX interruptions, exact deltas and no old session contamination',()=>{const s=new SessionSeries();for(let i=0;i<100;i++)s.append(snapshot(1000+i*1000,i,i),null,null);assert.equal(s.points.length,61);assert.equal(s.points.at(-1).bitrate,.16);assert.equal(s.summary().interruptions,0);assert.equal(s.points.at(-1).jitter,null);assert.equal(s.summary().noRxSeconds,0);s.append({...snapshot(102000),session:'next'},null,null);assert.equal(s.points.length,1);assert.ok(sessionArchive.length<=8)});
test('counter resets and timestamps never generate negative or invented throughput',()=>{const s=new SessionSeries();s.append(snapshot(1000,20,20),null,null);const p=s.append(snapshot(2000,0,0),null,null);assert.equal(p.bitrate,null);assert.equal(percentile([1,2,3,4,5,6,7,8,9,10]),10);assert.equal(numberMetric(NaN),null);assert.equal(numberMetric(-1),null)});
test('pipeline fields absent from WebRTC remain unknown, encoded and decoded are independent',()=>{const p=audioSample([{type:'outbound-rtp',kind:'audio',packetsSent:9},{type:'inbound-rtp',kind:'audio',totalSamplesReceived:48000}],{at:1000,available:true,capture:true,processing:'running',mode:'ptt',mute:false,deaf:false,outputSamples:0,outputContext:'running',playbackPaused:false,gate:'n/d'});assert.equal(p.encodedFrames,null);assert.equal(p.decodedFrames,null);assert.equal(p.bridgePackets,9);assert.equal(p.decodedSamples,48000);assert.equal(p.captureSeconds,null)});
test('evidence distinguishes blocked playback, authorization and speculation without assigning blame',()=>{const s=new SessionSeries();const p=s.append(snapshot(1000,2,2),null,null);assert.deepEqual(evidence(p),[]);p.snapshot.authorization='paused';assert.match(evidence(p).join(' '),/autorização/);assert.ok(!evidence(p).join(' ').includes('EasyTier'))});

test('lab import accepts numeric summaries only and strips credentials, identifiers and arbitrary text',()=>{assert.equal(importLabSummary({mode:'tailscale-lab',at:Date.now(),seconds:600,samples:10}),null);const summary=importLabSummary({mode:'tailscale-lab',at:Date.now(),seconds:10,samples:10,rttMean:12,rttP95:15,credential:'secret',peer:'private identity',conditions:'malicious text'});assert.equal(summary.rttMean,12);assert.ok(!JSON.stringify(summary).includes('secret'));assert.ok(!JSON.stringify(summary).includes('private identity'));assert.ok(!JSON.stringify(summary).includes('malicious text'))});

test('cached snapshots cannot duplicate RTT samples or distort P95',()=>{const s=new SessionSeries();s.append(snapshot(),null,null);s.append(snapshot(),null,null);assert.equal(s.points.length,1);s.append(snapshot(900),null,null);assert.equal(s.points.length,1)});

test('time without received packets is observed even before the first packet, without inferring loss',()=>{const s=new SessionSeries();s.append(snapshot(1000,0,0),null,null);s.append(snapshot(21000,0,0),null,null);assert.equal(s.summary().noRxSeconds,20);assert.equal(s.summary().interruptions,0);assert.equal(s.points.at(-1).jitter,null)});

test('audio buffer averages only rows with both WebRTC counters',()=>{
 const audio=audioSample([
  {type:'inbound-rtp',kind:'audio',jitterBufferDelay:.024,jitterBufferEmittedCount:12,concealmentEvents:2},
  {type:'inbound-rtp',kind:'audio',jitterBufferDelay:.042,jitterBufferEmittedCount:3,concealmentEvents:3},
  {type:'inbound-rtp',kind:'audio',jitterBufferDelay:2},
 ],{});
 assert.ok(Math.abs(audio.jitterBufferMeanMs-4.4)<1e-9);
 assert.equal(audio.concealmentEvents,5);
 assert.equal(audioSample([],{}).jitterBufferMeanMs,null);
});

test('equivalent load import accepts capped data and strips unrelated fields',()=>{
 const valid={mode:'tailscale-lab',testType:'limited-load',at:Date.now(),seconds:10,samples:38,rttMean:17,rttP95:20,kbps:121.6,variationMs:2,credential:'secret'};
 const summary=importLabSummary(valid);
 assert.equal(summary.testType,'limited-load');
 assert.equal(summary.kbps,121.6);
 assert.equal(summary.interruptions,null);
 assert.ok(!JSON.stringify(summary).includes('secret'));
 assert.equal(importLabSummary({...valid,kbps:129}),null);
 assert.equal(importLabSummary({...valid,testType:'other'}),null);
});

test('half-hour history retains one compact sample every five seconds and remains exportable',()=>{
 const series=new SessionSeries();
 for(let i=0;i<2400;i++){
  const sample=snapshot(1000+i*1000,i,i);
  sample.cpuPercent=15;
  sample.rssBytes=104857600;
  series.append(sample,null,null);
 }
 assert.equal(series.history.length,360);
 assert.equal(series.points.length,61);
 assert.ok(series.history.at(-1).time-series.history[0].time<=30*60_000);
 assert.ok(series.history.at(-1).time-series.history[0].time>29*60_000);
 assert.equal(series.history.at(-1).cpuPercent,15);
 assert.equal(series.history.at(-1).voiceRx,2399);
 assert.ok(!('snapshot' in series.history[0]));
 const report=series.report();
 assert.equal(report.history30Minutes.intervalMs,5000);
 assert.equal(report.history30Minutes.summary.samples,360);
 assert.ok(Buffer.byteLength(JSON.stringify(report))<2*1024*1024);
});

test('time gaps remain gaps, and recent observation does not clear half-hour history',()=>{
 const series=new SessionSeries();
 const a=snapshot(1000,1,1);
 const b=snapshot(61_000,2,2);
 const c=snapshot(62_000,3,3);
 series.append(a,null,null);
 series.append(b,null,null);
 series.append(c,null,null);
 assert.deepEqual(series.history.map(point=>point.time),[1000,62000]);
 assert.equal(series.summary(61_000).samples,2);
 assert.equal(series.report().history30Minutes.points.length,2);
});

test('transport report import uses only permitted numeric summary fields',()=>{
 const data={version:2,history30Minutes:{summary:{at:Date.now(),mode:'radmin',seconds:1700,samples:340,rttMean:18,rttP95:25,videoTxMbps:1.2,videoSendFps:30,brightness:'private'},points:[{alias:'private'}]}};
 const summary=importDiagnosticReport(data);
 assert.equal(summary.samples,340);
 assert.equal(summary.videoSendFps,30);
 assert.ok(!JSON.stringify(summary).includes('private'));
 data.history30Minutes.summary.samples=361;
 assert.equal(importDiagnosticReport(data),null);
});
