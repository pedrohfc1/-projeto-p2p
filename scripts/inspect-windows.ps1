param([Parameter(Mandatory=$true)][string[]]$Files,[switch]$RequireSigned)
$ErrorActionPreference='Stop'
$defenderStatus='unavailable'
$detections=@()
if (Get-Command Get-MpThreatDetection -ErrorAction SilentlyContinue) {
 try {$detections=@(Get-MpThreatDetection -ErrorAction Stop);$defenderStatus='queried'} catch {$defenderStatus='unavailable'}
}
$results=@()
$blocked=$false
foreach ($file in $Files) {
 $path=(Resolve-Path -LiteralPath $file).Path
 $signature=Get-AuthenticodeSignature -LiteralPath $path
 $matching=@($detections | Where-Object {@($_.Resources | Where-Object {$_ -like "*$path*"}).Count -gt 0})
 if ($matching.Count -gt 0 -or $signature.Status -eq 'HashMismatch') {$blocked=$true}
 if ($RequireSigned -and $signature.Status -ne 'Valid') {$blocked=$true}
 $results+=[pscustomobject]@{file=[IO.Path]::GetFileName($path);sha256=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLower();authenticode=$signature.Status.ToString();publisher=$signature.SignerCertificate.Subject;defender=$defenderStatus;detections=@($matching | Select-Object -ExpandProperty ThreatID)}
}
$results | ConvertTo-Json -Depth 5
if ($blocked) {throw 'Interrompa a distribuicao e investigue a deteccao ou assinatura invalida. Nao desative protecoes.'}
