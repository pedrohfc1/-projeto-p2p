export type WatchPeer = {id: string; sharing: boolean; connected: boolean; accepted: boolean; statusKnown?: boolean};

export function watchStatus(id: string, active: boolean, members: string[] | undefined, peers: WatchPeer[]) {
 if (!id) return 'idle';
 if (!active) return 'reconnecting';
 if (members && !members.includes(id)) return 'ended';
 const peer = peers.find(p => p.id === id);
 if (peer?.connected && peer.statusKnown && !peer.sharing) return 'ended';
 return peer?.connected && peer.accepted && peer.sharing && peer.statusKnown ? 'playing' : 'reconnecting';
}

// An existing explicit session can survive hide only when native Stop controls exist.
export function screenVisibilityAction(hidden: boolean, nativeControls: boolean, publishing: boolean, watching: boolean) {
 return !hidden ? 'restore' : nativeControls && (publishing || watching) ? 'keep' : 'stop';
}

export type VideoFault = 'capture' | 'encoder' | 'send' | 'receive' | 'decoder' | 'render' | 'bridge';
export type VideoFlowSample = {
 publishing: boolean;
 viewing: boolean;
 watchers: number;
 sourceMuted: boolean;
 capture: number | null;
 encoded: number | null;
 sent: number | null;
 receiving: number | null;
 localReceived: number | null;
 decoded: number | null;
 rendered: number | null;
 sendQueueDepth: number | null;
 videoPausedForVoice: boolean;
};

export function detectVideoFault(s: VideoFlowSample): VideoFault | null {
 if (s.publishing && s.watchers > 0) {
  if (s.sourceMuted) return 'capture';
  if ((s.capture ?? 0) > 0 && s.encoded === 0) return 'encoder';
  if ((s.encoded ?? 0) > 0 && s.sent === 0 && (s.sendQueueDepth ?? 0) > 0 && !s.videoPausedForVoice) return 'send';
 }
 if (s.viewing) {
  if ((s.receiving ?? 0) > 0 && s.localReceived === 0) return 'receive';
  if ((s.localReceived ?? 0) > 0 && s.decoded === 0) return 'decoder';
  if ((s.decoded ?? 0) > 0 && s.rendered === 0) return 'render';
 }
 // Stages without positive upstream evidence may be idle (static desktop),
 // unmeasurable or suspended by an intentional priority policy.
 return null;
}

export class VideoRecoveryBudget {
 private fault: VideoFault | null = null;
 private since = 0;
 private lastSample = 0;
 private lastAttempt = -Infinity;
 private healthySince: number | null = null;
 private attempts = 0;

 observe(fault: VideoFault | null, now: number, hasProgress: boolean) {
  if (!fault) {
   this.fault = null;
   if (!hasProgress) this.healthySince = null;
   else {
    this.healthySince ??= now;
    if (now - this.healthySince >= 12_000) {
     this.attempts = 0;
     this.lastAttempt = -Infinity;
    }
   }
   this.lastSample = now;
   return null;
  }
  this.healthySince = null;
  if (fault !== this.fault || now - this.lastSample > 3500) {
   this.fault = fault;
   this.since = now;
  }
  this.lastSample = now;
  const threshold = fault === 'bridge' ? 0 : 4000;
  if (now - this.since < threshold || now - this.lastAttempt < 12_000 || this.attempts >= 3) return null;
  this.attempts++;
  this.lastAttempt = now;
  return {fault, attempt: this.attempts};
 }
}
