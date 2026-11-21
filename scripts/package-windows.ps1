param([string]$GoCommand='go',[string]$SigningThumbprint='',[string]$DesktopExecutable='build/bin/PeeTwoPee.exe')
$ErrorActionPreference='Stop'
Push-Location (Join-Path $PSScriptRoot '..')
$oldCGO=$env:CGO_ENABLED; $oldOS=$env:GOOS; $oldArch=$env:GOARCH
try {
 $env:GOOS='windows';$env:GOARCH='amd64';$env:CGO_ENABLED='0'
 if (-not (Test-Path $DesktopExecutable)) {throw 'Execute o build Wails antes de empacotar.'}
 $versionFile=Join-Path (Resolve-Path build/bin).Path 'app-version.txt'
 $versionProc=Start-Process -FilePath (Resolve-Path $DesktopExecutable).Path -ArgumentList '--version' -RedirectStandardOutput $versionFile -WindowStyle Hidden -Wait -PassThru
 if ($versionProc.ExitCode -ne 0 -or (Get-Content $versionFile -Raw).Trim() -ne '0.10.7') {throw 'Expected a Wails desktop build matching VERSION (0.10.7).'}
 $policyFile=Join-Path (Resolve-Path build/bin).Path 'app-permission-policy.txt'
 $policyProc=Start-Process -FilePath (Resolve-Path $DesktopExecutable).Path -ArgumentList '--permission-policy' -RedirectStandardOutput $policyFile -WindowStyle Hidden -Wait -PassThru
 if ($policyProc.ExitCode -ne 0 -or (Get-Content $policyFile -Raw).Trim() -ne 'prompt-v1'){throw 'Desktop sem politica de consentimento verificada'}
 if ($SigningThumbprint) {./scripts/sign-windows.ps1 -Files @($DesktopExecutable) -Thumbprint $SigningThumbprint}
 $runtime='build/windows/installer/tmp/MicrosoftEdgeWebview2Setup.exe'
 if (-not (Test-Path $runtime)) {
  New-Item -ItemType Directory -Force (Split-Path $runtime) | Out-Null
  Invoke-WebRequest 'https://go.microsoft.com/fwlink/p/?LinkId=2124703' -OutFile $runtime
 }
 $sig=Get-AuthenticodeSignature $runtime
 if ($sig.Status -ne 'Valid' -or $sig.SignerCertificate.Subject -notmatch 'Microsoft') {throw 'Bootstrapper WebView2 sem assinatura Microsoft vÃ¡lida.'}
 python scripts/collect-licenses.py
 if ($LASTEXITCODE -ne 0) {throw 'Coleta de licenças falhou'}
 $obsoletePayload=Join-Path (Resolve-Path cmd/p2p-setup/payload).Path 'ProjetoP2P.exe.gz'
 if(Test-Path -LiteralPath $obsoletePayload){Remove-Item -LiteralPath $obsoletePayload -Force}
 $helper='build/bin/easytier/etcat-p2p.exe'
 $expected=(Get-Content internal/overlay/manifest.json -Raw | ConvertFrom-Json).sha256
 if ((Get-FileHash $helper -Algorithm SHA256).Hash.ToLower() -ne $expected) {throw 'Auxiliar diferente do manifesto auditado'}
 if (-not(Test-Path 'dist/release/EasyTier-corresponding-source.zip')) {throw 'Fontes correspondentes LGPL obrigatorias'}
 python scripts/audit-pe.py $helper $DesktopExecutable
 if ($LASTEXITCODE -ne 0) {throw 'Auditoria PE falhou'}
 Copy-Item $helper cmd/p2p-setup/payload/etcat-p2p.exe -Force
 Copy-Item dist/release/EasyTier-corresponding-source.zip cmd/p2p-setup/payload/EasyTier-corresponding-source.zip -Force
 Copy-Item LICENSE cmd/p2p-setup/payload/LICENSE.txt -Force
 Copy-Item dist/release/PeeTwoPee-v0.10.7-licencas-windows.zip cmd/p2p-setup/payload/THIRD_PARTY_LICENSES.zip -Force
 $legacyPayload=Join-Path (Resolve-Path cmd/p2p-setup/payload).Path 'PeeTwoPee.exe'
 if (Test-Path -LiteralPath $legacyPayload) {Remove-Item -LiteralPath $legacyPayload -Force}
 $source=[IO.File]::OpenRead((Resolve-Path $DesktopExecutable).Path)
 $archive=[IO.File]::Create((Join-Path (Resolve-Path cmd/p2p-setup/payload).Path 'PeeTwoPee.exe.gz'))
 $gzip=[IO.Compression.GZipStream]::new($archive,[IO.Compression.CompressionLevel]::Optimal)
 try {$source.CopyTo($gzip)} finally {$source.Dispose();$gzip.Dispose();$archive.Dispose()}
 Copy-Item $runtime cmd/p2p-setup/payload/MicrosoftEdgeWebview2Setup.exe -Force
 & $GoCommand build -buildvcs=false -trimpath -ldflags '-H windowsgui' -o cmd/p2p-setup/payload/uninstall.exe ./cmd/p2p-uninstall
 if ($LASTEXITCODE -ne 0) {throw 'Build do desinstalador falhou'}
 & $GoCommand run ./cmd/p2p-resources (Resolve-Path cmd/p2p-setup/payload/uninstall.exe).Path
 if ($LASTEXITCODE -ne 0) {throw 'Manifest do desinstalador falhou'}
 if ($SigningThumbprint) {./scripts/sign-windows.ps1 -Files @('cmd/p2p-setup/payload/uninstall.exe') -Thumbprint $SigningThumbprint}
 & $GoCommand build -buildvcs=false -trimpath -ldflags '-H windowsgui' -o build/bin/PeeTwoPee-Setup-v0.10.7.exe ./cmd/p2p-setup
 if ($LASTEXITCODE -ne 0) {throw 'Build do instalador falhou'}
 & $GoCommand run ./cmd/p2p-resources (Resolve-Path build/bin/PeeTwoPee-Setup-v0.10.7.exe).Path
 if ($LASTEXITCODE -ne 0) {throw 'Manifest do instalador falhou'}
 if ($SigningThumbprint) {./scripts/sign-windows.ps1 -Files @('build/bin/PeeTwoPee-Setup-v0.10.7.exe') -Thumbprint $SigningThumbprint}
 $setup=(Resolve-Path build/bin/PeeTwoPee-Setup-v0.10.7.exe).Path
 $output=Join-Path (Resolve-Path build/bin).Path 'setup-payload.json'
 $proc=Start-Process -FilePath $setup -ArgumentList '--verify-payload' -RedirectStandardOutput $output -WindowStyle Hidden -Wait -PassThru
 if ($proc.ExitCode -ne 0) {throw 'Falha ao verificar payload do setup'}
 $report=Get-Content $output -Raw | ConvertFrom-Json
 if ($report.appSHA256 -ne (Get-FileHash $DesktopExecutable -Algorithm SHA256).Hash.ToLower() -or ($report.hamachiBundled -or $report.radminBundled -or $report.vpnBundled)) {throw 'Payload do instalador invÃ¡lido'}
 if ($report.licenseSHA256 -ne (Get-FileHash LICENSE -Algorithm SHA256).Hash.ToLower() -or $report.thirdPartySHA256 -ne (Get-FileHash cmd/p2p-setup/payload/THIRD_PARTY_LICENSES.zip -Algorithm SHA256).Hash.ToLower()) {throw 'Licenças do instalador inválidas'}
 if (-not $report.integratedHelperBundled -or $report.helperSHA256 -ne $expected) {throw "Auxiliar nao corresponde ao setup"}
 $report | ConvertTo-Json
} finally {$env:CGO_ENABLED=$oldCGO;$env:GOOS=$oldOS;$env:GOARCH=$oldArch;Pop-Location}
