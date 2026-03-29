import {
    MouseButton,
    QUERY_KEY_DATA_TYPE_TRANSFORMS,
    QUERY_KEY_DATA_TYPES,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import type { DataTypeTransformCreate, DataTypeTransformRecord } from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
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
    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
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

    function create(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const data = new FormData(e.target);
        const label = data.get("label")!.toString().trim();
        const source = data.get("source")!.toString();
        const destination = data.get("destination")!.toString();
        let description = data.get("description")!.toString().trim();
        const transform = {
            Source: source,
            Destination: destination,
            Label: label,
            Description: description,
        } satisfies DataTypeTransformCreate;

        dataService.dataTypeTransformCreate(transform);
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
                <form onSubmit={create} className="flex flex-col gap-2 px-4">
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
                                    required
                                />
                            </label>
                        </div>
                        <div>
                            <label className="flex gap-2">
                                <span>Source</span>
                                <select
                                    id="source"
                                    name="source"
                                    className="input-basic"
                                    onChange={validate_source_dest}
                                >
                                    {data_types.map((data_type) => (
                                        <option
                                            key={data_type.Id.toString()}
                                            value={data_type.Id.toString()}
                                            title={data_type.Description ?? ""}
                                        >
                                            {data_type.Label}
                                        </option>
                                    ))}
                                </select>
                            </label>
                        </div>
                        <div>
                            <label className="flex gap-2">
                                <span>Destination</span>
                                <select
                                    id="destination"
                                    name="destination"
                                    className="input-basic"
                                    onChange={validate_source_dest}
                                >
                                    {data_types.map((data_type) => (
                                        <option
                                            key={data_type.Id.toString()}
                                            value={data_type.Id.toString()}
                                            title={data_type.Description ?? ""}
                                        >
                                            {data_type.Label}
                                        </option>
                                    ))}
                                </select>
                            </label>
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
