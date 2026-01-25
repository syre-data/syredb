# SyreDB

A database for scientfiic data.

Read more at [syre.ai](https://syre.ai)

## Get started

### Setting up the database

1. Install [PostgreSQL](https://www.postgresql.org/).
2. From [PowerShell 7](https://learn.microsoft.com/en-us/powershell/scripting/install/install-powershell-on-windows), run the setup script [`init/init_db.ps1`](init/init_db.ps1).

## Development

### React Devtools

Connect [`react-devtools`](https://www.npmjs.com/package/react-devtools) by installing it with

```sh
npm install -g react-devtools
```

then running

```sh
react-devtools
```

and ensuring that

```html
<script src="http://localhost:8097"></script>
```

is included in the `<head>` of [`index.html`](frontend/index.html).
