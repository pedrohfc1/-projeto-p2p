param([string]$Setup,[string]$DefaultDirectory,[string]$Version,[string]$WorkDirectory)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[Windows.Forms.Application]::EnableVisualStyles()
$form=New-Object Windows.Forms.Form
$form.Text="PeeTwoPee — Instalação beta $Version"
$form.ClientSize=New-Object Drawing.Size(680,510)
$form.StartPosition='CenterScreen';$form.FormBorderStyle='FixedDialog';$form.MaximizeBox=$false
$form.BackColor=[Drawing.ColorTranslator]::FromHtml('#232428');$form.ForeColor=[Drawing.ColorTranslator]::FromHtml('#F2F3F5');$form.Font=New-Object Drawing.Font('Segoe UI',10)
$logo=New-Object Windows.Forms.PictureBox;$logo.Bounds=New-Object Drawing.Rectangle(24,22,54,54);$logo.SizeMode='Zoom';$logo.Image=[Drawing.Image]::FromFile((Join-Path $WorkDirectory 'logo.png'));$form.Controls.Add($logo)
function Label($text,$x,$y,$w,$h){$c=New-Object Windows.Forms.Label;$c.Text=$text;$c.Bounds=New-Object Drawing.Rectangle($x,$y,$w,$h);$form.Controls.Add($c);return $c}
$title=Label 'Bem-vindo ao PeeTwoPee' 92 22 550 35;$title.Font=New-Object Drawing.Font('Segoe UI',19,[Drawing.FontStyle]::Bold)
$sub=Label "Beta privado $Version · EasyTier integrado · seus dados serão preservados" 94 61 550 25
$installed=Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\ProjetoP2PProjetoP2P' -ErrorAction SilentlyContinue
$installedText=if($installed.DisplayVersion){"Versão instalada: $($installed.DisplayVersion). Será atualizada para $Version."}else{'Nova instalação para este usuário. Não requer administrador na pasta padrão.'}
$status=Label $installedText 24 100 630 28
$folderLabel=Label 'Pasta de instalação' 24 140 400 22
$folder=New-Object Windows.Forms.TextBox;$folder.Bounds=New-Object Drawing.Rectangle(24,168,515,28);$folder.Text=$DefaultDirectory;$form.Controls.Add($folder)
$browse=New-Object Windows.Forms.Button;$browse.Text='Procurar…';$browse.Bounds=New-Object Drawing.Rectangle(547,166,108,30);$form.Controls.Add($browse)
$browse.Add_Click({$d=New-Object Windows.Forms.FolderBrowserDialog;$d.Description='Escolha a pasta dos executáveis; nunca selecione seu perfil de dados.';$d.SelectedPath=$folder.Text;if($d.ShowDialog() -eq 'OK'){$folder.Text=$d.SelectedPath};$d.Dispose()})
function Check($text,$y,$checked){$c=New-Object Windows.Forms.CheckBox;$c.Text=$text;$c.Bounds=New-Object Drawing.Rectangle(24,$y,630,26);$c.Checked=$checked;$form.Controls.Add($c);return $c}
$desktop=Check 'Criar atalho na área de trabalho' 211 (Test-Path (Join-Path ([Environment]::GetFolderPath('Desktop')) 'PeeTwoPee.lnk'))
$startWanted=if($installed.DisplayVersion){Test-Path (Join-Path ([Environment]::GetFolderPath('Programs')) 'PeeTwoPee.lnk')}else{$true}
$start=Check 'Criar atalho no Menu Iniciar' 242 $startWanted
$startup=Check 'Iniciar PeeTwoPee com Windows' 273 (Test-Path (Join-Path ([Environment]::GetFolderPath('Startup')) 'PeeTwoPee.lnk'))
$radmin=Check 'Instalar Radmin VPN (opcional · abrir download oficial ao concluir)' 304 $false
$radminPresent=Get-Service -Name RvControlSvc -ErrorAction SilentlyContinue
if($radminPresent){$radmin.Text='Radmin VPN já detectado — nenhuma reinstalação será feita';$radmin.Enabled=$false}
$terms=New-Object Windows.Forms.LinkLabel;$terms.Text='Termos do Radmin VPN';$terms.LinkColor=[Drawing.ColorTranslator]::FromHtml('#AEB5FF');$terms.Bounds=New-Object Drawing.Rectangle(45,334,250,22);$form.Controls.Add($terms);$terms.Add_LinkClicked({Start-Process 'https://www.radmin-vpn.com/eula/'})
$note=Label 'Radmin não está incluído neste pacote. Se solicitado, o site oficial abrirá para você baixar, revisar os termos e instalar com consentimento. EasyTier continua padrão.' 24 362 630 46
$progress=New-Object Windows.Forms.ProgressBar;$progress.Bounds=New-Object Drawing.Rectangle(24,418,630,15);$form.Controls.Add($progress)
$install=New-Object Windows.Forms.Button;$install.Text='Instalar';$install.BackColor=[Drawing.ColorTranslator]::FromHtml('#5865F2');$install.ForeColor=[Drawing.Color]::White;$install.FlatStyle='Flat';$install.Bounds=New-Object Drawing.Rectangle(523,459,131,34);$form.Controls.Add($install)
$cancel=New-Object Windows.Forms.Button;$cancel.Text='Cancelar';$cancel.Bounds=New-Object Drawing.Rectangle(398,459,115,34);$form.Controls.Add($cancel);$cancel.Add_Click({$form.Close()})
$open=Check 'Abrir PeeTwoPee ao concluir' 416 $true;$open.Visible=$false
$script:child=$null;$script:finished=$false;$script:wantRadmin=$false;$timer=New-Object Windows.Forms.Timer;$timer.Interval=200
$form.Add_FormClosing({param($s,$e) if($script:child -and !$script:child.HasExited){$e.Cancel=$true}})
$install.Add_Click({
 if($script:finished){if($open.Checked){Start-Process -FilePath (Join-Path $folder.Text 'PeeTwoPee.exe') -WorkingDirectory $folder.Text};if($script:wantRadmin){try{Start-Process 'https://www.radmin-vpn.com/'}catch{[Windows.Forms.MessageBox]::Show('Abra https://www.radmin-vpn.com/ para baixar o Radmin. O PeeTwoPee já está instalado.')}};$form.Close();return}
 try{
  if(![IO.Path]::IsPathRooted($folder.Text) -or $folder.Text.Contains('"')){throw 'Escolha uma pasta absoluta válida.'}
  $options=Join-Path $WorkDirectory 'options.json'
  @{directory=$folder.Text;desktop=$desktop.Checked;startMenu=$start.Checked;startup=$startup.Checked}|ConvertTo-Json|Set-Content -Encoding UTF8 $options
  $script:wantRadmin=$radmin.Checked
  $confirmation="Instalar $Version em $($folder.Text)?`n`nAtalho na área de trabalho: $($desktop.Checked)`nMenu Iniciar: $($start.Checked)`nIniciar com Windows: $($startup.Checked)`nDownload opcional do Radmin: $($radmin.Checked)`n`nIdentidades e histórico serão preservados. Instâncias abertas serão encerradas somente após sua confirmação. WebView2 será instalado se necessário (internet)."
  if([Windows.Forms.MessageBox]::Show($confirmation,'Confirmar instalação','YesNo','Question') -ne 'Yes'){return}
  $psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName=$Setup;$psi.Arguments='--apply-options "'+$options+'"';$psi.UseShellExecute=$false;$psi.CreateNoWindow=$true
  $script:child=[Diagnostics.Process]::Start($psi)
  foreach($c in @($folder,$browse,$desktop,$start,$startup,$radmin,$cancel,$install)){$c.Enabled=$false}
  $status.Text='Instalando e verificando arquivos…';$progress.Style='Marquee';$timer.Start()
 }catch{[Windows.Forms.MessageBox]::Show($_.Exception.Message,'Instalação não iniciada','OK','Error')}
})
$timer.Add_Tick({
 if(!$script:child.HasExited){return};$timer.Stop();$progress.Style='Blocks'
 $resultPath=Join-Path $WorkDirectory 'options.json.result'
 $result=if(Test-Path $resultPath){Get-Content $resultPath -Raw|ConvertFrom-Json}else{@{error='O processo de instalação terminou sem confirmação. Seus dados não foram removidos.'}}
 if($script:child.ExitCode -ne 0 -or $result.error){$status.Text='Instalação não concluída.';$note.Text=$result.error;$cancel.Enabled=$true;$install.Enabled=$true;$script:child=$null;foreach($c in @($folder,$browse,$desktop,$start,$startup)){$c.Enabled=$true};$radmin.Enabled=!$radminPresent;return}
 $progress.Value=100;$progress.Visible=$false;$script:finished=$true;$status.Text='Instalação concluída e integridade verificada.';$note.Text='Identidade, histórico e configurações preservados. O Radmin, se solicitado, será obtido separadamente pelo site oficial.';$open.Visible=$true;$install.Text='Concluir';$install.Enabled=$true;$cancel.Visible=$false
})
try{[void]$form.ShowDialog()}finally{$timer.Dispose();$logo.Image.Dispose();$form.Dispose();if($script:child){$script:child.Dispose()}}
