"""Fail before building/publishing inconsistent versions. No generated files needed."""
from pathlib import Path
import json,re,sys
p=Path(__file__).resolve().parent.parent
v=(p/'VERSION').read_text(encoding='utf-8').strip()
assert re.fullmatch(r'0|[1-9][0-9]*',v.split('.')[0]) and re.fullmatch(r'(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)',v), 'Invalid SemVer'
for f,key in [('frontend/package.json','version'),('frontend/package-lock.json','version')]:
 assert json.loads((p/f).read_text(encoding='utf-8'))[key]==v,f
assert json.loads((p/'frontend/package-lock.json').read_text(encoding='utf-8'))['packages']['']['version']==v
assert json.loads((p/'wails.json').read_text(encoding='utf-8'))['info']['productVersion']==v
for f,marker in [('main.go',f'fmt.Println("{v}")'),('internal/installwin/install_windows.go',f'const Version = "{v}"'),('cmd/p2p-resources/main_windows.go',f'version="{v}.0"')]:
 assert marker in (p/f).read_text(encoding='utf-8'),f
for f in ['scripts/package-windows.ps1','tests/installer-smoke.ps1']:
 assert f'Setup-v{v}.exe' in (p/f).read_text(encoding='utf-8'),f
if len(sys.argv)>1:assert sys.argv[1]=='v'+v,'Tag/version mismatch'
assert f'ProjetoP2P-v{v}-licencas-windows.zip' in (p/'scripts/package-windows.ps1').read_text(encoding='utf-8')
print('PASS: version '+v+' matches app, frontend, installer and optional tag')
