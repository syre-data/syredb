import * as common from "@/common";
import type { MouseEvent } from "react";
import { Link } from "react-router";
import icon from "@/icon";
import { AppError } from "@/errors";

interface SuspenseErrorProps {
    resetErrorBoundary: (...args: unknown[]) => void;
    className?: any;
    children: any;
}
export function SuspenseError({
    resetErrorBoundary,
    className,
    children,
}: SuspenseErrorProps) {
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

interface StandardErrorProps {
    resetErrorBoundary: (...args: unknown[]) => void;
    error: Error;
    title: string;
}
export function StandardError({
    resetErrorBoundary,
    error,
    title,
}: StandardErrorProps) {
    console.error(error);

    let content = <GenericError title={title} error={error} />;
    if (error instanceof AppError) {
        content = <AppErrorView title={title} error={error} />;
    }

    return (
        <SuspenseError resetErrorBoundary={resetErrorBoundary}>
            {content}
        </SuspenseError>
    );
}

interface GenericErrorProps {
    title: string;
    error: Error;
}
function GenericError({ title, error }: GenericErrorProps) {
    return (
        <div className="py-2 text-center">
            <div className="text-lg">{title}</div>
            <div className="text-syre-red-700 dark:text-syre-red-400">
                {error.message}
            </div>
        </div>
    );
}

interface AppErrorProps {
    title: string;
    error: AppError;
}
function AppErrorView({ title, error }: AppErrorProps) {
    return (
        <div className="py-2 text-center">
            <div className="text-lg">{title}</div>
            <div className="text-syre-red-700 dark:text-syre-red-400">
                {error.message}
            </div>
        </div>
    );
}
