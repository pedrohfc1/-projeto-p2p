"""Inspect PE regular and delay imports plus driver-loader string references."""
import argparse,hashlib,json,struct,pathlib
parser=argparse.ArgumentParser();parser.add_argument("files",nargs="+");args=parser.parse_args()
def inspect(path):
 b=path.read_bytes();u16=lambda o:struct.unpack_from("<H",b,o)[0];u32=lambda o:struct.unpack_from("<I",b,o)[0]
 pe=u32(60);assert b[pe:pe+4]==b"PE\0\0";coff=pe+4;opt=coff+20;magic=u16(opt);assert magic in(267,523)
 dd=opt+(112 if magic==523 else 96);sections=opt+u16(coff+16)
 def offset(rva):
  for i in range(u16(coff+2)):
   s=sections+40*i;va=u32(s+12);size=max(u32(s+8),u32(s+16))
   if va<=rva<va+size:return u32(s+20)+rva-va
  if rva<u32(opt+60):return rva
  raise ValueError("unmapped RVA")
 def string(rva):
  start=offset(rva);return b[start:b.index(b"\0",start)].decode("ascii")
 imports=[];delays=[]
 for index,step,target,namefield in[(1,20,imports,12),(13,32,delays,4)]:
  rva=u32(dd+index*8)
  if not rva:continue
  pos=offset(rva)
  while any(b[pos:pos+step]):
   name=u32(pos+namefield)
   if index==13 and not(u32(pos)&1):raise ValueError("unsupported legacy VA delay imports")
   target.append(string(name));pos+=step
 blocked=[x for x in ("packet.dll","npcap","wintun.dll","windivert") if x.encode() in b.lower() or x.encode("utf-16le") in b.lower()]
 assert not blocked,(path.name,blocked)
 return {"file":path.name,"sha256":hashlib.sha256(b).hexdigest(),"imports":imports,"delayImports":delays,"forbiddenReferences":blocked}
print(json.dumps([inspect(pathlib.Path(x)) for x in args.files],indent=2))
