import {test} from 'node:test';
import assert from 'node:assert/strict';
import {inspectVideoCapabilities, selectCaptureProfile, VideoQualityGovernor} from '../src/videoCapabilities.ts';

test('known smooth VP8 gates higher profiles and H.264 cannot change the wire codec', async () => {
 const capabilities = await inspectVideoCapabilities({
  encodingInfo: async ({type, video}) => {
   assert.equal(type, 'webrtc');
   if(video.contentType==='video/VP8'&&video.framerate===60) {
    return {supported: true, smooth: false, powerEfficient: false};
   }
   return {supported: true,smooth:true,powerEfficient:video.contentType.includes('H264')};
  },
 });
 assert.equal(capabilities['h264-1080p60'].powerEfficient,true);
 assert.equal(selectCaptureProfile('1080p60',capabilities),'1080p30');
 assert.equal(selectCaptureProfile('1080p60',{}),'720p30');
 assert.equal(selectCaptureProfile('1080p60',{'vp8-1080p60':{supported:true,smooth:true,powerEfficient:false}}),'1080p60');
});

test('capture ladder drops after sustained pressure and recovers gradually', () => {
 const g = new VideoQualityGovernor();
 assert.equal(g.target('1080p60').fps,60);
 for(let i=0;i<3;i++)g.observe(true);
 assert.deepEqual(g.target('1080p60'),{width:1280,height:720,fps:30});
 for(let i=0;i<3;i++)g.observe(true);
 assert.deepEqual(g.target('1080p60'),{width:960,height:540,fps:24});
 for(let i=0;i<11;i++)g.observe(false);
 assert.equal(g.target('1080p60').fps,24);
 for(let i=0;i<13;i++)g.observe(false);
 assert.equal(g.target('1080p60').fps,60);
});
