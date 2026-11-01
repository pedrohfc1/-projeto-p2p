"""Explicit DEVELOPMENT provisioning only. App runtime never downloads executable code.
Do not redistribute downloaded helpers until the corresponding-source/dependency notice audit is complete.
"""
import argparse,hashlib,pathlib,tempfile,urllib.request,zipfile,json
VERSION='2.6.4'
ASSET='https://github.com/EasyTier/EasyTier/releases/download/v2.6.4/easytier-windows-x86_64-v2.6.4.zip'
ARCHIVE_SHA='27af91e270e554709b048bd32327fefd2dfce5062ae1e8701af7550c6f525f84'
FILES={'Packet.dll':'c7c03a87eac7243ccbe331554624b18803010b740e311fc8cfddb573096eacac','easytier-core.exe':'da7eb2d24b5416f3d3407636949e964a0750e3f9dc53a828cb6799a57ead445d','easytier-cli.exe':'d8783e851e944b44a9b71b39fd02f227ec0a2a82b3165c55ead5dd32dcde53a1'}
a=argparse.ArgumentParser();a.add_argument('--output',required=True);a.add_argument('--archive',help='optional previously downloaded official archive');args=a.parse_args()
print('LOCAL TEST ONLY: upstream Packet.dll is Npcap, not covered by EasyTier LGPL. Review https://npcap.com/license.html and https://npcap.com/oem/redist before use/redistribution.')
out=pathlib.Path(args.output).resolve();out.mkdir(parents=True,exist_ok=True)
with tempfile.TemporaryDirectory(prefix='verify-easytier-',dir=out) as tmp:
 archive=pathlib.Path(args.archive).resolve() if args.archive else pathlib.Path(tmp)/'official.zip'
 if not args.archive:
  with urllib.request.urlopen(ASSET,timeout=60) as src,archive.open('wb') as dst:
   size=0
   while b:=src.read(1024*1024):
    size+=len(b)
    if size>150*1024*1024:raise ValueError('archive exceeds size limit')
    dst.write(b)
 assert hashlib.sha256(archive.read_bytes()).hexdigest()==ARCHIVE_SHA,'archive SHA mismatch; nothing executed'
 verified={}
 with zipfile.ZipFile(archive) as z:
  for name,want in FILES.items():
   info=z.getinfo('easytier-windows-x86_64/'+name)
   assert info.file_size<100*1024*1024,'helper oversized'
   data=z.read(info);assert hashlib.sha256(data).hexdigest()==want,'helper SHA mismatch'
   verified[name]=data
 for name,data in verified.items():(out/name).write_bytes(data)
(out/'PROVENANCE.json').write_text(json.dumps({'version':VERSION,'upstream':ASSET,'archiveSHA256':ARCHIVE_SHA,'files':FILES,'scope':'local development only; no drivers extracted; redistribution audit pending'},indent=2),encoding='utf-8')
print('PASS: official archive, both helpers and required Packet.dll hashes verified; no code executed, drivers extracted, routes/firewall/DNS changed')
