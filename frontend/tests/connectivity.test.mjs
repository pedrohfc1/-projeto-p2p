import {test} from 'node:test';
import assert from 'node:assert/strict';
import {vpnName,messageByteLength,canSendMessage} from '../src/connectivity.ts';
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
