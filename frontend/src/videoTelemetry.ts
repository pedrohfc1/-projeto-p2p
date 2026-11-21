type Row = {id: string; type: string; kind?: string; [key: string]: unknown};
type NetworkCounters = {framesOut: number; framesIn: number; bytesOut: number; bytesIn: number};

const finite = (value: unknown): number | null =>
 typeof value === 'number' && Number.isFinite(value) && value >= 0 ? value : null;
const description = (value: unknown): string | null =>
 typeof value === 'string' && value.trim() ? value.slice(0, 80) : null;
const hardwareHint = (value: unknown): boolean | null => typeof value === 'boolean' ? value : null;

export class VideoTelemetry {
 private previous = new Map<string, number>();
 private at: number | null = null;

 reset() {
  this.previous.clear();
  this.at = null;
 }

 sample(rows: Row[], network: NetworkCounters, now: number) {
  const seconds = this.at === null ? 0 : (now - this.at) / 1000;
  this.at = now;
  const counters = new Map<string, number>();
  const rate = (key: string, raw: unknown) => {
   const value = finite(raw);
   if (value === null) return null;
   const before = this.previous.get(key);
   counters.set(key, value);
   return seconds > 0 && before !== undefined && value >= before ? (value - before) / seconds : null;
  };

  const video = rows.filter(row => row.kind === 'video');
  const source = video.find(row => row.type === 'media-source');
  const outgoing = video.find(row => row.type === 'outbound-rtp' && finite(row.framesEncoded) !== null);
  const incoming = video.find(row => row.type === 'inbound-rtp' && finite(row.framesDecoded) !== null)
   ?? video.find(row => row.type === 'inbound-rtp' && finite(row.frameWidth) !== null);
  const codecID = outgoing?.codecId ?? incoming?.codecId;
  const codec = rows.find(row => row.type === 'codec' && row.id === codecID);
  const capture = source ? rate(source.id + ':captured', source.frames) ?? finite(source.framesPerSecond) : null;
  const encode = outgoing ? rate(outgoing.id + ':encoded', outgoing.framesEncoded) : null;
  const decode = incoming ? rate(incoming.id + ':decoded', incoming.framesDecoded) : null;
  const encodeSeconds = outgoing ? rate(outgoing.id + ':encodeSeconds', outgoing.totalEncodeTime) : null;
  const decodeSeconds = incoming ? rate(incoming.id + ':decodeSeconds', incoming.totalDecodeTime) : null;
  const encodeMs = encode !== null && encode > 0 && encodeSeconds !== null ? encodeSeconds / encode * 1000 : null;
  const decodeMs = decode !== null && decode > 0 && decodeSeconds !== null ? decodeSeconds / decode * 1000 : null;
  const sent = rate('sendFrames', network.framesOut);
  const received = rate('receiveFrames', network.framesIn);
  const result = {
   codec: description(codec?.mimeType),
   width: finite(incoming?.frameWidth), height: finite(incoming?.frameHeight),
   captureWidth: finite(source?.width), captureHeight: finite(source?.height),
   encodeWidth: finite(outgoing?.frameWidth), encodeHeight: finite(outgoing?.frameHeight),
   capture, encode, decode, encodeMs, decodeMs, sent, received,
   localReceived: incoming ? rate(incoming.id + ':received', incoming.framesReceived) : null,
   uploadMbps: (rate('upload', network.bytesOut) ?? 0) * 8 / 1e6,
   downloadMbps: (rate('download', network.bytesIn) ?? 0) * 8 / 1e6,
   localBridgePacketLoss: typeof incoming?.packetsLost === 'number' && Number.isFinite(incoming.packetsLost) ? incoming.packetsLost : null,
   encoder: description(outgoing?.encoderImplementation),
   decoder: description(incoming?.decoderImplementation),
   efficientEncoder: hardwareHint(outgoing?.powerEfficientEncoder),
   efficientDecoder: hardwareHint(incoming?.powerEfficientDecoder),
   limitation: description(outgoing?.qualityLimitationReason),
  };
  this.previous = counters;
  return result;
 }
}
