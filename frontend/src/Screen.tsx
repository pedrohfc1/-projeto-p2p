import {liveTelemetry} from './mediaDiagnosticModel';
import {TransmissionFields} from './TransmissionSettings';
import {readTransmissionPreference, saveTransmissionPreference} from './transmissionPreferences';
import {SettingsSurface} from './SettingsCenter';
import {SystemAudioPanel, type SoundControls} from './SystemAudio';
import type {SoundAPI} from './screenSound';
import {ScreenViewerOverlay} from './ScreenViewer';
import {diagnostic} from './diagnostics';
import {detectVideoFault, VideoRecoveryBudget, watchStatus, screenVisibilityAction, type VideoFault} from './screenSession';
import {VideoTelemetry} from './videoTelemetry';
import {captureLimits, inspectVideoCapabilities, selectCaptureProfile, VideoQualityGovernor, type VideoCapabilities} from './videoCapabilities';
import type {ScreenControls, ScreenControlState} from './callPresentation';
import React, {useEffect, useRef, useState, useImperativeHandle, forwardRef} from 'react';
import {Monitor, Square, Maximize2, X} from 'lucide-react';
import {ScreenCapture, screenFailure, videoBitrate, screenProfiles, type ScreenProfile} from './screenCapture';
import {isBackground} from './background';

export type ScreenView = {
 dropReasons?: Record<string, number>;
 sendQueue?: {depth: number; capacity: number; lastWaitMs: number | null};
 audioPriorityPaused?: boolean;
 participants?: string[];
 playing?: string;
 roundTripMs: number;
 active: boolean;
 publishing: boolean;
 watching: string;
 viewers: number;
 bitrate: number;
 maxViewers: number;
 framesIn: number;
 framesOut: number;
 dropped: number;
 bytesIn: number;
 bytesOut: number;
 peers: {id: string; name: string; sharing: boolean; connected: boolean; accepted: boolean; statusKnown?: boolean; pipeGeneration?: number}[];
 error: string;
};
export type ScreenAPI = SoundAPI & {
 ScreenBackgroundAllowed?: () => Promise<boolean>;
 ScreenConnect: (offer: RTCSessionDescriptionInit) => Promise<RTCSessionDescriptionInit>;
 ScreenPublish: (publish: boolean, bitrate: number, maxViewers: number) => Promise<void>;
 ScreenWatch: (id: string) => Promise<void>;
 ScreenState: () => Promise<ScreenView>;
 ScreenHeartbeat: () => Promise<void>;
 ScreenStop: () => Promise<void>;
};

type RenderMetrics = { fps: number; width: number; height: number };
type ScreenRecoveryEvent = {at: number; message: string};
const faultLabels: Record<VideoFault, string> = {
 capture: 'captura suspensa',
 encoder: 'encoder sem quadros',
 send: 'envio sem avanço',
 receive: 'recepção sem avanço na ponte',
 decoder: 'decoder sem quadros',
 render: 'renderização sem quadros',
 bridge: 'ponte WebRTC de vídeo desconectada',
};

function Video({stream, preview = false, onRendered, retry = 0}: {
 stream: MediaStream | null;
 preview?: boolean;
 onRendered?: (metrics: RenderMetrics) => void;
 retry?: number;
}) {
 const element = useRef<HTMLVideoElement | null>(null);
 const rendered = useRef(onRendered);
 rendered.current = onRendered;
 useEffect(() => {
  const video = element.current;
  if (!video) return;
  video.srcObject = stream;
  let handle = 0;
  let last = 0;
  let at: number | null = null;
  let lastFrameAt = performance.now();
  let width = 0;
  let height = 0;
  let stalled: ReturnType<typeof setInterval> | undefined;
  const frame = (now: number, meta: VideoFrameCallbackMetadata) => {
   lastFrameAt = performance.now();
   width = meta.width;
   height = meta.height;
   if (at === null) {
    at = now;
    last = meta.presentedFrames;
   } else if (now - at >= 1000) {
    rendered.current?.({fps: (meta.presentedFrames - last) * 1000 / (now - at), width, height});
    at = now;
    last = meta.presentedFrames;
   }
   handle = video.requestVideoFrameCallback(frame);
  };
  if (stream && rendered.current && video.requestVideoFrameCallback) {
   handle = video.requestVideoFrameCallback(frame);
   stalled = setInterval(() => {
    if (at !== null && performance.now() - lastFrameAt > 2000 && !isBackground()) {
     rendered.current?.({fps: 0, width, height});
    }
   }, 1000);
  }
  if (stream) void video.play().catch(e =>
   diagnostic('WARN', 'video', 'frontend/reprodução', (e as Error).name || 'Falha ao reproduzir'));
  return () => {
   clearInterval(stalled);
   if (handle) video.cancelVideoFrameCallback(handle);
   video.pause();
   video.srcObject = null;
  };
 }, [stream, retry]);
 return <video ref={element} muted autoPlay playsInline aria-label={preview ? 'Prévia local da fonte selecionada' : 'Tela compartilhada pelo participante'}/>;
}

function waitForCandidates(connection: RTCPeerConnection) {
 if (connection.iceGatheringState === 'complete') return Promise.resolve();
 return new Promise<void>((resolve, reject) => {
  const timeout = setTimeout(() => {
   connection.removeEventListener('icegatheringstatechange', check);
   reject(Error('Tempo esgotado na ponte local de vídeo.'));
  }, 5000);
  function check() {
   if (connection.iceGatheringState !== 'complete') return;
   clearTimeout(timeout);
   connection.removeEventListener('icegatheringstatechange', check);
   resolve();
  }
  connection.addEventListener('icegatheringstatechange', check);
  check();
 });
}

async function setSenderLimits(send: RTCRtpSender, bitrate: number, fps: number, initial = false) {
 const params = send.getParameters();
 const encoding = params.encodings?.[0];
 if (!encoding) {
  if (initial) throw Error('Encoder não forneceu limites de transmissão.');
  return;
 }
 if (encoding.maxBitrate === bitrate && encoding.maxFramerate === fps
     && encoding.priority === 'low' && params.degradationPreference === 'maintain-framerate') return;
 encoding.maxBitrate = bitrate;
 encoding.maxFramerate = fps;
 encoding.priority = 'low';
 params.degradationPreference = 'maintain-framerate';
 await send.setParameters(params);
}

export const ScreenPanel = forwardRef<ScreenControls, {
 api: () => ScreenAPI;
 session: string;
 authorization?: string;
 expires?: number;
 output?: string;
 onControlState: (state: ScreenControlState) => void;
}>(function ScreenPanel({api, session, authorization, expires, output = '', onControlState}, controlRef) {
 const [defaults] = useState(readTransmissionPreference);
 const [state, setState] = useState<ScreenView | null>(null);
 const [preview, setPreview] = useState<MediaStream | null>(null);
 const [remote, setRemote] = useState<MediaStream | null>(null);
 const [error, setError] = useState('');
 const [busy, setBusy] = useState(false);
 const [ready, setReady] = useState(false);
 const [profile, setProfile] = useState<ScreenProfile>(defaults.profile);
 const [limit, setLimit] = useState(defaults.limit);
 const [maxViewers, setMaxViewers] = useState(defaults.viewers);
 const [pick, setPick] = useState(false);
 const [expanded, setExpanded] = useState(false);
 const [remotePreview, setRemotePreview] = useState('');
 const [compact, setCompact] = useState(true);
 const [endedNotice, setEndedNotice] = useState('');
 const [controlsVisible, setControlsVisible] = useState(true);
 const [capabilities, setCapabilities] = useState<VideoCapabilities>({});
 const [captureNote, setCaptureNote] = useState('');
 const [rates, setRates] = useState<ReturnType<VideoTelemetry['sample']> | null>(null);
 const [render, setRender] = useState<RenderMetrics | null>(null);
 const [playRetry, setPlayRetry] = useState(0);
 const [lastRecovery, setLastRecovery] = useState<ScreenRecoveryEvent | null>(null);

 const stage = useRef<HTMLDivElement | null>(null);
 const sound = useRef<SoundControls | null>(null);
 const mounted = useRef(true);
 const backgroundAllowed = useRef(false);
 const roomAuthorization = useRef({authorization, expires});
 roomAuthorization.current = {authorization, expires};
 const connection = useRef<RTCPeerConnection | null>(null);
 const sender = useRef<RTCRtpSender | null>(null);
 const capture = useRef<ScreenCapture | null>(null);
 const turn = useRef(0);
 const initializing = useRef(false);
 const readyRef = useRef(false);
 const nextIdleConnectAt = useRef(0);
 const watching = useRef('');
 const watchVersion = useRef(0);
 const publishing = useRef(false);
 const publishVersion = useRef(0);
 const publishPending = useRef(false);
 const stopPromise = useRef<Promise<unknown>>(Promise.resolve());
 const selectedQuality = useRef<ScreenProfile>('720p30');
 const captureQuality = useRef(screenProfiles['720p30']);
 const profileRef = useRef(profile);
 const limitRef = useRef(limit);
 const viewersRef = useRef(maxViewers);
 const renderRef = useRef<RenderMetrics | null>(null);
 const telemetry = useRef(new VideoTelemetry());
 const recovery = useRef(new VideoRecoveryBudget());
 const recovering = useRef<VideoFault | null>(null);
 const governor = useRef(new VideoQualityGovernor());
 const encoderRate = useRef(limit);
 const previousDrops = useRef(0);
 const lastEncoderLog = useRef(0);
 const controlsTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

 profileRef.current = profile;
 limitRef.current = limit;
 viewersRef.current = maxViewers;
 renderRef.current = render;

 useEffect(() => saveTransmissionPreference({profile, limit, viewers: maxViewers}), [profile, limit, maxViewers]);
 useEffect(() => {
  let disposed = false;
  void inspectVideoCapabilities().then(value => { if (!disposed) setCapabilities(value); });
  return () => { disposed = true; };
 }, []);

 function revealControls() {
  setControlsVisible(true);
  clearTimeout(controlsTimer.current);
  controlsTimer.current = setTimeout(() => setControlsVisible(false), 3000);
 }
 useEffect(() => () => clearTimeout(controlsTimer.current), []);

 function reportRecovery(fault: VideoFault, outcome: string, level: 'INFO' | 'WARN' = 'INFO') {
  if (!mounted.current) return;
  const message = faultLabels[fault] + ' · ' + outcome;
  setLastRecovery({at: Date.now(), message});
  diagnostic(level, 'video', 'frontend/recuperação', message);
 }

 function stopSharing(clearAudioConsent = true) {
  sound.current?.stop(clearAudioConsent);
  publishVersion.current++;
  publishPending.current = false;
  publishing.current = false;
  capture.current?.stop();
  setPreview(null);
  setPick(false);
  governor.current.reset();
  if (sender.current) void sender.current.replaceTrack(null).catch(() => {});
  stopPromise.current = stopPromise.current.catch(() => {}).then(() =>
   api().ScreenPublish(false, limitRef.current, viewersRef.current)).catch(() => {});
  setState(value => value ? {...value, publishing: false} : value);
 }
 if (!capture.current) capture.current = new ScreenCapture(() => stopSharing());

 function resetBridge() {
  turn.current++;
  readyRef.current = false;
  connection.current?.close();
  connection.current = null;
  sender.current = null;
  telemetry.current.reset();
  setRates(null);
  setRemote(null);
  setRender(null);
  setReady(false);
 }

 function hasRoomAuthorization() {
  const lease = roomAuthorization.current;
  return lease.authorization === 'valid' && (lease.expires ?? 0) > Date.now() / 1000;
 }

 function requiresConsentAgain() {
  const lease = roomAuthorization.current;
  return lease.authorization !== undefined && !hasRoomAuthorization();
 }

 function closeBridge() {
  stopSharing();
  watchVersion.current++;
  watching.current = '';
  setRemotePreview('');
  resetBridge();
  stopPromise.current = stopPromise.current.catch(() => {}).then(() => api().ScreenStop()).catch(() => {});
 }

 // A successful rebind may reuse only the very same live, user-approved track.
 // The backend verifies the current room lease again before Connect/Publish/Watch.
 async function reapplyIntent(send: RTCRtpSender, epoch: number) {
  const approval = publishVersion.current;
  const track = capture.current?.stream?.getVideoTracks()[0];
  if (publishing.current && track?.readyState === 'live' && !isBackground()) {
   await send.replaceTrack(track);
   if (approval !== publishVersion.current || epoch !== turn.current || isBackground()) return;
   await setSenderLimits(send, encoderRate.current, captureQuality.current.fps, true);
   if (approval !== publishVersion.current || epoch !== turn.current || isBackground()) return;
   const result = api().ScreenPublish(true, limitRef.current, viewersRef.current);
   stopPromise.current = result;
   await result;
  }
 }

 async function ensureBridge(resuming = false) {
  if (initializing.current || connection.current || isBackground() || !mounted.current ||
      (!resuming && performance.now() < nextIdleConnectAt.current)) return false;
  initializing.current = true;
  const epoch = turn.current;
  let pc: RTCPeerConnection | null = null;
  try {
   backgroundAllowed.current = await api().ScreenBackgroundAllowed?.() ?? false;
   await stopPromise.current.catch(() => {});
   if (epoch !== turn.current || isBackground() || !mounted.current) return false;
   pc = new RTCPeerConnection({iceServers: [], bundlePolicy: 'max-bundle'});
   connection.current = pc;
   const transceiver = pc.addTransceiver('video', {direction: 'sendrecv'});
   const codecs = RTCRtpSender.getCapabilities?.('video')?.codecs.filter(codec => codec.mimeType.toLowerCase() === 'video/vp8');
   if (codecs?.length) transceiver.setCodecPreferences(codecs);
   sender.current = transceiver.sender;
   pc.ontrack = event => {
    if (epoch !== turn.current || !mounted.current) return;
    renderRef.current = null;
    setRender(null);
    setRemote(event.streams[0] || new MediaStream([event.track]));
   };
   const offer = await pc.createOffer();
   await pc.setLocalDescription(offer);
   await waitForCandidates(pc);
   if (epoch !== turn.current || isBackground() || !mounted.current) return false;
   const answer = await api().ScreenConnect({type: 'offer', sdp: pc.localDescription!.sdp});
   if (epoch !== turn.current || !mounted.current) {
    // No successor bridge can open while this exchange owns initializing.
    await api().ScreenStop().catch(() => {});
    return false;
   }
   await pc.setRemoteDescription(answer);
   if (epoch !== turn.current || !mounted.current) return false;
   if (resuming) await reapplyIntent(transceiver.sender, epoch);
   if (epoch !== turn.current || !mounted.current) return false;
   readyRef.current = true;
   setReady(true);
   setError('');
   return true;
  } catch (e) {
   if (epoch === turn.current && mounted.current) {
    nextIdleConnectAt.current = performance.now() + 4000;
    setError('Ponte de vídeo: ' + String(e));
   }
   return false;
  } finally {
   if (pc && (!readyRef.current || connection.current !== pc)) {
    pc.close();
    if (connection.current === pc) {
     connection.current = null;
     sender.current = null;
    }
   }
   initializing.current = false;
  }
 }

 async function recoverVideo(fault: VideoFault, attempt: number) {
  if (isBackground() || !mounted.current || requiresConsentAgain()) return;
  reportRecovery(fault, 'tentativa ' + attempt);
  recovering.current = fault;
  try {
   if (fault === 'render') {
    setPlayRetry(value => value + 1);
    return;
   }
   if (fault === 'receive' || fault === 'decoder') {
    const id = watching.current;
    if (id) await api().ScreenWatch(id);
    return;
   }
   if (fault === 'capture' || fault === 'encoder') {
    const track = capture.current?.stream?.getVideoTracks()[0];
    const approvedSender = sender.current;
    const approval = publishVersion.current;
    if (!publishing.current || !track || track.readyState !== 'live' || !approvedSender) return;
    if (fault === 'capture') {
     await track.applyConstraints(captureLimits(captureQuality.current));
     return;
    }
    await approvedSender.replaceTrack(null);
    if (approval !== publishVersion.current || !publishing.current || sender.current !== approvedSender) return;
    await approvedSender.replaceTrack(track);
    if (approval === publishVersion.current && sender.current === approvedSender) {
     await setSenderLimits(approvedSender, encoderRate.current, captureQuality.current.fps, true);
    }
    return;
   }
   if (fault === 'send') {
    const approvedSender = sender.current;
    if (approvedSender?.track && publishing.current) {
     encoderRate.current = videoBitrate(limitRef.current, 1, 1, encoderRate.current);
     await setSenderLimits(approvedSender, encoderRate.current, captureQuality.current.fps);
    }
    return;
   }
   resetBridge();
   if (!await ensureBridge(true)) reportRecovery(fault, 'ponte indisponível; nova tentativa limitada', 'WARN');
  } catch (e) {
   const name = (e as Error).name || 'erro';
   reportRecovery(fault, 'falha na tentativa (' + name + ')', 'WARN');
  }
 }

 useEffect(() => {
  mounted.current = true;
  void ensureBridge();
  let inFlight = false;
  let disposed = false;
  let disconnectedAt: number | null = null;
  let heartbeatPending = false;

  async function refresh() {
   const hasIntent = publishing.current || !!watching.current;
   if (disposed || inFlight || (isBackground() && !hasIntent)) return;
   inFlight = true;
   try {
    if (requiresConsentAgain()) {
     if (connection.current || hasIntent) {
      closeBridge();
      setError('Sessão de vídeo encerrada pela autorização da sala.');
     }
     return;
    }
    const now = performance.now();
    const bridgeStatus = connection.current?.connectionState;
    if (bridgeStatus === 'disconnected') disconnectedAt ??= now;
    else disconnectedAt = null;
    const disconnected = disconnectedAt !== null && now - disconnectedAt >= 4500;
    if (!connection.current && !hasIntent) await ensureBridge();
    if (!isBackground() && hasIntent && (!connection.current || bridgeStatus === 'failed' || disconnected)) {
     const retry = recovery.current.observe('bridge', now, false);
     if (retry) await recoverVideo(retry.fault, retry.attempt);
     return;
    }

    const publication = publishVersion.current;
    const next = await api().ScreenState();
    if (disposed) return;
    if (initializing.current) return;
    if (!next.active && readyRef.current) {
     if (hasIntent && hasRoomAuthorization() && !isBackground()) {
      const retry = recovery.current.observe('bridge', now, false);
      if (retry) await recoverVideo(retry.fault, retry.attempt);
      return;
     }
     if (!hasIntent) {
      resetBridge();
      return;
     }
     closeBridge();
     setError('Sessão de vídeo encerrada. Confira a autorização da sala antes de voltar a transmitir.');
     return;
    }
    setState(next);
    if (publication === publishVersion.current && !publishPending.current && publishing.current && !next.publishing) {
     stopSharing();
     setError('Compartilhamento encerrado. Confira a autorização da sala antes de voltar a transmitir.');
     return;
    }

    const id = watching.current;
    const status = watchStatus(id, next.active, next.participants, next.peers);
    if (status === 'ended') {
     watchVersion.current++;
     watching.current = '';
     setRemotePreview('');
     setEndedNotice('Transmissão encerrada ou acesso indisponível.');
     void api().ScreenWatch('').catch(() => {});
    } else if (id && next.active && next.watching !== id && !isBackground() && next.peers.some(peer => peer.id === id)) {
     await api().ScreenWatch(id);
    }
    const currentBridge = connection.current;
    if (!currentBridge) return;
    const report = await currentBridge.getStats();
    if (disposed || currentBridge !== connection.current) return;
    const rows: Parameters<VideoTelemetry['sample']>[0] = [];
    report.forEach(row => rows.push(row));
    const observed = telemetry.current.sample(rows, next, now);
    setRates(observed);

    const track = capture.current?.stream?.getVideoTracks()[0];
    const send = sender.current;
    if (publishing.current && !publishPending.current && send && track) {
     const wanted = next.viewers > 0 ? track : null;
     if (send.track !== wanted) await send.replaceTrack(wanted);
     if (disposed || send !== sender.current || !publishing.current) return;
     if (wanted) {
      const dropped = Math.max(0, next.dropped - previousDrops.current);
      const congested = dropped > 0 || (next.sendQueue?.depth ?? 0) >= 2 || (next.sendQueue?.lastWaitMs ?? 0) >= 45;
      const wasTarget = captureQuality.current;
      previousDrops.current = next.dropped;
      const pressure = congested || observed.limitation === 'cpu' || observed.limitation === 'bandwidth' || !!next.audioPriorityPaused;
      const changedQuality = governor.current.observe(pressure);
      if (changedQuality) {
       const target = governor.current.target(selectedQuality.current);
       if (target.width !== wasTarget.width || target.height !== wasTarget.height || target.fps !== wasTarget.fps) {
        try {
         await wanted.applyConstraints(captureLimits(target));
         if (send.track !== wanted || !publishing.current) return;
         captureQuality.current = target;
         diagnostic('INFO', 'video', 'frontend/qualidade', `limites=${target.width}x${target.height}@${target.fps} pressão=${pressure}`);
        } catch {
         diagnostic('WARN', 'video', 'frontend/qualidade', 'Limites de captura recusados pelo runtime; perfil anterior conservado.');
        }
       }
      }
      encoderRate.current = videoBitrate(limitRef.current, next.viewers, pressure ? 1 : 0, encoderRate.current);
      await setSenderLimits(send, encoderRate.current, captureQuality.current.fps);
     }
    }

    const visible = !isBackground();
    const viewing = visible && !!watching.current && next.playing === watching.current;
    const fault = visible ? detectVideoFault({
     publishing: publishing.current && next.publishing,
     viewing,
     watchers: next.viewers,
     sourceMuted: !!track?.muted && track.readyState === 'live',
     capture: observed.capture,
     encoded: observed.encode,
     sent: observed.sent,
     receiving: observed.received,
     localReceived: observed.localReceived,
     decoded: observed.decode,
     rendered: renderRef.current?.fps ?? null,
     sendQueueDepth: next.sendQueue?.depth ?? null,
     videoPausedForVoice: !!next.audioPriorityPaused,
    }) : null;
    const progress = (!publishing.current || (observed.sent ?? 0) > 0)
      && (!watching.current || (observed.decode ?? 0) > 0)
      && (!!watching.current || publishing.current);
    if (!fault && progress && recovering.current) {
     reportRecovery(recovering.current, 'fluxo de vídeo voltou a avançar');
     recovering.current = null;
    }
    const retry = recovery.current.observe(fault, now, progress);
    if (retry) await recoverVideo(retry.fault, retry.attempt);
   } catch (e) {
    if (!disposed) setError('Diagnóstico de vídeo: ' + String(e));
   } finally { inFlight = false; }
  }
  const timer = setInterval(() => {
   if (!heartbeatPending && (publishing.current || watching.current || readyRef.current) &&
       (!isBackground() || (backgroundAllowed.current && (publishing.current || !!watching.current)))) {
    heartbeatPending = true;
    void api().ScreenHeartbeat().catch(() => {}).finally(() => { heartbeatPending = false; });
   }
   void refresh();
  }, 1000);
  const visibility = () => {
   const action = screenVisibilityAction(isBackground(), backgroundAllowed.current, publishing.current, !!watching.current);
   if (action === 'stop') closeBridge();
   else if (action === 'restore') void refresh();
  };
  const unload = () => { capture.current?.stop(); connection.current?.close(); };
  document.addEventListener('visibilitychange', visibility);
  window.addEventListener('peetwopee:visibility', visibility);
  window.addEventListener('peetwopee:screen-stop', closeBridge);
  window.addEventListener('beforeunload', unload);
  return () => {
   disposed = true;
   mounted.current = false;
   clearInterval(timer);
   document.removeEventListener('visibilitychange', visibility);
   window.removeEventListener('peetwopee:visibility', visibility);
   window.removeEventListener('peetwopee:screen-stop', closeBridge);
   window.removeEventListener('beforeunload', unload);
   turn.current++;
   publishVersion.current++;
   capture.current?.stop();
   connection.current?.close();
   connection.current = null;
   liveTelemetry.video = null;
   void api().ScreenStop().catch(() => {});
  };
 }, [session]);

 useEffect(() => {
  if (!state?.active || (!state.publishing && !state.watching)) {
   liveTelemetry.video = null;
   return;
  }
  const queue = state.publishing ? state.sendQueue : undefined;
  liveTelemetry.video = {
   at: Date.now(),
   capture: rates?.capture ?? null, encode: rates?.encode ?? null, sent: rates?.sent ?? null,
   received: rates?.received ?? null, decode: rates?.decode ?? null, render: state.watching ? render?.fps ?? null : null,
   uploadMbps: rates?.sent != null ? rates.uploadMbps : null,
   downloadMbps: rates?.received != null ? rates.downloadMbps : null,
   dropped: state.dropped, configured: state.bitrate,
   queue: queue?.depth ?? null, queueCapacity: queue?.capacity ?? null, queueWaitMs: queue?.lastWaitMs ?? null,
   encodeMs: rates?.encodeMs ?? null, decodeMs: rates?.decodeMs ?? null,
   codec: rates?.codec ?? null, encoder: rates?.encoder ?? null, decoder: rates?.decoder ?? null,
   powerEfficientEncoder: rates?.efficientEncoder ?? null, powerEfficientDecoder: rates?.efficientDecoder ?? null,
   captureWidth: rates?.captureWidth ?? null, captureHeight: rates?.captureHeight ?? null,
   encodeWidth: rates?.encodeWidth ?? null, encodeHeight: rates?.encodeHeight ?? null,
   renderWidth: state.watching ? render?.width ?? null : null, renderHeight: state.watching ? render?.height ?? null : null,
   recovery: lastRecovery?.message ?? null, recoveryAt: lastRecovery?.at ?? null,
  };
 }, [state, rates, render, lastRecovery]);

 useEffect(() => {
  if (!rates || Date.now() - lastEncoderLog.current < 3000 || (!state?.publishing && !state?.watching)) return;
  lastEncoderLog.current = Date.now();
  diagnostic('INFO', 'video', 'frontend/encoder', JSON.stringify({...rates, renderFPS: render?.fps ?? null}));
 }, [rates, render, state?.publishing, state?.watching]);

 function select() {
  stopSharing();
  const approval = publishVersion.current;
  const request = profileRef.current;
  const offered = selectCaptureProfile(request, capabilities);
  selectedQuality.current = offered;
  setLastRecovery(null);
  recovering.current = null;
  recovery.current = new VideoRecoveryBudget();
  setError('');
  setCaptureNote(offered !== request && request !== 'auto'
   ? 'O WebView não confirmou VP8 fluido no perfil escolhido. A captura usará um perfil compatível.'
   : '');
  setPick(true);
  setExpanded(true);
  setBusy(true);
  void capture.current!.select(offered).then(stream => {
   if (!stream || !mounted.current || approval !== publishVersion.current) return;
   const settings = stream.getVideoTracks()[0].getSettings?.();
   const effective = capture.current?.appliedProfile ?? offered;
   selectedQuality.current = effective;
   if (effective !== offered) setCaptureNote('A fonte aceitou o perfil de captura 720p/30.');
   const preferred = screenProfiles[effective];
   captureQuality.current = {
    width: Math.min(preferred.width, settings?.width || preferred.width),
    height: Math.min(preferred.height, settings?.height || preferred.height),
    fps: Math.min(preferred.fps, settings?.frameRate || preferred.fps),
   };
   setPreview(stream);
  }).catch(e => {
   if (mounted.current && approval === publishVersion.current) {
    setError(screenFailure(e));
    setPick(false);
   }
  }).finally(() => { if (mounted.current) setBusy(false); });
 }

 async function start() {
  const stream = capture.current?.stream;
  const send = sender.current;
  if (!stream || !send || !readyRef.current || isBackground() || publishPending.current || requiresConsentAgain()) return;
  setBusy(true);
  const approval = ++publishVersion.current;
  publishPending.current = true;
  const valid = () => approval === publishVersion.current && capture.current?.stream === stream
    && sender.current === send && mounted.current && !isBackground() && !requiresConsentAgain();
  try {
   await stopPromise.current;
   if (!valid()) return;
   const track = stream.getVideoTracks()[0];
   await send.replaceTrack(track);
   if (!valid()) return;
   encoderRate.current = limitRef.current;
   previousDrops.current = 0;
   await setSenderLimits(send, encoderRate.current, captureQuality.current.fps, true);
   if (!valid()) return;
   await api().ScreenHeartbeat();
   if (!valid()) return;
   const result = api().ScreenPublish(true, limitRef.current, viewersRef.current);
   stopPromise.current = result;
   await result;
   if (!valid()) return;
   publishing.current = true;
   setPick(false);
   setState(value => value ? {...value, publishing: true} : value);
   // Audio permission derives from the explicit start and the existing source.
   void sound.current?.automatic(track);
  } catch (e) {
   if (approval === publishVersion.current) { stopSharing(); setError(screenFailure(e)); }
  } finally {
   if (approval === publishVersion.current) publishPending.current = false;
   if (mounted.current) setBusy(false);
  }
 }

 async function watch(id: string) {
  setEndedNotice('');
  setError('');
  const epoch = ++watchVersion.current;
  watching.current = id;
  renderRef.current = null;
  setRender(null);
  if (id) {
   recovery.current = new VideoRecoveryBudget();
   setLastRecovery(null);
   recovering.current = null;
   void sound.current?.prepare().catch(e => diagnostic('WARN', 'video', 'áudio da tela', String(e)));
  }
  setState(value => value ? {...value, watching: id} : value);
  try { await api().ScreenWatch(id); } catch (e) {
   if (epoch === watchVersion.current && mounted.current) setError(String(e));
  }
 }

 useImperativeHandle(controlRef, () => ({
  toggle: () => {
   if (publishing.current || state?.publishing) stopSharing();
   else if (ready && !busy) select();
  },
  preview: (id: string) => {
   if (!ready) return;
   setRemotePreview(id);
   setExpanded(true);
   void watch(id);
  },
 }));

 const peerSignature = JSON.stringify(state?.peers?.map(peer => [peer.id, peer.name, peer.sharing, peer.connected]));
 useEffect(() => {
  onControlState({ready, sharing: publishing.current || !!state?.publishing, busy, peers: state?.peers || []});
 }, [ready, state?.publishing, busy, peerSignature, onControlState]);

 const selected = state?.peers.find(peer => peer.id === watching.current);
 const remoteVisible = !!watching.current;
 const renderFPS = render?.fps ?? null;
 const details = [
  rates?.encodeWidth && rates.encodeHeight ? `${rates.encodeWidth}×${rates.encodeHeight}` : 'Resolução do encoder n/d',
  rates?.encode != null ? `${rates.encode.toFixed(1)} FPS codificados` : 'FPS codificados n/d',
  rates?.codec || 'Codec em observação',
  rates?.encoder || 'encoder não informado',
 ].join(' · ');
 useEffect(() => {
  if (remoteVisible) revealControls();
  return () => clearTimeout(controlsTimer.current);
 }, [remoteVisible]);
 return <section className={'screen-panel '+(compact?'compact-stream':'wide-stream')} aria-label="Compartilhamento de tela"><header><Monitor size={17}/><b>Transmissões</b><button title={expanded?'Recolher telas':'Ver telas dos participantes'} className="secondary" onClick={()=>setExpanded(!expanded)}>{expanded?'Recolher':'Telas dos participantes'}</button>{(publishing.current||state?.publishing)?<><span className="screen-live" role="status"><span className="live-badge">AO VIVO</span> · {state?.viewers||0} espectador(es)</span><button title="Parar compartilhamento de tela" className="danger secondary" onClick={()=>stopSharing()}><Square size={14}/> Parar compartilhamento</button></>:<button title="Escolher monitor ou janela para compartilhar" className="primary" disabled={busy||!ready} onClick={select}>Compartilhar tela</button>}</header><details className="screen-consent-details"><summary>Permissões e detalhes técnicos</summary><p className="screen-consent">Confira a prévia e confirme Iniciar compartilhamento para transmitir a fonte e o áudio compatível. Janela: somente o aplicativo identificado; monitor: toda a saída padrão, inclusive notificações. Sem gravação. O áudio é interrompido se não for possível evitar eco.</p></details><SystemAudioPanel output={output} session={session} ref={sound} api={api} publishing={!!state?.publishing} watching={!!watching.current} watchedId={watching.current}/><div hidden={!expanded&&!watching.current}><SettingsSurface section="transmission"><TransmissionFields profile={profile} limit={limit} viewers={maxViewers} profileLocked={busy||!!state?.publishing||!!preview} limitsLocked={!!state?.publishing} onProfile={q=>{setProfile(q);setLimit(q==='1080p60'?10000000:q==='1080p30'||q==='720p60'?6000000:3000000)}} onLimit={setLimit} onViewers={setMaxViewers}/></SettingsSurface>{state?.peers.filter(p=>p.sharing&&p.connected).map(p=><button key={p.id} className="secondary" onClick={()=>{setRemotePreview(p.id);setExpanded(true);void watch(p.id)}}><span className="live-badge">AO VIVO</span> {p.name||p.id.slice(0,10)} · {state.watching===p.id?'Assistindo':'Assistir'}</button>)}{state?.publishing&&preview&&!remoteVisible&&<div className="screen-stage local-screen-stage"><Video stream={preview} preview/><span className="screen-local-caption"><span className="live-badge">AO VIVO</span> Sua transmissão · {state.viewers} espectador(es)</span></div>}{remoteVisible&&<div className={'screen-stage '+(controlsVisible?'show-controls':'')} ref={stage} onPointerMove={revealControls} onFocusCapture={revealControls} onKeyDown={revealControls}><Video stream={remote} retry={playRetry} onRendered={setRender}/><ScreenViewerOverlay name={selected?.name||'Transmissão selecionada'} connected={!!selected?.connected&&!!selected?.accepted&&state?.playing===watching.current} width={render?.width??rates?.width} height={render?.height??rates?.height} fps={selected?.connected&&selected?.accepted&&state?.playing===watching.current?renderFPS:null} mbps={rates?.downloadMbps}><button className="secondary" title="Alternar tamanho da transmissão" onClick={()=>setCompact(!compact)}>{compact?'Ampliar no aplicativo':'Modo compacto'}</button><button title="Assistir em tela cheia" className="secondary" onClick={()=>void (document.fullscreenElement?document.exitFullscreen():stage.current?.requestFullscreen())?.catch(e=>setError(String(e)))}><Maximize2 size={14}/> Tela cheia</button><button title="Parar de assistir a esta tela" className="secondary" onClick={()=>{setRemotePreview('');void watch('')}}>Parar de assistir</button></ScreenViewerOverlay></div>}{state?.publishing&&<details className="screen-technical"><summary>Qualidade da transmissão</summary><small>{details} · saída {state.bytesOut} bytes · {state.framesOut} frames · {state.dropped} descartados</small></details>}</div>{rates&&<details className="screen-technical"><summary>Métricas de vídeo</summary><small role="status">FPS reais · captura {rates.capture?.toFixed(1)??'n/d'} · encoder {rates.encode?.toFixed(1)??'n/d'} · envio {rates.sent?.toFixed(1)??'n/d'} · recepção {rates.received?.toFixed(1)??'n/d'} · decoder {rates.decode?.toFixed(1)??'n/d'} · render {renderFPS?.toFixed(1)??'n/d'}<br/>Rede: ↑ {rates.uploadMbps.toFixed(2)} / ↓ {rates.downloadMbps.toFixed(2)} Mbps · RTT ACK {state?.roundTripMs.toFixed(0)??'n/d'} ms · descartes {state?.dropped??0} · {Object.entries(state?.dropReasons||{}).map(([k,v])=>`${k}:${v}`).join(" ")} · perda RTP ponte local {rates.localBridgePacketLoss??'n/d'}; perda do túnel TCP não mensurável<br/>Codec {rates.codec??'n/d'} · encoder {rates.encoder??'n/d'} · decoder {rates.decoder??'n/d'}
 · eficiência reportada pelo WebView (encode/decode) {rates.efficientEncoder==null?'n/d':rates.efficientEncoder?'sim':'não'}/{rates.efficientDecoder==null?'n/d':rates.efficientDecoder?'sim':'não'}</small></details>}{lastRecovery&&<p role="status">{new Date(lastRecovery.at).toLocaleTimeString('pt-BR')} · {lastRecovery.message}</p>}{endedNotice&&<p className="stream-ended" role="status">{endedNotice}</p>}{state?.error&&!error&&<small>{state.error}</small>}{error&&<p className="error" role="alert">{error}</p>}{remotePreview&&<div className="modal-overlay"><section className="modal screen-preview" role="dialog" aria-label="Prévia da transmissão"><button className="close icon" title="Fechar prévia e parar de assistir" onClick={()=>{setRemotePreview('');void watch('')}}><X/></button><h2>{state?.peers.find(p=>p.id===remotePreview)?.name||'Transmissor'}</h2><span className="live-badge">AO VIVO</span>{state?.peers.some(p=>p.id===remotePreview&&p.sharing&&p.connected)?<Video stream={state?.playing===remotePreview?remote:null} preview/>:<p>Transmissão encerrada ou permissão indisponível.</p>}<button className="primary" onClick={()=>{setRemotePreview('');setExpanded(true)}}>Assistir</button></section></div>}{pick&&<div className="modal-overlay"><section className="modal screen-preview" role="dialog" aria-label="Prévia antes de transmitir"><button className="close icon" aria-label="Cancelar compartilhamento" onClick={()=>stopSharing()}><X/></button><h2>Confira a fonte selecionada</h2><p>Escolha o monitor inteiro ou a janela exata no seletor do Windows. Ao confirmar o compartilhamento, a fonte será transmitida com áudio compatível, sem gravação nem troca automática de fonte.</p><Video stream={preview} preview/>{captureNote&&<p role="status">{captureNote}</p>}<small>{preview?.getVideoTracks()[0]?.label||'Aguardando a sua escolha no seletor do sistema…'}</small><button className="primary" disabled={busy||!preview||!ready} onClick={()=>void start()}>Iniciar compartilhamento</button><button className="secondary" onClick={()=>stopSharing()}>Cancelar</button><p>Ao minimizar, a transmissão continua com indicador e opção Parar na bandeja. Captura de janela minimizada depende do Windows; use um monitor para maior compatibilidade.</p>{error&&<p role="alert">{error}</p>}</section></div>}</section>
});
