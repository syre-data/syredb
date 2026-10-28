import { useSuspenseQuery } from "@tanstack/react-query";
import * as common from "@/common";
import data_service from "@/service/data.service";
import {
    ErrorBoundary,
    type FallbackProps as ErrorBoundaryProps,
} from "react-error-boundary";
import { Loading, SuspenseError } from "@/components";
import { Suspense, useContext, type MouseEvent } from "react";
import * as types from "@/types";
import icon from "@/icon";
import classNames from "classnames";
import { Link, useNavigate } from "react-router";
import { Context } from "@/AppStateContext";

export default function () {
    return (
        <div>
            <ErrorBoundary FallbackComponent={DataTypesError}>
                <Suspense fallback={<Loading />}>
                    <DataTypes />
                </Suspense>
            </ErrorBoundary>
        </div>
    );
}

function DataTypesError({ error, resetErrorBoundary }: ErrorBoundaryProps) {
    const err = error as common.BackendError;
    console.error(err);

    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            <div>Could not get data types</div>
            <div>{err.message}</div>
        </SuspenseError>
    );
}

function DataTypes() {
    const { data: data_types } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_TYPES],
        queryFn: data_service.dataTypesGetAll,
    });
    const navigate = useNavigate();

    const ctx = useContext(Context);
    const user = ctx.user;

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    const canCreateType = common.hasDbPermission(
        types.DbPermissionDataTypeCreate,
        user.DbPermissions,
    );
    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <div className="flex gap-2 items-center">
                    <h1 className="title">Data types</h1>
                    <div>
                        {canCreateType ? (
                            <Link to="/data-type/create">
                                <button type="button" className="btn-cmd">
                                    <icon.Plus />
                                </button>
                            </Link>
                        ) : null}
                    </div>
                </div>
                <div>
                    <button
                        type="button"
                        title="Close"
                        className="btn-cmd"
                        onMouseDown={close}
                    >
                        <icon.Close />
                    </button>
                </div>
            </div>
            {data_types.length === 0 ? (
                <DataTypesEmpty />
            ) : (
                <DataTypesContent data_types={data_types} />
            )}
        </div>
    );
}

function DataTypesEmpty() {
    return (
        <div className="px-4">
            <p>No data types</p>
            <p>
                Create one by clicking the <icon.Plus className="inline" />{" "}
                above
            </p>
        </div>
    );
}

interface DataTypesContentProps {
    data_types: types.DataType[];
}
function DataTypesContent({ data_types }: DataTypesContentProps) {
    return (
        <table className="table-std">
            <tbody>
                {data_types.map((type) => (
                    <DataTypeItem key={type.Id.toString()} data_type={type} />
                ))}
            </tbody>
        </table>
    );
}

interface DataTypeItemProps {
    data_type: types.DataType;
}
function DataTypeItem({ data_type }: DataTypeItemProps) {
    return (
        <tr
            className={classNames({
                group: true,
                "text-secondary": !data_type.Active,
            })}
        >
            <td className="whitespace-nowrap w-0">{data_type.Label}</td>
            <td className="whitespace-nowrap w-0">
                {data_type.Storage === types.DataStorageInternal ? (
                    <icon.Columns
                        className="inline-block"
                        title="Internal storage"
                    />
                ) : (
                    <icon.Files
                        className="inline-block"
                        title="External storage"
                    />
                )}
            </td>
            <td className="">{data_type.Description}</td>
            <td>
                <Link
                    to={`/data-type/${data_type.Id}`}
                    className="invisible group-hover:visible"
                >
                    <button type="button" className="btn-cmd">
                        <icon.Eye />
                    </button>
                </Link>
            </td>
        </tr>
    );
}
