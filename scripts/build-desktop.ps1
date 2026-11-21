param([string]$Output='PeeTwoPee.exe')
$ErrorActionPreference='Stop'
Push-Location (Join-Path $PSScriptRoot '..')
$modPath=Join-Path (Get-Location) 'go.mod';$sumPath=Join-Path (Get-Location) 'go.sum'
$originalMod=[IO.File]::ReadAllBytes($modPath);$originalSum=[IO.File]::ReadAllBytes($sumPath)
try {
 ./scripts/build-loopback.ps1
 $localSDK=(& python scripts/prepare-webview-policy.py | Select-Object -Last 1)
 if($LASTEXITCODE){throw 'Falha ao preparar politica de permissao'}
 go mod edit "-replace=github.com/wailsapp/wails/v2=./build/permission-policy/wails"
 if($LASTEXITCODE){throw 'Falha ao selecionar SDK local'}
 wails build -trimpath -o $Output -ldflags '-X main.permissionPolicy=prompt-v1'
 if($LASTEXITCODE){throw 'Build desktop falhou'}
} finally {[IO.File]::WriteAllBytes($modPath,$originalMod);[IO.File]::WriteAllBytes($sumPath,$originalSum);Pop-Location}

