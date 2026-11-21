param()
$ErrorActionPreference='Stop'
Push-Location $PSScriptRoot
try {
 go test -tags ts_omit_logtail .
 if($LASTEXITCODE){throw 'Testes do laboratório falharam'}
 go build -tags ts_omit_logtail -trimpath -o ../../build/bin/PeeTwoPee-Tailscale-Lab.exe .
 if($LASTEXITCODE){throw 'Build do laboratório falhou'}
 python licenses.py
 if($LASTEXITCODE){throw 'Licenças pendentes; não redistribuir o laboratório'}
} finally {Pop-Location}
