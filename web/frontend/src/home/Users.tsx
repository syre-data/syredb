import { Link, useNavigate } from "react-router";
import icon from "../icon";
import { Suspense, useOptimistic, useState } from "react";
import type { Dispatch, MouseEvent, SetStateAction } from "react";
import { MouseButton } from "../common";
import { ErrorBoundary } from "react-error-boundary";
import type { FallbackProps } from "react-error-boundary";
import user_service from "@/service/user.service";
import * as types from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import * as common from "../common";
import classNames from "classnames";

export default function Users() {
    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    return (
        <div>
            <div className="flex gap-2 text-xl pt-2">
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
                <div className="pr-4">
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

function UserListError({ error, resetErrorBoundary }: FallbackProps) {
    const err = error as common.BackendError;
    return (
        <div className="flex flex-col gap-2 items-center pt-2">
            <div>Could not get users</div>
            <div className="text-red-600">{err.message}</div>
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
        queryKey: [common.QUERY_KEY_USER_LIST],
        queryFn: user_service.get_users,
    });
    const [usersOptimistic, setUsersOptimistic] = useOptimistic<
        types.User[],
        types.User
    >(users, (users, user) => {
        return users.map((u) => (u.Id === user.Id ? user : u));
    });

    return (
        <ul className="grid grid-cols-[repeat(3,min-content)_auto] gap-x-4">
            {usersOptimistic.map((user) => (
                <li
                    key={user.Id.toString()}
                    className={classNames({
                        "group/user-row col-span-full grid grid-cols-subgrid \
                      hover:bg-gray-100 dark:hover:bg-gray-900": true,
                        "text-gray-600": user.AccountStatus != "active",
                        "dark:text-gray-400": user.AccountStatus != "active",
                    })}
                >
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
    user: types.User;
    editing: string | null;
    setEditing: Dispatch<SetStateAction<string | null>>;
    setUsersOptimistic: (action: types.User) => void;
}
function UserItem({ user }: UserItemProps) {
    return (
        <div className="contents">
            <div className="col-1 pl-4">{user.Email}</div>
            <div className="col-2 whitespace-nowrap">{user.Name}</div>
            <div className="col-3 pr-4 invisible group-hover/user-row:visible">
                <Link to={`/user/${user.Id}`}>
                    <button type="button" className="btn-cmd">
                        <icon.Pen />
                    </button>
                </Link>
            </div>
        </div>
    );
}
