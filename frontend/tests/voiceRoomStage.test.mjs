import test from 'node:test';
import assert from 'node:assert/strict';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {build} from 'esbuild';
import {createRequire} from 'node:module';
import {mkdtemp,writeFile,rm} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import path from 'node:path';

const require=createRequire(import.meta.url);
const folder=await mkdtemp(fileURLToPath(new URL('../node_modules/.voice-stage-',import.meta.url)));
const file=path.join(folder,'stage.cjs');
let VoiceRoomStage;
try {
 const result=await build({
  entryPoints:[fileURLToPath(new URL('../src/VoiceRoomStage.tsx',import.meta.url))],
  bundle:true,write:false,platform:'node',format:'cjs',external:['react'],
  plugins:[{name:'isolated-presentation-fixture',setup(b){
   b.onLoad({filter:/[\\/]Voice\.tsx$/},()=>({contents:'export const useVoice=()=>globalThis.__stageVoice;',loader:'js'}));
  }}],
 });
 await writeFile(file,result.outputFiles[0].text);
 ({VoiceRoomStage}=require(file));
} finally {
 await rm(folder,{recursive:true,force:true});
}

const props={group:'fixture-group',channel:'fixture-voice',name:'Sala de fixture',community:'Amigos da fixture',owner:'self'};

function render({joined=true,authorization='valid',theater=false,disabled=false}={}) {
 globalThis.__stageVoice={
  speaking:{},
  view:{joined,community:props.group,channel:props.channel,authorization,
   participants:[{id:'self',name:'Local',self:true,state:'waiting',muted:true},{id:'remote',name:'Remoto',self:false,state:'connected'},{id:'remote',name:'Duplicado',self:false,state:'connected'}],
   media:{state:authorization==='paused'?'authorization-paused':'connected',packetsOut:10,packetsIn:10},
  },
  busy:false,screen:{sharing:false,ready:true,watching:'remote',playing:'remote',theater,quality:{width:1280,height:720,fps:29.6},peers:[{id:'remote',name:'Remoto',sharing:true,connected:true}]},
 };
 return renderToStaticMarkup(React.createElement(VoiceRoomStage,{...props,disabled}));
}

test('room integrates the in-stage outlet, participant filmstrip, owner and measured stream quality',()=>{
 const html=render();
 for (const text of ['voice-media-slot','Sala de fixture','Amigos da fixture','Proprietário','Assistindo transmissão','1280×720 · 30 FPS','AO VIVO','Detalhes técnicos','Abrir logs da chamada','Desconectar chamada']) assert.ok(html.includes(text),text);
 assert.equal((html.match(/<article class="voice-participant-tile /g)||[]).length,2);
 assert.doesNotMatch(html,/Duplicado|Encerrar PeeTwoPee|Captura local ativa/);
});

test('theater keeps the media outlet and call controls without the filmstrip',()=>{
 const html=render({theater:true});
 assert.match(html,/voice-room-stage is-theater/);
 assert.match(html,/voice-media-slot/);
 assert.match(html,/Sair do modo foco/);
 assert.match(html,/Desconectar chamada/);
 assert.doesNotMatch(html,/voice-participant-filmstrip/);
});

test('authorization pause takes precedence over stale playing state',()=>{
 const html=render({authorization:'paused'});
 assert.match(html,/Mídia pausada · aguardando autorização/);
 assert.match(html,/Você continua na sala/);
 assert.doesNotMatch(html,/Assistindo transmissão|AO VIVO/);
});

test('unjoined rooms never render stale participants or initialize media and entry honors permissions',()=>{
 const html=render({joined:false,disabled:true});
 assert.match(html,/Fora da sala/);
 assert.match(html,/disabled="">Entrar na sala de voz/);
 assert.doesNotMatch(html,/voice-media-slot|voice-stage-controls|voice-participant-tile/);
});
