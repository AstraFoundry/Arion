# Pure descriptor/parser contracts only: no accounts, native handles, DACL writes
# or GUI are created by this test, including on developer machines.
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'lib/windows-keyboard-ci-desktop.ps1')
function Assert-Contract([bool]$Value, [string]$Message) { if (!$Value) { throw $Message } }
foreach ($name in @('check-windows-keyboard-ci.ps1', 'lib/windows-keyboard-ci-desktop.ps1', 'test-windows-keyboard-ci-desktop.ps1')) {
    $parseErrors = $null; $tokens = $null
    [Management.Automation.Language.Parser]::ParseFile((Join-Path $PSScriptRoot $name), [ref]$tokens, [ref]$parseErrors) | Out-Null
    Assert-Contract ($parseErrors.Count -eq 0) "PowerShell parse errors in $name : $parseErrors"
}
$sid = 'S-1-5-21-123-456-789-1001'
$descriptor = [Security.AccessControl.RawSecurityDescriptor]::new('O:BAG:SYD:P(A;;GA;;;SY)(A;;GA;;;BA)')
$original = [byte[]]::new($descriptor.BinaryLength); $descriptor.GetBinaryForm($original, 0)
$saved = [Convert]::ToBase64String($original)
foreach ($rights in @([Arion.KeyboardCI.DesktopAccess]::StationRights, [Arion.KeyboardCI.DesktopAccess]::DesktopRights)) {
    $granted = [Arion.KeyboardCI.DesktopAccess]::GrantDescriptor($original, $sid, $rights)
    Assert-Contract ([Convert]::ToBase64String($original) -eq $saved) 'Grant mutated the saved original descriptor.'
    $actual = [Security.AccessControl.RawSecurityDescriptor]::new($granted, 0)
    Assert-Contract ($actual.Owner -eq $descriptor.Owner -and $actual.Group -eq $descriptor.Group) 'Owner/group changed.'
    Assert-Contract ($actual.ControlFlags -eq $descriptor.ControlFlags) 'Descriptor control flags changed.'
    Assert-Contract ($actual.DiscretionaryAcl.Count -eq 3) 'Grant did not add exactly one ACE.'
    $ace = $actual.DiscretionaryAcl[2]
    Assert-Contract ($ace.SecurityIdentifier.Value -eq $sid -and $ace.AccessMask -eq $rights) 'Grant SID or rights mismatch.'
    Assert-Contract ($ace.AceFlags -eq 0 -and $ace.AceQualifier -eq 'AccessAllowed') 'Grant ACE type mismatch.'
    for ($i = 0; $i -lt 2; $i++) {
        $before = [byte[]]::new($descriptor.DiscretionaryAcl[$i].BinaryLength)
        $after = [byte[]]::new($actual.DiscretionaryAcl[$i].BinaryLength)
        $descriptor.DiscretionaryAcl[$i].GetBinaryForm($before, 0); $actual.DiscretionaryAcl[$i].GetBinaryForm($after, 0)
        Assert-Contract ([Convert]::ToBase64String($before) -eq [Convert]::ToBase64String($after)) 'An existing ACE changed.'
    }
    Assert-Contract (![Arion.KeyboardCI.DesktopAccess]::DaclEquals($original, $granted)) 'Grant was not detected.'
    Assert-Contract ([Arion.KeyboardCI.DesktopAccess]::DaclEquals($original, $original)) 'Original DACL comparison failed.'
}
$nullDescriptor = [Security.AccessControl.RawSecurityDescriptor]::new('O:BAG:SYD:NO_ACCESS_CONTROL')
$nullBytes = [byte[]]::new($nullDescriptor.BinaryLength); $nullDescriptor.GetBinaryForm($nullBytes, 0)
Assert-Contract ([Arion.KeyboardCI.DesktopAccess]::DaclEquals($nullBytes, [Arion.KeyboardCI.DesktopAccess]::GrantDescriptor($nullBytes, $sid, 1))) 'Null DACL was narrowed.'
if ($env:GITHUB_ACTIONS -ne 'true' -or $env:RUNNER_ENVIRONMENT -ne 'github-hosted') {
    $refused = $false
    try { Grant-KeyboardCIDesktopAccess $sid | Out-Null } catch { $refused = $true }
    Assert-Contract $refused 'Desktop mutation entry point did not refuse this local host.'
    Write-Output 'PASS local desktop mutation refusal (no native changes).'
}
Write-Output 'PASS keyboard CI parser, descriptor grants and exact DACL comparison (no native changes).'
