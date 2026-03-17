import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import * as common from "@/common";
import data_service from "@/service/data.service";
import {
    ErrorBoundary,
    type FallbackProps as ErrorBoundaryProps,
} from "react-error-boundary";
import { Loading, SuspenseError } from "@/components";
import { Suspense, useState, type MouseEvent, type SubmitEvent } from "react";
import * as types from "@/types";
import icon from "@/icon";
import { StatusCodes } from "http-status-codes";
import classNames from "classnames";
import { Link } from "react-router";

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

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <div className="flex gap-2">
                    <h2 className="text-lg">Data types</h2>
                    <div>
                        <Link to="/data-type/create">
                            <button type="button" className="btn-cmd">
                                <icon.Plus />
                            </button>
                        </Link>
                    </div>
                </div>
                <div>
                    <Link to="/">
                        <icon.Home />
                    </Link>
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
        <ul className="grid grid-cols-[repeat(3,min-content)] gap-2">
            {data_types.map((type) => (
                <DataTypeListItem key={type.Id.toString()} data_type={type} />
            ))}
        </ul>
    );
}

interface DataTypeListItemProps {
    data_type: types.DataType;
}
function DataTypeListItem({ data_type }: DataTypeListItemProps) {
    return (
        <li
            className={classNames({
                "px-4 col-span-full grid grid-cols-subgrid": true,
                "text-gray-600 dark:text-gray-400": !data_type.Active,
            })}
        >
            <div className="col-1 whitespace-nowrap">{data_type.Label}</div>
            <div className="col-2 whitespace-nowrap">
                {data_type.Description}
            </div>
            <div className="col-3 whitespace-nowrap">{data_type.Transform}</div>
        </li>
    );
}
