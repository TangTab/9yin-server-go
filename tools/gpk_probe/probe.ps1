# GPKitSvr.dll 动态行为探针(32 位 PowerShell + P/Invoke,载荷为自制 ret 桩)
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

$src = @'
using System;
using System.Runtime.InteropServices;

public delegate int DInit(IntPtr t, IntPtr a, IntPtr b);          // slot1 ret 0xc
public delegate int DRand(IntPtr t);                               // slot2 ret 4
public delegate int DGet(IntPtr t, uint packed, IntPtr outp);      // slot3 ret 0xc
public delegate int DCall(IntPtr t, uint a, uint b, uint codeid);  // slot4/5 ret 0x10

public class G {
    [DllImport("GPKitSvr.dll", CallingConvention = CallingConvention.StdCall)]
    public static extern IntPtr GPKCreateSvrDynCode();

    public static Delegate Del(IntPtr vtbl, int slot, Type t) {
        IntPtr fn = (IntPtr)Marshal.ReadInt32(vtbl, slot * 4);
        return Marshal.GetDelegateForFunctionPointer(fn, t);
    }
    public static IntPtr Ptr(byte[] b) {
        IntPtr p = Marshal.AllocHGlobal(b.Length + 1);
        Marshal.Copy(b, 0, p, b.Length);
        Marshal.WriteByte(p, b.Length, 0);
        return p;
    }
    public static string ObjDump(IntPtr obj, string tag) {
        byte[] raw = new byte[0x414];
        Marshal.Copy(obj, raw, 0, 0x414);
        int flag = raw[4];
        int n = BitConverter.ToUInt16(raw, 0x3F0);
        int cnt = BitConverter.ToUInt16(raw, 0x3F2);
        var used = new System.Text.StringBuilder();
        for (int i = 0; i < (n > 0 ? n : 0) && i < 252; i++) {
            int s = BitConverter.ToInt32(raw, 8 + i * 4);
            if (s != 0) used.AppendFormat("[{0}]={1:X} ", i, s);
        }
        return string.Format("[{0}] flag={1} n@3f0={2} cnt@3f2={3} slots={4}", tag, flag, n, cnt, used);
    }
}
'@
Add-Type -TypeDefinition $src

$obj = [G]::GPKCreateSvrDynCode()
if ($obj -eq [IntPtr]::Zero) { throw "create failed" }
$vtbl = [IntPtr]([System.Runtime.InteropServices.Marshal]::ReadInt32($obj))
Write-Host ("obj={0:X} vtbl={1:X}" -f $obj, $vtbl)
Write-Host ([G]::ObjDump($obj, "before"))

$dInit = [DInit]([G]::Del($vtbl, 1, [DInit]))
$dRand = [DRand]([G]::Del($vtbl, 2, [DRand]))
$dGet  = [DGet]([G]::Del($vtbl, 3, [DGet]))
$dC1   = [DCall]([G]::Del($vtbl, 4, [DCall]))
$dC2   = [DCall]([G]::Del($vtbl, 5, [DCall]))

$pSrv = [G]::Ptr([System.Text.Encoding]::ASCII.GetBytes("DynCodeBin\Server"))
$pClt = [G]::Ptr([System.Text.Encoding]::ASCII.GetBytes("DynCodeBin\Client"))
$r = $dInit.Invoke($obj, $pClt, $pSrv)
Write-Host ("slot1(Init Server,Client) ret={0}" -f $r)
Write-Host ([G]::ObjDump($obj, "after"))

if ($r -ge 0) {
    $idx = $dRand.Invoke($obj)
    Write-Host ("slot2(RandomPick) idx={0}" -f $idx)
    $out = [System.Runtime.InteropServices.Marshal]::AllocHGlobal(4)
    $packed = [uint32]($idx -shl 16)
    $sz = $dGet.Invoke($obj, $packed, $out)
    $ptr = [System.Runtime.InteropServices.Marshal]::ReadInt32($out)
    Write-Host ("slot3(GetData idx) ret={0} outPtr={1:X}" -f $sz, $ptr)
    if ($ptr -ne 0 -and $sz -gt 0) {
        $data = New-Object byte[] ([Math]::Min($sz, 64))
        [System.Runtime.InteropServices.Marshal]::Copy([IntPtr]$ptr, $data, 0, $data.Length)
        Write-Host ("  data[0..15]=" + (($data[0..15] | ForEach-Object { $_.ToString('X2') }) -join ' '))
    }
    $a = $dC1.Invoke($obj, [uint32]0xDEAD0001, [uint32]0xDEAD0002, [uint32]$idx)
    Write-Host ("slot4(Call1 a1,a2,idx) ret={0:X8}" -f $a)
    $s = [G]::Ptr([System.Text.Encoding]::ASCII.GetBytes("HELLO-GPK"))
    $b = $dC2.Invoke($obj, [uint32]0, [uint32]$s, [uint32]$idx)
    Write-Host ("slot5(Call2 0,str,idx) ret={0:X8}" -f $b)
}
