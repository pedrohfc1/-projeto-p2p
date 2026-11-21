import React, {useId, useState} from 'react';
import {Activity, Focus, Monitor, Users, Volume2} from 'lucide-react';
import {AssetIcon,IdentityName} from './ProfileIdentity';
import {CallControlsBar} from './CallPanel';
import {openCallLogs} from './DiagnosticConsole';
import {ParticipantFilmstrip, ParticipantSidebar, voiceParticipants} from './VoiceParticipants';
import {callStatus, type ScreenControlState} from './callPresentation';
import {useVoice, type VoiceView} from './Voice';

type VoiceRoomProps = {
 group: string;
 channel: string;
 name: string;
 community: string;
 owner?: string;
 iconHash?:string;
 members?: {id:string;role:string}[];
 disabled?: boolean;
};

function TechnicalDetailsPanel({view, onDiagnostics}: {view: VoiceView; onDiagnostics: () => void}) {
 const authorization = view.media.state==='authorization-paused' ? 'Aguardando autorização da sala' : view.authorization==='valid' ? 'Autorizada' : 'Em verificação';
 return <details className="voice-technical">
  <summary title="Detalhes técnicos" aria-label="Detalhes técnicos"><Activity size={17}/></summary>
  <div className="voice-technical-card">
   <h3>Detalhes técnicos</h3>
   <dl>
    <div><dt>Mídia</dt><dd>{callStatus(view.media.state,view.participants.length,view.media.packetsOut,view.media.packetsIn)}</dd></div>
    <div><dt>Autorização</dt><dd>{authorization}</dd></div>
    <div><dt>Transporte</dt><dd>{view.media.transport||'Indisponível'}</dd></div>
    <div><dt>Pacotes de rede</dt><dd>TX {view.media.packetsOut??0} · RX {view.media.packetsIn??0}</dd></div>
   </dl>
   <button type="button" className="secondary" onClick={onDiagnostics}>Diagnóstico em tempo real</button>
  </div>
 </details>;
}

function VoiceRoomHeader({name,community,iconHash,status,screen,view,subject,subjectID,peopleOpen,onPeople,sidebarId,onDiagnostics,onFocus}: {
 name: string; community: string; iconHash?:string; status: string; screen: ScreenControlState; view: VoiceView|null;
 subject: string; subjectID?:string; peopleOpen: boolean; onPeople: () => void; sidebarId: string;
 onDiagnostics: () => void; onFocus: () => void;
}) {
 const {quality} = screen;
 return <header className="voice-room-header">
  <AssetIcon hash={iconHash} fallback={<Volume2 size={20}/>} className="voice-header-icon"/>
  <div className="voice-room-heading"><h2 title={name}>{name}</h2><span title={community}>{community}</span></div>
  <span className="voice-room-status" role="status" title={status}>{status}</span>
  {subject&&<div className="voice-room-live"><span className="live-badge">AO VIVO</span><span className="voice-live-name" title={subject}><IdentityName id={subjectID||''} name={subject}/></span>{quality?.width&&quality.height ? <span className="voice-room-quality">{quality.width}×{quality.height}{quality.fps!=null?' · '+Math.round(quality.fps)+' FPS':''}</span>:null}</div>}
  {view&&<div className="voice-header-actions">
   {screen.theater&&<button type="button" className="icon" title="Sair do modo foco" aria-label="Sair do modo foco" onClick={onFocus}><Focus size={18}/></button>}
   <button type="button" className="icon" title="Participantes da sala" aria-label="Participantes da sala" aria-expanded={peopleOpen&&!screen.theater} aria-controls={sidebarId} onClick={onPeople}><Users size={18}/></button>
   <TechnicalDetailsPanel view={view} onDiagnostics={onDiagnostics}/>
  </div>}
 </header>;
}

export function VoiceRoomStage({group,channel,name,community,owner,iconHash,members,disabled=false}: VoiceRoomProps) {
 const voice = useVoice();
 const [peopleOpen,setPeopleOpen] = useState(() => typeof window==='undefined'||!!window.matchMedia?.('(min-width: 1180px)').matches);
 const sidebarId = useId();
 const view = voice.view?.joined&&voice.view.community===group&&voice.view.channel===channel ? voice.view : null;
 const screen: ScreenControlState = view ? voice.screen : {ready:false,sharing:false,busy:false};
 const people = voiceParticipants({view,speaking:voice.speaking,screen,owner,members,watching:screen.playing});
 const live = people.filter(person=>person.live);
 const watched = live.find(person=>person.id===screen.watching);
 const ownStream = live.find(person=>person.self);
 const subject = watched||ownStream;
 const joining = voice.busy;
 const status = !view ? 'Fora da sala'
  : view.media.state==='authorization-paused' ? 'Mídia pausada · aguardando autorização'
  : screen.playing ? 'Assistindo transmissão'
  : screen.watching ? 'Reconectando transmissão'
  : screen.sharing ? 'Transmitindo'
  : people.length<=1 ? 'Aguardando participantes'
  : 'Na sala';

 const showPeople = () => {
  if(screen.theater){voice.focusStream();setPeopleOpen(true);return}
  setPeopleOpen(open=>!open);
 };

 return <section className={'voice-room-stage'+(screen.theater?' is-theater':'')} aria-label={'Sala de voz '+name}>
  <VoiceRoomHeader name={name} community={community} iconHash={iconHash} status={status} screen={screen} view={view} subject={subject?.name||''} subjectID={subject?.id} peopleOpen={peopleOpen} sidebarId={sidebarId} onPeople={showPeople} onDiagnostics={voice.openDiagnostics} onFocus={voice.focusStream}/>
  {view ? <div className="voice-workspace">
   <div className="voice-stage-main">
    <div className="voice-media">
     <div className="voice-media-slot" ref={voice.setStageContainer}/>
     {view.media.state==='authorization-paused'&&<p className="voice-authorization-hint">Você continua na sala. A mídia será retomada quando a autorização for renovada.</p>}
    </div>
    {!screen.theater&&<ParticipantFilmstrip people={people} onWatch={voice.previewStream}>
     {live.length>0&&live.map(person=><button type="button" key={person.id} className={'voice-stream-tile'+(screen.watching===person.id?' is-selected':'')} title={person.self?'Sua transmissão':'Ver transmissão de '+person.name} aria-label={person.self?'Sua transmissão':'Ver transmissão de '+person.name} aria-current={screen.watching===person.id?'true':undefined} disabled={person.self} onClick={()=>voice.previewStream(person.id)}>
      <Monitor size={23}/><span className="live-badge">AO VIVO</span><strong><IdentityName id={person.id} name={person.name}/></strong>
     </button>)}
    </ParticipantFilmstrip>}
    <div className="voice-call-footer">
     <CallControlsBar floating muted={view.muted} deaf={view.deaf} sharing={screen.sharing} screenReady={screen.ready} screenBusy={screen.busy}
      onMute={()=>void voice.control(!view.muted,view.deaf)} onDeaf={()=>void voice.control(view.muted,!view.deaf)}
      onScreen={voice.screenAction} onStreams={voice.showStreams} streamsOpen={screen.streamsOpen}
      onLogs={openCallLogs} onSettings={voice.settings} onLeave={()=>void voice.leave()}/>
    </div>
   </div>
   <div className="voice-sidebar-slot" id={sidebarId} hidden={!peopleOpen||screen.theater}>
    <ParticipantSidebar people={people} onWatch={voice.previewStream} onClose={()=>setPeopleOpen(false)}/>
   </div>
  </div>:<div className="voice-room-lobby">
   <span className="voice-lobby-icon"><Volume2 size={30}/></span>
   <h3>{disabled?'Entrada indisponível':'Entre na sala de voz'}</h3>
   <p>{disabled?'Seu acesso a este canal ainda não está disponível.':'Converse e assista a transmissões com a comunidade.'}</p>
   <button type="button" className="primary" disabled={disabled||joining} onClick={()=>void voice.join(group,channel,name)}>{joining?'Conectando…':'Entrar na sala de voz'}</button>
  </div>}
 </section>;
}
