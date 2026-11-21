import {screenProfiles, type ScreenProfile} from './screenCapture.ts';

export type CodecCapability = {
 supported: boolean;
 smooth: boolean;
 powerEfficient: boolean;
};

type CapabilityID = 'vp8-1080p30' | 'vp8-1080p60' | 'vp8-720p60' | 'h264-1080p60';
export type VideoCapabilities = Partial<Record<CapabilityID, CodecCapability | null>>;

const queries: [CapabilityID, string, number, number, number, number][] = [
 ['vp8-1080p30', 'video/VP8', 1920, 1080, 30, 6_000_000],
 ['vp8-1080p60', 'video/VP8', 1920, 1080, 60, 10_000_000],
 ['vp8-720p60', 'video/VP8', 1280, 720, 60, 6_000_000],
 ['h264-1080p60', 'video/H264; profile-level-id=42e02a; packetization-mode=1', 1920, 1080, 60, 10_000_000],
];

export async function inspectVideoCapabilities(media = navigator.mediaCapabilities): Promise<VideoCapabilities> {
 if (!media?.encodingInfo) return {};
 const result = await Promise.all(queries.map(async ([id, contentType, width, height, framerate, bitrate]) => {
  try {
   const configuration = { type: 'webrtc', video: { contentType, width, height, framerate, bitrate } };
   const {supported, smooth, powerEfficient} = await media.encodingInfo(configuration as MediaEncodingConfiguration);
   return [id, {supported, smooth, powerEfficient}] as const;
  } catch {
   return [id, null] as const;
  }
 }));
 return Object.fromEntries(result);
}

function smooth(capabilities: VideoCapabilities, id: CapabilityID) {
 return capabilities[id]?.supported === true && capabilities[id]?.smooth === true;
}

export function selectCaptureProfile(profile: ScreenProfile, capabilities: VideoCapabilities): ScreenProfile {
 if (profile === 'auto' || profile === '720p30') return '720p30';
 if (profile === '1080p60' && !smooth(capabilities, 'vp8-1080p60')) {
  return smooth(capabilities, 'vp8-1080p30') ? '1080p30' : '720p30';
 }
 return smooth(capabilities, `vp8-${profile}`) ? profile : '720p30';
}

export type CaptureQuality = (typeof screenProfiles)[ScreenProfile];

export function captureLimits(quality: CaptureQuality): MediaTrackConstraints {
 return {width: {max: quality.width}, height: {max: quality.height}, frameRate: {max: quality.fps}};
}

const pressureFallback = {width: 960, height: 540, fps: 24};

export class VideoQualityGovernor {
 private level: 0 | 1 | 2 = 0;
 private badSamples = 0;
 private goodSamples = 0;

 reset() {
  this.level = 0;
  this.badSamples = 0;
  this.goodSamples = 0;
 }

 observe(pressure: boolean) {
  if (pressure) {
   this.badSamples++;
   this.goodSamples = 0;
   if (this.badSamples >= 3 && this.level < 2) {
    this.level = this.level === 0 ? 1 : 2;
    this.badSamples = 0;
    return true;
   }
  } else {
   this.goodSamples++;
   this.badSamples = 0;
   if (this.goodSamples >= 12 && this.level > 0) {
    this.level = this.level === 2 ? 1 : 0;
    this.goodSamples = 0;
    return true;
   }
  }
  return false;
 }

 target(profile: ScreenProfile): CaptureQuality {
  const preferred = screenProfiles[profile];
  if (this.level === 2) return pressureFallback;
  if (this.level === 1) return {
   width: Math.min(preferred.width, 1280),
   height: Math.min(preferred.height, 720),
   fps: Math.min(preferred.fps, 30),
  };
  return preferred;
 }
}
