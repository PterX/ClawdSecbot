#Requires -Version 5.1
param(
    [int]$PprofPort = 0,
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$GoLibDir = Join-Path $ProjectRoot "go_lib"
$PluginsDir = Join-Path $ProjectRoot "plugins"
$OutputName = "botsec"
$DllName = "$OutputName.dll"

if ($PprofPort -le 0) {
    if ($env:BOTSEC_PPROF_PORT) {
        $parsed = 0
        if ([int]::TryParse($env:BOTSEC_PPROF_PORT, [ref]$parsed) -and $parsed -gt 0) {
            $PprofPort = $parsed
        } else {
            $PprofPort = 6060
        }
    } else {
        $PprofPort = 6060
    }
}

Set-Location $ProjectRoot

function Stop-RepoRuntimeProcesses {
    $targets = Get-CimInstance Win32_Process -ErrorAction SilentlyContinue | Where-Object {
        $_.Name -match 'bot_sec_manager|flutter|dart|dartaotruntime' -and
        $_.CommandLine -like "*$ProjectRoot*"
    }
    foreach ($p in $targets) {
        try {
            Stop-Process -Id $p.ProcessId -Force -ErrorAction SilentlyContinue
        } catch {}
    }
    if ($targets) {
        Start-Sleep -Milliseconds 600
    }
}

function Resolve-BuiltDll {
    $candidates = @(
        (Join-Path $GoLibDir "lib$DllName"),
        (Join-Path $GoLibDir $DllName)
    )
    foreach ($c in $candidates) {
        if (Test-Path $c) { return $c }
    }
    return $null
}

function Invoke-FlutterPubGetWithFallback {
    $oldStorage = $env:FLUTTER_STORAGE_BASE_URL
    $oldPub = $env:PUB_HOSTED_URL
    $oldGit = $env:FLUTTER_GIT_URL

    try {
        # Primary: official overseas sources
        $env:FLUTTER_STORAGE_BASE_URL = "https://storage.googleapis.com"
        $env:PUB_HOSTED_URL = "https://pub.dev"
        if (Test-Path Env:\FLUTTER_GIT_URL) {
            Remove-Item Env:\FLUTTER_GIT_URL -ErrorAction SilentlyContinue
        }
        & flutter pub get
        if ($LASTEXITCODE -eq 0) {
            Write-Host "flutter pub get succeeded with official sources." -ForegroundColor Green
            return
        }

        Write-Host "flutter pub get failed with official sources, retrying China mirrors..." -ForegroundColor Yellow

        # Fallback: China mirrors
        $env:FLUTTER_STORAGE_BASE_URL = "https://storage.flutter-io.cn"
        $env:PUB_HOSTED_URL = "https://pub.flutter-io.cn"
        $env:FLUTTER_GIT_URL = "https://gitee.com/mirrors/flutter.git"
        & flutter pub get
        if ($LASTEXITCODE -ne 0) { throw "flutter pub get failed with both official and China mirrors" }
        Write-Host "flutter pub get succeeded with China mirrors." -ForegroundColor Green
    } finally {
        if ($oldStorage) { $env:FLUTTER_STORAGE_BASE_URL = $oldStorage } else { Remove-Item Env:\FLUTTER_STORAGE_BASE_URL -ErrorAction SilentlyContinue }
        if ($oldPub) { $env:PUB_HOSTED_URL = $oldPub } else { Remove-Item Env:\PUB_HOSTED_URL -ErrorAction SilentlyContinue }
        if ($oldGit) { $env:FLUTTER_GIT_URL = $oldGit } else { Remove-Item Env:\FLUTTER_GIT_URL -ErrorAction SilentlyContinue }
    }
}

function Test-NeedFlutterPubGet {
    param(
        [string]$ProjectRootPath
    )
    $packageConfig = Join-Path $ProjectRootPath ".dart_tool\package_config.json"
    if (-not (Test-Path $packageConfig)) { return $true }

    $lockFile = Join-Path $ProjectRootPath "pubspec.lock"
    if (-not (Test-Path $lockFile)) { return $true }

    $packageConfigTime = (Get-Item $packageConfig).LastWriteTimeUtc
    $lockTime = (Get-Item $lockFile).LastWriteTimeUtc
    if ($packageConfigTime -lt $lockTime) { return $true }

    $overrides = Join-Path $ProjectRootPath "pubspec_overrides.yaml"
    if (Test-Path $overrides) {
        $overrideTime = (Get-Item $overrides).LastWriteTimeUtc
        if ($packageConfigTime -lt $overrideTime) { return $true }
    }

    return $false
}

Write-Host "============================================" -ForegroundColor White
Write-Host "  BotSecManager - pprof mode (Windows)" -ForegroundColor White
Write-Host "============================================" -ForegroundColor White
Write-Host ""

if (-not $SkipBuild) {
    Write-Host "[1/2] Building Go plugin..." -ForegroundColor Cyan
    if (-not (Test-Path $PluginsDir)) {
        New-Item -ItemType Directory -Path $PluginsDir | Out-Null
    }

    Stop-RepoRuntimeProcesses
    Push-Location $GoLibDir
    try {
        $env:CGO_ENABLED = "1"
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"
        & go build -buildvcs=false -buildmode=c-shared -o $DllName .
        if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }
    } finally {
        Pop-Location
    }

    $builtDll = Resolve-BuiltDll
    if (-not $builtDll) {
        throw "Build output not found: expected $DllName or lib$DllName in $GoLibDir"
    }

    $destDll = Join-Path $PluginsDir $DllName
    Copy-Item -Force $builtDll $destDll
    $headerPath = [System.IO.Path]::ChangeExtension($builtDll, ".h")
    if (Test-Path $headerPath) {
        Copy-Item -Force $headerPath (Join-Path $PluginsDir "$OutputName.h")
    }
    Write-Host ("Built and copied: {0}" -f $destDll) -ForegroundColor Green
    Write-Host ""
} else {
    Write-Host "[1/2] Skipping Go plugin build (-SkipBuild)." -ForegroundColor Yellow
    Write-Host ""
}

Write-Host ("[2/2] Starting Flutter with pprof port: {0}" -f $PprofPort) -ForegroundColor Cyan
Write-Host ("pprof URL: http://127.0.0.1:{0}/debug/pprof/" -f $PprofPort) -ForegroundColor DarkGray
Write-Host "Common commands:" -ForegroundColor DarkGray
Write-Host ("  go tool pprof http://127.0.0.1:{0}/debug/pprof/heap" -f $PprofPort) -ForegroundColor DarkGray
Write-Host ("  go tool pprof ""http://127.0.0.1:{0}/debug/pprof/profile?seconds=30""" -f $PprofPort) -ForegroundColor DarkGray
Write-Host ("  go tool pprof http://127.0.0.1:{0}/debug/pprof/goroutine" -f $PprofPort) -ForegroundColor DarkGray
Write-Host ""
Write-Host "============================================" -ForegroundColor White
Write-Host ""

if (Test-NeedFlutterPubGet -ProjectRootPath $ProjectRoot) {
    Write-Host "Flutter dependencies are missing or stale. Running flutter pub get..." -ForegroundColor Yellow
    Invoke-FlutterPubGetWithFallback
} else {
    Write-Host "Flutter dependencies are up-to-date. Skipping flutter pub get." -ForegroundColor DarkGray
}

$env:BOTSEC_PPROF_PORT = "$PprofPort"
$env:FLUTTER_STORAGE_BASE_URL = "https://storage.flutter-io.cn"
$env:PUB_HOSTED_URL = "https://pub.flutter-io.cn"
$env:FLUTTER_GIT_URL = "https://gitee.com/mirrors/flutter.git"
$env:FLUTTER_ALREADY_LOCKED = "true"

& flutter run -d windows --no-pub
exit $LASTEXITCODE
