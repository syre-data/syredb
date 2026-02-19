import { useState } from "react";
import type { SubmitEvent, MouseEvent } from "react";
import { useNavigate } from "react-router";
import { MouseButton } from "../common";
import isEmail from "validator/lib/isEmail";
import { StatusCodes } from "http-status-codes";
import icon from "../icon";
import * as types from "@/types";
import user_service from "@/service/user.service";

export default function UserCreate() {
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

    async function create_user(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        setError("");
        const btn_submit = document.getElementById(
            "submit",
        )! as HTMLButtonElement;
        const emailInput = document.getElementById(
            "email",
        )! as HTMLInputElement;
        btn_submit.disabled = true;
        emailInput.setCustomValidity("");

        const data = new FormData(e.target as HTMLFormElement);
        const email = data.get("email")!.toString();
        const name = data.get("name")!.toString();
        const role = data.get("role")!.toString();

        if (!isEmail(email)) {
            emailInput.setCustomValidity("invalid email");
            emailInput.reportValidity();
            return;
        }

        const user = {
            Email: email,
            Name: name,
            Role: role,
            Password: "",
        };

        setPending(true);
        await user_service.create_user(user).then(async (resp: Response) => {
            if (resp.ok) {
                return navigate(-1);
            }
            if (resp.status == StatusCodes.CONFLICT) {
                setError("User already exists.");
                emailInput.setCustomValidity("Email already exists");
                emailInput.reportValidity();
                btn_submit.disabled = false;
                emailInput.setCustomValidity("");
                return;
            }

            const err = (await resp.json()) as types.AppError;
            if (err.Code == types.AppErrCodeUserWelcomeEmailNotSent) {
                setPassword(err.Payload);
                return;
            }

            setError(err.Message);
            btn_submit.disabled = false;
        });
        setPending(false);
    }

    return (
        <div className="flex flex-col gap-2 items-center">
            <h2 className="px-4 pt-2 text-xl font-bold">New user</h2>
            {error.length > 0 ? (
                <div className="text-red-600">
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
