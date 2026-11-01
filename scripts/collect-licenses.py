"""Collect module/package notices without including source code or profiles."""
import subprocess,json,pathlib,zipfile,platform
p=pathlib.Path(__file__).resolve().parent.parent;v=(p/'VERSION').read_text().strip()
out=p/'dist/release';out.mkdir(parents=True,exist_ok=True)
raw=subprocess.check_output(['go','list','-m','-json','all'],cwd=p).decode();decoder=json.JSONDecoder();mods=[]
while raw.strip():
 raw=raw.lstrip();m,n=decoder.raw_decode(raw);mods.append(m);raw=raw[n:]
def put(z,name,data):
 info=zipfile.ZipInfo(name.replace('\\','/'),date_time=(1980,1,1,0,0,0));info.compress_type=zipfile.ZIP_DEFLATED;info.external_attr=0o100644<<16;z.writestr(info,data)
def notices(z,folder,prefix):
 if not folder.is_dir():return
 for f in sorted(folder.iterdir()):
  if f.is_file() and f.name.upper().startswith(('LICENSE','LICENCE','COPYING','NOTICE','PATENTS','AUTHORS')):put(z,prefix+'/'+f.name,f.read_bytes())
with zipfile.ZipFile(out/f'ProjetoP2P-v{v}-licencas-{platform.system().lower()}.zip','w',zipfile.ZIP_DEFLATED) as z:
 put(z,'ProjetoP2P-LICENSE.txt',(p/'LICENSE').read_bytes());put(z,'THIRD_PARTY_NOTICES.md',(p/'THIRD_PARTY_NOTICES.md').read_bytes())

 for f in (p/'third_party/etcat').iterdir():
  if f.is_file() and f.name!='Cargo.lock':put(z,'integrated-helper/'+f.name,f.read_bytes())
 put(z,'go-modules.json',json.dumps([{k:m[k] for k in ('Path','Version','Sum') if k in m} for m in mods],indent=2))
 for m in sorted(mods,key=lambda x:x['Path']):
  if not m.get('Main') and m.get('Dir'):notices(z,pathlib.Path(m['Dir']),'go/'+m['Path']+'@'+m.get('Version',''))
 goroot=pathlib.Path(subprocess.check_output(['go','env','GOROOT']).decode().strip());notices(z,goroot,'go-runtime')
 packages=[]
 for f in sorted((p/'frontend/node_modules').rglob('package.json')):
  if f.parent.name=='.vite':continue
  try:meta=json.loads(f.read_text(encoding='utf-8'))
  except (ValueError,UnicodeError):continue
  if meta.get('name') and meta.get('version'):
   packages.append({k:meta[k] for k in ('name','version','license') if k in meta});notices(z,f.parent,'npm/'+str(f.parent.relative_to(p/'frontend/node_modules')))
 put(z,'npm-packages.json',json.dumps(packages,indent=2))
print('PASS: dependency notices and inventories packaged')
