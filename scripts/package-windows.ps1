param([string]$GoCommand='go',[string]$SigningThumbprint='')
$ErrorActionPreference='Stop'
Push-Location (Join-Path $PSScriptRoot '..')
$oldCGO=$env:CGO_ENABLED; $oldOS=$env:GOOS; $oldArch=$env:GOARCH
try {
 $env:GOOS='windows';$env:GOARCH='amd64';$env:CGO_ENABLED='0'
 if (-not (Test-Path build/bin/ProjetoP2P.exe)) {throw 'Execute o build Wails antes de empacotar.'}
 $versionFile=Join-Path (Resolve-Path build/bin).Path 'app-version.txt'
 $versionProc=Start-Process -FilePath (Resolve-Path build/bin/ProjetoP2P.exe).Path -ArgumentList '--version' -RedirectStandardOutput $versionFile -WindowStyle Hidden -Wait -PassThru
 if ($versionProc.ExitCode -ne 0 -or (Get-Content $versionFile -Raw).Trim() -ne '0.5.2') {throw 'Expected a Wails desktop build matching VERSION (0.5.2).'}
 if ($SigningThumbprint) {./scripts/sign-windows.ps1 -Files @('build/bin/ProjetoP2P.exe') -Thumbprint $SigningThumbprint}
 $runtime='build/windows/installer/tmp/MicrosoftEdgeWebview2Setup.exe'
 if (-not (Test-Path $runtime)) {
  New-Item -ItemType Directory -Force (Split-Path $runtime) | Out-Null
  Invoke-WebRequest 'https://go.microsoft.com/fwlink/p/?LinkId=2124703' -OutFile $runtime
 }
 $sig=Get-AuthenticodeSignature $runtime
 if ($sig.Status -ne 'Valid' -or $sig.SignerCertificate.Subject -notmatch 'Microsoft') {throw 'Bootstrapper WebView2 sem assinatura Microsoft vÃ¡lida.'}
 python scripts/collect-licenses.py
 if ($LASTEXITCODE -ne 0) {throw 'Coleta de licenças falhou'}
 Copy-Item LICENSE cmd/p2p-setup/payload/LICENSE.txt -Force
 Copy-Item dist/release/ProjetoP2P-v0.5.2-licencas-windows.zip cmd/p2p-setup/payload/THIRD_PARTY_LICENSES.zip -Force
 $legacyPayload=Join-Path (Resolve-Path cmd/p2p-setup/payload).Path 'ProjetoP2P.exe'
 if (Test-Path -LiteralPath $legacyPayload) {Remove-Item -LiteralPath $legacyPayload -Force}
 $source=[IO.File]::OpenRead((Resolve-Path build/bin/ProjetoP2P.exe).Path)
 $archive=[IO.File]::Create((Join-Path (Resolve-Path cmd/p2p-setup/payload).Path 'ProjetoP2P.exe.gz'))
 $gzip=[IO.Compression.GZipStream]::new($archive,[IO.Compression.CompressionLevel]::Optimal)
 try {$source.CopyTo($gzip)} finally {$source.Dispose();$gzip.Dispose();$archive.Dispose()}
 Copy-Item $runtime cmd/p2p-setup/payload/MicrosoftEdgeWebview2Setup.exe -Force
 & $GoCommand build -buildvcs=false -trimpath -ldflags '-H windowsgui' -o cmd/p2p-setup/payload/uninstall.exe ./cmd/p2p-uninstall
 if ($LASTEXITCODE -ne 0) {throw 'Build do desinstalador falhou'}
 & $GoCommand run ./cmd/p2p-resources (Resolve-Path cmd/p2p-setup/payload/uninstall.exe).Path
 if ($LASTEXITCODE -ne 0) {throw 'Manifest do desinstalador falhou'}
 if ($SigningThumbprint) {./scripts/sign-windows.ps1 -Files @('cmd/p2p-setup/payload/uninstall.exe') -Thumbprint $SigningThumbprint}
 & $GoCommand build -buildvcs=false -trimpath -ldflags '-H windowsgui' -o build/bin/ProjetoP2P-Setup-v0.5.2.exe ./cmd/p2p-setup
 if ($LASTEXITCODE -ne 0) {throw 'Build do instalador falhou'}
 & $GoCommand run ./cmd/p2p-resources (Resolve-Path build/bin/ProjetoP2P-Setup-v0.5.2.exe).Path
 if ($LASTEXITCODE -ne 0) {throw 'Manifest do instalador falhou'}
 if ($SigningThumbprint) {./scripts/sign-windows.ps1 -Files @('build/bin/ProjetoP2P-Setup-v0.5.2.exe') -Thumbprint $SigningThumbprint}
 $setup=(Resolve-Path build/bin/ProjetoP2P-Setup-v0.5.2.exe).Path
 $output=Join-Path (Resolve-Path build/bin).Path 'setup-payload.json'
 $proc=Start-Process -FilePath $setup -ArgumentList '--verify-payload' -RedirectStandardOutput $output -WindowStyle Hidden -Wait -PassThru
 if ($proc.ExitCode -ne 0) {throw 'Falha ao verificar payload do setup'}
 $report=Get-Content $output -Raw | ConvertFrom-Json
 if ($report.appSHA256 -ne (Get-FileHash build/bin/ProjetoP2P.exe -Algorithm SHA256).Hash.ToLower() -or ($report.hamachiBundled -or $report.radminBundled -or $report.vpnBundled)) {throw 'Payload do instalador invÃ¡lido'}
 if ($report.licenseSHA256 -ne (Get-FileHash LICENSE -Algorithm SHA256).Hash.ToLower() -or $report.thirdPartySHA256 -ne (Get-FileHash cmd/p2p-setup/payload/THIRD_PARTY_LICENSES.zip -Algorithm SHA256).Hash.ToLower()) {throw 'Licenças do instalador inválidas'}
 $report | ConvertTo-Json
} finally {$env:CGO_ENABLED=$oldCGO;$env:GOOS=$oldOS;$env:GOARCH=$oldArch;Pop-Location}
