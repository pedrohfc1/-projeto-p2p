import {test} from 'node:test';
import assert from 'node:assert/strict';
import {canManage,canPost,roleName,textBytes} from '../src/communityPolicy.ts';
test('papéis refletem estado verificado e revogação bloqueia controles',()=>{
 assert.equal(canManage('owner'),true);assert.equal(canManage('moderator'),true);assert.equal(canManage('member'),false);
 assert.equal(canManage('owner',true),false);assert.equal(canManage('moderator',false,true),false);
 assert.equal(roleName('member'),'Membro');assert.equal(roleName(''),'Ingresso pendente');
});
test('envio de texto exige canal de texto e participação ativa',()=>{
 assert.equal(canPost('member','text',false,false,false),true);
 for(const args of [['member','voice',false,false,false],['member','text',true,false,false],['member','text',false,true,false],['member','text',false,false,true],['','text',false,false,false]])assert.equal(canPost(...args),false);
 assert.equal(textBytes('🇧🇷'),8);
});
