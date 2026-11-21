"""Generate a non-personal mono Opus/20ms test fixture; requires ffmpeg in PATH."""
from pathlib import Path
import argparse,struct,subprocess,tempfile
parser=argparse.ArgumentParser();parser.add_argument('output',type=Path);args=parser.parse_args()
output=args.output.resolve();output.parent.mkdir(parents=True,exist_ok=True)
with tempfile.TemporaryDirectory(prefix='audio-codec-',dir=output.parent) as folder:
 ogg=Path(folder)/'tone.ogg'
 subprocess.run(['ffmpeg','-hide_banner','-loglevel','error','-f','lavfi','-i','sine=frequency=440:sample_rate=48000','-t','1','-ac','1','-c:a','libopus','-frame_duration','20','-b:a','32k','-page_duration','20000','-y',str(ogg)],check=True)
 raw=ogg.read_bytes();pos=0;pending=b'';frames=[]
 while pos<len(raw):
  assert raw[pos:pos+4]==b'OggS','invalid Ogg page'
  count=raw[pos+26];sizes=raw[pos+27:pos+27+count];offset=pos+27+count
  for size in sizes:
   pending+=raw[offset:offset+size];offset+=size
   if size<255:
    if not pending.startswith((b'OpusHead',b'OpusTags')):frames.append(pending)
    pending=b''
  pos=offset
 assert frames and not pending
 output.write_bytes(b''.join(struct.pack('>H',len(frame))+frame for frame in frames))
 print('PASS:',len(frames),'generated Opus frames; no microphone captured')
