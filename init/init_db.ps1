$DB_NAME = "syredb"
$CMD_SET_DB_OWNER = "db-owner"
$CMD_SET_APP_EMAIL = "app-email"
$APP_EMAIL_URL_KEY = "app:email:url"
$APP_EMAIL_USERNAME_KEY = "app:email:username"
$APP_EMAIL_PASSWORD_KEY = "app:email:password"
$APP_EMAIL_FROM_KEY = "app:email:from"

Write-Host "Initializing Postgres database $DB_NAME"
$pgUser = Read-Host -Prompt "Postgres user"
$pgPassword = Read-Host -Prompt "Password" -AsSecureString

$pgPasswordPlainText = ConvertFrom-SecureString -SecureString $pgPassword -AsPlainText
$pgpassword_o = $env:PGPASSWORD
$env:PGPASSWORD = $pgPasswordPlainText

$databaseExists = psql -U $pgUser -tAc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'"
if (-not $databaseExists) {
    psql -U $pgUser --command="CREATE DATABASE $DB_NAME"
}

psql -U $pgUser -d $DB_NAME -f syredb.sql
Write-Host "Postgres database initialized"

# ---

$ownerExists = psql -U $pgUser -d $DB_NAME -tAc "SELECT 1 FROM user_ WHERE role='owner'"
if (-not $ownerExists) {
    Write-Host "Initializing $DB_NAME owner"
    $userEmail = Read-Host -Prompt "Email"
    $userName = Read-Host -Prompt "Name"
    $userPassword = Read-Host -Prompt "Password" -AsSecureString
    $userPasswordPlainText = ConvertFrom-SecureString -SecureString $userPassword -AsPlainText

    .\init_db\init_syredb.exe `
        --cmd $CMD_SET_DB_OWNER `
        --pg-user $pgUser `
        --pg-password $pgPasswordPlainText `
        --db-owner-email $userEmail `
        --db-owner-name $userName `
        --db-owner-password $userPasswordPlainText

    if ($LASTEXITCODE -eq 1) {
        Write-Error "Invalid command $CMD_SET_DB_OWNER"
        exit 1
    }
    if ($LASTEXITCODE -eq 2) {
        Write-Error "Could not connect to database"
        exit 1
    }
    if ($LASTEXITCODE -eq 10) {
        Write-Error "Invalid email"
        exit 1
    }
    if ($LASTEXITCODE -eq 11) {
        Write-Error "Could not create user"
        exit 1
    }
    Write-Host "$DB_NAME owner initialized"
}
# ---

$appEmailUrlExists = psql -U $pgUser -d $DB_NAME -tAc "SELECT 1 FROM _app_data_ WHERE key='$APP_EMAIL_URL_KEY'"
$appEmailUsernameExists = psql -U $pgUser -d $DB_NAME -tAc "SELECT 1 FROM _app_data_ WHERE key='$APP_EMAIL_USERNAME_KEY'"
$appEmailPasswordExists = psql -U $pgUser -d $DB_NAME -tAc "SELECT 1 FROM _app_data_ WHERE key='$APP_EMAIL_PASSWORD_KEY'"
$appEmailFromExists = psql -U $pgUser -d $DB_NAME -tAc "SELECT 1 FROM _app_data_ WHERE key='$APP_EMAIL_FROM_KEY'"
$appEmailSet = $appEmailUrlExists && $appEmailUsernameExists && $appEmailPasswordExists && $appEmailFromExists
if (-not $appEmailSet) {
    Write-Host "Initializing $DB_NAME email client"
    $appEmailUrl = Read-Host -Prompt "SMTP server URL"
    $appEmailUsername = Read-Host -Prompt "Username"
    $appEmailPassword = Read-Host -Prompt "Password" -AsSecureString
    $appEmailFrom = Read-Host -Prompt "From address"
    $appEmailPasswordPlainText = ConvertFrom-SecureString -SecureString $appEmailPassword -AsPlainText

    .\init_db\init_syredb.exe `
        --cmd $CMD_SET_APP_EMAIL `
        --pg-user $pgUser `
        --pg-password $pgPasswordPlainText `
        --app-email-url $appEmailUrl `
        --app-email-username $appEmailUsername `
        --app-email-password $appEmailPasswordPlainText `
        --app-email-from-address $appEmailFrom

    if ($LASTEXITCODE -eq 1) {
        Write-Error "Invalid command $CMD_SET_DB_OWNER"
        exit 1
    }
    if ($LASTEXITCODE -eq 2) {
        Write-Error "Could not connect to database"
        exit 1
    }
    if ($LASTEXITCODE -eq 20) {
        Write-Error "Invalid from address"
        exit 1
    }
    if ($LASTEXITCODE -eq 21) {
        Write-Error "Invalid email"
        exit 1
    }
    Write-Host "$DB_NAME email client initialized"
}

$env:PGPASSWORD = $pgpassword_o
