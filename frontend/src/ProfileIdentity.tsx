import React, {createContext,useCallback,useContext,useEffect,useId,useRef,useState,useSyncExternalStore,type ReactNode} from 'react';
import {Check, ShieldCheck} from 'lucide-react';
import {LocalImagePicker} from './LocalImagePicker';
import {cleanName,editName,fontId,fonts,imageAssetURL,initials,profileNameLimit,type DisplayFontID,type ImageAsset} from './profileIdentityModel';
import {ProfileIdentityStore,type ProfileAPI,type ProfileMetadata} from './profileIdentityStore';

export type {ImageAsset} from './profileIdentityModel';
export type {ProfileAPI,ProfileEdit,ProfileMetadata} from './profileIdentityStore';

const Context=createContext<ProfileIdentityStore|null>(null);
const emptySubscribe=()=>()=>{};
const emptySnapshot=()=>undefined;

export function ProfileIdentityProvider({api,children}:{api:()=>Partial<ProfileAPI>;children:ReactNode}){
 const apiRef=useRef(api);
 apiRef.current=api;
 const [store]=useState(()=>new ProfileIdentityStore(()=>apiRef.current()));
 useEffect(()=>{store.start();return()=>store.stop()},[store]);
 return <Context.Provider value={store}>{children}</Context.Provider>;
}

function useRecord(id:string){
 const store=useContext(Context);
 const subscribe=useCallback((notify:()=>void)=>store?.subscribeProfile(id,notify)??emptySubscribe(),[store,id]);
 const snapshot=useCallback(()=>store?.profile(id),[store,id]);
 return useSyncExternalStore(subscribe,snapshot,emptySnapshot);
}

function useAsset(hash?:string) {
 const store=useContext(Context);
 const key=hash||'';
 const subscribe=useCallback((notify:()=>void)=>store?.subscribeAsset(key,notify)??emptySubscribe(),[store,key]);
 const snapshot=useCallback(()=>store?.asset(key),[store,key]);
 return useSyncExternalStore(subscribe,snapshot,emptySnapshot);
}

export function useIdentity(id:string,legacyName=''){
 const profile=useRecord(id);
 const src=useAsset(profile?.avatarHash);
 const [last,setLast]=useState<{id:string;src:string}|null>(null);
 useEffect(()=>{
  if(src){setLast({id,src});return}
  if(!profile?.avatarHash)setLast(null);
 },[id,profile?.avatarHash,src]);
 return {profile,displayName:profile?.displayName||legacyName||'Participante',displayFontId:fontId(profile?.displayFontId),
  avatarSrc:src||(profile?.avatarHash&&last?.id===id ? last.src : undefined)};
}

function Portrait({src,label,className='',children}:{src?:string;label?:string;className?:string;children:ReactNode}){
 const [broken,setBroken]=useState('');
 return <span className={'identity-portrait '+className} aria-hidden={!label||undefined} aria-label={label}>
  {src&&src!==broken?<img src={src} alt="" draggable={false} onError={()=>setBroken(src)}/>:children}
 </span>;
}

export function IdentityAvatar({id,name,className='',fallback}:{id:string;name?:string;className?:string;fallback?:ReactNode}){
 const identity=useIdentity(id,name);
 return <Portrait src={identity.avatarSrc} className={className}>{fallback||initials(identity.displayName)}</Portrait>;
}

export function IdentityName({id,name,className='',title}:{id:string;name?:string;className?:string;title?:string}){
 const profile=useRecord(id);
 const displayName=profile?.displayName||name||'Participante';
 return <span className={'identity-name '+className} data-display-font={fontId(profile?.displayFontId)} title={title??displayName}>{displayName}</span>;
}

export function AssetIcon({hash,className='',fallback}:{hash?:string;className?:string;fallback?:ReactNode}){
 return <Portrait src={useAsset(hash)} className={'identity-asset-icon '+className}>{fallback}</Portrait>;
}

export function ImageAssetPicker({existingHash,...props}:React.ComponentProps<typeof LocalImagePicker>&{existingHash?:string}){
 const src=useAsset(existingHash);
 return <LocalImagePicker {...props} existingSrc={props.existingSrc||src} hasExisting={props.hasExisting||!!existingHash}/>;
}

type Draft={name:string;font:DisplayFontID;image:ImageAsset|null;remove:boolean};
function draftFor(profile:ProfileMetadata|undefined,name:string):Draft {
 return {name:cleanName(profile?.displayName||name),font:fontId(profile?.displayFontId),image:null,remove:false};
}

export function ProfileEditor({selfID,legacyName,status}:{selfID:string;legacyName:string;status?:string}){
 const store=useContext(Context);
 const {profile,avatarSrc}=useIdentity(selfID,legacyName);
 const [draft,setDraft]=useState<Draft>(()=>draftFor(profile,legacyName));
 const [dirty,setDirty]=useState(false),[busy,setBusy]=useState(false),[notice,setNotice]=useState(''),[error,setError]=useState('');
 const fontGroup=useId(),liveStatus=useId();
 useEffect(()=>{if(!dirty)setDraft(draftFor(profile,legacyName))},[profile,legacyName,dirty]);

 const update=(value:Partial<Draft>)=>{
  setDraft(previous=>({...previous,...value}));
  setDirty(true);setNotice('');setError('');
 };
 const save=async(e:React.FormEvent)=>{
  e.preventDefault();
  const displayName=cleanName(draft.name);
  if(!displayName){setError('Digite pelo menos um caractere para o nome.');return}
  if(busy||!store||!selfID)return;
  setBusy(true);setError('');setNotice('');
  try {
   const saved=await store.save({displayName,displayFontId:draft.font,avatar:draft.image??undefined,removeAvatar:draft.remove});
   setDraft(draftFor(saved,displayName));setDirty(false);setNotice('Perfil atualizado.');
  } catch(cause){setError(cause instanceof Error?cause.message:String(cause))}
  finally{setBusy(false)}
 };
 const previewName=cleanName(draft.name)||'Seu nome';
 const savedURL=draft.remove?undefined:avatarSrc;
 const previewURL=imageAssetURL(draft.image)||savedURL;

 return <div className="profile-editor-layout">
  <form className="settings-card profile-editor-form" onSubmit={save} aria-label="Editar seu perfil">
   <h3>Avatar</h3>
   <ImageAssetPicker label="avatar" asset={draft.image} existingSrc={savedURL} hasExisting={!draft.remove&&!!profile?.avatarHash} fallback={initials(previewName)} disabled={busy}
    onChange={image=>update({image,remove:!image})}/>
   <label className="profile-name-field">Nome de exibição
    <input value={draft.name} onChange={e=>update({name:editName(e.target.value)})} aria-describedby={liveStatus} placeholder="Seu nome" autoComplete="nickname" required disabled={busy}/>
    <small id={liveStatus}>{Array.from(draft.name).length}/{profileNameLimit} caracteres</small>
   </label>
   <fieldset className="profile-fonts">
    <legend>Fonte do nome</legend>
    {fonts.map(font=><label key={font.id} className="profile-font-row">
     <input type="radio" name={fontGroup} value={font.id} checked={draft.font===font.id} disabled={busy} onChange={()=>update({font:font.id})}/>
     <span>{font.name}</span><span className="identity-name profile-font-sample" data-display-font={font.id} aria-hidden="true">{previewName}</span>
    </label>)}
   </fieldset>
   <div className="profile-editor-footer">
    <button type="submit" className="primary" disabled={!selfID||!dirty||busy||!cleanName(draft.name)}><Check size={16}/>{busy?'Salvando…':'Salvar perfil'}</button>
    {dirty&&<button type="button" className="secondary" disabled={busy} onClick={()=>{setDraft(draftFor(profile,legacyName));setDirty(false);setNotice('');setError('')}}>Descartar alterações</button>}
   </div>
   {notice&&<p className="profile-editor-notice" role="status">{notice}</p>}
   {error&&<p className="profile-editor-error" role="alert">{error}</p>}
  </form>
  <section className="profile-preview-card" aria-label="Como outras pessoas verão você">
   <div className="profile-preview-cover"/>
   <div className="profile-preview-content">
    <Portrait src={previewURL} className="profile-preview-avatar">{initials(previewName)}</Portrait>
    <h3 className="identity-name profile-preview-name" data-display-font={draft.font} title={previewName}>{previewName}</h3>
    <p>{status||'Presença não verificada'}</p>
   </div>
   <div className="profile-preview-caption"><ShieldCheck size={16}/><span>Como outras pessoas verão você</span></div>
  </section>
 </div>;
}
