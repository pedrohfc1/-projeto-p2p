import {CallSoundTracker, cueEnabled, soundStorageSlot, soundPreferences, streamCue, type SoundCue, type SoundPreferences, type SoundVoiceSnapshot} from './callSounds.ts';
import type {ScreenControlState} from './callPresentation.ts';
import {diagnostic} from './diagnostics.ts';
import {LocalCuePlayer, type CuePlayer} from './localCuePlayer.ts';

const transitionLog: Record<SoundCue, string> = {
 'call-connect': 'voice_call_connected',
 'call-disconnect': 'voice_call_disconnected',
 'voice-user-join': 'voice_user_join',
 'voice-user-leave': 'voice_user_leave',
 'stream-start': 'screen_share_started local=false',
 'stream-stop': 'screen_share_stopped local=false',
 'local-share-start': 'screen_share_started local=true',
 'local-share-stop': 'screen_share_stopped local=true',
};

export class SoundEffects {
 private readonly player: CuePlayer;
 private readonly storage?: Pick<Storage, 'getItem' | 'setItem'>;
 private readonly tracker = new CallSoundTracker();
 private readonly subscribers = new Set<() => void>();
 private preferences: SoundPreferences;
 private generation = 0;
 private pending = 0;
 private queue = Promise.resolve();

 constructor(player: CuePlayer, storage?: Pick<Storage, 'getItem' | 'setItem'>) {
  this.player = player;
  this.storage = storage;
  try {
   const raw = this.storage?.getItem(soundStorageSlot);
   this.preferences = soundPreferences(raw ? JSON.parse(raw) : null);
  } catch {
   this.preferences = soundPreferences(null);
  }
  this.player.volume(this.preferences.volume / 100);
 }

 get = () => this.preferences;

 subscribe = (listener: () => void) => {
  this.subscribers.add(listener);
  return () => { this.subscribers.delete(listener); };
 };

 update(patch: Partial<SoundPreferences>) {
  const previous = this.preferences;
  this.preferences = soundPreferences({...previous, ...patch});
  try {
   this.storage?.setItem(soundStorageSlot, JSON.stringify(this.preferences));
  } catch {/* Keep the choice for the current session if storage is unavailable. */}
  const {enabled, people, sharing, volume} = this.preferences;
  if (!enabled || previous.people !== people || previous.sharing !== sharing || !volume) {
   this.generation++;
   this.player.cancel();
  }
  this.player.volume(volume / 100);
  for (const listener of this.subscribers) listener();
 }

 output(sink: string) {
  this.player.output(sink);
 }

 prepare() {
  const {enabled, volume} = this.preferences;
  if (enabled && volume > 0) void this.player.prepare().catch(() => {});
 }

 observe(voice: SoundVoiceSnapshot, screen: ScreenControlState) {
  for (const cue of this.tracker.observe(voice, screen)) {
   diagnostic('INFO', streamCue(cue) ? 'video' : 'voice', 'frontend/eventos de chamada', transitionLog[cue]);
   void this.play(cue);
  }
 }

 async preview() {
  this.prepare();
  const cue = this.preferences.people ? 'call-connect' : 'stream-start';
  return this.play(cue);
 }

 private async play(cue: SoundCue) {
  const generation = this.generation;
  const allowed = () => generation === this.generation && cueEnabled(cue, this.preferences);
  if (!allowed() || this.pending >= 12) return false;
  this.pending++;
  const result = this.queue.then(async () => {
   if (!allowed()) return false;
   try {
    const played = await this.player.play(cue, allowed);
    if (played) diagnostic('INFO', streamCue(cue) ? 'video' : 'voice', 'frontend/efeitos', 'sound_event_played ' + cue);
    return played;
   } catch (error) {
    diagnostic('WARN', streamCue(cue) ? 'video' : 'voice', 'frontend/efeitos', (error as Error).name || 'saída indisponível');
    return false;
   }
  });
  this.queue = result.then(() => { this.pending--; });
  return result;
 }

}

function localPreferences() {
 try { return typeof window === 'undefined' ? undefined : window.localStorage; }
 catch { return undefined; }
}

export const callSounds = new SoundEffects(new LocalCuePlayer(), localPreferences());
