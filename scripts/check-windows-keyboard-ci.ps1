# Hosted Windows runners are administrators. Native acceptance must instead
# execute under a real standard user's loaded profile, including DPAPI.
[CmdletBinding()]
param([switch]$Child, [string]$Repository, [string]$Node, [string]$GoExecutable, [string]$GoVersion,
    [string]$ExpectedSID, [string]$Station, [string]$Desktop)
$ErrorActionPreference = 'Stop'

if ($Child) {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    if (!$ExpectedSID -or $identity.User.Value -ne $ExpectedSID) { throw 'Unexpected keyboard CI test identity.' }
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    if ($principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw 'Keyboard acceptance must not run as an administrator.'
    }
    $env:APPDATA = [Environment]::GetFolderPath('ApplicationData')
    $env:LOCALAPPDATA = [Environment]::GetFolderPath('LocalApplicationData')
    $env:USERPROFILE = [Environment]::GetFolderPath('UserProfile')
    if (!$env:APPDATA -or !$env:LOCALAPPDATA -or !$env:USERPROFILE) {
        throw 'The standard test user profile was not loaded.'
    }
    $env:TEMP = Join-Path $env:LOCALAPPDATA 'Temp'
    $env:TMP = $env:TEMP
    New-Item -ItemType Directory -Path $env:TEMP -Force | Out-Null
    Set-Location -LiteralPath $Repository
    # Credentialed process creation rebuilds PATH. Select the parent's installed
    # SDK explicitly instead of letting an older image SDK download a toolchain.
    if (![IO.Path]::IsPathRooted($GoExecutable) -or !(Test-Path -LiteralPath $GoExecutable -PathType Leaf)) {
        throw 'The parent must provide its installed Go executable.'
    }
    $env:GO = $GoExecutable
    $env:PATH = (Split-Path -Parent $GoExecutable) + [IO.Path]::PathSeparator + $env:PATH
    $env:GOTOOLCHAIN = 'local'
    if (!$GoVersion) { throw 'The parent must provide its actual Go SDK version.' }
    $actualGoVersion = & $GoExecutable version
    if ($LASTEXITCODE -ne 0 -or $actualGoVersion -ne $GoVersion) {
        throw "Unexpected standard-user Go toolchain: $actualGoVersion; expected $GoVersion"
    }
    $resolvedGo = (Get-Command go -CommandType Application | Select-Object -First 1).Source
    if ($resolvedGo -ne $GoExecutable) { throw 'The standard-user PATH does not select the pinned Go executable.' }
    . (Join-Path $PSScriptRoot 'lib/windows-ci-go-cache.ps1')
    $cacheSeed = Join-Path $Repository '.workspace/ci-standard-user-go-cache'
    $cache = Join-Path $Repository '.workspace/windows-standard-user-go'
    New-Item -ItemType Directory -Path $cache -Force | Out-Null
    $env:GOCACHE = Join-Path $cache 'build'
    $env:GOMODCACHE = Join-Path $cache 'mod'
    foreach ($name in @('build', 'mod')) {
        $destination = Join-Path $cache $name
        New-Item -ItemType Directory -Path $destination -Force | Out-Null
        Copy-CIGoCache (Join-Path $cacheSeed $name) $destination
        $owner = (Get-Acl -LiteralPath $destination).GetOwner([Security.Principal.SecurityIdentifier]).Value
        if ($owner -ne $ExpectedSID) { throw 'The active Go cache must be owned by the standard test user.' }
    }
    $env:XDG_CACHE_HOME = Join-Path $Repository '.workspace/native-download-cache'
    Write-Output "Standard-user Go caches: $cache (owner $ExpectedSID)"
    $testExitCode = 1
    try {
        $preflightDirectory = Join-Path $Repository '.workspace/tests/windows-keyboard-ci'
        New-Item -ItemType Directory -Path $preflightDirectory -Force | Out-Null
        @{ executable = $GoExecutable; version = $actualGoVersion; toolchain = $env:GOTOOLCHAIN;
            sha256 = (Get-FileHash -LiteralPath $GoExecutable -Algorithm SHA256).Hash } |
            ConvertTo-Json | Tee-Object -FilePath (Join-Path $preflightDirectory 'go-toolchain.json')
        $preflightSource = Join-Path $Repository 'apps/desktop/ime/windows/ci_preflight.cpp'
        $preflightBinary = Join-Path $preflightDirectory 'preflight.exe'
        $compiler = (Get-Command g++ -CommandType Application | Select-Object -First 1).Source
        $compileArguments = @('-std=c++17', '-DINITGUID', '-O2', '-Wall', '-Wextra', '-Werror', '-static', '-municode',
            $preflightSource, '-o', $preflightBinary, '-lole32', '-loleaut32', '-luuid', '-ladvapi32', '-lcrypt32', '-lgdi32')
        @{ compiler = $compiler; arguments = $compileArguments; sid = $ExpectedSID; profile = $env:USERPROFILE;
            sourceSHA256 = (Get-FileHash -LiteralPath $preflightSource -Algorithm SHA256).Hash } |
            ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $preflightDirectory 'command.json')
        & $compiler @compileArguments
        if ($LASTEXITCODE -ne 0) { throw 'Could not compile the GUI/TSF/DPAPI preflight.' }
        $preflight = & $preflightBinary $ExpectedSID $Station $Desktop
        if ($LASTEXITCODE -ne 0) { throw 'Standard-user GUI/TSF/DPAPI preflight failed.' }
        $preflight | Tee-Object -FilePath (Join-Path $preflightDirectory 'preflight.json')
        # Pipe identity, private ACL and canonical-path checks require this
        # ordinary token and its real profile temp directory, too.
        # Validate the private native ABI before the longer Go desktop suite.
        $timer = [Diagnostics.Stopwatch]::StartNew()
        & $Node (Join-Path $Repository 'scripts/check-windows-keyboard.mjs')
        if ($LASTEXITCODE -ne 0) { throw 'Standard-user native keyboard checks failed.' }
        Write-Output "Standard-user native keyboard checks: $($timer.Elapsed.TotalSeconds) seconds"
        $timer.Restart()
        & $Node (Join-Path $Repository 'scripts/check-go.mjs') desktop
        $testExitCode = $LASTEXITCODE
        Write-Output "Standard-user desktop tests and vet: $($timer.Elapsed.TotalSeconds) seconds"
        if ($testExitCode -eq 0) {
            foreach ($name in @('build', 'mod')) { Copy-CIGoCache (Join-Path $cache $name) (Join-Path $cacheSeed $name) }
            Write-Output 'Exported standard-user Go compiler and module caches without profile or ACL data.'
        }
    } finally {
        # Private DPAPI/options fixtures deny the parent runner access. Archive
        # them as their owner before deleting this account; the archive itself
        # inherits the workspace ACL and remains readable by artifact upload.
        if (Test-Path -LiteralPath (Join-Path $Repository '.workspace/tests')) {
            & (Join-Path $env:SystemRoot 'System32/tar.exe') -a -cf `
                (Join-Path $Repository '.workspace/keyboard-evidence.zip') `
                -C (Join-Path $Repository '.workspace') tests
            if ($LASTEXITCODE -ne 0) { throw 'Could not archive private keyboard test evidence.' }
        }
    }
    exit $testExitCode
}

# Never provision accounts on a developer workstation or a persistent runner.
if ($env:GITHUB_ACTIONS -ne 'true' -or $env:RUNNER_ENVIRONMENT -ne 'github-hosted') {
    throw 'This launcher is restricted to disposable GitHub-hosted Windows runners.'
}
$Repository = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
if ($Repository -ne (Resolve-Path -LiteralPath $env:GITHUB_WORKSPACE).Path) {
    throw 'The test repository must be the current GitHub workspace.'
}
& (Join-Path $PSScriptRoot 'test-windows-keyboard-ci-desktop.ps1')
$Node = (Get-Command node -CommandType Application | Select-Object -First 1).Source
$GoExecutable = (Get-Command go -CommandType Application | Select-Object -First 1).Source
$env:GOTOOLCHAIN = 'local'
$GoVersion = & $GoExecutable version
if ($LASTEXITCODE -ne 0 -or $GoVersion -notmatch '^go version go\d+\.\d+\.\d+ windows/amd64$') {
    throw "The installed native Go SDK is unavailable: $GoVersion"
}
$account = 'arion-test-' + [Guid]::NewGuid().ToString('N').Substring(0, 8)
$password = ConvertTo-SecureString ('Aa1!' + [Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))) -AsPlainText -Force
$originalACL = Get-Acl -LiteralPath $Repository
$created = $false
$desktopAccess = $null
$testProcess = $null
$failure = $null
$cleanupFailures = [Collections.Generic.List[string]]::new()
try {
    $user = New-LocalUser -Name $account -Password $password -Description 'Disposable Arion keyboard CI test user'
    $created = $true
    Add-LocalGroupMember -SID 'S-1-5-32-545' -Member $user
    $acl = Get-Acl -LiteralPath $Repository
    $rule = [Security.AccessControl.FileSystemAccessRule]::new($user.SID, 'Modify', 'ContainerInherit,ObjectInherit', 'None', 'Allow')
    $acl.AddAccessRule($rule)
    Set-Acl -LiteralPath $Repository -AclObject $acl
    . (Join-Path $PSScriptRoot 'lib/windows-keyboard-ci-desktop.ps1')
    $desktopAccess = Grant-KeyboardCIDesktopAccess $user.SID.Value
    Write-Output "Temporary keyboard CI desktop: $($desktopAccess.StationName)\$($desktopAccess.DesktopName), SID $($user.SID.Value)"
    $credential = [Management.Automation.PSCredential]::new("$env:COMPUTERNAME\$account", $password)
    $arguments = '-NoProfile -File "{0}" -Child -Repository "{1}" -Node "{2}" -ExpectedSID "{3}" -Station "{4}" -Desktop "{5}" -GoExecutable "{6}" -GoVersion "{7}"' -f `
        $PSCommandPath, $Repository, $Node, $user.SID.Value, $desktopAccess.StationName, $desktopAccess.DesktopName, $GoExecutable, $GoVersion
    # Only the tool search path is forwarded; release credentials and other
    # workflow environment variables are not inherited by the test process.
    $testProcess = Start-Process -FilePath (Join-Path $PSHOME 'pwsh.exe') -ArgumentList $arguments `
        -Credential $credential -LoadUserProfile -UseNewEnvironment -Environment @{ PATH = $env:PATH } `
        -WorkingDirectory $Repository -WindowStyle Hidden -PassThru `
        -RedirectStandardOutput (Join-Path $env:RUNNER_TEMP 'keyboard-standard-user.stdout.log') `
        -RedirectStandardError (Join-Path $env:RUNNER_TEMP 'keyboard-standard-user.stderr.log')
    # Bound the entire owned suite, including a hung build, before restoring its
    # desktop permissions. Kill(true) is restricted to this spawned process tree.
    if (!$testProcess.WaitForExit(30 * 60 * 1000)) {
        $testProcess.Kill($true)
        if (!$testProcess.WaitForExit(15000)) { throw 'Timed-out keyboard CI process did not terminate.' }
        throw 'Standard-user keyboard checks exceeded 30 minutes.'
    }
    Get-Content -LiteralPath (Join-Path $env:RUNNER_TEMP 'keyboard-standard-user.stdout.log')
    Get-Content -LiteralPath (Join-Path $env:RUNNER_TEMP 'keyboard-standard-user.stderr.log')
    if ($testProcess.ExitCode -ne 0) { throw "Standard-user keyboard checks failed ($($testProcess.ExitCode))." }
} catch {
    $failure = $_
} finally {
    if ($testProcess -and !$testProcess.HasExited) {
        try { $testProcess.Kill($true); if (!$testProcess.WaitForExit(15000)) { throw 'Keyboard CI process is still running.' } }
        catch { $cleanupFailures.Add("Child cleanup: $_") }
    }
    if ($desktopAccess) {
        try { $desktopAccess.Dispose(); Write-Output 'Original window station and desktop DACLs restored and verified.' }
        catch { $cleanupFailures.Add("Desktop DACL cleanup: $_") }
    }
    if ($created) {
        try { Set-Acl -LiteralPath $Repository -AclObject $originalACL }
        catch { $cleanupFailures.Add("Workspace ACL cleanup: $_") }
        try {
            # The SID belongs to this invocation, not a pre-existing account.
            Remove-LocalUser -SID $user.SID
            if (Get-LocalUser -SID $user.SID -ErrorAction SilentlyContinue) { throw 'Test account still exists.' }
            Write-Output "Disposable keyboard CI account removed: $($user.SID.Value)"
        } catch { $cleanupFailures.Add("Account cleanup: $_") }
    }
    $password.Dispose()
}
if ($cleanupFailures.Count) { throw "Keyboard CI cleanup failed: $($cleanupFailures -join '; '). Original failure: $failure" }
if ($failure) { throw $failure }
