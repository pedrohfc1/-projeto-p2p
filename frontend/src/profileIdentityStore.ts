import {imageAssetURL, validAssetHash, type ImageAsset} from './profileIdentityModel.ts';

export type ProfileMetadata = {
 peerId:string;
 displayName:string;
 displayFontId:string;
 avatarHash?:string;
 avatarMime?:string;
 revision:number;
 updatedAt:number;
};
export type ProfileEdit = {
 displayName:string;
 displayFontId:string;
 avatar?:ImageAsset;
 removeAvatar?:boolean;
};
export type ProfileAPI = {
 Profiles: (ids:string[])=>Promise<ProfileMetadata[]>;
 ProfileAsset: (hash:string)=>Promise<ImageAsset|null>;
 SaveProfile: (input:ProfileEdit)=>Promise<ProfileMetadata>;
};

const profileLimit=512;
const imageLimit=72;
const maxAssetReads=4;
const maxProfilesPerRead=64;
const retryDelay=2_000;

export class ProfileIdentityStore {
 private profiles=new Map<string,ProfileMetadata>();
 private assets=new Map<string,string>();
 private profileListeners=new Map<string,Set<()=>void>>();
 private assetListeners=new Map<string,Set<()=>void>>();
 private assetRetries=new Map<string,number>();
 private assetPending=new Set<string>();
 private refreshPending=false;
 private running=false;
 private wake:ReturnType<typeof setTimeout>|undefined;
 private timer:ReturnType<typeof setInterval>|undefined;

 private api:()=>Partial<ProfileAPI>;
 constructor(api:()=>Partial<ProfileAPI>){this.api=api}

 start(){
  if(this.running) return;
  this.running=true;
  this.kick();
  this.timer=setInterval(()=>{void this.refresh();this.readAssets()},retryDelay);
 }
 stop(){
  this.running=false;
  clearTimeout(this.wake);
  clearInterval(this.timer);
  this.wake=undefined;
 }
 private kick() {
  if(!this.running||this.wake!==undefined) return;
  this.wake=setTimeout(()=>{this.wake=undefined;void this.refresh();this.readAssets()},60);
 }
 private subscribe(listeners: Map<string,Set<()=>void>>, id:string, notify:()=>void){
  const current=listeners.get(id)??new Set<()=>void>();
  current.add(notify);
  listeners.set(id,current);
  this.kick();
  return ()=>{
   current.delete(notify);
   if(!current.size) {
    listeners.delete(id);
    if(listeners===this.assetListeners)this.assetRetries.delete(id);
   }
  };
 }
 subscribeProfile(id:string,notify:()=>void){return id?this.subscribe(this.profileListeners,id,notify):()=>{}}
 subscribeAsset(hash:string,notify:()=>void){return validAssetHash(hash)?this.subscribe(this.assetListeners,hash,notify):()=>{}}
 profile(id:string){return this.profiles.get(id)}
 asset(hash:string){return this.assets.get(hash)}
 private prune<T>(items: Map<string,T>,limit:number,active:Map<string,Set<()=>void>>){
  if(items.size<=limit) return;
  for(const key of items.keys()){
   if(!active.has(key)) items.delete(key);
   if(items.size<=limit) break;
  }
 }
 publish(profile:ProfileMetadata,avatar?:ImageAsset){
  if(!profile?.peerId||!Number.isSafeInteger(profile.revision)||profile.revision<1) return;
  const old=this.profiles.get(profile.peerId);
  if(old && old.revision>=profile.revision)return;
  if(avatar&&profile.avatarHash&&profile.avatarMime===avatar.mime)this.putAsset(profile.avatarHash,avatar);
  this.profiles.delete(profile.peerId);
  this.profiles.set(profile.peerId,profile);
  this.prune(this.profiles,profileLimit,this.profileListeners);
  this.profileListeners.get(profile.peerId)?.forEach(notify=>notify());
 }
 private putAsset(hash:string,asset:ImageAsset){
  const url=imageAssetURL(asset);
  if(!url||!validAssetHash(hash)) return;
  this.assets.set(hash,url);
  this.assetRetries.delete(hash);
  this.prune(this.assets,imageLimit,this.assetListeners);
  this.assetListeners.get(hash)?.forEach(notify=>notify());
 }
 private async refresh() {
  if(!this.running||this.refreshPending||!this.profileListeners.size)return;
  this.refreshPending=true;
  try{
   const service=this.api();
   if(!service.Profiles)return;
   const ids=[...this.profileListeners.keys()];
   for(let offset=0;this.running&&offset<ids.length;offset+=maxProfilesPerRead){
    const chunk=ids.slice(offset,offset+maxProfilesPerRead);
    const records=await service.Profiles(chunk);
    if(this.running&&Array.isArray(records)) for(const record of records)if(record&&chunk.includes(record.peerId))this.publish(record);
   }
  } catch {/* Existing names and avatars are usable while the desktop service is unavailable. */}
  finally{this.refreshPending=false}
 }
 private readAssets() {
  if(!this.running||this.assetPending.size>=maxAssetReads||!this.assetListeners.size) return;
  let service: Partial<ProfileAPI>;
  try { service=this.api(); } catch { return; }
  if(!service.ProfileAsset)return;
  const now=Date.now();
  for(const hash of this.assetListeners.keys()){
   if(this.assets.has(hash)||this.assetPending.has(hash)||(this.assetRetries.get(hash)??0)>now)continue;
   this.assetPending.add(hash);
   void this.readAsset(hash,service.ProfileAsset);
   if(this.assetPending.size>=maxAssetReads) break;
  }
 }
 private async readAsset(hash:string,read:ProfileAPI['ProfileAsset']){
  try {
   const asset=await read(hash);
   if(this.running&&asset)this.putAsset(hash,asset);
  } catch {/* The asset may still be arriving or the owner may be offline. */}
  finally {
   this.assetPending.delete(hash);
   if(this.running&&this.assetListeners.has(hash)&&!this.assets.has(hash))this.assetRetries.set(hash,Date.now()+retryDelay);
   if(this.running)queueMicrotask(()=>this.readAssets());
  }
 }
 async save(input:ProfileEdit) {
  const api=this.api();
  if(!api.SaveProfile)throw Error('O editor de perfil precisa de uma versão compatível do aplicativo.');
  const profile=await api.SaveProfile(input);
  this.publish(profile,input.avatar);
  return profile;
 }
}
