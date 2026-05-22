$json = Get-Content -Path "C:\Users\humas\.gemini\antigravity\brain\61ffaf32-ada3-4528-a77b-ce026a7d4234\.system_generated\logs\overview.txt" -Raw
$step0 = ($json -split "`n")[0]
$obj = ConvertFrom-Json $step0
[System.IO.File]::WriteAllText("e:\nvide\bEck_NVide\scratch\step0_content.txt", $obj.content, [System.Text.Encoding]::UTF8)
Write-Host "Success in UTF8!"
