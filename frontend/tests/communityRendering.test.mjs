import {test} from 'node:test';
import assert from 'node:assert/strict';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {build} from 'esbuild';
import {mkdtemp,writeFile,mkdir,rm,readFile} from 'node:fs/promises';
import {resolve,dirname} from 'node:path';
import {pathToFileURL} from 'node:url';
test('conteúdo remoto é texto escapado e hub permite criar/entrar sem captura de mídia',async()=>{
 const root=resolve('dist/.ssr-tests');await mkdir(root,{recursive:true});const dir=await mkdtemp(resolve(root,'community-'));
 try{
  const result=await build({entryPoints:['src/CommunityPanel.tsx'],bundle:true,platform:'node',format:'esm',write:false,external:['react','react-dom','lucide-react']});
  const file=resolve(dir,'components.mjs');await writeFile(file,result.outputFiles[0].contents);
  const {PlainMessage,CommunityHub}=await import(pathToFileURL(file).href);
  const dangerous='<script>alert(1)</script><img src=x onerror=alert(2)>';
  const html=renderToStaticMarkup(React.createElement(PlainMessage,{text:dangerous}));
  assert.ok(html.includes('&lt;script&gt;'));assert.ok(!html.includes('<script>'));assert.ok(!html.includes('<img'));
  const hub=renderToStaticMarkup(React.createElement(CommunityHub,{api:()=>({}),onClose:()=>{},onOpen:()=>{},onNetwork:async()=>{}}));
  assert.ok(hub.includes('Criar comunidade'));assert.ok(hub.includes('Entrar por convite'));assert.ok(hub.includes('proprietário precisa estar online'));
  assert.ok(!hub.includes('<audio'));assert.ok(!hub.includes('<video'));
  const index=await readFile('index.html','utf8');assert.ok(index.includes('Content-Security-Policy'));assert.ok(index.includes("script-src 'self'"));assert.ok(index.includes("media-src 'none'"));
 }finally{assert.equal(dirname(dir),root);await rm(dir,{recursive:true,force:true})}
});
