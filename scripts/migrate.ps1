param(
    [string]$HostName = "localhost",
    [string]$Port = "5432",
    [string]$User = "crm",
    [string]$Database = "crm"
)

$ErrorActionPreference = "Stop"

psql -h $HostName -p $Port -U $User -d $Database -v ON_ERROR_STOP=1 -f migrations/001_init.sql

