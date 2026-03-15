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

    const [showAddNew, setShowAddNew] = useState(false);

    function openNew(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        setShowAddNew(true);
    }

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <div className="flex gap-2">
                    <h2 className="text-lg">Data types</h2>
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            onMouseDown={openNew}
                        >
                            <icon.Plus />
                        </button>
                    </div>
                </div>
                <div>
                    <Link to="/">
                        <icon.Home />
                    </Link>
                </div>
            </div>
            {data_types.length === 0 && !showAddNew ? (
                <DataTypesEmpty />
            ) : (
                <div>
                    {showAddNew ? (
                        <NewDataTypeForm
                            onSuccess={() => setShowAddNew(false)}
                            onCancel={() => setShowAddNew(false)}
                        />
                    ) : null}
                    <DataTypesContent data_types={data_types} />
                </div>
            )}
        </div>
    );
}

function DataTypesEmpty() {
    return (
        <div className="px-4">
            <p>You don't have any data types yet.</p>
            <p>
                Create your first one by clicking the{" "}
                <icon.Plus className="inline" /> above
            </p>
        </div>
    );
}

interface NewDataTypeFormProps {
    onSuccess: () => void;
    onCancel: () => void;
}
function NewDataTypeForm({ onSuccess, onCancel }: NewDataTypeFormProps) {
    const queryClient = useQueryClient();

    async function submit(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const data = new FormData(e.target);
        const transform = data.get("transform")! as File;
        const label = data.get("label")!.toString().trim();
        const description = data.get("description")!.toString().trim();

        if (!label) {
            const input = document.getElementById("label")! as HTMLInputElement;
            input.setCustomValidity("Label can not be empty");
            input.reportValidity();
            return;
        }

        await data_service
            .dataTypeCreate(transform, label, description)
            .then((resp) => {
                if (resp.status === StatusCodes.OK) {
                    queryClient.invalidateQueries({
                        queryKey: [common.QUERY_KEY_DATA_TYPES],
                    });
                    onSuccess();
                }
            });
    }

    function cancel(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        onCancel();
    }

    return (
        <form className="px-4" onSubmit={submit}>
            <div className="w-min flex flex-col gap-2">
                <div className="flex flex-col gap-2">
                    <div>
                        <label>
                            <span className="sr-only">Label</span>
                            <input
                                id="label"
                                name="label"
                                type="text"
                                placeholder="Label"
                                className="input-basic"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="sr-only">Transform</span>
                            <input
                                id="transform"
                                name="transform"
                                type="file"
                                placeholder="Transform"
                                className="input-basic"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="sr-only">Description</span>
                            <textarea
                                id="description"
                                name="description"
                                placeholder="Description"
                                className="input-basic"
                            ></textarea>
                        </label>
                    </div>
                </div>
                <div className="flex gap-2 justify-center">
                    <button type="submit" className="btn-submit">
                        Save
                    </button>
                    <button
                        type="button"
                        onMouseDown={cancel}
                        className="btn-submit"
                    >
                        Cancel
                    </button>
                </div>
            </div>
        </form>
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
