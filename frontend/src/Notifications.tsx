import React,{useEffect,useState,useSyncExternalStore} from 'react';
import {Volume2} from 'lucide-react';
import {callSounds} from './soundEffects';
import {useSettings} from './SettingsCenter';

export type NoticePrefs = {enabled: boolean; preview: boolean; typing: boolean; muted: Record<string, boolean>};
const api = () => window.go!.main.App as unknown as {
 NotificationPreferences: () => Promise<NoticePrefs>;
 SaveNotificationPreferences: (value: NoticePrefs) => Promise<void>;
};

export function NotificationSettings() {
 const settings = useSettings();
 const [value,setValue] = useState<NoticePrefs|null>(null);
 const [error,setError] = useState('');
 useEffect(() => {
  let alive = true;
  void api().NotificationPreferences().then(preferences => {
   if (alive) setValue(preferences);
  }).catch(e => { if (alive) setError(String(e)); });
  return () => { alive = false; };
 }, []);
 const save = async (next: NoticePrefs) => {
  setValue(next);
  try {
   await api().SaveNotificationPreferences(next);
   setError('');
  } catch(e) { setError(String(e)); }
 };
 return <section className="card">
  <h3>Notificações e privacidade</h3>
  {value&&<>
   <label className="toggle"><input type="checkbox" checked={value.enabled} onChange={e=>void save({...value,enabled:e.target.checked})}/>Ativar notificações nativas</label>
   <label className="toggle"><input type="checkbox" checked={value.preview} onChange={e=>void save({...value,preview:e.target.checked})}/>Mostrar conteúdo na notificação (pode aparecer na tela bloqueada)</label>
   <label className="toggle"><input type="checkbox" checked={value.typing} onChange={e=>void save({...value,typing:e.target.checked})}/>Compartilhar indicador de digitação com participantes autorizados</label>
   <p>Alertas são desativados por padrão e não exibem conteúdo sem esta opção. Mensagens da conversa visível não geram alerta. O Windows pode silenciar alertas por suas próprias regras.</p>
   {Object.entries(value.muted).filter(([,muted])=>muted).map(([key])=><button className="secondary" key={key} onClick={()=>void save({...value,muted:{...value.muted,[key]:false}})}>Reativar {key.slice(0,24)}</button>)}
  </>}
  {error&&<p role="alert">{error}</p>}
  <button className="secondary" type="button" onClick={()=>settings.open('sounds')}><Volume2 size={15}/> Configurar efeitos sonoros</button>
 </section>;
}

export function SoundSettings() {
 const value = useSyncExternalStore(callSounds.subscribe,callSounds.get,callSounds.get);
 const [previewStatus,setPreviewStatus] = useState('');
 const playPreview = async () => {
  setPreviewStatus('');
  const played = await callSounds.preview();
  setPreviewStatus(played?'Prévia reproduzida.':'A saída de áudio selecionada está indisponível.');
 };
 return <section className="settings-card">
  <h3>Efeitos sonoros da chamada</h3>
  <p>Os efeitos tocam na sua saída de áudio e têm um volume próprio.</p>
  <label className="toggle"><input type="checkbox" checked={value.enabled} onChange={e=>{
   callSounds.update({enabled:e.target.checked});
   if (e.target.checked) callSounds.prepare();
   setPreviewStatus('');
  }}/>Ativar efeitos sonoros</label>
  <label>Volume dos efeitos · {value.volume}%<input aria-label="Volume dos efeitos" aria-valuetext={value.volume+'%'} type="range" min="0" max="100" value={value.volume} disabled={!value.enabled} onChange={e=>callSounds.update({volume:Number(e.target.value)})}/></label>
  <label className="toggle"><input type="checkbox" checked={value.people} disabled={!value.enabled} onChange={e=>callSounds.update({people:e.target.checked})}/>Entrada e saída da sala de voz</label>
  <label className="toggle"><input type="checkbox" checked={value.sharing} disabled={!value.enabled} onChange={e=>callSounds.update({sharing:e.target.checked})}/>Início e fim de compartilhamento de tela</label>
  <button type="button" className="secondary" disabled={!value.enabled||!value.volume||(!value.people&&!value.sharing)} onClick={()=>void playPreview()}><Volume2 size={15}/> Testar efeito sonoro</button>
  {previewStatus&&<p role="status">{previewStatus}</p>}
 </section>;
}

export async function muteContext(context: string) {
 const backend = api();
 const value = await backend.NotificationPreferences();
 await backend.SaveNotificationPreferences({...value,muted:{...value.muted,[context]:!value.muted[context]}});
}
