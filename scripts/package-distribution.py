"""Create runtime packages using explicit allowlists; never include user profiles."""
from pathlib import Path
import argparse,tarfile,zipfile,hashlib,io,subprocess,json,platform
p=Path(__file__).resolve().parent.parent
v=(p/'VERSION').read_text(encoding='utf-8').strip()
a=argparse.ArgumentParser();a.add_argument('kind',choices=['windows','linux','linux-cli','source','checksums']);args=a.parse_args()
out=p/'dist/release';out.mkdir(parents=True,exist_ok=True)
docs=['LICENSE','THIRD_PARTY_NOTICES.md','README.md','CHANGELOG.md','docs/INSTALACAO.md','docs/CONEXAO-RADMIN-VPN.md','docs/CONEXAO-HAMACHI.md','SECURITY.md','docs/COMUNIDADES-v0.5.md','docs/SEGURANCA-v0.5.md','docs/METRICAS-v0.5.md','docs/PLANO-v0.6-VOZ.md','docs/CRIPTOGRAFIA-E2EE-PLANO.md','docs/ASSINATURA-WINDOWS.md','docs/VALIDACAO-v0.5.md','docs/CONVITES-v0.5.1.md','docs/EASYTIER-v0.5.2.md','docs/VALIDACAO-v0.5.1-v0.5.2.md']
docs += ['docs/EASYTIER-INTEGRADO-BETA-v0.5.2.md','docs/SEGURANCA-DHT-BETA.md','docs/VALIDACAO-BETA-v0.5.2.md','docs/METRICAS-BETA-v0.5.2.json']
docs += ['docs/MIDIA-ESTABILIDADE-v0.10.7.md','docs/DIAGNOSTICO-TSNET-v0.10.6.md','docs/DIAGNOSTICO-TSNET-v0.10.5.md','docs/ETAPA3-v0.10.4.md']

docs += ['docs/VOZ-PTT-v0.10.3.md','docs/VOZ-PTT-v0.10.3-VALIDACAO.json']

docs += ['docs/DIAGNOSTICO-REDE-v0.10.2.md','docs/DIAGNOSTICO-REDE-v0.10.2-VALIDACAO.json']

docs += ['docs/MICROFONE-MENU-v0.10.1.md','docs/MICROFONE-MENU-v0.10.1-VALIDACAO.json']

docs += ['docs/CENTRAL-CONFIGURACOES-v0.10.0.md','docs/CENTRAL-CONFIGURACOES-v0.10.0-VALIDACAO.json']

docs += ['docs/REPRODUCAO-VOZ-v0.9.9.md','docs/REPRODUCAO-VOZ-v0.9.9-VALIDACAO.json']

docs += ['docs/AUDIO-INTERFACE-v0.9.8.md','docs/AUDIO-INTERFACE-v0.9.8-VALIDACAO.json']

docs += ['docs/AUDIO-TELA-v0.9.7.md','docs/AUDIO-TELA-v0.9.7-VALIDACAO.json']

docs += ['docs/HOTFIX-v0.9.6.md','docs/HOTFIX-v0.9.6-VALIDACAO.json','docs/VOICE-HOTFIX-v0.9.5.md','docs/VOICE-HOTFIX-v0.9.5-VALIDACAO.json']

docs += ['docs/DIAGNOSTICO-v0.9.4.md','docs/DIAGNOSTICO-v0.9.4-VALIDACAO.json']

docs += ['docs/PEETWOPEE-v0.9.2.md', 'docs/PEETWOPEE-v0.9.3.md', 'docs/PEETWOPEE-v0.9.3-VALIDACAO.json','docs/PEETWOPEE-v0.9.2-VALIDACAO.json']

docs += ['docs/PEETWOPEE-v0.9.1.md','docs/PEETWOPEE-v0.9.1-VALIDACAO.json']

docs += ['docs/PEETWOPEE-v0.9.0.md','docs/PEETWOPEE-v0.9.0-VALIDACAO.json']

docs += ['docs/PEETWOPEE-v0.8.2.md','docs/PEETWOPEE-v0.8.2-VALIDACAO.json']

docs += ['docs/PEETWOPEE-v0.8.1.md','docs/PEETWOPEE-v0.8.1-VALIDACAO.json']

docs += ['docs/PEETWOPEE-v0.8.0.md','docs/PEETWOPEE-v0.8.0-VALIDACAO.json']

docs += ['docs/PEETWOPEE-v0.7.0.md','docs/PEETWOPEE-v0.7.0-VALIDACAO.json']
docs += ['docs/PEETWOPEE-v0.6.2.md','docs/PEETWOPEE-v0.6.2-VALIDACAO.json']
docs += ['docs/PEETWOPEE-v0.6.1.md','docs/PEETWOPEE-v0.6.1-METRICAS.json']
docs += ['docs/BETA-v0.6.0.md','docs/VALIDACAO-v0.6.0.md']
if args.kind=='checksums':
 files=sorted(f for f in out.iterdir() if f.is_file() and f.name!='SHA256SUMS.txt')
 assert files,'No release files'
 (out/'SHA256SUMS.txt').write_text(''.join(hashlib.sha256(f.read_bytes()).hexdigest()+'  '+f.name+'\n' for f in files),encoding='utf-8')
 print('PASS: SHA256SUMS.txt');raise SystemExit
if args.kind=='source':
 subprocess.run(['python' if __import__('os').name=='nt' else 'python3','scripts/audit-public.py'],cwd=p,check=True)
 files=subprocess.check_output(['git','ls-files','-z','--cached','--others','--exclude-standard'],cwd=p).split(b'\0')
 assert any(files),'No audited source candidates'
 target=out/f'PeeTwoPee-v{v}-fonte.zip'
 with zipfile.ZipFile(target,'w',zipfile.ZIP_DEFLATED) as z:
  for name in sorted(set(files)):
   if name:
    f=p/name.decode('utf-8');info=zipfile.ZipInfo('projeto-p2p/'+name.decode('utf-8'));info.compress_type=zipfile.ZIP_DEFLATED
    info.external_attr=(0o100755 if f.suffix=='.sh' else 0o100644)<<16;z.writestr(info,f.read_bytes())
 with zipfile.ZipFile(target) as z:assert z.testzip() is None
elif args.kind=='windows':
 import shutil
 setup=p/f'build/bin/PeeTwoPee-Setup-v{v}.exe';assert setup.stat().st_size>1000000
 shutil.copy2(setup,out/setup.name)
 target=out/f'PeeTwoPee-v{v}-windows-amd64-portatil.zip'
 with zipfile.ZipFile(target,'w',zipfile.ZIP_DEFLATED) as z:
  z.write(p/'build/bin/PeeTwoPee.exe','PeeTwoPee.exe')
  z.write(p/'build/bin/easytier/etcat-p2p.exe','easytier/etcat-p2p.exe')
  z.write(out/'EasyTier-corresponding-source.zip','EasyTier-corresponding-source.zip')
  for name in docs:z.write(p/name,name)
  notices=out/f'PeeTwoPee-v{v}-licencas-windows.zip';z.write(notices,'THIRD_PARTY_LICENSES.zip')
else:
 cli=args.kind=='linux-cli'
 binary=p/('dist/projetop2p-linux-amd64' if cli else 'build/bin/PeeTwoPee')
 assert binary.read_bytes()[:4]==b'\x7fELF','Expected Linux ELF'
 target=out/f'PeeTwoPee-v{v}-linux-amd64{"-cli" if cli else ""}.tar.gz'
 with tarfile.open(target,'w:gz') as t:
  notices=out/f'PeeTwoPee-v{v}-licencas-{platform.system().lower()}.zip'
  for name,f,mode in [('projetop2p' if cli else 'PeeTwoPee',binary,0o755),('THIRD_PARTY_LICENSES.zip',notices,0o644)]+[(n,p/n,0o644) for n in docs]:
   data=f.read_bytes();info=tarfile.TarInfo(f'PeeTwoPee-v{v}/'+name);info.size=len(data);info.mode=mode;info.mtime=0;t.addfile(info,io.BytesIO(data))
 with tarfile.open(target) as t:assert t.getmembers()[0].mode==0o755
print('PASS:',target.name)
