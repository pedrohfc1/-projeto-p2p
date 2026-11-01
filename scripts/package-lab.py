"""Package our experimental app, never third-party EasyTier/Npcap executables."""
from pathlib import Path
import zipfile
p=Path(__file__).resolve().parent.parent;v=(p/'VERSION').read_text().strip();out=p/'dist/release';out.mkdir(parents=True,exist_ok=True)
target=out/f'ProjetoP2P-v{v}-laboratorio-windows-amd64.zip'
files=['LICENSE','THIRD_PARTY_NOTICES.md','docs/EASYTIER-v0.5.2.md','docs/CONVITES-v0.5.1.md','docs/VALIDACAO-v0.5.1-v0.5.2.md','scripts/prepare-easytier.py']
guide="""# Laboratorio EasyTier experimental

Este pacote nao altera o transporte dos contatos/comunidades. Para uso normal prefira o instalador 0.5.1.

Leia docs/EASYTIER-v0.5.2.md (seguranca, licencas e limites) antes de preparar componentes.
O binario oficial exige Packet.dll do Npcap: nao esta incluida e nao e coberta pela LGPL do EasyTier.
Nao redistribua componentes baixados sem resolver as licencas/fontes/notices correspondentes.

Preparacao explicita de DESENVOLVIMENTO, Python 3, sem executar downloads:

    python scripts/prepare-easytier.py --output ./easytier

Executar app com perfil separado, sem tocar no perfil pessoal:

    ./ProjetoP2P.exe --data ./perfil-laboratorio

Exige WebView2. Configuracoes > Rede P2P integrada (laboratorio experimental).
Use nome/segredo iguais, IPs virtuais distintos e somente nos conhecidos autorizados.
Nao ha relay publico ou STUN embutido. Processo ativo nao comprova conexao.
Pare o laboratorio antes de remover a pasta. Nao apagar %APPDATA%/ProjetoP2P.
Sem voz, tela, blockchain ou E2EE de grupo novo. Nenhuma release publicada.
"""
with zipfile.ZipFile(target,'w',zipfile.ZIP_DEFLATED) as z:
 z.write(p/'build/bin/ProjetoP2P.exe','ProjetoP2P.exe')
 z.writestr('LEIA-ME-LAB.md',guide)
 for name in files:z.write(p/name,name)
 z.write(out/f'ProjetoP2P-v{v}-licencas-windows.zip','THIRD_PARTY_LICENSES.zip')
with zipfile.ZipFile(target) as z:
 assert z.testzip() is None
 assert all(Path(n).name not in ('easytier-core.exe','easytier-cli.exe','Packet.dll','wintun.dll','WinDivert64.sys') for n in z.namelist())
print('PASS:',target.name,'contains no EasyTier/Npcap binary or driver')
