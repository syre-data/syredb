$DB_NAME = "syredb"
$CMD_SET_DB_OWNER = "db-owner"
$CMD_SET_APP_EMAIL = "app-email"
$CMD_SET_ACCOUNT_INFO = "account-info"
$CMD_SET_DATA = "data"
$APP_EMAIL_URL_KEY = "app:email:url"
$APP_EMAIL_USERNAME_KEY = "app:email:username"
$APP_EMAIL_PASSWORD_KEY = "app:email:password"
$APP_EMAIL_FROM_KEY = "app:email:from"
$APP_ACCOUNT_NAME = "app:account:name"
$APP_ACCOUNT_LOGO = "app:account:logo"
$APP_DATA_PATH = "app:data:path"

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

$ownerExists = psql -U $pgUser -d $DB_NAME -tAc `
    "SELECT 1 
    FROM db_user_permission_ as p
    JOIN user_ as u
    ON p._user=u._id
    WHERE p._permission='owner' AND u.account_status='active'"
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

$appAccountNameExists = psql -U $pgUser -d $DB_NAME -tAc "SELECT 1 FROM _app_data_ WHERE key='$APP_ACCOUNT_NAME'"
$appAccountLogoExists = psql -U $pgUser -d $DB_NAME -tAc "SELECT 1 FROM _app_data_ WHERE key='$APP_ACCOUNT_LOGO'"
if (-not ($appAccountNameExists || $appAccountLogoExists)) {
    Write-Host "Initializing $DB_NAME account settings"
    $appAccountName = Read-Host -Prompt "Account name"
    $appAccountLogo = Read-Host -Prompt "Account logo path"

    .\init_db\init_syredb.exe `
        --cmd $CMD_SET_ACCOUNT_INFO `
        --pg-user $pgUser `
        --pg-password $pgPasswordPlainText `
        --account-name $appAccountName `
        --account-logo $appAccountLogo `

    if ($LASTEXITCODE -eq 1) {
        Write-Error "Invalid command $CMD_SET_DB_OWNER"
        exit 1
    }
    if ($LASTEXITCODE -eq 2) {
        Write-Error "Could not connect to database"
        exit 1
    }
    if ($LASTEXITCODE -eq 30) {
        Write-Error "Could not set account info"
        exit 1
    }

    Write-Host "$DB_NAME account settings initialized"
}

$appDataPathExists = psql -U $pgUser -d $DB_NAME -tAc "SELECT 1 FROM _app_data_ WHERE key='$APP_DATA_PATH'"
if (-not ($appDataPathExists)) {
    Write-Host "Initializing $DB_NAME data settings"
    $appDataPath = Read-Host -Prompt "Data path"
    .\init_db\init_syredb.exe `
        --cmd $CMD_SET_DATA `
        --pg-user $pgUser `
        --pg-password $pgPasswordPlainText `
        --data-path $appDataPath `

    Write-Host "$DB_NAME data settings initialized"
    if ($LASTEXITCODE -eq 1) {
        Write-Error "Invalid command $CMD_SET_DB_OWNER"
        exit 1
    }
    if ($LASTEXITCODE -eq 2) {
        Write-Error "Could not connect to database"
        exit 1
    }
    if ($LASTEXITCODE -eq 40) {
        Write-Error "Could not set data settings"
        exit 1
    }
}

$env:PGPASSWORD = $pgpassword_o
Write-Host "$DB_NAME initialized"
