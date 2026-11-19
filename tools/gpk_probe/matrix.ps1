# x86 Init 参数序 × 文件布局矩阵
$ErrorActionPreference = 'Continue'
Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;
public delegate int DInit(IntPtr t, IntPtr a, IntPtr b);
public class G {
  [DllImport("GPKitSvr.dll")] public static extern IntPtr GPKCreateSvrDynCode();
  public static IntPtr P(string s){ byte[] b=System.Text.Encoding.ASCII.GetBytes(s); IntPtr p=Marshal.AllocHGlobal(b.Length+1); Marshal.Copy(b,0,p,b.Length); return p; }
}'

function Try-Init($pa, $pb) {
    $obj = [G]::GPKCreateSvrDynCode()
    $vt = [System.Runtime.InteropServices.Marshal]::ReadInt32($obj)
    $dInit = [DInit][System.Runtime.InteropServices.Marshal]::GetDelegateForFunctionPointer([IntPtr][System.Runtime.InteropServices.Marshal]::ReadInt32([IntPtr]$vt, 4), [DInit])
    $r = $dInit.Invoke($obj, [G]::P($pa), [G]::P($pb))
    $raw = New-Object byte[] 0x414
    [System.Runtime.InteropServices.Marshal]::Copy($obj, $raw, 0, 0x414)
    $mc = [BitConverter]::ToInt32($raw, 8)
    $cnt = -1; $detail = ""
    if ($mc -ne 0) {
        $mr = New-Object byte[] 0x414
        [System.Runtime.InteropServices.Marshal]::Copy([IntPtr]$mc, $mr, 0, 0x414)
        $cnt = [BitConverter]::ToUInt16($mr, 0x404)
        for ($i = 0; $i -lt [Math]::Min($cnt, 3); $i++) {
            $e = [BitConverter]::ToInt32($mr, 4 + $i*4)
            if ($e -ne 0) {
                $er = New-Object byte[] 0x60
                [System.Runtime.InteropServices.Marshal]::Copy([IntPtr]$e, $er, 0, 0x60)
                $q4 = [BitConverter]::ToUInt32($er, 0x10); $q5 = [BitConverter]::ToUInt32($er, 0x14)
                $nm = [System.Text.Encoding]::ASCII.GetString($er[0x18..0x57]).Split([char]0)[0]
                $detail += "  entry[$i] fn1=0x$($q4.ToString('X')) fn2=0x$($q5.ToString('X')) name=$nm`n"
            }
        }
    }
    Write-Host ("Init('{0}','{1}') ret={2} mc={3} count={4}`n{5}" -f $pa, $pb, $r, $mc, $cnt, $detail)
}

Write-Host "=== layout C: Server[TestA.dat+x86载荷, TestA.bin+载荷] Client[TestA.bin 数据] ==="
Try-Init 'DynCodeBin\Server' 'DynCodeBin\Client'
Try-Init 'DynCodeBin\Client' 'DynCodeBin\Server'
