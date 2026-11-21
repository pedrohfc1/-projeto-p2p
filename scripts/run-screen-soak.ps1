param(
    [ValidateSet('EasyTier','Loopback')][string]$Transport = 'EasyTier',
    [ValidateSet('720p30','1080p30','1080p60')][string]$Profile = '720p30',
    [ValidateRange(1,1800)][int]$DurationSeconds = 1800,
    [ValidateRange(0,10)][int]$RejoinCycles = 3,
    [ValidateRange(0,12000000)][int]$Bitrate = 0,
    [switch]$ScreenAudio,
    [string]$FixturePath = '',
    [string]$SwitchFixturePath = '',
    [string]$HelperDirectory = 'build/bin/easytier',
    [string]$OutputDirectory = 'data/media-qa'
)

$ErrorActionPreference = 'Stop'
Push-Location (Join-Path $PSScriptRoot '..')
$keys = @('P2P_SCREEN_IVF', 'P2P_SCREEN_TEST_FPS', 'P2P_SCREEN_TEST_BITRATE', 'P2P_SCREEN_SOAK_SECONDS', 'P2P_SCREEN_LOOPBACK', 'P2P_ETCAT_TEST_DIR', 'P2P_SCREEN_REJOIN_CYCLES', 'P2P_SCREEN_SWITCH_IVF', 'P2P_SCREEN_AUDIO_INTEGRATION')
$previous = @{}
foreach ($key in $keys) {
    $previous[$key] = [Environment]::GetEnvironmentVariable($key, 'Process')
}
try {
    $null = Get-Command go, ffmpeg
    $metadata = @{
        '720p30'  = @{ Width = 1280; Height = 720; FPS = 30; Bitrate = 3000000 }
        '1080p30' = @{ Width = 1920; Height = 1080; FPS = 30; Bitrate = 5000000 }
        '1080p60' = @{ Width = 1920; Height = 1080; FPS = 60; Bitrate = 7500000 }
    }[$Profile]
    if ($Bitrate) {
        if ($Bitrate -lt 256000) { throw 'O limite mínimo do teste é 256 kbit/s.' }
        $metadata.Bitrate = $Bitrate
    }
    $output = (New-Item -ItemType Directory -Path $OutputDirectory -Force).FullName
    if (!$FixturePath) {
        $FixturePath = Join-Path $output ('testsrc2-{0}-{1}.ivf' -f $Profile, $metadata.Bitrate)
        if (!(Test-Path -LiteralPath $FixturePath)) {
            $source = 'testsrc2=size={0}x{1}:rate={2}' -f $metadata.Width, $metadata.Height, $metadata.FPS
            ffmpeg -hide_banner -loglevel error -f lavfi -i $source -t 5 -an -c:v libvpx -deadline realtime -cpu-used 8 -threads 4 -g $metadata.FPS -b:v $metadata.Bitrate -f ivf -n $FixturePath
            if ($LASTEXITCODE) { throw 'Não foi possível gerar o VP8 sintético.' }
        }
    }
    $env:P2P_SCREEN_IVF = (Resolve-Path -LiteralPath $FixturePath).Path
    $env:P2P_SCREEN_SWITCH_IVF = if ($SwitchFixturePath) { (Resolve-Path -LiteralPath $SwitchFixturePath).Path } else { '' }
    $env:P2P_SCREEN_TEST_FPS = [string]$metadata.FPS
    $env:P2P_SCREEN_TEST_BITRATE = [string]$metadata.Bitrate
    $env:P2P_SCREEN_SOAK_SECONDS = [string]$DurationSeconds
    $env:P2P_SCREEN_REJOIN_CYCLES = [string]$RejoinCycles
    $env:P2P_SCREEN_LOOPBACK = if($Transport -eq 'Loopback') { '1' } else { '0' }
    if ($ScreenAudio) {
        $env:P2P_SCREEN_AUDIO_INTEGRATION = '1'
    }
    if ($Transport -eq 'Loopback') {
        $test = '^TestScreenAndVoiceOverLoopback$'
    } else {
        $env:P2P_ETCAT_TEST_DIR = (Resolve-Path -LiteralPath $HelperDirectory).Path
        $test = '^TestRealScreenAndVoiceOverIntegratedTransport$'
    }
    $run = '{0}-{1}-{2}' -f (Get-Date -Format 'yyyyMMdd-HHmmss'), $Transport.ToLowerInvariant(), $Profile
    $log = Join-Path $output ($run + '.log')
    $timeout = ($DurationSeconds + 180).ToString() + 's'
    # Keep the full trace local; publish only the sanitized metric record.
    if ($env:OS -eq 'Windows_NT') {
        ./scripts/build-loopback.ps1
    }
    go test ./internal/desktop -run $test -timeout $timeout -count 1 -v *> $log
    $code = $LASTEXITCODE
    $row = Select-String -Path $log -Pattern 'SCREEN_SOAK_METRICS (\{.+\})' | Select-Object -Last 1
    if ($row) {
        $metrics = $row.Matches[0].Groups[1].Value | ConvertFrom-Json
        $metrics | Add-Member -NotePropertyName 'configuredBitrateBps' -NotePropertyValue $metadata.Bitrate
        $metrics | Add-Member -NotePropertyName 'testPassed' -NotePropertyValue ($code -eq 0)
        $metrics | ConvertTo-Json -Depth 32 | Set-Content -Encoding utf8 (Join-Path $output ($run + '.json'))
    }
    Get-Content -LiteralPath $log -Tail 6
    if ($code) { throw "Teste não concluído. Consulte $log" }
    if (!$row) { throw "O teste terminou sem medir a sessão. Consulte $log" }
} finally {
    foreach ($key in $keys) {
        [Environment]::SetEnvironmentVariable($key, $previous[$key], 'Process')
    }
    Pop-Location
}
