param(
    [int]$Port = 19060,
    [string]$OutputDir = ''
)

$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($OutputDir)) {
    $OutputDir = Join-Path $PSScriptRoot 'logs\lister'
}
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, $Port)
$listener.Start()
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
            $responseText = '$svrlist #\u672C\u5730\u6D4B\u8BD5 $127.0.0.1 19061 0' + "`r`n"
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
