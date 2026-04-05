import {
    MouseButton,
    QUERY_KEY_DATA_TYPE_TRANSFORMS,
    QUERY_KEY_DATA_TYPES,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import type { DataTypeTransformCreate, DataTypeTransformRx } from "@/types";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { StatusCodes } from "http-status-codes";
import {
    Suspense,
    type ChangeEvent,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, useNavigate } from "react-router";

export default function () {
    return (
        <ErrorBoundary FallbackComponent={DataTypeTransformCreateError}>
            <Suspense fallback={<Loading />}>
                <DataTypeTransformCreate />
            </Suspense>
        </ErrorBoundary>
    );
}

function DataTypeTransformCreateError({
    error,
    resetErrorBoundary,
}: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            <div>Could not load data type transforms</div>
        </SuspenseError>
    );
}

function DataTypeTransformCreate() {
    const { data: data_types } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPES],
        queryFn: dataService.dataTypesGetAll,
    });
    const { data: transforms } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPE_TRANSFORMS],
        queryFn: dataService.dataTypeTransformsGetAll,
    });
    const queryClient = useQueryClient();
    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    function validate_label(e: ChangeEvent<HTMLInputElement>) {
        const input = e.target;
        const label = input.value;
        input.setCustomValidity("");
        if (label.length === 0) {
            input.setCustomValidity("Label is required");
        }
        if (
            transforms.findIndex((transform) => transform.Label === label) > -1
        ) {
            input.setCustomValidity("Label already exists");
        }
    }

    function validate_source_dest(e: ChangeEvent<HTMLSelectElement>) {
        e.target.setCustomValidity("");
        const srcInput = document.getElementById(
            "source",
        )! as HTMLSelectElement;
        const dstInput = document.getElementById(
            "destination",
        )! as HTMLSelectElement;
        const src = srcInput.value;
        const dst = dstInput.value;
        if (src === dst) {
            e.target.setCustomValidity(
                "Source and destination must be different",
            );
        }
    }

    function create_data_type_transform(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const data = new FormData(e.target);
        dataService.dataTypeTransformCreate(data).then((resp) => {
            if (resp.status == StatusCodes.OK) {
                queryClient.invalidateQueries({
                    queryKey: [QUERY_KEY_DATA_TYPE_TRANSFORMS],
                });

                navigate(-1);
            }
        });
    }

    return (
        <div>
            <div className="flex justify-between px-4 pt-2">
                <h2 className="text-lg">Create data type transform</h2>
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
            <div className="pt-4">
                <form
                    onSubmit={create_data_type_transform}
                    className="flex flex-col gap-2 px-4"
                >
                    <div className="flex flex-col gap-2">
                        <div>
                            <label>
                                <span className="sr-only">Label</span>
                                <input
                                    type="text"
                                    id="label"
                                    name="label"
                                    className="input-basic"
                                    placeholder="Label"
                                    onChange={validate_label}
                                    required
                                />
                            </label>
                        </div>
                        <div className="flex gap-2 items-center">
                            <div>
                                <label>
                                    <span className="sr-only">
                                        Source data type
                                    </span>
                                    <select
                                        id="source"
                                        name="source"
                                        className="input-basic"
                                        onChange={validate_source_dest}
                                        required
                                    >
                                        <option disabled hidden selected>
                                            Source
                                        </option>
                                        {data_types.map((data_type) => (
                                            <option
                                                key={data_type.Id.toString()}
                                                value={data_type.Id.toString()}
                                                title={
                                                    data_type.Description ?? ""
                                                }
                                            >
                                                {data_type.Label}
                                            </option>
                                        ))}
                                    </select>
                                </label>
                            </div>
                            <div>
                                <Icon.ArrowRight />
                            </div>
                            <div>
                                <label>
                                    <span className="sr-only">
                                        Destination data type
                                    </span>
                                    <select
                                        id="destination"
                                        name="destination"
                                        className="input-basic"
                                        onChange={validate_source_dest}
                                        required
                                    >
                                        <option disabled hidden selected>
                                            Destination
                                        </option>
                                        {data_types.map((data_type) => (
                                            <option
                                                key={data_type.Id.toString()}
                                                value={data_type.Id.toString()}
                                                title={
                                                    data_type.Description ?? ""
                                                }
                                            >
                                                {data_type.Label}
                                            </option>
                                        ))}
                                    </select>
                                </label>
                            </div>
                        </div>
                        <div>
                            <label className="flex gap-2">
                                <span>Script</span>
                                <input
                                    type="file"
                                    id="script"
                                    name="script"
                                    accept=".py"
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
                    <div>
                        <button type="submit" className="btn-submit">
                            Create
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}
