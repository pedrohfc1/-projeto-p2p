import test from 'node:test';
import assert from 'node:assert/strict';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {build} from 'esbuild';
import {fileURLToPath} from 'node:url';
import {unlink} from 'node:fs/promises';

const path = new URL(`../.screen-viewer-test-${process.pid}.mjs`, import.meta.url);
await build({
 entryPoints: [fileURLToPath(new URL('../src/ScreenViewer.tsx', import.meta.url))],
 outfile: fileURLToPath(path),
 bundle: true,
 platform: 'node',
 format: 'esm',
 external: ['react','react-dom','lucide-react'],
 logLevel: 'silent',
});

try {
 const {ScreenShareDirectory, ScreenShareEmpty, ScreenViewerModeControls, ScreenViewerOverlay} = await import(path.href);
 const render = (component, props, child) => renderToStaticMarkup(React.createElement(component, props, child));

 test('viewer shows broadcaster, observed geometry/FPS and reconnection without removing controls', () => {
  const html = render(ScreenViewerOverlay, {name:'Ana', connected:false, width:960, height:540, fps:28.7, mbps:1.83}, React.createElement('button', null, 'Parar de assistir'));
  for (const value of ['screen-overlay','stream-avatar','Ana','AO VIVO','Reconectando','960×540','28.7','1.83','Parar de assistir']) assert.ok(html.includes(value), value);
  const unknown = render(ScreenViewerOverlay, {name:'B', connected:true}, null);
  assert.match(unknown,/Resolução n\/d/);
  assert.match(unknown,/n\/d.*FPS/);
  assert.doesNotMatch(unknown,/1920|Reconectando/);
 });

 test('screen directory makes watch explicit and labels the active source', () => {
  const streams = [
   {id:'a',name:'Ana',connected:true,sharing:true},
   {id:'b',name:'Bruno',connected:true,sharing:true},
  ];
  const directory = render(ScreenShareDirectory, {streams, selected:'b', watching:'a', ready:true});
  assert.match(directory,/Telas dos participantes/);
  assert.ok(directory.indexOf('Bruno') < directory.indexOf('Ana'));
  assert.match(directory,/Assistir/);
  assert.match(directory,/Voltar à tela/);
  assert.doesNotMatch(directory, /<video|<audio/);
 });

 test('empty stage distinguishes an unavailable stream from a stream that can be selected', () => {
  const empty = render(ScreenShareEmpty, {ready:false, busy:false, remoteStreams:0});
  assert.match(empty,/Nenhuma transmissão ativa/);
  assert.match(empty,/disabled="".*Compartilhar tela/);
  assert.doesNotMatch(empty,/Telas dos participantes/);
  const live = render(ScreenShareEmpty, {ready:true, busy:false, remoteStreams:2});
  assert.match(live,/Telas dos participantes/);
  assert.match(live,/Compartilhar tela/);
 });

 test('mode controls distinguish theater from the compact player and fullscreen', () => {
  const room = render(ScreenViewerModeControls, {inRoom:true, theater:true, fullscreen:false});
  assert.match(room,/Sair do modo teatro/);
  assert.match(room,/Tela cheia/);
  const compact = render(ScreenViewerModeControls, {inRoom:false, compact:true, fullscreen:true});
  assert.match(compact,/Ampliar transmissão/);
  assert.match(compact,/Sair da tela cheia/);
 });
} finally {
 await unlink(path);
}
