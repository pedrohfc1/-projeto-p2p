param([switch]$Installer)
$ErrorActionPreference='Stop'
Push-Location (Join-Path $PSScriptRoot '..')
try {
 if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {throw 'Instale Wails v2.17.0 e adicione o diretório de binários do Go ao PATH.'}
 wails build -trimpath -webview2 embed
 if ($LASTEXITCODE -ne 0) {throw 'Build desktop falhou; confira o diagnóstico acima.'}
 if ($Installer) {./scripts/package-windows.ps1}
} finally {Pop-Location}
