import { FormEvent, MouseEvent, useState } from "react";
import { useNavigate } from "react-router";
import { MouseButton } from "../common";
import * as app from "../../bindings/syredb/app";
import isEmail from "validator/lib/isEmail";
import icon from "../icon";

export default function UserCreate() {
    const PASSWORD_LENGTH = 10;
    const navigate = useNavigate();
    const [error, setError] = useState("");
    const [pending, setPending] = useState(false);
    const [password, setPassword] = useState("");

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    async function create_user(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
        setError("");
        const btn_submit = document.getElementById(
            "submit"
        )! as HTMLButtonElement;
        btn_submit.disabled = true;

        const data = new FormData(e.target as HTMLFormElement);
        const email = data.get("email")!.toString();
        const name = data.get("name")!.toString();
        const role = data.get("role")!.toString();

        if (!isEmail(email)) {
            const input = document.getElementById("email")! as HTMLInputElement;
            input.setCustomValidity("invalid email");
            return;
        }

        const user = new app.UserCreate({
            Email: email,
            Name: name,
            Role: role,
            Password: randomString(PASSWORD_LENGTH),
        });

        setPending(true);
        await app.AppService.CreateUser(user)
            .then(() => navigate(-1))
            .catch((err: string) => {
                if (err.startsWith("WELCOME_EMAIL_NOT_SENT")) {
                    const PASSWORD_KEY = "{password: ";
                    const pw_start = err.indexOf(PASSWORD_KEY);
                    const pw_end = err.indexOf("}");
                    if (pw_start < 0 || pw_end < 0) {
                        console.error(
                            "user creation email not sent, and password could not be parsed"
                        );
                        setError(
                            "Something went wrong. Please contact your system administrator."
                        );
                    }

                    const pw = err.substring(
                        pw_start + PASSWORD_KEY.length,
                        pw_end
                    );
                    setPassword(pw);
                    return;
                }

                if (err.includes("(SQLSTATE 23505)")) {
                    setError("User already exists.");
                } else {
                    setError(err);
                }
                btn_submit.disabled = false;
            });
        setPending(false);
    }

    return (
        <div className="flex flex-col gap-2 items-center">
            <h2 className="px-4 text-xl font-bold">New user</h2>
            {error.length > 0 ? (
                <div className="text-red-600">
                    <div>Could not create user</div>
                    <div>{error}</div>
                </div>
            ) : null}
            {password.length > 0 ? (
                <div className="text-blue-700 dark:text-blue-300 text-center">
                    <div>Could not send welcome email</div>
                    <div>
                        Please inform the user their password is&nbsp;
                        <span className="font-bold cursor-pointer select-all">
                            {password}
                        </span>
                    </div>
                </div>
            ) : null}
            <form
                onSubmit={create_user}
                className="flex flex-col gap-2 items-center"
            >
                <div className="flex flex-col gap-2 items-center">
                    <div>
                        <label>
                            <span className="hidden">Email</span>
                            <input
                                id="email"
                                name="email"
                                type="text"
                                placeholder="Email"
                                className="input-basic"
                                autoComplete="email"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="hidden">Name</span>
                            <input
                                id="name"
                                name="name"
                                type="text"
                                placeholder="Name"
                                className="input-basic"
                                autoComplete="name"
                                required
                            />
                        </label>
                    </div>
                    <div className="w-full">
                        <label>
                            <span className="hidden">Role</span>
                            <select
                                id="role"
                                name="role"
                                defaultValue={"user"}
                                className="input-basic w-full"
                            >
                                <option value="user">User</option>
                                <option value="admin">Admin</option>
                                <option value="owner">Owner</option>
                            </select>
                        </label>
                    </div>
                </div>
                <div className="flex gap-2 justify-center">
                    {password.length > 0 ? (
                        <div>
                            <button
                                type="button"
                                id="submit"
                                onMouseDown={close}
                                className="btn-submit flex gap-2 items-center"
                            >
                                <icon.LeftArrow />
                                Back
                            </button>
                        </div>
                    ) : (
                        <>
                            <div>
                                <button
                                    type="submit"
                                    className="btn-submit"
                                    disabled={pending}
                                >
                                    {pending ? (
                                        <>
                                            <span className="pr-2 inline-block">
                                                <icon.Spinner className="animate-spin" />
                                            </span>
                                            Creating user
                                        </>
                                    ) : (
                                        <>Add user</>
                                    )}
                                </button>
                            </div>
                            <div>
                                <button
                                    type="button"
                                    id="submit"
                                    onMouseDown={close}
                                    className="btn-submit"
                                >
                                    Cancel
                                </button>
                            </div>
                        </>
                    )}
                </div>
            </form>
        </div>
    );
}

// Taken from https://stackoverflow.com/a/79761047/2961550
function randomString(
    length: number,
    allowed = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
): string {
    const characterAmount = allowed.length;
    return Array.from(
        crypto.getRandomValues(new Uint32Array(Number(length))),
        (randomValue) => allowed[randomValue % characterAmount]
    ).join(""); // randomValue?.constructor(characterAmount)
}
