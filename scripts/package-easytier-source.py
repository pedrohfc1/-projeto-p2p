"""Package full modified source + vendored dependencies for LGPL relinking.
Input work tree contains the exact audited sources used for the helper build.
Never include runtime credentials, executables, drivers or build outputs.
"""
from pathlib import Path
import argparse,zipfile,hashlib,json
ap=argparse.ArgumentParser();ap.add_argument("work",type=Path);ap.add_argument("output",type=Path);a=ap.parse_args()
w=a.work.resolve();root=Path(__file__).resolve().parent.parent
sources=[("etcat-source",w/"etcat-source"),("easytier-minimal",w/"easytier-minimal"),("vendor",w/"etcat-vendor"),("protodesc",w/"protodesc")]
assert all(p.is_dir() for _,p in sources)
a.output.parent.mkdir(parents=True,exist_ok=True)
count=0
with zipfile.ZipFile(a.output,"w",zipfile.ZIP_DEFLATED,compresslevel=6,strict_timestamps=False) as z:
 for label,folder in sources:
  for f in sorted(folder.rglob("*")):
   rel=f.relative_to(folder)
   if not f.is_file() or f.is_symlink() or any(x in {".git","target","node_modules",".github"} for x in rel.parts):continue
   if f.suffix.lower() in {".exe",".dll",".sys",".pdb",".log"} or any(x in f.name.lower() for x in ["packet.dll","npcap","wintun.dll","windivert.dll"]):continue
   z.write(f,"work/"+label+"/"+rel.as_posix());count+=1
 for f in (root/"third_party/etcat").iterdir():
  if f.is_file():z.write(f,f.name)
 z.writestr("README-SOURCE.txt","""Projeto P2P EasyTier modified helper: full corresponding source.
Etcat ccb45cd84c20b2aa1e6615a3f62a83827e961695 Apache-2.0.
EasyTier 164e2db6aecd13117d840821d5b889b51cb7c463 LGPL-3.0.
Changes: no TUN/capture/UPnP; token via environment; path monitoring;
independent client instance/address allocation; compatible dependency updates.
This archive accompanies every private installer and is installed alongside it.
You may modify/rebuild/replace the LGPL components and relink this helper.
Runtime verifies the helper digest: rebuild Projeto P2P with the new manifest
hash after modifying/relinking your helper. No signing key is needed.
Build: see BUILD.md. Compiler/build tools are not application runtime dependencies.
Driver/capture binaries and unrelated executable fixtures are deliberately excluded.
""")
print(json.dumps({"files":count,"bytes":a.output.stat().st_size,"sha256":hashlib.sha256(a.output.read_bytes()).hexdigest()}))
