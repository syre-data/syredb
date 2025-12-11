$dbName = "syredb"
Write-Host "Initializing Postgres database $dbName"
$pgUser = Read-Host -Prompt "Postgres user"
$pgPassword = Read-Host -Prompt "Password" -AsSecureString

$pgPasswordPlainText = ConvertFrom-SecureString -SecureString $pgPassword -AsPlainText
$env:PGPASSWORD = $pgPasswordPlainText

$databaseExists = psql -U $pgUser -tAc "SELECT 1 FROM pg_database WHERE datname = '$dbName'"
if (-not $databaseExists) {
    psql -U $pgUser --command="CREATE DATABASE $dbName"
}

psql -U $pgUser -d $dbName -f syredb.sql
Write-Host "Postgres database initialized"

Write-Host "Initializing $dbName owner"
$userEmail = Read-Host -Prompt "Email"
$userName = Read-Host -Prompt "Name"
$userPassword = Read-Host -Prompt "Password" -AsSecureString
$userPasswordPlainText = ConvertFrom-SecureString -SecureString $userPassword -AsPlainText

.\init_user\init_syredb_user.exe --pg-user $pgUser --pg-password $pgPasswordPlainText --email $userEmail --name $userName --password $userPasswordPlainText
if ($LASTEXITCODE -eq 1) {
    Write-Error "Could not connect to database"
    exit 1
}
if ($LASTEXITCODE -eq 2) {
    Write-Error "Invalid email"
    exit 1
}
if ($LASTEXITCODE -eq 3) {
    Write-Error "Could not create user"
    exit 1
}

Write-Host "$dbName owner initialized"

$env:PGPASSWORD = ""
