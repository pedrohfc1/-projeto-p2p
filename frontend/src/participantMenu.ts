import type React from 'react';
import type {MenuAction} from './interface';
import type {ListenerPreference} from './participantAudio';
export type ParticipantTarget={id:string;name:string;self?:boolean;profile?:()=>void;mention?:()=>void;message?:()=>void;extra?:MenuAction[]};
let directMessages=new Map<string,()=>void>();
export function participantMessages(messages:Map<string,()=>void>){directMessages=messages;return()=>{if(directMessages===messages)directMessages=new Map()}}
export function participantContext(event:Pick<React.MouseEvent,'clientX'|'clientY'|'currentTarget'|'preventDefault'|'stopPropagation'>,target:ParticipantTarget){event.preventDefault();event.stopPropagation();window.dispatchEvent(new CustomEvent('peetwopee:participant-menu',{detail:{event:{clientX:event.clientX,clientY:event.clientY,currentTarget:event.currentTarget,preventDefault(){}},target}}))}
export function participantKeyboard(event:React.KeyboardEvent<HTMLElement>,target:ParticipantTarget){if(event.key==='ContextMenu'||(event.shiftKey&&event.key==='F10')){const rect=event.currentTarget.getBoundingClientRect();participantContext({clientX:rect.left+12,clientY:rect.bottom,currentTarget:event.currentTarget,preventDefault:()=>event.preventDefault(),stopPropagation:()=>event.stopPropagation()},target)}}
export function participantActions(target:ParticipantTarget,p:ListenerPreference,update:(patch:Partial<ListenerPreference>)=>void,copy:()=>void,profile:()=>void):MenuAction[]{
 const items:MenuAction[]=[{label:'Interação',section:true},{label:'Ver perfil',icon:'profile',run:target.profile||profile}];
 if(target.mention)items.push({label:'Mencionar',icon:'mention',run:target.mention});
 if(!target.self&&(target.message||directMessages.has(target.id)))items.push({label:'Enviar mensagem privada',icon:'message',run:target.message||directMessages.get(target.id)});
 items.push({label:'Copiar ID do usuário',icon:'copy',run:copy});
 if(!target.self)items.push({label:'Áudio · somente neste computador',section:true},{label:'Volume do usuário',range:{value:p.volume,change:volume=>update({volume})}},{label:'Silenciar localmente',icon:'mute',checked:p.muted,run:()=>update({muted:!p.muted})});
 if(target.extra?.length)items.push({label:'Comunidade e ações',section:true},...target.extra);
 return items;
}
