import {selectAudioOutput} from './audioDevices.ts';
import {activateVoice} from './voiceActivation.ts';
import type {SoundCue} from './callSounds.ts';

const clips: Record<SoundCue, () => string> = {
 'call-connect': () => new URL('./assets/sounds/call-connect.wav', import.meta.url).href,
 'call-disconnect': () => new URL('./assets/sounds/call-disconnect.wav', import.meta.url).href,
 'voice-user-join': () => new URL('./assets/sounds/voice-user-join.wav', import.meta.url).href,
 'voice-user-leave': () => new URL('./assets/sounds/voice-user-leave.wav', import.meta.url).href,
 'stream-start': () => new URL('./assets/sounds/stream-start.wav', import.meta.url).href,
 'stream-stop': () => new URL('./assets/sounds/stream-stop.wav', import.meta.url).href,
 'local-share-start': () => new URL('./assets/sounds/local-share-start.wav', import.meta.url).href,
 'local-share-stop': () => new URL('./assets/sounds/local-share-stop.wav', import.meta.url).href,
};

export type CuePlayer = {
 prepare(): Promise<void>;
 play(cue: SoundCue, allowed: () => boolean): Promise<boolean>;
 volume(value: number): void;
 output(value: string): void;
 cancel(): void;
};

type Playback = {
 audio: HTMLAudioElement;
 context: AudioContext;
 destination: MediaStreamAudioDestinationNode;
 clips: Map<SoundCue, Promise<AudioBuffer>>;
 sources: Set<AudioBufferSourceNode>;
 nextStart: number;
 output?: string;
 route?: {id: string; task: Promise<void>};
 activation?: Promise<void>;
};

// The destination is connected only to a local playback element. In particular
// it is never connected to the microphone, WebRTC, or the system-audio bridge.
export class LocalCuePlayer implements CuePlayer {
 private playback?: Playback;
 private sink = '';
 private level = .4;
 private idle?: ReturnType<typeof setTimeout>;

 private create() {
  const context = new AudioContext();
  const destination = context.createMediaStreamDestination();
  const audio = new Audio();
  audio.muted = !!this.sink;
  audio.srcObject = destination.stream;
  audio.volume = this.level;
  const playback: Playback = {context, destination, audio, clips: new Map(), sources: new Set(), nextStart: 0, output: ''};
  this.playback = playback;
  return playback;
 }

 async prepare() {
  const playback = this.playback ?? this.create();
  this.touch();
  if (!playback.activation && (playback.context.state !== 'running' || playback.audio.paused)) {
   playback.activation = activateVoice([playback.context], playback.audio, 800).finally(() => {
    playback.activation = undefined;
   });
  }
  await Promise.all([playback.activation, this.route(playback)]);
 }

 async play(cue: SoundCue, allowed: () => boolean) {
  await this.prepare();
  const playback = this.playback;
  if (!playback || !allowed()) return false;
  const buffer = await this.buffer(playback, cue);
  if (this.playback !== playback || playback.output !== this.sink || playback.audio.muted || !allowed()) return false;
  const now = playback.context.currentTime;
  const start = Math.max(now, playback.nextStart);
  if (start > now + 2) return false;
  const source = playback.context.createBufferSource();
  source.buffer = buffer;
  source.connect(playback.destination);
  source.onended = () => {
   source.disconnect();
   playback.sources.delete(source);
  };
  source.start(start);
  playback.sources.add(source);
  playback.nextStart = start + buffer.duration + .025;
  this.touch();
  return true;
 }

 volume(value: number) {
  this.level = value;
  if (this.playback) this.playback.audio.volume = value;
 }

 output(value: string) {
  if (this.sink === value) return;
  this.sink = value;
  const playback = this.playback;
  if (!playback) return;
  this.cancel();
  void this.route(playback).catch(() => {});
 }

 cancel() {
  const playback = this.playback;
  if (!playback) return;
  for (const source of playback.sources) {
   source.onended = null;
   source.stop();
   source.disconnect();
  }
  playback.sources.clear();
  playback.nextStart = 0;
 }

 private route(playback: Playback): Promise<void> {
  const id = this.sink;
  if (playback.route?.id === id) return playback.route.task;
  if (!playback.route && playback.output === id) return Promise.resolve();

  playback.audio.muted = true;
  const prior = playback.route?.task.catch(() => {}) ?? Promise.resolve();
  const task = prior.then(async () => {
   if (this.playback !== playback || this.sink !== id) return;
   // Leave output suspended when selection fails. Do not fall back to an output
   // currently being captured for screen sharing.
   await selectAudioOutput(playback.audio, id);
   if (this.playback === playback && this.sink === id) {
    playback.output = id;
    playback.audio.muted = false;
   }
  }).finally(() => {
   if (playback.route?.task === task) playback.route = undefined;
  });
  playback.route = {id, task};
  return task;
 }

 private buffer(playback: Playback, cue: SoundCue) {
  const cached = playback.clips.get(cue);
  if (cached) return cached;
  const pending = fetch(clips[cue]()).then(async result => {
   if (!result.ok) throw Error('Efeito sonoro indisponível.');
   return playback.context.decodeAudioData(await result.arrayBuffer());
  }).catch(error => {
   playback.clips.delete(cue);
   throw error;
  });
  playback.clips.set(cue, pending);
  return pending;
 }

 private touch() {
  clearTimeout(this.idle);
  this.idle = setTimeout(() => {
   const playback = this.playback;
   if (!playback) return;
   this.cancel();
   this.playback = undefined;
   playback.audio.pause();
   playback.audio.srcObject = null;
   for (const track of playback.destination.stream.getTracks()) track.stop();
   void playback.context.close().catch(() => {});
  }, 60_000);
 }
}
