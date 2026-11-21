import test from 'node:test';
import assert from 'node:assert/strict';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {build} from 'esbuild';
import {createRequire} from 'node:module';
import {mkdtemp, writeFile, rm} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import path from 'node:path';

const require = createRequire(import.meta.url);
const workspace = await mkdtemp(fileURLToPath(new URL('../node_modules/.voice-participants-', import.meta.url)));
let voiceParticipants, ParticipantSidebar, ParticipantFilmstrip;
try {
 const result = await build({entryPoints: [fileURLToPath(new URL('../src/VoiceParticipants.tsx', import.meta.url))], bundle: true, write: false, platform: 'node', format: 'cjs', external: ['react']});
 const file = path.join(workspace, 'participants.cjs');
 await writeFile(file, result.outputFiles[0].text);
 ({voiceParticipants, ParticipantSidebar, ParticipantFilmstrip} = require(file));
} finally {
 await rm(workspace, {recursive: true, force: true});
}

function fixture() {
 return {
  view: {
   joined: true, authorization: 'valid', muted: false, deaf: true, media: {state: 'connected'},
   participants: [
    {id: 'local', name: 'Pessoa local', self: true, state: 'local'},
    {id: 'host', name: 'Proprietário teste', self: false, state: 'connected'},
    {id: 'host', name: 'Identidade duplicada', self: false, state: 'waiting'},
    {id: 'stale', name: 'Caminho em reconexão', self: false, state: 'reconnecting'},
   ],
  },
  speaking: {local: .8, host: .19},
  screen: {sharing: true, peers: [
   {id: 'host', sharing: true, connected: true},
   {id: 'stale', sharing: true, connected: false},
  ]},
  owner: 'host', watching: 'host',
 };
}

test('roster uses real room controls, authorized speaking and locally confirmed playback', () => {
 const props = fixture();
 const [local, host, stale] = voiceParticipants(props);
 assert.deepEqual(voiceParticipants(props).map(person => person.id), ['local', 'host', 'stale']);
 assert.deepEqual([local.group, local.muted, local.deaf, local.speaking, local.live], ['Assistindo', true, true, false, true]);
 assert.deepEqual([host.group, host.speaking, host.owner, host.muted, host.live], ['Em voz', true, true, undefined, true]);
 assert.deepEqual([stale.group, stale.status, stale.live], ['Aguardando', 'Reconectando', false]);
 assert.equal(voiceParticipants({...props, watching: 'stale'})[0].group, 'Em voz');
 assert.deepEqual(voiceParticipants({...props, view: {...props.view, joined: false}}), []);
});

test('authorization pause suppresses speaking and live affordances without dropping the roster', () => {
 const props = fixture();
 const people = voiceParticipants({...props, view: {...props.view, authorization: 'paused'}});
 assert.equal(people.length, 3);
 for (const person of people) assert.deepEqual([person.group, person.status, person.speaking, person.live], ['Aguardando', 'Aguardando autorização', false, false]);
});

test('sidebar and tiles expose keyboard-addressable profile, options and confirmed remote live actions', () => {
 const people = voiceParticipants(fixture());
 const onWatch = () => assert.fail('list rendering must never start another stream');
 const sidebar = renderToStaticMarkup(React.createElement(ParticipantSidebar, {people, onWatch, onClose: () => {}}));
 assert.match(sidebar, /aria-label="Participantes da sala"/);
 assert.match(sidebar, /aria-label="Ocultar participantes"/);
 assert.match(sidebar, /aria-label="Assistindo"/);
 assert.match(sidebar, /aria-label="Aguardando"/);
 assert.match(sidebar, /voice-participant-owner/);
 for (const text of ['Ver perfil de Proprietário teste', 'Opções de Proprietário teste', 'Ver transmissão de Proprietário teste']) assert.ok(sidebar.includes(text));
 assert.doesNotMatch(sidebar, /Ver transmissão de Caminho em reconexão/);
 assert.doesNotMatch(sidebar, /Microfone habilitado/);
 const strip = renderToStaticMarkup(React.createElement(ParticipantFilmstrip, {people, onWatch}, React.createElement('button', null, 'Transmissão em destaque')));
 assert.match(strip, /aria-label="Miniaturas da sala"/);
 assert.equal((strip.match(/class="voice-participant-tile /g) ?? []).length, 3);
 assert.ok(strip.indexOf('Transmissão em destaque') < strip.indexOf('Pessoa local'));
 assert.doesNotMatch(strip, /Identidade duplicada/);
});
