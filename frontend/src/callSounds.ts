import type {ScreenControlState} from './callPresentation.ts';

export type SoundCue =
 'call-connect' | 'call-disconnect' | 'voice-user-join' | 'voice-user-leave' |
 'stream-start' | 'stream-stop' | 'local-share-start' | 'local-share-stop';

export type SoundPreferences = {
 enabled: boolean;
 volume: number;
 people: boolean;
 sharing: boolean;
};

export const soundStorageSlot = 'peetwopee.sound-effects.v1';
export const soundDefaults: SoundPreferences = {
 enabled: true, volume: 40, people: true, sharing: true,
};

export function soundPreferences(value: unknown): SoundPreferences {
 const item = value && typeof value === 'object' ? value as Partial<SoundPreferences> : {};
 return {
  enabled: typeof item.enabled === 'boolean' ? item.enabled : soundDefaults.enabled,
  volume: typeof item.volume === 'number' && Number.isFinite(item.volume) ? Math.min(100, Math.max(0, Math.round(item.volume))) : soundDefaults.volume,
  people: typeof item.people === 'boolean' ? item.people : soundDefaults.people,
  sharing: typeof item.sharing === 'boolean' ? item.sharing : soundDefaults.sharing,
 };
}

export function streamCue(cue: SoundCue) {
 return cue.startsWith('stream-') || cue.startsWith('local-share-');
}

export function cueEnabled(cue: SoundCue, preferences: SoundPreferences) {
 return preferences.enabled && preferences.volume > 0 && (streamCue(cue) ? preferences.sharing : preferences.people);
}

type VoiceSnapshot = {
 joined: boolean;
 session: string;
 community: string;
 channel: string;
 authorization?: string;
 participants: readonly {id: string; self: boolean}[];
 media: {state: string};
};

export type SoundVoiceSnapshot = VoiceSnapshot | null;

type RoomState = {
 session: string;
 community: string;
 channel: string;
 people: Set<string> | null;
 streams: Map<string, boolean>;
 localSharing: boolean;
};

// Audio uses the signed room roster for membership and peer acknowledgements for
// publications. Media pipe state only tells us whether a publication can be observed.
export class CallSoundTracker {
 private room?: RoomState;

 observe(voice: SoundVoiceSnapshot, screen: ScreenControlState): SoundCue[] {
  const events: SoundCue[] = [];
  const previous = this.room;
  if (!voice?.joined || !voice.session) {
   if (previous) {
    if (previous.localSharing) events.push('local-share-stop');
    events.push('call-disconnect');
   }
   this.room = undefined;
   return events;
  }

  const switched = previous && (previous.community !== voice.community || previous.channel !== voice.channel);
  if (switched) {
   if (previous.localSharing) events.push('local-share-stop');
   events.push('call-disconnect');
  }
  if (!previous || switched) events.push('call-connect');
  if (!previous || switched || previous.session !== voice.session) {
   this.room = {
    session: voice.session, community: voice.community, channel: voice.channel,
    people: null, streams: new Map(), localSharing: !switched && !!previous?.localSharing,
   };
  }

  const room = this.room!;
  if (screen.session === voice.session && screen.sharing !== room.localSharing) {
   room.localSharing = screen.sharing;
   events.push(screen.sharing ? 'local-share-start' : 'local-share-stop');
  }

  const reliable = voice.authorization !== 'paused' && voice.media.state !== 'authorization-paused' &&
   voice.participants.some(participant => participant.self);
  if (!reliable) {
   // Skip catch-up sounds after a roster that was not valid for media.
   room.people = null;
   room.streams.clear();
   return events;
  }

  const people = new Set(voice.participants.filter(participant => !participant.self && participant.id).map(participant => participant.id));
  if (room.people) {
   for (const id of people) if (!room.people.has(id)) events.push('voice-user-join');
   for (const id of room.people) {
    if (people.has(id)) continue;
    if (room.streams.get(id)) events.push('stream-stop');
    room.streams.delete(id);
    events.push('voice-user-leave');
   }
  }
  room.people = people;

  if (screen.session === voice.session) {
   for (const peer of screen.peers ?? []) {
    if (!people.has(peer.id) || !peer.connected || peer.statusKnown !== true) continue;
    const previousSharing = room.streams.get(peer.id);
    room.streams.set(peer.id, peer.sharing);
    if (previousSharing !== undefined && previousSharing !== peer.sharing) {
     events.push(peer.sharing ? 'stream-start' : 'stream-stop');
    }
   }
  }
  return events;
 }
}
