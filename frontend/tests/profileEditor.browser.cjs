/* Browser-only QA fixture. Uses generated images and controlled local IPC doubles. */
const {strict:assert}=require('node:assert');
const {createServer}=require('node:http');
const fs=require('node:fs/promises');
const path=require('node:path');
const {build}=require('esbuild');
const {chromium}=require(process.env.P2P_PLAYWRIGHT_PATH||'playwright');
const frontend=path.resolve(__dirname,'..');

const fixtureSource=`
import React from 'react';
import {createRoot} from 'react-dom/client';
import {IdentityAvatar,IdentityName,ProfileEditor,ProfileIdentityProvider} from './src/ProfileIdentity';
import './src/profileIdentity.css';
const profiles={remote:{peerId:'remote',displayName:'Peer sincronizado',displayFontId:'unknown-font',revision:1,updatedAt:1}};
const assets={};
const calls=[];
const api={
 Profiles: async ids=>ids.map(id=>profiles[id]).filter(Boolean),
 ProfileAsset: async hash=>assets[hash]||null,
 SaveProfile: async edit=>{
  calls.push(edit);
  const hash=edit.removeAvatar?'': edit.avatar?'fa'.repeat(32):(profiles.self?.avatarHash||'');
  if(edit.avatar)assets[hash]=edit.avatar;
  return profiles.self={peerId:'self',displayName:edit.displayName,displayFontId:edit.displayFontId,avatarHash:hash,avatarMime:assets[hash]?.mime,revision:(profiles.self?.revision||0)+1,updatedAt:2};
 },
};
window.profileFixture={profiles,assets,calls};
createRoot(document.getElementById('root')).render(<ProfileIdentityProvider api={()=>api}><div id="self"><IdentityAvatar id="self" name="Nome anterior"/><IdentityName id="self" name="Nome anterior"/></div><div id="remote"><IdentityAvatar id="remote" name="Legado"/><IdentityName id="remote" name="Legado"/></div><ProfileEditor selfID="self" legacyName="Nome anterior" status="Conectado"/></ProfileIdentityProvider>);
`;

(async()=>{
 let browser,server;
 try {
  const bundled=await build({stdin:{contents:fixtureSource,loader:'tsx',resolveDir:frontend},bundle:true,write:false,outfile:'fixture.js',external:['/fonts/profile/*']});
  const media=Object.fromEntries(bundled.outputFiles.map(asset=>[path.basename(asset.path),asset.contents]));
  server=createServer(async(req,res)=>{
   const target=req.url;
   if(target==='/'||target==='/index.html'){
    res.writeHead(200,{'content-type':'text/html'}).end('<!doctype html><html lang="pt-BR"><meta charset="utf-8"><link rel="stylesheet" href="/fixture.css"><div id="root"></div><script src="/fixture.js"></script>');
   }else if(media[target.slice(1)]){
    res.writeHead(200,{'content-type':target.endsWith('.css')?'text/css':'text/javascript'}).end(media[target.slice(1)]);
   } else if(/^\/fonts\/profile\/[a-z0-9-]+\.woff2$/.test(target)){
    res.writeHead(200,{'content-type':'font/woff2'}).end(await fs.readFile(path.join(frontend,'public',target)));
   }else res.writeHead(404).end();
  });
  await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
  browser=await chromium.launch({executablePath:process.env.P2P_EDGE_PATH||'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',headless:true,chromiumSandbox:true,args:['--no-first-run']});
  const page=await browser.newPage({viewport:{width:1280,height:900}});
  const errors=[];
  page.on('pageerror',error=>errors.push(error.message));
  await page.goto('http://127.0.0.1:'+server.address().port);
  await page.locator('#remote [data-display-font=inter]').getByText('Peer sincronizado').waitFor();
  const picker=page.locator('input[type=file]');
  await picker.setInputFiles({name:'arquivo.png',mimeType:'image/png',buffer:Buffer.from('<svg xmlns="http://www.w3.org/2000/svg"/>')});
  await page.getByRole('alert').getByText('Arquivo inválido. Escolha uma imagem PNG, JPEG ou WebP.').waitFor();
  await picker.setInputFiles({name:'grande.png',mimeType:'image/png',buffer:Buffer.alloc(5*1024*1024+1)});
  await page.getByRole('alert').getByText('A imagem excede o limite de 5 MB.').waitFor();

  const image=await page.evaluate(()=>{
   const canvas=document.createElement('canvas');
   canvas.width=440;
   canvas.height=220;
   const ctx=canvas.getContext('2d');
   ctx.fillStyle='#fc0313';
   ctx.fillRect(0,0,220,220);
   ctx.fillStyle='#2b29ef';
   ctx.fillRect(220,0,220,220);
   return canvas.toDataURL('image/png').split(',')[1];
  });
  await picker.setInputFiles({name:'teste.png',mimeType:'image/png',buffer:Buffer.from(image,'base64')});
  const dialog=page.getByRole('dialog',{name:'Recortar imagem'});
  await dialog.waitFor();
  await page.keyboard.press('Escape');
  await dialog.waitFor({state:'hidden'});
  await picker.setInputFiles({name:'teste.png',mimeType:'image/png',buffer:Buffer.from(image,'base64')});
  await dialog.waitFor();
  await dialog.getByRole('slider',{name:'Posição horizontal'}).fill('100');
  const pixel=await dialog.locator('canvas').evaluate(canvas=>Array.from(canvas.getContext('2d').getImageData(30,30,1,1).data));
  assert.ok(pixel[2]>220&&pixel[0]<70,'square preview follows selected pan');
  await dialog.getByRole('button',{name:'Usar recorte'}).click();
  await dialog.waitFor({state:'hidden'});
  await page.getByRole('textbox',{name:'Nome de exibição'}).fill('João 💙');
  await page.getByRole('radio',{name:'Space Grotesk'}).check();
  await page.getByRole('region',{name:'Como outras pessoas verão você'}).getByText('João 💙').waitFor();
  await page.getByRole('button',{name:'Salvar perfil'}).click();
  await page.locator('#self [data-display-font=space-grotesk]').getByText('João 💙').waitFor();
  await page.locator('#self img').waitFor();
  const result=await page.evaluate(async()=>{
   const asset=window.profileFixture.calls[0].avatar;
   const blob=await (await fetch('data:'+asset.mime+';base64,'+asset.data)).blob();
   const image=await createImageBitmap(blob);
   const result={mime:asset.mime,bytes:blob.size,width:image.width,height:image.height};
   image.close();
   return result;
  });
  assert.equal(result.width,256);
  assert.equal(result.height,256);
  assert.equal(result.mime,'image/webp');
  assert.ok(result.bytes>0&&result.bytes<256*1024);

  await page.getByRole('button',{name:'Remover imagem'}).click();
  await page.getByRole('button',{name:'Salvar perfil'}).click();
  await page.locator('#self img').waitFor({state:'detached'});
  assert.deepEqual(await page.evaluate(()=>window.profileFixture.calls.map(x=>x.removeAvatar)),[false,true]);
  assert.deepEqual(errors,[]);
  console.log('PASS: limite, rejeição de SVG disfarçado, recorte 256×256, WebP, preview, fonte e avatar atualizados e remoção');
 } finally {
  await browser?.close();
  if(server)await new Promise(resolve=>server.close(resolve));
 }
})().catch(error=>{console.error(error);process.exitCode=1});
