"""Report SDK loader references transparently; does NOT relax the application's strict PE audit."""
from pathlib import Path
import json,struct,hashlib
p=Path(__file__).resolve().parents[2]/'build/bin/PeeTwoPee-Tailscale-Lab.exe';b=p.read_bytes();u16=lambda o:struct.unpack_from('<H',b,o)[0];u32=lambda o:struct.unpack_from('<I',b,o)[0]
pe=u32(60);assert b[pe:pe+4]==b'PE\0\0';coff=pe+4;opt=coff+20;magic=u16(opt);dd=opt+(112 if magic==523 else 96);sections=opt+u16(coff+16)
def offset(rva):
 for i in range(u16(coff+2)):
  s=sections+40*i;va=u32(s+12);size=max(u32(s+8),u32(s+16))
  if va<=rva<va+size:return u32(s+20)+rva-va
 if rva<u32(opt+60):return rva
 raise ValueError('unmapped RVA')
imports=[];delays=[]
for index,step,target,namefield in [(1,20,imports,12),(13,32,delays,4)]:
 rva=u32(dd+index*8)
 if not rva:continue
 pos=offset(rva)
 while any(b[pos:pos+step]):
  name=u32(pos+namefield)
  if index==13:assert u32(pos)&1
  at=offset(name);target.append(b[at:b.index(b'\0',at)].decode('ascii'));pos+=step
references=[name for name in ['packet.dll','npcap','wintun.dll','windivert'] if name.encode() in b.lower() or name.encode('utf-16le') in b.lower()]
assert not any(name in references for name in ['packet.dll','npcap','windivert'])
assert not any('wintun' in name.lower() or 'packet' in name.lower() for name in imports+delays)
print(json.dumps({'file':p.name,'sha256':hashlib.sha256(b).hexdigest(),'imports':imports,'delayImports':delays,'loaderReferences':references,'strictAppAudit':'reference to wintun.dll remains; no main auditor rules changed','runtimeEvidence':'SDK started with tun=false, wintunLoaded=false and NeedsLogin; authenticated tailnet traffic pending'},indent=2))
