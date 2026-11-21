import {test} from 'node:test';import assert from 'node:assert/strict';import {watchStatus,screenVisibilityAction} from '../src/screenSession.ts';
test('transient media/bridge/focus recovery cannot turn a chosen stream into an ended transmission',()=>{const p={id:'owner',sharing:true,connected:true,accepted:true,statusKnown:true};assert.equal(watchStatus('owner',true,['owner'],[p]),'playing');assert.equal(watchStatus('owner',false,undefined,[]),'reconnecting');assert.equal(watchStatus('owner',true,['owner'],[{...p,connected:false,statusKnown:false,sharing:false}]),'reconnecting');assert.equal(watchStatus('owner',true,['owner'],[{...p,statusKnown:false,sharing:false}]),'reconnecting');assert.equal(watchStatus('owner',true,['owner'],[{...p,sharing:false}]),'ended');assert.equal(watchStatus('owner',true,['other'],[]),'ended');assert.equal(watchStatus('',true,[],[]),'idle')});

test('minimize preserves only explicit sessions with native stop controls; previews never survive hide',()=>{
 assert.equal(screenVisibilityAction(true,true,true,false),'keep');assert.equal(screenVisibilityAction(true,true,false,true),'keep');
 assert.equal(screenVisibilityAction(true,true,false,false),'stop');assert.equal(screenVisibilityAction(true,false,true,true),'stop');
 assert.equal(screenVisibilityAction(false,false,true,true),'restore');
});

test('recovery uses upstream progress, excludes static screens and deliberate voice prioritization',async()=>{
 const {detectVideoFault}=await import('../src/screenSession.ts');
 const base={publishing:true,viewing:false,watchers:1,sourceMuted:false,capture:0,encoded:0,sent:0,receiving:null,localReceived:null,decoded:null,rendered:null,sendQueueDepth:0,videoPausedForVoice:false};
 assert.equal(detectVideoFault(base),null);
 assert.equal(detectVideoFault({...base,sourceMuted:true}),'capture');
 assert.equal(detectVideoFault({...base,capture:30,encoded:0}),'encoder');
 assert.equal(detectVideoFault({...base,capture:30,encoded:30,sendQueueDepth:2,videoPausedForVoice:true}),null);
 assert.equal(detectVideoFault({...base,capture:30,encoded:30,sendQueueDepth:2}),'send');
 const viewer={...base,publishing:false,viewing:true};
 assert.equal(detectVideoFault({...viewer,receiving:30,localReceived:0}),'receive');
 assert.equal(detectVideoFault({...viewer,localReceived:30,decoded:0}),'decoder');
 assert.equal(detectVideoFault({...viewer,decoded:30,rendered:0}),'render');
 assert.equal(detectVideoFault({...viewer,decoded:30,rendered:null}),null);
});

test('recovery waits for sustained evidence and imposes a retry cap until healthy', async()=>{
 const {VideoRecoveryBudget}=await import('../src/screenSession.ts');
 const budget=new VideoRecoveryBudget();
 assert.equal(budget.observe('decoder',0,false),null);
 for(let t=1000;t<4000;t+=1000)assert.equal(budget.observe('decoder',t,false),null);
 assert.deepEqual(budget.observe('decoder',4000,false),{fault:'decoder',attempt:1});
 for(let t=5000;t<=20000;t+=1000)budget.observe('decoder',t,false);
 for(let t=21000;t<=36000;t+=1000)budget.observe('decoder',t,false);
 assert.equal(budget.observe('bridge',40000,false),null);
 budget.observe(null,41000,true);
 budget.observe(null,53000,true);
 assert.deepEqual(budget.observe('bridge',54000,false),{fault:'bridge',attempt:1});
});
