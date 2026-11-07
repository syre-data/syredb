import { Link, useNavigate } from "react-router";
import icon from "../icon";
import { Suspense, useOptimistic, useState } from "react";
import type { Dispatch, MouseEvent, SetStateAction } from "react";
import { MouseButton } from "../common";
import { ErrorBoundary } from "react-error-boundary";
import type { FallbackProps } from "react-error-boundary";
import user_service from "@/service/user.service";
import { useSuspenseQuery } from "@tanstack/react-query";
import * as common from "../common";
import classNames from "classnames";
import { StandardError } from "@/components";
import type { AppError, User } from "@/types";

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
    return (
        <StandardError
            resetErrorBoundary={resetErrorBoundary}
            error={error as Error}
            title={"Could not get users"}
        />
    );
}

function UserList() {
    const [editing, setEditing] = useState<string | null>(null);
    const { data: users } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_USER_LIST],
        queryFn: user_service.get_users,
    });
    const [usersOptimistic, setUsersOptimistic] = useOptimistic<User[], User>(
        users,
        (users, user) => {
            return users.map((u) => (u.Id === user.Id ? user : u));
        },
    );

    return (
        <table className="table-std">
            <tbody>
                {usersOptimistic.map((user) => (
                    <UserItem
                        key={user.Id.toString()}
                        user={user}
                        editing={editing}
                        setEditing={setEditing}
                        setUsersOptimistic={setUsersOptimistic}
                    />
                ))}
            </tbody>
        </table>
    );
}

interface UserItemProps {
    user: User;
    editing: string | null;
    setEditing: Dispatch<SetStateAction<string | null>>;
    setUsersOptimistic: (action: User) => void;
}
function UserItem({ user }: UserItemProps) {
    return (
        <tr
            className={classNames({
                group: true,
                "text-secondary": user.AccountStatus != "active",
            })}
        >
            <td className="whitespace-nowrap font-semibold w-0">
                {user.Email}
            </td>
            <td className="whitespace-nowrap w-0">{user.Name}</td>
            <td>
                <div className="invisible group-hover:visible">
                    <Link to={`/user/${user.Id}`}>
                        <button type="button" className="btn-cmd">
                            <icon.Eye />
                        </button>
                    </Link>
                </div>
            </td>
        </tr>
    );
}
