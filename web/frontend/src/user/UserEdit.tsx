import { Loading, SuspenseError } from "@/components";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import {
    Suspense,
    useContext,
    useEffect,
    useState,
    type ChangeEvent,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Navigate, useNavigate, useParams } from "react-router";
import * as uuid from "uuid";
import icon from "@/icon";
import * as common from "@/common";
import app_service from "@/service/app.service";
import user_service from "@/service/user.service";
import isEmail from "validator/lib/isEmail";
import { StatusCodes } from "http-status-codes";
import * as types from "@/types";
import Icon from "@/icon";
import { Context } from "@/AppStateContext";

export default function () {
    const { user_id: user_id_value } = useParams();
    if (!user_id_value) {
        return <Navigate to="/" replace />;
    }
    const user_id = uuid.parse(user_id_value);

    const ctx = useContext(Context);
    const user = ctx.user;
    const canModifyUsers = common.hasDbPermission(
        types.DbPermissionUserModify,
        user.DbPermissions,
    );
    if (!canModifyUsers) {
        console.debug("insufficient permissions to edit user");
        return <Navigate to="/" replace />;
    }

    return (
        <ErrorBoundary FallbackComponent={UserEditError}>
            <Suspense fallback={<Loading />}>
                <UserEdit user_id={user_id} />
            </Suspense>
        </ErrorBoundary>
    );
}

function UserEditError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            <div>Could not load user.</div>
        </SuspenseError>
    );
}

interface UserEditProps {
    user_id: uuid.UUIDTypes;
}
function UserEdit({ user_id }: UserEditProps) {
    const { data: user } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_USER, user_id],
        queryFn: async () => user_service.user(user_id),
    });
    const { data: db_permissions } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DB_PERMISSIONS],
        queryFn: app_service.getDbPermissions,
    });
    const queryClient = useQueryClient();

    const navigate = useNavigate();
    const [error, setError] = useState("");
    const [pending, setPending] = useState(false);

    const owner_permission = db_permissions.find(
        (permission) => permission.Id === types.DbPermissionOwner,
    )!;

    function cancel(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    function disable_standard_db_permissions(disable: boolean) {
        const others = document.querySelectorAll(
            "input[name='permission'][data-standard-permission]",
        ) as NodeListOf<HTMLInputElement>;
        for (const other of others) {
            other.disabled = disable;
        }
    }

    function owner_permission_toggled(e: ChangeEvent<HTMLInputElement>) {
        const disable_standard = e.target.checked;
        disable_standard_db_permissions(disable_standard);
    }

    useEffect(
        () =>
            disable_standard_db_permissions(
                user.DbPermissions.includes(
                    common.db_permission_id_string_to_variant(
                        owner_permission.Id,
                    )!,
                ),
            ),
        [],
    );

    async function update_user(e: SubmitEvent<HTMLFormElement>) {
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
        const permission_values = data
            .getAll("permission")
            .map((permission) =>
                common.db_permission_id_string_to_variant(
                    permission.toString(),
                ),
            );

        if (!isEmail(email)) {
            emailInput.setCustomValidity("Invalid email");
            emailInput.reportValidity();
            return;
        }

        for (const permission of permission_values) {
            if (permission === undefined) {
                const input = document.getElementById(
                    `permission-${permission}`,
                ) as HTMLInputElement | null;
                if (input === null) {
                    console.error(`invalid db permission: ${permission}`);
                    return;
                }

                input.setCustomValidity("Invalid value");
                return;
            }
        }
        const permissions = permission_values.filter(
            // user for typing
            (permission) => permission !== undefined,
        );

        const update = {
            Id: user.Id,
            AccountStatus: user.AccountStatus,
            Email: email,
            Name: name,
            DbPermissions: permissions,
        };

        setPending(true);
        await user_service.userUpdate(update).then(async (resp: Response) => {
            if (resp.ok) {
                queryClient.invalidateQueries({
                    queryKey: [common.QUERY_KEY_USER, user.Id.toString()],
                });
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
            setError(err.Message);
            btn_submit.disabled = false;
        });
        setPending(false);
    }

    return (
        <div>
            <div className="px-4 py-2 flex justify-between">
                <h2 className="text-xl">Edit user</h2>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={cancel}
                    >
                        <Icon.Close />
                    </button>
                </div>
            </div>
            <form onSubmit={update_user}>
                <div className="flex flex-col gap-2">
                    <div className="px-4">
                        <label>
                            <span className="sr-only">Email</span>
                            <input
                                type="text"
                                id="email"
                                name="email"
                                placeholder="Email"
                                className="input-basic"
                                defaultValue={user.Email}
                                onChange={(e) => e.target.setCustomValidity("")}
                            />
                        </label>
                    </div>
                    <div className="px-4">
                        <label>
                            <span className="sr-only">Name</span>
                            <input
                                type="text"
                                id="name"
                                name="name"
                                placeholder="Name"
                                className="input-basic"
                                defaultValue={user.Name}
                            />
                        </label>
                    </div>
                    <div className="w-full">
                        <fieldset>
                            <label
                                className="flex gap-2 px-4"
                                title={owner_permission.Description}
                            >
                                <input
                                    type="checkbox"
                                    id={`permission-${owner_permission.Id}`}
                                    name="permission"
                                    value={owner_permission.Id}
                                    onChange={owner_permission_toggled}
                                    defaultChecked={user.DbPermissions.includes(
                                        common.db_permission_id_string_to_variant(
                                            owner_permission.Id,
                                        )!,
                                    )}
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
                                        className="flex gap-2 px-4"
                                        title={permission.Description}
                                    >
                                        <input
                                            type="checkbox"
                                            id={`permission-${permission.Id}`}
                                            name="permission"
                                            value={permission.Id}
                                            data-standard-permission
                                            defaultChecked={user.DbPermissions.includes(
                                                common.db_permission_id_string_to_variant(
                                                    permission.Id,
                                                )!,
                                            )}
                                        />

                                        <span className="whitespace-nowrap">
                                            {permission.Label}
                                        </span>
                                    </label>
                                ))}
                        </fieldset>
                    </div>
                </div>
                <div className="flex gap-2 pt-4 px-4">
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
                                    Updating user
                                </>
                            ) : (
                                <>Update user</>
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
                </div>
            </form>
            <div className="pt-4">
                <PasswordReset user={user} />
            </div>
        </div>
    );
}

interface PasswordResetProps {
    user: types.User;
}
function PasswordReset({ user }: PasswordResetProps) {
    const [pending, setPending] = useState(false);
    const [message, setMessage] = useState("");

    async function confirm_reset(e: MouseEvent) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }
        setPending(true);

        const confirm = window.confirm(
            `Reset ${user.Name}'s <${user.Email}> password?`,
        );
        if (!confirm) {
            return;
        }

        await user_service
            .passwordReset(user.Id)
            .then(async (resp: Response) => {
                setPending(false);
                if (resp.ok) {
                    return;
                }

                const err = (await resp.json()) as types.AppError;
                if (err.Code == types.AppErrCodeEmailNotSent) {
                    setMessage(
                        `Could not send user email. Alert them their new password is ${err.Payload}`,
                    );
                } else {
                    console.debug(err);
                    setMessage("Could not reset password");
                }
            });
    }

    return (
        <div className="px-4">
            <button
                id="password-reset"
                type="button"
                className="btn-submit"
                onMouseDown={confirm_reset}
                disabled={pending}
            >
                Reset password
            </button>
            <div>{message}</div>
        </div>
    );
}
