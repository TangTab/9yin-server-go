# slot2-5 全链验证(x86 参数探针载荷)
$ErrorActionPreference = 'Stop'
Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;
public delegate int DInit(IntPtr t, IntPtr a, IntPtr b);
public delegate int DRnd(IntPtr t);
public delegate int DGet(IntPtr t, uint packed, IntPtr outp);
public delegate int DCall(IntPtr t, uint a, uint b, uint c);
public class G {
  [DllImport("GPKitSvr.dll")] public static extern IntPtr GPKCreateSvrDynCode();
  public static IntPtr P(string s){ byte[] b=System.Text.Encoding.ASCII.GetBytes(s); IntPtr p=Marshal.AllocHGlobal(b.Length+1); Marshal.Copy(b,0,p,b.Length); return p; }
  public static Delegate S(IntPtr vt, int i, Type t){ return Marshal.GetDelegateForFunctionPointer((IntPtr)Marshal.ReadInt32((IntPtr)vt, i*4), t); }
}'
$obj = [G]::GPKCreateSvrDynCode()
$vt = [System.Runtime.InteropServices.Marshal]::ReadInt32($obj)
$di=[DInit][G]::S($vt,1,[DInit]); $dr=[DRnd][G]::S($vt,2,[DRnd]); $dg=[DGet][G]::S($vt,3,[DGet]); $dc=[DCall][G]::S($vt,4,[DCall]); $dd=[DCall][G]::S($vt,5,[DCall])
$r = $di.Invoke($obj, [G]::P('DynCodeBin\Server'), [G]::P('DynCodeBin\Client'))
Write-Host ('init ret=' + $r)
$idx = $dr.Invoke($obj)
Write-Host ('RandomPick idx=' + $idx)
$out = [System.Runtime.InteropServices.Marshal]::AllocHGlobal(4)
$sz = $dg.Invoke($obj, [uint32]($idx -shl 16), $out)
$ptr = [System.Runtime.InteropServices.Marshal]::ReadInt32($out)
Write-Host ('GetData size=' + $sz + ' ptr=0x' + $ptr.ToString('X'))
if ($ptr -ne 0) { $b=New-Object byte[] 16; [System.Runtime.InteropServices.Marshal]::Copy([IntPtr]$ptr,$b,0,16); Write-Host ('  data: ' + (($b | ForEach-Object { $_.ToString('X2') }) -join ' ')) }
$a1 = $dc.Invoke($obj, [uint32]3721184257, [uint32]3721184258, [uint32]($idx -shl 16))
Write-Host ('slot4 Call1(a,b,codeid) ret=0x' + $a1.ToString('X8'))
$a2 = $dd.Invoke($obj, [uint32]3203386371, [uint32]3203386372, [uint32]($idx -shl 16))
Write-Host ('slot5 Call2(a,b,codeid) ret=0x' + $a2.ToString('X8'))
# 逆向验证:slot5 的 FsGame 用法 (this, outbuf, 0xc4, str)
$strp = [G]::P('hello-gpk-msg')
$a3 = $dd.Invoke($obj, [uint32]0, [uint32]$strp.ToInt64(), [uint32]($idx -shl 16))
Write-Host ('slot5 (0,str,codeid) ret=0x' + $a3.ToString('X8'))
