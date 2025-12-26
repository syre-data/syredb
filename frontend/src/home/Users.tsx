import { Link, useNavigate } from "react-router";
import icon from "../icon";
import {
    Dispatch,
    FormEvent,
    MouseEvent,
    SetStateAction,
    startTransition,
    Suspense,
    useOptimistic,
    useState,
} from "react";
import { MouseButton } from "../common";
import { ErrorBoundary, FallbackProps } from "react-error-boundary";
import * as app from "../../bindings/syredb/app";
import {
    useMutation,
    useQueryClient,
    useSuspenseQuery,
} from "@tanstack/react-query";
import isEmail from "validator/lib/isEmail";
import * as common from "../common";
import classNames from "classnames";

export default function Users() {
    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        console.log("back");
        navigate(-1);
    }

    return (
        <div>
            <div className="flex gap-2 text-xl">
                <div className="grow flex gap-2 px-4">
                    <h2>Users</h2>
                    <div>
                        <Link
                            to="/user/create"
                            title="Add user"
                            className="align-middle"
                        >
                            <button type="button" className="btn-cmd">
                                <icon.Plus />
                            </button>
                        </Link>
                    </div>
                </div>
                <div>
                    <button
                        type="button"
                        onMouseDown={close}
                        className="btn-cmd"
                        title="Close"
                    >
                        <icon.Close />
                    </button>
                </div>
            </div>
            <ErrorBoundary FallbackComponent={UserListError}>
                <Suspense fallback={<Loading />}>
                    <UserList />
                </Suspense>
            </ErrorBoundary>
        </div>
    );
}

function Loading() {
    return <div className="text-center pt-2">Loading users</div>;
}

const QUERY_KEY_USER_LIST = "get_users_list";
function UserListError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <div className="flex flex-col gap-2 items-center pt-2">
            <div>Could not get users</div>
            <div className="text-red-600">{error}</div>
            <div>
                <button
                    type="button"
                    onMouseDown={resetErrorBoundary}
                    className="btn-submit"
                >
                    Try again
                </button>
            </div>
        </div>
    );
}

function UserList() {
    const [editing, setEditing] = useState<string | null>(null);
    const { data: users } = useSuspenseQuery({
        queryKey: [QUERY_KEY_USER_LIST],
        queryFn: app.AppService.GetUsers,
    });
    const [usersOptimistic, setUsersOptimistic] = useOptimistic<
        app.User[],
        app.User
    >(users, (users, user) => {
        return users.map((u) => (u.Id === user.Id ? user : u));
    });

    return (
        <ul>
            {usersOptimistic.map((user) => (
                <li key={user.Id.toString()}>
                    <UserItem
                        user={user}
                        editing={editing}
                        setEditing={setEditing}
                        setUsersOptimistic={setUsersOptimistic}
                    />
                </li>
            ))}
        </ul>
    );
}

interface UserItemProps {
    user: app.User;
    editing: string | null;
    setEditing: Dispatch<SetStateAction<string | null>>;
    setUsersOptimistic: (action: app.User) => void;
}
function UserItem({
    user,
    editing,
    setEditing,
    setUsersOptimistic,
}: UserItemProps) {
    function set_editing(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        setEditing(user.Id.toString());
    }

    if (editing === user.Id.toString()) {
        return (
            <UserItemEditing
                user={user}
                setEditing={setEditing}
                setUsersOptimistic={setUsersOptimistic}
            />
        );
    } else {
        return (
            <div
                className={classNames({
                    flex: true,
                    "gap-2": true,
                    "px-4": true,
                    group: true,
                    "hover:bg-gray-100": true,
                    "dark:hover:bg-gray-900": true,
                    "text-gray-600": user.AccountStatus != "active",
                    "dark:text-gray-400": user.AccountStatus != "active",
                })}
            >
                <div className="grow flex gap-2">
                    <div>{user.Email}</div>
                    <div>{user.Name}</div>
                    <div>{user.Role}</div>
                </div>
                <div className="invisible group-hover:visible">
                    <button
                        type="button"
                        onMouseDown={set_editing}
                        className="btn-cmd"
                    >
                        <icon.Pen />
                    </button>
                </div>
            </div>
        );
    }
}

interface UserItemEditingProps {
    user: app.User;
    setEditing: Dispatch<SetStateAction<string | null>>;
    setUsersOptimistic: (action: app.User) => void;
}
function UserItemEditing({
    user,
    setEditing,
    setUsersOptimistic,
}: UserItemEditingProps) {
    const queryClient = useQueryClient();
    const [error, setError] = useState("");
    const updateUserMutation = useMutation({
        mutationFn: app.AppService.UpdateUser,
        onSettled: () =>
            queryClient.invalidateQueries({ queryKey: [QUERY_KEY_USER_LIST] }),
    });
    const deactivateUserMutation = useMutation({
        mutationFn: app.AppService.DeactivateUser,
        onSettled: () =>
            queryClient.invalidateQueries({ queryKey: [QUERY_KEY_USER_LIST] }),
    });

    function enable_btns(enabled: boolean) {
        const submit = document.getElementById("submit")! as HTMLButtonElement;
        const cancel = document.getElementById("cancel")! as HTMLButtonElement;
        const del = document.getElementById("deactivate")! as HTMLButtonElement;
        submit.disabled = !enabled;
        cancel.disabled = !enabled;
        del.disabled = !enabled;
    }

    async function onsubmit(e: FormEvent) {
        e.preventDefault();
        setError("");
        enable_btns(false);

        const data = new FormData(e.target as HTMLFormElement);
        const email = data.get("email")!.toString();
        const name = data.get("name")!.toString();
        const role_str = data.get("role")!.toString();
        const role = common.user_role_string_to_variant(role_str);
        if (!role) {
            console.error(`invalid user role: ${role_str}`);
            return;
        }

        if (!isEmail(email)) {
            const input = document.getElementById("email")! as HTMLInputElement;
            input.setCustomValidity("invalid email");
            enable_btns(true);
            return;
        }

        const update = new app.User({
            Id: user.Id,
            AccountStatus: user.AccountStatus,
            Email: email,
            Name: name,
            Role: role,
        });

        startTransition(() => setUsersOptimistic(update));
        await updateUserMutation
            .mutateAsync(update)
            .then(() => {
                setEditing(null);
            })
            .catch((err: string) => {
                if (err.includes("NO_USER_WITH_OWNER_ROLE")) {
                    setError("Must have at least one owner");
                } else {
                    setError(err);
                }
            });
        enable_btns(true);
    }

    function cancel(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        setEditing(null);
    }

    async function deactivate_user(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != MouseButton.Primary) {
            return;
        }
        setError("");

        const update = new app.User({
            Id: user.Id,
            AccountStatus: "disabled",
            Email: user.Email,
            Name: user.Name,
            Role: user.Role,
        });

        startTransition(() => setUsersOptimistic(update));
        await deactivateUserMutation
            .mutateAsync(user.Id)
            .then(() => {
                setEditing(null);
            })
            .catch((err: string) => {
                setError(err);
            });
    }

    return (
        <div className="px-4">
            <form onSubmit={onsubmit} className="flex gap-2">
                <div className="grow flex gap-2">
                    <div>
                        <label>
                            <span className="hidden">Email</span>
                            <input
                                type="email"
                                id="email"
                                name="email"
                                defaultValue={user.Email}
                                placeholder="Email"
                                className="input-basic"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="hidden">Email</span>
                            <input
                                type="text"
                                id="name"
                                name="name"
                                defaultValue={user.Name}
                                placeholder="Name"
                                className="input-basic"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="hidden">Role</span>
                            <select name="role" defaultValue={user.Role}>
                                <option value="user">User</option>
                                <option value="admin">Admin</option>
                                <option value="owner">Owner</option>
                            </select>
                        </label>
                    </div>
                </div>
                <div className="flex gap-2">
                    {user.AccountStatus === "active" ? (
                        <div>
                            <button
                                type="button"
                                id="deactivate"
                                onMouseDown={deactivate_user}
                                title="Deactivate user"
                                className="btn-cmd hover:text-red-700"
                            >
                                <icon.PowerOff />
                            </button>
                        </div>
                    ) : null}
                    <div>
                        <button
                            type="submit"
                            id="submit"
                            name="submit"
                            title="Save changes"
                            className="btn-cmd hover:text-green-700"
                        >
                            <icon.Check />
                        </button>
                    </div>
                    <div>
                        <button
                            type="button"
                            id="cancel"
                            onMouseDown={cancel}
                            title="Cancel"
                            className="btn-cmd hover:text-gray-600 dark:hover:text-gray-400"
                        >
                            <icon.Close />
                        </button>
                    </div>
                </div>
            </form>
            <div className="text-red-600">
                <small>{error}</small>
            </div>
        </div>
    );
}
