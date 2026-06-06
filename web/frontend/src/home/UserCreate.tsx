import { Suspense, useState } from "react";
import type { SubmitEvent, MouseEvent, ChangeEvent } from "react";
import { useNavigate } from "react-router";
import { MouseButton } from "../common";
import isEmail from "validator/lib/isEmail";
import { StatusCodes } from "http-status-codes";
import icon from "../icon";
import * as common from "@/common";
import * as types from "@/types";
import user_service from "@/service/user.service";
import app_service from "@/service/app.service";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { useSuspenseQuery } from "@tanstack/react-query";
import { SuspenseError } from "@/components";
import { Loading } from "@/components";

export default function () {
    return (
        <ErrorBoundary FallbackComponent={LoadError}>
            <Suspense fallback={<Loading />}>
                <UserCreate />
            </Suspense>
        </ErrorBoundary>
    );
}

function LoadError({ resetErrorBoundary, error }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            <div>Could not load database permissions</div>
        </SuspenseError>
    );
}

function UserCreate() {
    const { data: db_permissions } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DB_PERMISSIONS],
        queryFn: app_service.getDbPermissions,
    });

    const navigate = useNavigate();
    const [error, setError] = useState("");
    const [pending, setPending] = useState(false);
    const [password, setPassword] = useState("");

    function cancel(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    function owner_permission_toggled(e: ChangeEvent<HTMLInputElement>) {
        const others = document.querySelectorAll(
            "input[name='permission'][data-standard-permission]",
        ) as NodeListOf<HTMLInputElement>;
        const disable_others = e.target.checked;
        for (const other of others) {
            other.disabled = disable_others;
        }
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
        const permissions = data
            .getAll("permission")
            .map((permission) =>
                common.db_permission_id_string_to_variant(
                    permission.toString(),
                ),
            );

        if (permissions.includes(undefined)) {
            console.debug("invalid permissions", permissions);
            throw new Error(`invalid permissions: ${permissions}`);
        }

        if (!isEmail(email)) {
            emailInput.setCustomValidity("Invalid email");
            emailInput.reportValidity();
            return;
        }

        const user = {
            email,
            name,
            password: "",
            db_permissions: permissions,
        };

        setPending(true);
        await user_service.userCreate(user).then(async (resp: Response) => {
            setPending(false);
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
            if (err.Code == types.AppErrCodeEmailNotSent) {
                setPassword(err.Payload);
                return;
            }

            setError(err.Message);
            btn_submit.disabled = false;
        });
    }

    const owner_permission = db_permissions.find(
        (permission) => permission.Id === types.DbPermissionOwner,
    )!;
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
                            <span className="sr-only">Email</span>
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
                            <span className="sr-only">Name</span>
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
                        <fieldset>
                            <label
                                className="flex gap-2"
                                title={owner_permission.Description}
                            >
                                <input
                                    type="checkbox"
                                    id={`permission-${owner_permission.Id}`}
                                    name="permission"
                                    value={owner_permission.Id}
                                    onChange={owner_permission_toggled}
                                />

                                <span className="whitespace-nowrap">
                                    {owner_permission.Label}
                                </span>
                            </label>
                            {db_permissions
                                .filter(
                                    (permission) =>
                                        permission.Id !==
                                        types.DbPermissionOwner,
                                )
                                .map((permission) => (
                                    <label
                                        key={permission.Id}
                                        className="flex gap-2"
                                        title={permission.Description}
                                    >
                                        <input
                                            type="checkbox"
                                            id={`permission-${permission.Id}`}
                                            name="permission"
                                            value={permission.Id}
                                            data-standard-permission
                                        />

                                        <span className="whitespace-nowrap">
                                            {permission.Label}
                                        </span>
                                    </label>
                                ))}
                        </fieldset>
                    </div>
                </div>
                <div className="flex gap-2 justify-center">
                    {password.length > 0 ? (
                        <div>
                            <button
                                type="button"
                                id="submit"
                                onMouseDown={cancel}
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
                                    onMouseDown={cancel}
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
