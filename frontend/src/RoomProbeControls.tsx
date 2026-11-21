import React, {useEffect, useRef, useState} from 'react';
import type {PeerSnapshot, SessionSummary} from './mediaDiagnosticModel';

export type RoomProbeResult = {
 durationSeconds: number;
 totalBlocks: number;
 payloadBytes: number;
 rttMeanMs: number;
 rttP95Ms: number;
 variationMs: number | null;
 payloadKbps: number;
};

export type RoomProbeAPI = {
 ApproveRoomProbe?: (alias: string) => Promise<void>;
 RunRoomProbe?: (alias: string, seconds: number) => Promise<RoomProbeResult>;
 CancelRoomProbe?: () => Promise<void>;
};

type Props = {
 api: () => RoomProbeAPI;
 peers: PeerSnapshot[];
 ready: boolean;
 mode: string;
 onResult: (summary: SessionSummary) => void;
};

export function RoomProbeControls({api, peers, ready, mode, onResult}: Props) {
 const [alias, setAlias] = useState('');
 const [duration, setDuration] = useState(10);
 const [role, setRole] = useState<'idle' | 'starting' | 'receiving' | 'sending'>('idle');
 const [message, setMessage] = useState('');
 const [error, setError] = useState('');
 const operation = useRef(0);
 const active = useRef(false);
 const expiry = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
 const target = peers.find(peer => peer.alias === alias && peer.connections > 0);
 const available = !!(api().ApproveRoomProbe && api().RunRoomProbe && api().CancelRoomProbe);

 function finish() {
  active.current = false;
  clearTimeout(expiry.current);
  setRole('idle');
 }

 async function stop() {
  const revision = ++operation.current;
  clearTimeout(expiry.current);
  setRole('starting');
  setError('');
  try {
   await api().CancelRoomProbe?.();
   if (revision === operation.current) setMessage('Teste ou permissão cancelados.');
  } catch {
   if (revision === operation.current) setError('Falha ao cancelar. A permissão expira em até 90 segundos.');
  } finally {
   if (revision === operation.current) finish();
  }
 }

 useEffect(() => () => {
  operation.current++;
  clearTimeout(expiry.current);
  if (active.current) void api().CancelRoomProbe?.().catch(() => {});
 }, []);

 useEffect(() => {
  if ((!ready || !target) && active.current) void stop();
 }, [ready, target?.alias]);

 async function start(receive: boolean) {
  if (!ready || !target || !available || role !== 'idle') return;
  const description = receive
   ? 'Permitir um teste deste participante nos próximos 90 segundos?'
   : `Iniciar um teste de ${duration} segundos com este participante?`;
  if (!window.confirm(description + ' O teste usa até 128 kbit/s por direção e pode disputar banda com voz ou vídeo.')) return;

  const revision = ++operation.current;
  active.current = true;
  setRole(receive ? 'starting' : 'sending');
  setMessage(receive ? '' : 'Aguardando respostas do participante…');
  setError('');
  try {
   if (receive) {
    await api().ApproveRoomProbe!(alias);
    if (revision !== operation.current) {
     await api().CancelRoomProbe?.();
     return;
    }
    setRole('receiving');
    setMessage('Permissão de uso único ativa por até 90 segundos.');
    expiry.current = setTimeout(() => {
     finish();
     setMessage('Permissão encerrada.');
    }, 90_000);
    return;
   }
   const result = await api().RunRoomProbe!(alias, duration);
   if (revision !== operation.current) return;
   onResult({
    at: Date.now(), mode, seconds: result.durationSeconds, samples: result.totalBlocks,
    rttMean: result.rttMeanMs, rttP95: result.rttP95Ms, jitter: null,
    kbps: result.payloadKbps, fps: null, interruptions: null, noRxSeconds: null,
    testType: 'limited-load', variationMs: result.variationMs,
    conditions: 'Eco de 4 kB a cada 250 ms pela conexão da sala. Vazão de payload por direção e RTT do eco sob carga limitada.'
   });
   setMessage('Teste concluído. Resumo disponível para comparação.');
  } catch (err) {
   if (revision === operation.current) {
    setMessage('');
    setError(String(err));
   }
  } finally {
   if (revision === operation.current && (!receive || !expiry.current)) finish();
  }
 }

 if (!available) return null;
 return <div className="diagnostic-load">
  <h3>Teste com carga limitada</h3>
  <p>Escolham um ao outro. Peça ao participante para permitir o teste antes de iniciar. O limite é de 128 kbit/s por direção.</p>
  <label>Participante
   <select value={alias} disabled={role !== 'idle'} onChange={event => setAlias(event.target.value)}>
    <option value="">Escolha um participante</option>
    {peers.filter(peer => peer.connections > 0).map(peer =>
     <option key={peer.alias} value={peer.alias}>Peer {peer.alias}</option>)}
   </select>
  </label>
  <label>Duração do teste
   <select value={duration} disabled={role !== 'idle'} onChange={event => setDuration(Number(event.target.value))}>
    {[10, 30, 60].map(value => <option key={value} value={value}>{value} s</option>)}
   </select>
  </label>
  <div className="diagnostic-actions">
   <button disabled={!ready || !target || role !== 'idle'} onClick={() => void start(true)}>Permitir teste deste participante</button>
   <button disabled={!ready || !target || role !== 'idle'} onClick={() => void start(false)}>Iniciar teste com carga limitada</button>
   {role !== 'idle' && <button onClick={() => void stop()}>Cancelar teste ou permissão</button>}
  </div>
  {message && <p role="status">{message}</p>}
  {error && <p role="alert">{error}</p>}
 </div>;
}
