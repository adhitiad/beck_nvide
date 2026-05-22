$bytes = [System.IO.File]::ReadAllBytes("C:\Users\humas\.gemini\antigravity\conversations\61ffaf32-ada3-4528-a77b-ce026a7d4234.pb")
$chars = [System.Text.Encoding]::UTF8.GetChars($bytes)
$sb = New-Object System.Text.StringBuilder
$strings = New-Object System.Collections.Generic.List[string]

foreach ($c in $chars) {
    # If it is a printable char or space or tab or newline
    $val = [int]$c
    if (($val -ge 32 -and $val -le 126) -or $val -eq 10 -or $val -eq 13 -or $val -eq 9) {
        $sb.Append($c) | Out-Null
    } else {
        if ($sb.Length -gt 200) {
            $strings.Add($sb.ToString())
        }
        $sb.Clear() | Out-Null
    }
}
if ($sb.Length -gt 200) {
    $strings.Add($sb.ToString())
}

$i = 0
foreach ($s in $strings) {
    if ($s -like "*beck_nvide*" -or $s -like "*live streaming*") {
        [System.IO.File]::WriteAllText("e:\nvide\bEck_NVide\scratch\prompt_extracted_$i.txt", $s, [System.Text.Encoding]::UTF8)
        Write-Host "Extracted matching string $i of length: $($s.Length)"
        $i++
    }
}
Write-Host "Done, found $i matching strings!"
