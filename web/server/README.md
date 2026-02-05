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
