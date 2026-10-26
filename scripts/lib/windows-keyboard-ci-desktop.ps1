# Merely loading this file does not open or change native objects. The mutation
# entry point below is restricted to the disposable hosted runner.
if (-not ('Arion.KeyboardCI.DesktopAccess' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Security.AccessControl;
using System.Security.Principal;
using System.Text;

namespace Arion.KeyboardCI {
    public sealed class DesktopAccess : IDisposable {
        // No desktop creation, journal hooks, shutdown, or desktop switching.
        public const int StationRights = 0x327;
        public const int DesktopRights = 0xcf;
        private const uint DaclInformation = 4, SecurityAccess = 0x60000;
        private IntPtr station, desktop;
        private byte[] stationOriginal, desktopOriginal;
        private bool stationChanged, desktopChanged;
        public string StationName { get; private set; }
        public string DesktopName { get; private set; }

        [DllImport("user32.dll")] private static extern IntPtr GetProcessWindowStation();
        [DllImport("user32.dll")] private static extern IntPtr GetThreadDesktop(uint thread);
        [DllImport("kernel32.dll")] private static extern uint GetCurrentThreadId();
        [DllImport("user32.dll", CharSet=CharSet.Unicode, SetLastError=true)] private static extern bool GetUserObjectInformation(IntPtr handle, int index, StringBuilder value, uint length, out uint needed);
        [DllImport("user32.dll", CharSet=CharSet.Unicode, SetLastError=true)] private static extern IntPtr OpenWindowStation(string name, bool inherit, uint access);
        [DllImport("user32.dll", CharSet=CharSet.Unicode, SetLastError=true)] private static extern IntPtr OpenDesktop(string name, uint flags, bool inherit, uint access);
        [DllImport("user32.dll", SetLastError=true)] private static extern bool CloseWindowStation(IntPtr handle);
        [DllImport("user32.dll", SetLastError=true)] private static extern bool CloseDesktop(IntPtr handle);
        [DllImport("user32.dll", SetLastError=true)] private static extern bool GetUserObjectSecurity(IntPtr handle, ref uint information, byte[] value, uint length, out uint needed);
        [DllImport("user32.dll", SetLastError=true)] private static extern bool SetUserObjectSecurity(IntPtr handle, ref uint information, byte[] value);

        private static string Name(IntPtr handle) {
            uint needed;
            GetUserObjectInformation(handle, 2, null, 0, out needed);
            if (needed == 0) throw new Win32Exception();
            var value = new StringBuilder(checked((int)needed / 2));
            if (!GetUserObjectInformation(handle, 2, value, needed, out needed)) throw new Win32Exception();
            return value.ToString();
        }
        private static byte[] Read(IntPtr handle) {
            uint information = DaclInformation, needed;
            GetUserObjectSecurity(handle, ref information, null, 0, out needed);
            if (needed == 0) throw new Win32Exception();
            var result = new byte[needed];
            if (!GetUserObjectSecurity(handle, ref information, result, needed, out needed)) throw new Win32Exception();
            return result;
        }
        private static void Write(IntPtr handle, byte[] value) {
            uint information = DaclInformation;
            if (!SetUserObjectSecurity(handle, ref information, value)) throw new Win32Exception();
        }
        public static byte[] GrantDescriptor(byte[] original, string sid, int rights) {
            var descriptor = new RawSecurityDescriptor(original, 0);
            // A null DACL already grants access; never narrow it as a side effect.
            if (descriptor.DiscretionaryAcl == null) return (byte[])original.Clone();
            var acl = descriptor.DiscretionaryAcl;
            acl.InsertAce(acl.Count, new CommonAce(AceFlags.None, AceQualifier.AccessAllowed, rights, new SecurityIdentifier(sid), false, null));
            var result = new byte[descriptor.BinaryLength];
            descriptor.GetBinaryForm(result, 0);
            return result;
        }
        public static bool DaclEquals(byte[] first, byte[] second) {
            var a = new RawSecurityDescriptor(first, 0).DiscretionaryAcl;
            var b = new RawSecurityDescriptor(second, 0).DiscretionaryAcl;
            if (a == null || b == null) return a == null && b == null;
            if (a.BinaryLength != b.BinaryLength) return false;
            var x = new byte[a.BinaryLength]; var y = new byte[b.BinaryLength];
            a.GetBinaryForm(x, 0); b.GetBinaryForm(y, 0);
            for (int i = 0; i < x.Length; ++i) if (x[i] != y[i]) return false;
            return true;
        }
        public DesktopAccess(string sid) {
            // Defense in depth: callers cannot bypass the PowerShell guard.
            if (Environment.GetEnvironmentVariable("GITHUB_ACTIONS") != "true" || Environment.GetEnvironmentVariable("RUNNER_ENVIRONMENT") != "github-hosted")
                throw new InvalidOperationException("Desktop grants require a disposable GitHub-hosted runner.");
            try {
                StationName = Name(GetProcessWindowStation());
                DesktopName = Name(GetThreadDesktop(GetCurrentThreadId()));
                station = OpenWindowStation(StationName, false, SecurityAccess);
                if (station == IntPtr.Zero) throw new Win32Exception();
                desktop = OpenDesktop(DesktopName, 0, false, SecurityAccess);
                if (desktop == IntPtr.Zero) throw new Win32Exception();
                stationOriginal = Read(station); desktopOriginal = Read(desktop);
                var granted = GrantDescriptor(stationOriginal, sid, StationRights);
                stationChanged = true; Write(station, granted);
                if (!DaclEquals(Read(station), granted)) throw new InvalidOperationException("Window station DACL grant did not persist.");
                granted = GrantDescriptor(desktopOriginal, sid, DesktopRights);
                desktopChanged = true; Write(desktop, granted);
                if (!DaclEquals(Read(desktop), granted)) throw new InvalidOperationException("Desktop DACL grant did not persist.");
            } catch (Exception original) {
                try { Dispose(); } catch (Exception cleanup) { throw new AggregateException(original, cleanup); }
                throw;
            }
        }
        public void Dispose() {
            var failures = new List<Exception>();
            // Attempt every restoration even when another restoration fails.
            if (desktopChanged) try {
                Write(desktop, desktopOriginal);
                if (!DaclEquals(Read(desktop), desktopOriginal)) throw new InvalidOperationException("Desktop DACL restoration mismatch.");
                desktopChanged = false;
            } catch (Exception error) { failures.Add(error); }
            if (stationChanged) try {
                Write(station, stationOriginal);
                if (!DaclEquals(Read(station), stationOriginal)) throw new InvalidOperationException("Window station DACL restoration mismatch.");
                stationChanged = false;
            } catch (Exception error) { failures.Add(error); }
            if (desktop != IntPtr.Zero) { if (!CloseDesktop(desktop)) failures.Add(new Win32Exception()); desktop = IntPtr.Zero; }
            if (station != IntPtr.Zero) { if (!CloseWindowStation(station)) failures.Add(new Win32Exception()); station = IntPtr.Zero; }
            if (failures.Count != 0) throw new AggregateException("Could not restore keyboard CI desktop access.", failures);
        }
    }
}
'@
}

function Grant-KeyboardCIDesktopAccess([string]$SID) {
    if ($env:GITHUB_ACTIONS -ne 'true' -or $env:RUNNER_ENVIRONMENT -ne 'github-hosted') {
        throw 'Desktop access grants are restricted to disposable GitHub-hosted runners.'
    }
    [Arion.KeyboardCI.DesktopAccess]::new($SID)
}
