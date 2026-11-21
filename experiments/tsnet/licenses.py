"""Collect notices for packages actually linked by the isolated tsnet prototype."""
from pathlib import Path
import subprocess,json,zipfile,hashlib
base=Path(__file__).resolve().parent
raw=subprocess.check_output(['go','list','-tags','ts_omit_logtail','-deps','-json','.'],cwd=base).decode('utf-8');decoder=json.JSONDecoder();modules={}
while raw.strip():
 raw=raw.lstrip();p,n=decoder.raw_decode(raw);raw=raw[n:];m=p.get('Module')
 if m and m.get('Dir'):modules[m['Path']]=m
out=base/'../../build/bin/Tailscale-Lab-licencas.zip';inventory=[];missing=[]
with zipfile.ZipFile(out,'w',zipfile.ZIP_DEFLATED) as z:
 for name,m in sorted(modules.items()):
  folder=Path(m['Dir']);files=[p for p in folder.iterdir() if p.is_file() and p.name.upper().startswith(('LICENSE','LICENCE','COPYING','NOTICE','PATENTS','COPYRIGHT')) and p.stat().st_size<2<<20]
  if not files:missing.append(name)
  for p in files:z.writestr(name+'/'+p.name,p.read_bytes())
  inventory.append({'module':name,'version':m.get('Version','local source'),'notices':[p.name for p in files]})
 goroot=Path(subprocess.check_output(['go','env','GOROOT'],cwd=base).decode().strip())
 z.writestr('Go/LICENSE',(goroot/'LICENSE').read_bytes())
 z.writestr('INVENTARIO.json',json.dumps({'modules':inventory,'missingNotices':missing},indent=2))
print(json.dumps({'modules':len(inventory),'missingNotices':missing,'sha256':hashlib.sha256(out.read_bytes()).hexdigest()}))
if missing:raise SystemExit('Notices missing; do not distribute prototype binary yet')
