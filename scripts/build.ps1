param([string]$GoCommand = 'go')
$ErrorActionPreference = 'Stop'
Push-Location (Join-Path $PSScriptRoot '..')
$oldOS=$env:GOOS; $oldArch=$env:GOARCH; $oldCGO=$env:CGO_ENABLED
try {
 New-Item -ItemType Directory -Force dist | Out-Null
 $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
 foreach ($target in @('windows','linux')) {
  $env:GOOS=$target
  $name="dist/projetop2p-$target-amd64"; if ($target -eq 'windows') {$name+='.exe'}
  & $GoCommand build -buildvcs=false -trimpath -o $name ./cmd/projetop2p
  if ($LASTEXITCODE -ne 0) {throw "Build $target falhou"}
 }
} finally { $env:GOOS=$oldOS; $env:GOARCH=$oldArch; $env:CGO_ENABLED=$oldCGO; Pop-Location }
