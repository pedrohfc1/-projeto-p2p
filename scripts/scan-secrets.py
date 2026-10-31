"""Run gitleaks on exactly the Git candidates, excluding ignored local profiles/builds."""
import pathlib,subprocess,tempfile,shutil,sys
p=pathlib.Path(__file__).resolve().parent.parent
subprocess.run([sys.executable,str(p/'scripts/audit-public.py')],cwd=p,check=True)
files=subprocess.check_output(['git','ls-files','-z','--cached','--others','--exclude-standard'],cwd=p).split(b'\0')
(p/'dist').mkdir(exist_ok=True)
with tempfile.TemporaryDirectory(prefix='p2p-audit-',dir=p/'dist') as tmp:
 for name in set(files):
  if not name:continue
  f=p/name.decode();target=pathlib.Path(tmp)/name.decode();target.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(f,target)
 subprocess.run(['gitleaks','dir',tmp,'--redact','--no-banner'],check=True)
