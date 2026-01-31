# SyreDB server

## Development

### Tailwind

Run `npx @tailwindcss/cli -i ./main.css -o ./public/main.css --watch`

### Server

Use [`air`](https://github.com/air-verse/air) to watch for file changes.
Run `air -- [flags]` from the server root.
e.g.

```sh
air -- --pg-username="postgres" --pg-password="root" --pg-url="localhost:5432" --pg-dbname="syredb"
```
