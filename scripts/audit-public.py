"""Audit Git candidates before add/commit and tracked files in CI; never print secrets."""
import pathlib,subprocess,re,sys,json
p=pathlib.Path(__file__).resolve().parent.parent
raw=subprocess.check_output(['git','ls-files','-z','--cached','--others','--exclude-standard'],cwd=p)
files=sorted(set(x.decode('utf-8') for x in raw.split(b'\0') if x))
blocked=[]
secret=re.compile(rb'-----BEGIN (?:[A-Z ]*PRIVATE KEY)-----|(?:gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,}|AKIA[0-9A-Z]{16})|https://[^/\s:]+:[^/\s@]+@')
for name in files:
 f=p/name;parts=pathlib.PurePosixPath(name).parts
 if f.is_symlink():blocked.append((name,'symlink requires review'));continue
 if not f.is_file():continue
 low=f.name.lower()
 if (any(x in {'data','profiles','node_modules','wailsjs','.git','__pycache__'} for x in parts)
     or (low.startswith('.env') and low!='.env.example')
     or low.endswith(('.key','.pem','.p12','.pfx','.db','.sqlite','.sqlite3','.exe','.gz','.zip','.syso','.log'))
     or '.db-' in low or low in {'identity.key','settings.local.json','credentials.json'}
     or name.startswith(('docs/evidence/','build/bin/','build/windows/installer/tmp/'))):
  blocked.append((name,'private/generated path'));continue
 data=f.read_bytes()
 if len(data)>2*1024*1024:blocked.append((name,'oversize file requires review'))
 if secret.search(data):blocked.append((name,'possible credential/private key'))
 # Prevent accidentally committing exported application profiles even under another name.
 if f.suffix=='.json' and low not in {'package-lock.json'}:
  try:
   obj=json.loads(data)
   if isinstance(obj,dict) and ('privateKey' in obj or ('bootstrap' in obj and 'relays' in obj and 'name' in obj)):
    blocked.append((name,'possible application profile'))
  except (ValueError,UnicodeDecodeError):pass
if blocked:
 for name,reason in blocked:print('BLOCKED:',name,reason)
 sys.exit(1)
print(f'PASS: {len(files)} Git candidates; no forbidden paths or credential patterns found. Review and gitleaks remain required.')
