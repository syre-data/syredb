# SyreDB server

## Development

### Tailwind

Run `npx @tailwindcss/cli -i ./main.css -o ./public/main.css --watch`

### Server

Use [`air`](https://github.com/air-verse/air) to watch for file changes.
Run `air -- [flags]` from the server root.
e.g.

```sh
air -- --db-username="postgres" --db-password="root" --db-host="localhost:5432" --db-name="syredb"
```

## Production build

The entire app is built into a single executable that includes
+ The Go `echo` server
+ Vendored React frontend

1. Build the React app by running `npm run build` from the `frontend` directory.
2. Copy the contents in `frontend/dist` to `server/dist/frontend`.
3. Build the `echo` server by running `go build -o bin/syredb[.exe]` (where the `.exe` is included for Windows executables)
4. Run the app with `<path>/<to>/syredb[.exe] --db-username=<db user> --db-password=<db password> --db-host=<db address> --db-name=<db name>`.