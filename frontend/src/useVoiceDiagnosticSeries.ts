import {useEffect, useRef, useState} from 'react';
import {isBackground} from './background';
import {liveTelemetry, SessionSeries, type Snapshot} from './mediaDiagnosticModel';

export type VoiceDiagnosticAPI = {
 VoiceDiagnosticSample?: (probe: boolean) => Promise<Snapshot>;
 VoiceDiagnosticSnapshot?: () => Promise<Snapshot>;
};

export function useVoiceDiagnosticSeries(api: () => VoiceDiagnosticAPI, session: string, visible: boolean) {
 const [series] = useState(() => new SessionSeries());
 const options = useRef({api, visible});
 options.current = {api, visible};
 const wake = useRef<() => void>(() => {});

 useEffect(() => {
  if (!session) return;
  let disposed = false;
  let inFlight = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const active = () => options.current.visible && !isBackground();

  function schedule() {
   clearTimeout(timer);
   if (!disposed) timer = setTimeout(() => void collect(), active() ? 1000 : 5000);
  }

  async function collect() {
   if (disposed || inFlight) return;
   inFlight = true;
   const backend = options.current.api();
   const probe = active();
   try {
    const snapshot = backend.VoiceDiagnosticSample
     ? await backend.VoiceDiagnosticSample(probe)
     : probe ? await backend.VoiceDiagnosticSnapshot?.() : undefined;
    if (disposed) return;
    if (snapshot?.joined && snapshot.session) {
     series.append(snapshot, liveTelemetry.audio, liveTelemetry.video);
     series.setError('');
    } else if (probe && !backend.VoiceDiagnosticSample && !backend.VoiceDiagnosticSnapshot) {
     series.setError('O diagnóstico requer um backend compatível.');
    }
   } catch {
    if (!disposed && probe) series.setError('A última coleta não respondeu. O histórico foi preservado.');
   } finally {
    inFlight = false;
    schedule();
   }
  }

  function requestSample() {
   clearTimeout(timer);
   if (!inFlight) void collect();
  }

  wake.current = requestSample;
  window.addEventListener('peetwopee:visibility', requestSample);
  document.addEventListener('visibilitychange', requestSample);
  requestSample();
  return () => {
   disposed = true;
   clearTimeout(timer);
   wake.current = () => {};
   window.removeEventListener('peetwopee:visibility', requestSample);
   document.removeEventListener('visibilitychange', requestSample);
   series.finish();
  };
 }, [series, session]);

 useEffect(() => wake.current(), [visible]);
 return series;
}
