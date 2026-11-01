param(
 [Parameter(Mandatory=$true)][string[]]$Files,
 [Parameter(Mandatory=$true)][string]$Thumbprint,
 [string]$TimestampServer='http://timestamp.digicert.com'
)
$ErrorActionPreference='Stop'
if ($Thumbprint -notmatch '^[0-9a-fA-F]{40}$') {throw 'Informe um thumbprint valido do armazenamento de certificados do usuario.'}
$cert=Get-Item -LiteralPath "Cert:\CurrentUser\My\$Thumbprint" -ErrorAction Stop
if (-not $cert.HasPrivateKey -or $cert.NotAfter -lt (Get-Date) -or $cert.NotBefore -gt (Get-Date)) {throw 'Certificado expirado ou sem chave privada.'}
$eku=@($cert.Extensions | Where-Object {$_.Oid.Value -eq '2.5.29.37'} | ForEach-Object {$_.EnhancedKeyUsages} | ForEach-Object {$_.Value})
if ($eku -notcontains '1.3.6.1.5.5.7.3.3') {throw 'O certificado precisa permitir Code Signing.'}
$tool=Get-Command signtool.exe -ErrorAction SilentlyContinue
if (-not $tool) {
 $sdk=Join-Path ${env:ProgramFiles(x86)} 'Windows Kits/10/bin/*/x64/signtool.exe'
 $candidate=Get-ChildItem -Path $sdk -ErrorAction SilentlyContinue | Sort-Object FullName -Descending | Select-Object -First 1
 if (-not $candidate) {throw 'Instale o Windows SDK com SignTool para assinar.'}
 $signTool=$candidate.FullName
} else {$signTool=$tool.Source}
foreach ($file in $Files) {
 $path=(Resolve-Path -LiteralPath $file).Path
 $old=Get-AuthenticodeSignature -LiteralPath $path
 if ($old.Status -ne 'Valid' -or $old.SignerCertificate.Thumbprint -ne $Thumbprint) {
  & $signTool sign /fd SHA256 /sha1 $Thumbprint /s My /tr $TimestampServer /td SHA256 $path
  if ($LASTEXITCODE -ne 0) {throw 'SignTool falhou; nao publique o artefato.'}
 }
 $signature=Get-AuthenticodeSignature -LiteralPath $path
 if ($signature.Status -ne 'Valid' -or $signature.SignerCertificate.Thumbprint -ne $Thumbprint) {throw 'A assinatura final nao e valida para o certificado solicitado.'}
 "Assinatura Authenticode valida: $([IO.Path]::GetFileName($path))"
}
