import type { MouseEvent, ReactNode } from 'react';
import {IdentityAvatar,IdentityName} from './ProfileIdentity';
import { ShieldCheck,AudioLines, Crown, Ellipsis, HeadphoneOff, Headphones, Mic, MicOff, Users, X } from 'lucide-react';
import type { ScreenControlState } from './callPresentation';
import { participantContext, participantKeyboard, type ParticipantTarget } from './participantMenu';
import type { VoiceView } from './Voice';

export type VoiceParticipant = {
 id: string;
 name: string;
 self: boolean;
 owner: boolean;
 moderator?:boolean;
 speaking: boolean;
 muted?: boolean;
 deaf?: boolean;
 live: boolean;
 status: string;
 group: 'Em voz' | 'Assistindo' | 'Aguardando';
};

export function voiceParticipants({view, speaking, screen, owner, members, watching}: {
 view: VoiceView | null;
 speaking: Record<string, number>;
 screen: ScreenControlState;
 owner?: string;
 members?: {id:string;role:string}[];
 watching?: string;
}): VoiceParticipant[] {
 if (!view?.joined) return [];
 const pending = view.authorization === 'paused' || view.media.state === 'authorization-paused';
 const livePeers = new Set(screen.peers?.filter(peer => peer.sharing && peer.connected === true).map(peer => peer.id));
 const unique = new Set<string>();
 const people: VoiceParticipant[] = [];
 for (const peer of view.participants) {
  if (!peer.id || unique.has(peer.id)) continue;
  unique.add(peer.id);
  const muted = peer.self ? view.muted || view.deaf : peer.muted;
  const deaf = peer.self ? view.deaf : peer.deaf;
  const talking = !pending && !muted && !deaf && (speaking[peer.id] ?? 0) > .08;
  const watchingNow = !pending && peer.self && !!watching && livePeers.has(watching);
  const inVoice = peer.self || peer.state === 'connected' || talking;
  const group = pending ? 'Aguardando' : watchingNow ? 'Assistindo' : inVoice ? 'Em voz' : 'Aguardando';
  let status: string = group;
  if (pending) status = 'Aguardando autorização';
  else if (talking) status = 'Falando';
  else if (group === 'Aguardando') {
   if (peer.state === 'reconnecting') status = 'Reconectando';
   else if (peer.state === 'connecting') status = 'Conectando';
   else status = 'Aguardando conexão';
  }
  people.push({
   id: peer.id, name: peer.name?.trim() || 'Participante', self: peer.self, owner: peer.id === owner,
   moderator: members?.some(member=>member.id===peer.id&&member.role==='moderator'),
   speaking: talking, muted, deaf, live: !pending && (peer.self ? screen.sharing : livePeers.has(peer.id)),
   status, group,
  });
 }
 return people;
}

function participantTarget(person: VoiceParticipant, onWatch: (id: string) => void): ParticipantTarget {
 return {
  id: person.id, name: person.name, self: person.self,
  profile: () => window.dispatchEvent(new CustomEvent('peetwopee:profile', {detail: {id: person.id, name: person.name}})),
  extra: person.live && !person.self ? [{label: 'Ver transmissão', run: () => onWatch(person.id)}] : undefined,
 };
}

function openOptions(event: MouseEvent<HTMLButtonElement>, target: ParticipantTarget) {
 const rect = event.currentTarget.getBoundingClientRect();
 participantContext({
  currentTarget: event.currentTarget, clientX: rect.left, clientY: rect.bottom + 4,
  preventDefault: () => event.preventDefault(), stopPropagation: () => event.stopPropagation(),
 }, target);
}

const avatarTones = ['mauve', 'blue', 'clay', 'aqua', 'indigo'] as const;
export function ParticipantAvatar({name, id}: {name: string; id: string}) {
 const tone = Array.from(id).reduce((sum, character) => (sum * 31 + character.charCodeAt(0)) % avatarTones.length, 0);
 return <IdentityAvatar name={name} id={id} className={'voice-participant-avatar tone-' + avatarTones[tone]}/>;
}

function ParticipantAudioIcons({person}: {person: VoiceParticipant}) {
 const Microphone = person.muted ? MicOff : Mic;
 const Headset = person.deaf ? HeadphoneOff : Headphones;
 return <span className="voice-participant-audio">
  {person.muted !== undefined && <span className={person.muted ? 'is-off' : undefined} title={person.muted ? 'Microfone silenciado' : 'Microfone habilitado'}><Microphone size={13} role="img" aria-label={person.muted ? 'Microfone silenciado' : 'Microfone habilitado'}/></span>}
  {person.deaf !== undefined && <span className={person.deaf ? 'is-off' : undefined} title={person.deaf ? 'Saída de áudio desativada' : 'Saída de áudio habilitada'}><Headset size={13} role="img" aria-label={person.deaf ? 'Saída de áudio desativada' : 'Saída de áudio habilitada'}/></span>}
 </span>;
}

function ParticipantLive({person, onWatch}: {person: VoiceParticipant; onWatch: (id: string) => void}) {
 if (!person.live) return null;
 if (person.self) return <span className="voice-participant-live">AO VIVO</span>;
 return <button type="button" className="voice-participant-live" title={'Ver transmissão de ' + person.name} aria-label={'Ver transmissão de ' + person.name} onClick={() => onWatch(person.id)}>AO VIVO</button>;
}

function participantClass(person: VoiceParticipant) {
 return [person.self && 'is-self', person.speaking && 'is-speaking', person.live && 'is-live'].filter(Boolean).join(' ');
}

export function ParticipantTile({person, onWatch}: {person: VoiceParticipant; onWatch: (id: string) => void}) {
 const target = participantTarget(person, onWatch);
 return <article className={'voice-participant-tile ' + participantClass(person)} aria-label={'Participante ' + person.name}
  tabIndex={-1} onContextMenu={event => participantContext(event, target)} onKeyDown={event => participantKeyboard(event, target)}>
  <button type="button" className="voice-tile-profile" title={'Ver perfil de ' + person.name} aria-label={'Ver perfil de ' + person.name} onClick={target.profile}>
   <ParticipantAvatar id={person.id} name={person.name}/>
   <span className="voice-tile-name"><b><IdentityName id={person.id} name={person.name}/></b>{person.self && <small>Você</small>}</span>
  </button>
  <div className="voice-tile-top">
   <ParticipantLive person={person} onWatch={onWatch}/>
   <button type="button" className="voice-participant-options" aria-haspopup="menu" aria-label={'Opções de ' + person.name} title={'Opções de ' + person.name} onClick={event => openOptions(event, target)}><Ellipsis size={16}/></button>
  </div>
  <div className="voice-tile-status">
   {person.speaking && <span className="voice-participant-speaking" title="Falando"><AudioLines size={13} role="img" aria-label="Falando"/></span>}
   <ParticipantAudioIcons person={person}/>
  </div>
 </article>;
}

export function ParticipantFilmstrip({people, onWatch, children}: {people: readonly VoiceParticipant[]; onWatch: (id: string) => void; children?: ReactNode}) {
 if (!people.length && !children) return null;
 return <div className="voice-participant-filmstrip" role="region" aria-label="Miniaturas da sala">
  <div className="voice-participant-filmstrip-inner">{children}{people.map(person => <ParticipantTile key={person.id} person={person} onWatch={onWatch}/>)}</div>
 </div>;
}

function ParticipantRow({person, onWatch}: {person: VoiceParticipant; onWatch: (id: string) => void}) {
 const target = participantTarget(person, onWatch);
 return <li className={'voice-participant-row ' + participantClass(person)} tabIndex={-1}
  onContextMenu={event => participantContext(event, target)} onKeyDown={event => participantKeyboard(event, target)}>
  <button type="button" className="voice-participant-profile" title={'Ver perfil de ' + person.name} aria-label={'Ver perfil de ' + person.name} onClick={target.profile}>
   <ParticipantAvatar id={person.id} name={person.name}/>
   <span className="voice-participant-label"><strong><IdentityName id={person.id} name={person.name}/>{person.self && <small>Você</small>}</strong>
    <span className={person.speaking ? 'voice-participant-speaking' : undefined}>{person.speaking && <AudioLines size={12} aria-hidden="true"/>}{person.status}</span>
   </span>
  </button>
  <ParticipantLive person={person} onWatch={onWatch}/>
  <button type="button" className="voice-participant-options" aria-haspopup="menu" aria-label={'Opções de ' + person.name} title={'Opções de ' + person.name} onClick={event => openOptions(event, target)}><Ellipsis size={16}/></button>
  {(person.owner || person.moderator || person.muted !== undefined || person.deaf !== undefined) && <div className="voice-participant-meta">
   {person.owner&&<span className="voice-participant-owner"><Crown size={12} aria-hidden="true"/>Proprietário</span>}
   {person.moderator&&!person.owner&&<span className="voice-participant-owner"><ShieldCheck size={12} aria-hidden="true"/>Moderador</span>}
   <ParticipantAudioIcons person={person}/>
  </div>}
 </li>;
}

const participantGroups = ['Em voz', 'Assistindo', 'Aguardando'] as const;
export function ParticipantSidebar({people, onWatch, onClose}: {people: readonly VoiceParticipant[]; onWatch: (id: string) => void; onClose?: () => void}) {
 return <aside className="voice-participant-sidebar" aria-label="Participantes da sala">
  <div className="voice-participant-sidebar-heading"><Users size={17} aria-hidden="true"/><h3>Participantes</h3><span className="voice-participant-count" aria-label={`${people.length} na sala`}>{people.length}</span>
   {onClose && <button type="button" className="voice-participant-options" title="Ocultar participantes" aria-label="Ocultar participantes" onClick={onClose}><X size={17}/></button>}
  </div>
  <div className="voice-participant-sidebar-list">
   {participantGroups.map(group => {
    const members = people.filter(person => person.group === group);
    if (!members.length) return null;
    return <section className="voice-participant-section" key={group} aria-label={group}><h4>{group} <span>{members.length}</span></h4><ul>{members.map(person => <ParticipantRow key={person.id} person={person} onWatch={onWatch}/>)}</ul></section>;
   })}
   {!people.length && <p className="voice-participant-empty">Aguardando participantes da sala.</p>}
  </div>
 </aside>;
}
