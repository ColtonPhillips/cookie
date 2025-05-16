param (
    [Parameter(Mandatory = $true)]
    [string[]]$Scripts
)

foreach ($file in $Scripts) {
    if (-not (Test-Path $file)) {
        Write-Warning "File not found: $file"
        continue
    }

    $absolutePath = Resolve-Path $file
    $workingDir = Split-Path $absolutePath -Parent
    $fileName = Split-Path $absolutePath -Leaf
    $extension = [System.IO.Path]::GetExtension($fileName).ToLowerInvariant()

    switch ($extension) {
        ".go" {
            $command = @"
cd "$workingDir"
Write-Host 'Running: go run "$fileName"'
go run "$fileName"
"@

            Write-Host "Launching Go file: $fileName (in $workingDir)"
            Start-Process powershell -ArgumentList "-NoExit", "-Command", $command `
                -WorkingDirectory $workingDir `
                -WindowStyle Normal
        }

        ".ps1" {
            $command = @"
cd "$workingDir"
Write-Host 'Running: powershell -ExecutionPolicy Bypass -File "$fileName"'
.\$fileName
"@

            Write-Host "Launching PowerShell script: $fileName (in $workingDir)"
            Start-Process powershell -ArgumentList "-NoExit", "-ExecutionPolicy", "Bypass", "-Command", $command `
                -WorkingDirectory $workingDir `
                -WindowStyle Normal
        }

        default {
            Write-Warning "Unsupported file type: $file"
        }
    }
}
