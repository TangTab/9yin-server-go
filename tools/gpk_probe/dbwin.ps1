# DBWIN 捕获 GPKitSvrD.dll(Debug 版)的 OutputDebugStringA 诊断
$ErrorActionPreference = 'Continue'
Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;
public delegate int DInit(IntPtr t, IntPtr a, IntPtr b);
public class W {
  [DllImport("kernel32", SetLastError=true, CharSet=CharSet.Ansi)] public static extern IntPtr CreateFileMappingA(IntPtr hFile, IntPtr sa, uint prot, uint maxHi, uint maxLo, string name);
  [DllImport("kernel32", SetLastError=true)] public static extern IntPtr MapViewOfFile(IntPtr hMap, uint access, uint offHi, uint offLo, UIntPtr bytes);
  [DllImport("kernel32", SetLastError=true, CharSet=CharSet.Ansi)] public static extern IntPtr CreateEventA(IntPtr sa, bool manual, bool init, string name);
  [DllImport("kernel32", SetLastError=true)] public static extern uint WaitForSingleObject(IntPtr h, uint ms);
  [DllImport("GPKitSvrD.dll")] public static extern IntPtr GPKCreateSvrDynCode();
  public static IntPtr P(string s){ byte[] b=System.Text.Encoding.ASCII.GetBytes(s); IntPtr p=Marshal.AllocHGlobal(b.Length+1); Marshal.Copy(b,0,p,b.Length); return p; }
}'
# DBWIN buffer: [0]=pid, [4..]=string
$map = [W]::CreateFileMappingA([IntPtr](-1), [IntPtr]::Zero, 0x4, 0, 4096, "DBWIN_BUFFER")
$buf = [W]::MapViewOfFile($map, 0xF001F, 0, 0, [UIntPtr][uint64]4096)
$evReady = [W]::CreateEventA([IntPtr]::Zero, $false, $false, "DBWIN_BUFFER_READY")
$evData = [W]::CreateEventA([IntPtr]::Zero, $false, $false, "DBWIN_DATA_READY")
Write-Host ("dbwin map={0:X} buf={1:X} ready={2:X} data={3:X}" -f $map, $buf, $evReady, $evData)
# signal buffer ready

Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;
public class N { [DllImport("kernel32")] public static extern bool SetEvent(IntPtr h); }'
[N]::SetEvent($evReady) | Out-Null

$obj = [W]::GPKCreateSvrDynCode()
$vt = [System.Runtime.InteropServices.Marshal]::ReadInt32($obj)
$dInit = [DInit][System.Runtime.InteropServices.Marshal]::GetDelegateForFunctionPointer([IntPtr][System.Runtime.InteropServices.Marshal]::ReadInt32([IntPtr]$vt, 4), [DInit])
$sw = [Diagnostics.Stopwatch]::StartNew()
$r = $dInit.Invoke($obj, [W]::P('DynCodeBin\Client'), [W]::P('DynCodeBin\Server'))
Write-Host ("init ret = " + $r)
while ($sw.ElapsedMilliseconds -lt 1200) {
    if ([W]::WaitForSingleObject($evData, 100) -eq 0) {
        $bytes = New-Object byte[] 512
        [System.Runtime.InteropServices.Marshal]::Copy($buf, $bytes, 0, 512)
        $txt = [System.Text.Encoding]::ASCII.GetString($bytes, 4, 400).Split([char]0)[0]
        $pid2 = [BitConverter]::ToInt32($bytes, 0)
        Write-Host ("[dbg pid=$pid2] $txt")
        [N]::SetEvent($evReady) | Out-Null
    }
}
