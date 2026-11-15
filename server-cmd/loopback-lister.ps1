param(
    [int]$Port = 19060,
    # 绑定 IP：IPv4/IPv6 字面量或通配（*、any、0.0.0.0、::）；方括号可省。
    [string]$BindIP = '127.0.0.1',
    # svrlist 通告的游戏服端口（与游戏服 -listen 的端口保持一致）。
    [int]$GamePort = 19061,
    # svrlist 通告的游戏服地址：留空 = BindIP 为具体 IP 时通告它本身，
    # 通配时自动挑本机首个非回环 IPv4（客户端拿到的是具体地址，不是通配）。
    [string]$AdvertiseHost = '',
    [string]$OutputDir = ''
)

$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($OutputDir)) {
    $OutputDir = Join-Path $PSScriptRoot 'logs\lister'
}
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

# 解析绑定 IP -> TcpListener 地址。::/any/* 绑 IPv6Any 双栈（IPv4 以
# v4-mapped 接入，一个监听同时收两种连接）；0.0.0.0 仅 IPv4 全接口。
$bind = $BindIP.Trim().Trim('[]')
$listenIP = if ($bind -eq '*' -or $bind -ieq 'any' -or $bind -eq '::' -or $bind -eq '') {
    [System.Net.IPAddress]::IPv6Any
} elseif ($bind -eq '0.0.0.0') {
    [System.Net.IPAddress]::Any
} else {
    [System.Net.IPAddress]::Parse($bind)
}

# 通告地址：通配绑定时挑本机首个非回环、非链路本地 IPv4，回退 127.0.0.1。
$advertise = $AdvertiseHost.Trim().Trim('[]')
if ($advertise -eq '' -or $advertise -eq '*' -or $advertise -ieq 'any' -or $advertise -eq '::' -or $advertise -eq '0.0.0.0') {
    $advertise = $bind
    if ($advertise -eq '' -or $advertise -eq '*' -or $advertise -ieq 'any' -or $advertise -eq '::' -or $advertise -eq '0.0.0.0') {
        $lan = Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue |
            Where-Object { $_.IPAddress -ne '127.0.0.1' -and $_.IPAddress -notlike '169.254*' } |
            Select-Object -First 1 -ExpandProperty IPAddress
        if ($lan) { $advertise = $lan } else { $advertise = '127.0.0.1' }
    }
}

$listener = [System.Net.Sockets.TcpListener]::new($listenIP, $Port)
$listener.Start()
Write-Host "svrlist listening on $BindIP`:$Port -> game $advertise`:$GamePort"
try {
    while ($true) {
        $client = $listener.AcceptTcpClient()
        try {
            $stream = $client.GetStream()
            $buffer = New-Object byte[] 4096
            $capture = [System.Collections.Generic.List[byte]]::new()
            do {
                $count = $stream.Read($buffer, 0, $buffer.Length)
                if ($count -le 0) { break }
                for ($i = 0; $i -lt $count; $i++) { $capture.Add($buffer[$i]) }
                $seenTerminator = $false
                foreach ($b in $capture) {
                    if ($b -eq 10 -or $b -eq 13) { $seenTerminator = $true; break }
                }
            } while (-not $seenTerminator)

            [System.IO.File]::WriteAllBytes((Join-Path $OutputDir 'request.bin'), $capture.ToArray())
            $responseText = '$svrlist #\u672C\u5730\u6D4B\u8BD5 $' + $advertise + ' ' + $GamePort + ' 0' + "`r`n"
            $response = [System.Text.Encoding]::ASCII.GetBytes($responseText)
            [System.IO.File]::WriteAllBytes((Join-Path $OutputDir 'response.bin'), $response)
            $stream.Write($response, 0, $response.Length)
            $stream.Flush()
        } finally {
            $client.Dispose()
        }
    }
} finally {
    $listener.Stop()
}
