param([string]$Setup='build/bin/ProjetoP2P-Setup-v0.5.0.exe')
$ErrorActionPreference='Stop'
$setupPath=(Resolve-Path $Setup).Path
$output=Join-Path (Split-Path $setupPath) 'setup-verify.json'
$p=Start-Process -FilePath $setupPath -ArgumentList '--verify-payload' -RedirectStandardOutput $output -WindowStyle Hidden -Wait -PassThru
if ($p.ExitCode -ne 0) {throw 'Setup payload check failed'}
$r=Get-Content $output -Raw | ConvertFrom-Json
$app=Join-Path (Split-Path $setupPath) 'ProjetoP2P.exe'
if ($r.version -ne '0.5.0' -or ($r.hamachiBundled -or $r.radminBundled -or $r.vpnBundled) -or $r.appSHA256 -ne (Get-FileHash $app -Algorithm SHA256).Hash.ToLower()) {throw 'Payload mismatch'}
'PASS: setup contains the exact desktop build, uninstaller and runtime bootstrapper; No VPN software is bundled. Installation was not performed.'
if ($r.licenseSHA256 -ne (Get-FileHash LICENSE -Algorithm SHA256).Hash.ToLower() -or $r.thirdPartySHA256 -ne (Get-FileHash cmd/p2p-setup/payload/THIRD_PARTY_LICENSES.zip -Algorithm SHA256).Hash.ToLower()) {throw 'Installer licenses mismatch'}
'PASS: MIT and third-party license payload hashes match'
