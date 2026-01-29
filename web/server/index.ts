import { SQL } from "bun";
import index from "../public/index.html";
import { SignJWT, jwtVerify } from "jose";

// TODO: This probably needs to be more secure.
const JWT_SECRET = new TextEncoder().encode(Bun.randomUUIDv7().toString());

const DB_USERNAME = process.env["SYRE_DB_USERNAME"];
const DB_PASSWORD = process.env["SYRE_DB_PASSWORD"];
if (DB_USERNAME === undefined) {
  throw new Error("SYRE_DB_USERNAME env var is not set");
}
if (DB_PASSWORD === undefined) {
  throw new Error("SYRE_DB_PASSWORD env var is not set");
}

const DB_URL = process.env["SYRE_DB_URL"] ?? "localhost:5432";
const DB_NAME = process.env["SYRE_DB_URL"] ?? "syredb";
const pg = new SQL(
  `postgres://${DB_USERNAME}:${DB_PASSWORD}@${DB_URL}/${DB_NAME}`,
);

const server = Bun.serve({
  routes: {
    "/": index,
    "/api/login": { POST: login },
  },
  development: {
    hmr: true,
    console: true,
  },
});

console.log(`listening on ${server.url.href}`);

async function login(req: Request) {
  const data = await req.formData();
  const email = data.get("email");
  const password = data.get("password");
  if (!email || !password) {
    return new Response("", { status: 401 });
  }

  const user_rows =
    await pg`SELECT _id FROM user_ WHERE email=${email} AND account_status='active'`.values();
  if (user_rows.length === 0) {
    return new Response("", { status: 401 });
  }
  if (user_rows.length > 1) {
    console.error(`multiple user records with email ${email}`);
    return new Response("", { status: 401 });
  }
  const user_id = user_rows[0][0];

  const auth_rows =
    await pg`SELECT auth FROM user_auth_ WHERE _id=${user_id}`.values();
  if (auth_rows.length !== 1) {
    console.error(`invalid user auth for user ${user_id}`);
    return new Response("", { status: 401 });
  }
  const hash = auth_rows[0][0];

  const valid_credentials = await Bun.password.verify(
    password.toString(),
    hash,
  );
  if (valid_credentials) {
    const token = await new SignJWT({ user_id: user_id, email: email })
      .setProtectedHeader({ alg: "HS256" })
      .setExpirationTime("24h")
      .sign(JWT_SECRET);

    return new Response("OK");
  } else {
    return new Response("", { status: 401 });
  }
}
