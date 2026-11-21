$ErrorActionPreference='Stop'
$root=Split-Path $PSScriptRoot -Parent
$csc=Join-Path $env:WINDIR 'Microsoft.NET/Framework64/v4.0.30319/csc.exe'
if(!(Test-Path -LiteralPath $csc)){throw '.NET Framework C# compiler required to build Windows WASAPI helper'}
& $csc /nologo /optimize+ /platform:x64 /target:exe ("/out:"+(Join-Path $root 'internal/loopback/capture.bin')) (Join-Path $root 'native/loopback/LoopbackCapture.cs')
if($LASTEXITCODE){throw 'WASAPI helper build failed'}
