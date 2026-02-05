import type { FallbackProps } from "react-error-boundary";
import * as common from "@/common";
import { Link, useNavigate } from "react-router";
import type { MouseEvent } from "react";
import icon from "@/icon";

export function Loading() {
    return <div className="text-center pt-4">Loading</div>;
}

interface ErrorProps extends FallbackProps {
    children: any;
}
export function Error({ error, resetErrorBoundary, children }: ErrorProps) {
    const err = error as common.BackendError;
    const navigate = useNavigate();

    if (err.message === common.USER_NOT_AUTHENTICATED_ERROR) {
        console.error(common.USER_NOT_AUTHENTICATED_ERROR);
        navigate("/");
        return null;
    } else {
        console.error(err);
    }

    function reload(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        resetErrorBoundary();
    }

    return (
        <div className="flex flex-col gap-2 items-center pt-4">
            <div>{children}</div>
            <div className="flex gap-2 items-center">
                <div>
                    <Link to="/">
                        <button type="button" className="btn-cmd">
                            <icon.Home />
                        </button>
                    </Link>
                </div>
                <div>
                    <button
                        type="button"
                        onMouseDown={reload}
                        className="btn-cmd"
                    >
                        <icon.Reload />
                    </button>
                </div>
            </div>
        </div>
    );
}
