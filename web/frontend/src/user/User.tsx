import { Loading, StandardError } from "@/components";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
    Suspense,
    useContext,
    type ComponentProps,
    type MouseEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, redirect, useNavigate, useParams } from "react-router";
import * as uuid from "uuid";
import icon from "@/icon";
import * as common from "@/common";
import user_service from "@/service/user.service";
import {
    DbPermissionUserModify,
    type DbPermissionRecord,
    type User,
} from "@/types";
import { Context } from "@/AppStateContext";
import Icon from "@/icon";
import { DbPermissionBadges } from "./components";
import appService from "@/service/app.service";

export default function () {
    const { user_id: paramUserId } = useParams();
    if (!paramUserId) {
        throw redirect("/");
    }
    const userId = uuid.parse(paramUserId);
    return (
        <ErrorBoundary FallbackComponent={UserError}>
            <Suspense fallback={<Loading />}>
                <User userId={userId} />
            </Suspense>
        </ErrorBoundary>
    );
}

function UserError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <StandardError
            resetErrorBoundary={resetErrorBoundary}
            error={error as Error}
            title={"Could not load user"}
        />
    );
}

interface UserProps {
    userId: uuid.UUIDTypes;
}
function User({ userId }: UserProps) {
    const ctx = useContext(Context);
    const authUser = ctx.user;
    const { data: user } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_USER, userId],
        queryFn: async () => user_service.user(userId),
    });
    const { data: dbPermissions } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DB_PERMISSIONS],
        queryFn: appService.getDbPermissions,
    });

    const permissions = user.DbPermissions.map((perm) => {
        const info = dbPermissions.find((p) => p.Id === perm);
        if (!info) {
            throw new Error(`invalid database permission ${perm}`);
        }

        return info;
    });

    const canModifyUser = common.hasDbPermission(
        DbPermissionUserModify,
        authUser.DbPermissions,
    );
    return (
        <div>
            <Header canModifyUser={canModifyUser} userId={user.Id} />
            <div className="pt-4 px-4">
                <div className="card py-4 flex gap-4 divide-x justify-start rounded-lg">
                    <UserDetails user={user} />
                    <UserDbPermissions permissions={permissions} />
                </div>
            </div>
        </div>
    );
}

interface HeaderProps {
    canModifyUser: boolean;
    userId: uuid.UUIDTypes;
}
function Header({ canModifyUser, userId }: HeaderProps) {
    const navigate = useNavigate();
    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    return (
        <div className="px-4 pt-2 flex justify-between">
            <div className="flex gap-2">
                <h1 className="text-xl">User profile</h1>
                <div>
                    {canModifyUser ? (
                        <Link
                            to={`/user/${common.uuidToString(userId)}/edit`}
                            className="align-middle"
                        >
                            <button type="button" className="btn-cmd">
                                <icon.Pen />
                            </button>
                        </Link>
                    ) : null}
                </div>
            </div>
            <div>
                <button type="button" className="btn-cmd" onMouseDown={close}>
                    <icon.Close />
                </button>
            </div>
        </div>
    );
}

interface UserDetailsProps {
    user: User;
}
function UserDetails({ user }: UserDetailsProps) {
    return (
        <div>
            <div className="px-4 flex gap-2 items-baseline">
                <Icon.User />
                <h2 className="font-bold">User details</h2>
            </div>
            <div className="px-4 pt-2">
                {user.Name ? (
                    <>
                        Name: <span className="font-bold">{user.Name}</span>
                    </>
                ) : (
                    <span className="text-secondary">
                        "(User name not set)"
                    </span>
                )}
            </div>
            <div className="px-4 pt-2 text-nowrap flex gap-2 items-baseline">
                Email:{" "}
                <span className="font-bold cursor-pointer hover:underline select-all">
                    {user.Email}
                </span>
                <a
                    target="_blank"
                    href={`mailto:${user.Email}`}
                    className="cursor-pointer"
                >
                    <Icon.Mail />
                </a>
            </div>
        </div>
    );
}

interface UserDbPermissionsProps extends ComponentProps<"div"> {
    permissions: DbPermissionRecord[];
}
function UserDbPermissions({ permissions, ...props }: UserDbPermissionsProps) {
    return (
        <div {...props}>
            <div className="pb-2">
                <div className="px-4 flex gap-2 items-center text-lg">
                    <Icon.Shield />
                    <h2 className="font-bold">Database permissions</h2>
                </div>
            </div>
            {permissions.length > 0 ? (
                <DbPermissionBadges
                    permissions={permissions}
                    className="px-4 flex gap-2"
                />
            ) : (
                <div className="px-4 text-sm text-secondary">
                    User does not have any database permissions
                </div>
            )}
        </div>
    );
}
