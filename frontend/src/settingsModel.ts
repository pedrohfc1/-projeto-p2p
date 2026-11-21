export const settingsGroups=[
 {label:'Você',items:[['account','Conta'],['profile','Perfil'],['privacy','Dados e privacidade'],['messages','Permissões de mensagens'],['notifications','Notificações']]},
 {label:'Voz e vídeo',items:[['voice','Voz'],['transmission','Transmissão'],['sounds','Sons'],['soundboard','Painel de efeitos sonoros'],['audio-advanced','Avançado']]},
 {label:'Aplicativo',items:[['appearance','Aparência'],['communities','Comunidades'],['invites','Convites'],['connectivity','Conectividade'],['about','Sobre / versão']]}
] as const;
export type SettingsSection=typeof settingsGroups[number]['items'][number][0];
export const settingsSections=settingsGroups.flatMap(g=>g.items.map(([id,label])=>({id,label})));
export function validSettingsSection(value:unknown):SettingsSection{return settingsSections.some(s=>s.id===value)?value as SettingsSection:'account'}
export type SettingsState={visible:boolean;section:SettingsSection;target:string};
export function settingsTransition(state:SettingsState,action:{type:'close'}|{type:'open';section?:SettingsSection;target?:string}):SettingsState{return action.type==='close'?{...state,visible:false,target:''}:{visible:true,section:action.section||state.section,target:action.target||''}}
export type Appearance={density:'comfortable'|'compact';scale:'normal'|'large';reduceMotion:boolean};
export function appearance(value:unknown):Appearance{const p=(value&&typeof value==='object'?value:{}) as Partial<Appearance>;return {density:p.density==='compact'?'compact':'comfortable',scale:p.scale==='large'?'large':'normal',reduceMotion:p.reduceMotion===true}}
export function dayKey(seconds:number){const d=new Date(seconds*1000);return Number.isFinite(d.getTime())?`${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`:'unknown'}
