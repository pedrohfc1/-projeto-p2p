import test from 'node:test';import assert from 'node:assert/strict';
import {DiagnosticBuffer,redact,filtered,formatEvent,captureMediaConsole,diagnosticBuffer,copyDiagnosticText} from '../src/diagnostics.ts';
const row=(i=0)=>({time:1700000000000+i,level:'INFO',category:'voice',component:'microphone',message:'state '+i});
test('circular buffer retains newest 2000 immutable snapshots, monotonic keys and bounded strings',()=>{const b=new DiagnosticBuffer();for(let i=0;i<2500;i++)b.add(row(i));const rows=b.snapshot();assert.equal(rows.length,2000);assert.equal(rows[0].id,501);assert.equal(rows.at(-1).id,2500);b.add({...row(),message:'! '.repeat(5000)});assert.equal(rows[0].id,501);assert.ok(b.snapshot().at(-1).message.length<=768);b.clear();assert.equal(b.snapshot().length,0);assert.equal(b.add(row()).id,2502)});
test('redaction removes invite, key, bearer, peer and address without exposing media contents',()=>{const s=redact('token=hidden Bearer abc peetwopee://join/secret 10.2.3.4 12D3KooWabcdefghijklmnop '+ 'a'.repeat(80));for(const secret of ['hidden','Bearer abc','join/secret','10.2.3.4','abcdefghijklmnop','a'.repeat(80)])assert.ok(!s.includes(secret));assert.equal(redact('INFO\nnext'),'INFO next')});
test('filters distinguish category from errors and preserve timestamp milliseconds',()=>{const b=new DiagnosticBuffer();b.add(row());b.add({...row(),category:'video',level:'ERROR'});assert.equal(filtered(b.snapshot(),'all').length,2);assert.equal(filtered(b.snapshot(),'voice').length,1);assert.equal(filtered(b.snapshot(),'network').length,0);assert.equal(filtered(b.snapshot(),'errors')[0].category,'video');assert.match(formatEvent(b.snapshot()[0]),/\d\d:\d\d:\d\d\.000 INFO  \[microphone\]/)});
test('console capture accepts only actual media event names and allowed structured counters',()=>{diagnosticBuffer.clear();const original=console.info;console.info=()=>{};captureMediaConsole();console.info('chat_text',{text:'never capture private text'});console.info('voice_codec_io',{bytes:300,packets:15,token:'secret',text:'private content'});const r=diagnosticBuffer.snapshot();assert.equal(r.length,1);assert.match(r[0].message,/packets=15/);assert.ok(!r[0].message.includes('private'));assert.ok(!r[0].message.includes('secret'));console.info=original});

test('confirmed copy uses the browser user gesture, falls back to native and reports failure honestly',async()=>{const calls=[];await copyDiagnosticText('safe',async()=>calls.push('native'),async s=>calls.push('browser:'+s));assert.deepEqual(calls,['browser:safe']);await copyDiagnosticText('safe',async s=>calls.push('native:'+s),async()=>{throw Error('Denied')});assert.equal(calls.at(-1),'native:safe');await assert.rejects(copyDiagnosticText('safe',async()=>{throw Error('Clipboard unavailable')}),/unavailable/)});

test('authorization filter isolates structured voice authorization events',()=>{
 const b=new DiagnosticBuffer();
 b.add({...row(),message:'voice_authorization result=renewed'});
 b.add({...row(),message:'Mídia pausada durante renovação da autorização'});
 b.add({...row(),message:'Opus RX'});
 b.add({...row(),category:'network',message:'authorization'});
 assert.equal(filtered(b.snapshot(),'authorization').length,2);
});
