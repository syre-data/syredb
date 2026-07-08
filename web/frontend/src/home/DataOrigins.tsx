import { MouseButton, QUERY_KEY_DATA, QUERY_KEY_DATA_ORIGINS } from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import { useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";
import { Suspense, type MouseEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, useNavigate } from "react-router";

export default function () {
    return (
        <ErrorBoundary FallbackComponent={DataOriginsError}>
            <Suspense fallback={<Loading />}>
                <DataOrigins />
            </Suspense>
        </ErrorBoundary>
    );
}

function DataOriginsError({ resetErrorBoundary, error }: FallbackProps) {
    return (
        <SuspenseError resetErrorBoundary={resetErrorBoundary}>
            "Could not load data origins"
        </SuspenseError>
    );
}

function DataOrigins() {
    const { data: origins } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_ORIGINS],
        queryFn: dataService.dataOrigins,
    });
    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <div className="flex gap-2 items-center">
                    <h1 className="title">Data origins</h1>
                    <div className="h-full">
                        <Link to="/data-origin/create" className="align-middle">
                            <button
                                type="button"
                                className="btn-cmd align-middle"
                            >
                                <Icon.Plus />
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
                        <Icon.Close />
                    </button>
                </div>
            </div>
            <div className="pt-2">
                <table className="table-std">
                    <tbody>
                        {origins.map((origin) => (
                            <tr
                                key={origin.Id.toString()}
                                className={classNames({
                                    group: true,
                                    "gray-600 dark:gray-300": !origin.Active,
                                })}
                            >
                                <td className="font-semibold text-nowrap w-0">
                                    {origin.Label}
                                </td>
                                <td>{origin.Description}</td>
                                <td>
                                    <div className="invisible group-hover:visible">
                                        <Link
                                            to={`/data-origin/${origin.Id}/edit`}
                                            title="Edit"
                                        >
                                            <button
                                                type="button"
                                                className="btn-cmd"
                                            >
                                                <Icon.Pen />
                                            </button>
                                        </Link>
                                    </div>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}
