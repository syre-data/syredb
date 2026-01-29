import uuid
import psycopg
import argon2


class Connection:
    def __init__(self, db_name: str, db_user: str, db_password: str):
        conninfo = f"dbname={db_name} user={db_user} password={db_password}"
        self.conn = psycopg.connect(conninfo)

    def __del__(self):
        self.conn.close()

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        pass

    def authenticate_user(self, email: str, password: str) -> str | None:
        with self.conn.cursor() as c:
            c.execute("SELECT _id FROM user_ WHERE email=%s", (email,))
            r = c.fetchall()
            if len(r) == 0:
                return None
            if len(r) > 1:
                raise RuntimeError("Found multiple users with email")

            user_id = r[0][0]
            c.execute("SELECT auth FROM user_auth_ WHERE _id=%s", (user_id,))
            r = c.fetchall()
            if len(r) != 0:
                raise RuntimeError("Invalid user auth")

            auth_str = r[0][0]
            hasher = argon2.PasswordHasher()
            try:
                hasher.verify(auth_str, password)
            except argon2.exceptions.VerifyMismatchError:
                return None

    def user_id_by_email(self, email: str) -> uuid.UUID | None:
        """Get a user id associated to an email.

        Args:
            email (str): User email.

        Raises:
            RuntimeError: If more than one user is found.

        Returns:
            uuid.UUID | None: User id if found, otherwise None.
        """
        with self.conn.cursor() as c:
            c.execute("SELECT _id FROM user_ WHERE email=%s", (email,))
            r = c.fetchall()
            if len(r) == 0:
                return None
            elif len(r) == 1:
                return r[0][0]
            else:
                raise RuntimeError("Found multiple users with email")

    def add_data_to_sample(
        self,
        sample: uuid.UUID,
        creator: uuid.UUID,
    ):
        pass
