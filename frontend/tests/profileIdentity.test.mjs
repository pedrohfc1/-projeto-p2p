import {test} from 'node:test';
import assert from 'node:assert/strict';
import {cleanName, fontId, imageAssetURL, imageInputLimit, inspectImage, squareCrop} from '../src/profileIdentityModel.ts';
import {ProfileIdentityStore} from '../src/profileIdentityStore.ts';

const digest='ab'.repeat(32);
const photo={mime:'image/png',data:'iVBORw0KGgo='};
const profile={peerId:'p',displayName:'Pessoa',displayFontId:'space-grotesk',avatarHash:digest,avatarMime:photo.mime,revision:2,updatedAt:100};

function pngHeader(width,height) {
 const bytes=new Uint8Array([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82,0,0,0,0,0,0,0,0]);
 const view=new DataView(bytes.buffer);
 view.setUint32(16,width);
 view.setUint32(20,height);
 return bytes.buffer;
}

test('name validation preserves normal Unicode, trims unsafe controls and clamps code points',()=>{
 assert.equal(cleanName(' \u0000João\u202e 💙 あ\u0007 '),'João 💙 あ');
 assert.equal(Array.from(cleanName('🙂'.repeat(40))).length,32);
 assert.equal(cleanName('\u0007 \n'),'');
 assert.equal(fontId('space-grotesk'),'space-grotesk');
 assert.equal(fontId('uploaded-remote-font'),'inter');
});

test('image headers are checked before decode and only supported square output assets get safe data URLs',()=>{
 assert.deepEqual(inspectImage(pngHeader(400,200)),{mime:'image/png',width:400,height:200});
 assert.throws(()=>inspectImage(pngHeader(50_000,50_000)),/Dimensões/);
 assert.throws(()=>inspectImage(new ArrayBuffer(imageInputLimit+1)),/5 MB/);
 assert.throws(()=>inspectImage(new TextEncoder().encode('<svg xmlns="http://www.w3.org/2000/svg"><image/></svg>').buffer),/Arquivo inválido/);
 assert.equal(imageAssetURL(photo),'data:image/png;base64,iVBORw0KGgo=');
 assert.equal(imageAssetURL({...photo,mime:'image/svg+xml'}),undefined);
 assert.equal(imageAssetURL({...photo,data:'<img src=x>'}),undefined);
});

test('square crop keeps the source rectangle within image bounds at arbitrary pan and zoom',()=>{
 assert.deepEqual(squareCrop(440,220,{zoom:1,x:0,y:0}),{x:110,y:0,size:220});
 assert.deepEqual(squareCrop(440,220,{zoom:1,x:100,y:0}),{x:220,y:0,size:220});
 assert.deepEqual(squareCrop(220,440,{zoom:2,x:-100,y:100}),{x:0,y:330,size:110});
});

test('saving updates visible profiles immediately and replayed revisions cannot evict the current avatar',async()=>{
 const store=new ProfileIdentityStore(()=>({SaveProfile:async()=>profile}));
 let updates=0;
 const unsubscribe=store.subscribeProfile('p',()=>updates++);
 await store.save({displayName:'Pessoa',displayFontId:'space-grotesk',avatar:photo});
 assert.equal(updates,1);
 assert.equal(store.profile('p')?.displayName,'Pessoa');
 assert.equal(store.asset(digest),imageAssetURL(photo));
 store.publish({...profile,revision:1,displayName:'Retransmissão'});
 assert.equal(updates,1);
 assert.equal(store.profile('p')?.displayName,'Pessoa');
 unsubscribe();
 store.stop();
});

test('one read per hash for multiple interested components and no read of unreferenced assets',async()=>{
 const reads=[];
 const store=new ProfileIdentityStore(()=>({ProfileAsset:async key=>{reads.push(key);return photo}}));
 const ready=new Promise(resolve=>store.subscribeAsset(digest,resolve));
 const unsubscribe=store.subscribeAsset(digest,()=>{});
 store.subscribeAsset('../untrusted.png',()=>assert.fail('invalid hash must fail before IPC'));
 store.start();
 try {
  await ready;
  assert.deepEqual(reads,[digest]);
  assert.equal(store.asset(digest),imageAssetURL(photo));
 } finally {
  unsubscribe();
  store.stop();
 }
});
