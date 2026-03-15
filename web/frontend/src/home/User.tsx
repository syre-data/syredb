import { Loading, SuspenseError } from "@/components";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Suspense, type MouseEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, redirect, useNavigate, useParams } from "react-router";
import * as uuid from "uuid";
import icon from "@/icon";
import * as common from "@/common";
import user_service from "@/service/user.service";

export default function () {
    const { user_id: user_id_value } = useParams();
    if (!user_id_value) {
        throw redirect("/");
    }
    const user_id = uuid.parse(user_id_value);
    return (
        <ErrorBoundary FallbackComponent={UserError}>
            <Suspense fallback={<Loading />}>
                <User user_id={user_id} />
            </Suspense>
        </ErrorBoundary>
    );
}

function UserError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            <div>Could not load user.</div>
        </SuspenseError>
    );
}

interface UserProps {
    user_id: uuid.UUIDTypes;
}
function User({ user_id }: UserProps) {
    const { data: user } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_USER, user_id],
        queryFn: async () => user_service.user(user_id),
    });
    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        console.log("back");
        navigate(-1);
    }

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <div className="flex gap-2">
                    <h2 className="text-xl">User profile</h2>
                    <div>
                        <Link to={`/user/${user.Id.toString()}/edit`}>
                            <button type="button" className="btn-cmd">
                                <icon.Pen />
                            </button>
                        </Link>
                    </div>
                </div>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={close}
                    >
                        <icon.Close />
                    </button>
                </div>
            </div>
            <div>
                <div className="px-4 pt-2">{user.Name}</div>
                <div className="px-4 pt-2">{user.Email}</div>
                <div className="pt-2">
                    <div className="px-4">
                        <h3 className="text-lg">Permissions</h3>
                    </div>
                    {user.DbPermissions.length > 0 ? (
                        <ul>
                            {user.DbPermissions.map((permission) => (
                                <li key={permission} className="px-4">
                                    {permission}
                                </li>
                            ))}
                        </ul>
                    ) : (
                        <div className="px-4">No permissions</div>
                    )}
                </div>
            </div>
        </div>
    );
}
