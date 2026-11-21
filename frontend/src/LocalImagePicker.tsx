import React, {useEffect, useId, useRef, useState, type ReactNode} from 'react';
import {Check, ImagePlus, X} from 'lucide-react';
import {centerCrop, imageAssetLimit, imageAssetURL, imageInputLimit, imageOutputSide, inspectImage, squareCrop, type ImageAsset, type ImageMime} from './profileIdentityModel';

type PickerProps = {
 asset: ImageAsset|null;
 existingSrc?: string;
 hasExisting?: boolean;
 label?: string;
 disabled?: boolean;
 onChange: (asset:ImageAsset|null)=>void;
 fallback?: ReactNode;
};

function toBlob(canvas: HTMLCanvasElement,mime:ImageMime,quality?:number) {
 return new Promise<Blob|null>(resolve=>canvas.toBlob(resolve,mime,quality));
}

async function encodeCrop(canvas:HTMLCanvasElement):Promise<ImageAsset> {
 let blob:Blob|null=null;
 const candidates: [ImageMime,number?][] = [['image/webp',.87],['image/png'],['image/jpeg',.87]];
 for (const [mime,quality] of candidates) {
  const result=await toBlob(canvas,mime,quality);
  if(result?.type===mime&&result.size<=imageAssetLimit){blob=result;break}
 }
 if(!blob) throw Error('Não foi possível reduzir a imagem. Tente escolher outro recorte.');
 const mime=blob.type as ImageMime;
 const bytes=new Uint8Array(await blob.arrayBuffer());
 let binary='';
 for(let offset=0;offset<bytes.length;offset+=8192) binary+=String.fromCharCode(...bytes.subarray(offset,offset+8192));
 return {mime,data:btoa(binary)};
}

export function LocalImagePicker({asset,existingSrc,hasExisting,label='Avatar',disabled=false,onChange,fallback}:PickerProps) {
 const [bitmap,setBitmap]=useState<ImageBitmap|null>(null),[busy,setBusy]=useState(false),[error,setError]=useState(''),[crop,setCrop]=useState(centerCrop);
 const fileRef=useRef<HTMLInputElement>(null),dialog=useRef<HTMLDialogElement>(null),canvas=useRef<HTMLCanvasElement>(null),request=useRef(0);
 const imageURL=imageAssetURL(asset) || existingSrc;
 const heading=useId(),description=useId();

 useEffect(()=>()=>{++request.current},[]);
 useEffect(()=>{
  if(!bitmap) return;
  return ()=>bitmap.close();
 },[bitmap]);
 useEffect(()=>{
  if(!bitmap) return;
  const element=dialog.current;
  element?.showModal();
  return ()=>element?.close();
 },[bitmap]);
 useEffect(()=>{
  if(!bitmap || !canvas.current) return;
  const context=canvas.current.getContext('2d');
  if(!context) return;
  const {x,y,size}=squareCrop(bitmap.width,bitmap.height,crop);
  context.clearRect(0,0,imageOutputSide,imageOutputSide);
  context.imageSmoothingQuality='high';
  context.drawImage(bitmap,x,y,size,size,0,0,imageOutputSide,imageOutputSide);
 },[bitmap,crop]);

 const open=async(file:File)=>{
  const version=++request.current;
  setError('');setBusy(true);
  try {
   if(file.size>imageInputLimit) throw Error('A imagem excede o limite de 5 MB.');
   const buffer=await file.arrayBuffer();
   const header=inspectImage(buffer);
   const decoded=await createImageBitmap(new Blob([buffer],{type:header.mime}),{imageOrientation:'from-image'});
   if(version!==request.current){decoded.close();return}
   if(!decoded.width||!decoded.height||decoded.width*decoded.height>32_000_000){
    decoded.close();throw Error('Não foi possível abrir esta imagem no tamanho indicado.');
   }
   setCrop(centerCrop);setBitmap(decoded);
  } catch(cause) {
   if(version===request.current) setError(cause instanceof Error&&/^(A imagem|Dimensões|Arquivo inválido|Não foi possível)/.test(cause.message)?cause.message:'Não foi possível decodificar a imagem selecionada.');
  } finally {
   if(version===request.current) setBusy(false);
  }
 };

 const apply=async()=>{
  if(!canvas.current) return;
  const version=++request.current;
  setBusy(true);setError('');
  try {
   const media=await encodeCrop(canvas.current);
   if(version===request.current){dialog.current?.close();onChange(media);setBitmap(null)}
  } catch(cause) {
   if(version===request.current) setError(cause instanceof Error?cause.message:'Não foi possível processar a imagem.');
  } finally{
   if(version===request.current) setBusy(false);
  }
 };
 const cancel=()=>{++request.current;dialog.current?.close();setBusy(false);setBitmap(null);setError('')};

 return <div className="local-image-picker">
  <input ref={fileRef} hidden tabIndex={-1} type="file" accept=".png,.jpg,.jpeg,.webp,image/png,image/jpeg,image/webp"
   onChange={e=>{const file=e.target.files?.[0];e.target.value='';if(file) void open(file)}}/>
  <button type="button" className="local-image-picker-preview" aria-label={'Alterar '+label} title={'Alterar '+label} disabled={disabled||busy} onClick={()=>fileRef.current?.click()}>
   {imageURL?<img src={imageURL} alt=""/>:<span className="local-image-picker-placeholder">{fallback||<ImagePlus size={23}/>}</span>}
  </button>
  <div className="local-image-picker-actions">
   <button type="button" className="secondary" disabled={disabled||busy} onClick={()=>fileRef.current?.click()}><ImagePlus size={15}/>{busy?'Processando…':'Escolher imagem'}</button>
   {(hasExisting||imageURL)&&<button type="button" className="profile-text-action" disabled={disabled||busy} onClick={()=>{setError('');onChange(null)}}>Remover imagem</button>}
  </div>
  {error&&!bitmap&&<p className="profile-editor-error" role="alert">{error}</p>}
  {bitmap&&<dialog ref={dialog} className="identity-image-dialog" aria-labelledby={heading} aria-describedby={description} onCancel={e=>{e.preventDefault();cancel()}}
   onKeyDown={e=>{if(e.key==='Escape'){e.stopPropagation();e.preventDefault();cancel()}else if(e.key==='Tab')e.stopPropagation()}}>
   <div className="identity-image-dialog-heading"><h3 id={heading}>Recortar imagem</h3><button type="button" className="icon" aria-label="Fechar recorte" onClick={cancel}><X size={18}/></button></div>
   <p id={description}>Ajuste a área quadrada que será exibida no aplicativo.</p>
   <canvas ref={canvas} width={imageOutputSide} height={imageOutputSide} role="img" aria-label="Prévia do recorte quadrado"/>
   <div className="identity-crop-controls" aria-label="Ajustes do recorte">
    <label>Zoom <input type="range" min={1} max={4} step={.01} value={crop.zoom} disabled={busy} onChange={e=>setCrop({...crop,zoom:e.target.valueAsNumber})}/></label>
    <label>Posição horizontal <input type="range" min={-100} max={100} value={crop.x} disabled={busy} onChange={e=>setCrop({...crop,x:e.target.valueAsNumber})}/></label>
    <label>Posição vertical <input type="range" min={-100} max={100} value={crop.y} disabled={busy} onChange={e=>setCrop({...crop,y:e.target.valueAsNumber})}/></label>
   </div>
   {error&&<p className="profile-editor-error" role="alert">{error}</p>}
   <div className="identity-image-dialog-actions"><button type="button" className="secondary" onClick={cancel}>Cancelar</button><button type="button" className="primary" disabled={busy} onClick={()=>void apply()}><Check size={16}/>{busy?'Processando…':'Usar recorte'}</button></div>
  </dialog>}
 </div>;
}
