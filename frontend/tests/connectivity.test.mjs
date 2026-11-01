import {test} from 'node:test';
import assert from 'node:assert/strict';
import {vpnName,messageByteLength,canSendMessage,needsInviteChoice,adaptersForMode} from '../src/connectivity.ts';
test('VPNs independentes e compatibilidade do modo nativo',()=>{
 assert.equal(vpnName('radmin'),'Radmin VPN');assert.equal(vpnName('hamachi'),'Hamachi');
 assert.equal(vpnName('interface'),'Interface manual');assert.equal(vpnName('auto'),'Automatico (libp2p)');
});
test('limite de mensagem em bytes UTF-8, inclusive caracteres multibyte',()=>{
 assert.equal(messageByteLength('  ação  '),6);
 assert.equal(canSendMessage('😀'.repeat(1024),'peer',false),true);
 assert.equal(canSendMessage('😀'.repeat(1025),'peer',false),false);
 assert.equal(canSendMessage('a'.repeat(4096),'peer',false),true);
 assert.equal(canSendMessage('a'.repeat(4097),'peer',false),false);
});
test('bloqueia envio vazio, sem contato ou durante operação',()=>{
 assert.equal(canSendMessage('   ','peer',false),false);
 assert.equal(canSendMessage('texto','',false),false);
 assert.equal(canSendMessage('texto','peer',true),false);
});

test('convite nativo exige escolha explícita quando VPN detectada, inclusive ambas',()=>{
 assert.equal(needsInviteChoice('auto',{radmin:true,hamachi:true}),true);
 assert.equal(needsInviteChoice('auto',{radmin:true,hamachi:false}),true);
 assert.equal(needsInviteChoice('auto',{}),false);
 assert.equal(needsInviteChoice('radmin',{radmin:true,hamachi:true}),false);
});
test('seletor Radmin não oferece adaptador Hamachi/LAN',()=>{
 const items=[{id:'h',provider:'hamachi'},{id:'r',provider:'radmin'},{id:'lan',provider:'unknown'}];
 assert.deepEqual(adaptersForMode(items,'radmin').map(x=>x.id),['r']);
 assert.deepEqual(adaptersForMode(items,'hamachi').map(x=>x.id),['h']);
 assert.equal(adaptersForMode(items,'interface').length,3);
});
