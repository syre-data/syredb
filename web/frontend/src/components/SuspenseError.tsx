import * as common from "@/common";
import type { MouseEvent } from "react";
import { Link } from "react-router";
import icon from "@/icon";

interface Props {
    resetErrorBoundary: (...args: unknown[]) => void;
    className?: any;
    children: any;
}

export default function ({ resetErrorBoundary, className, children }: Props) {
    function reload(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        resetErrorBoundary();
    }

    return (
        <div className={className}>
            <div>{children}</div>
            <div className="flex gap-2 items-center justify-center">
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
