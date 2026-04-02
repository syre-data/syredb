import { ErrorBoundary } from "react-error-boundary";
import { useSuspenseQuery, useQueryClient } from "@tanstack/react-query";
import {
    Suspense,
    type MouseEvent,
    type SubmitEvent,
    type ChangeEvent,
    useState,
    useRef,
} from "react";
import { useNavigate, useParams, Link } from "react-router";
import Loading from "../components/Loading";
import { SuspenseError } from "@/components";
import type { FallbackProps } from "react-error-boundary";
import * as types from "@/types";
import * as uuid from "uuid";
import data_service from "@/service/data.service";
import icon from "@/icon";
import * as common from "@/common";
import dataService from "@/service/data.service";
import { StatusCodes } from "http-status-codes";

export default function () {
    const navigate = useNavigate();
    const { data_schema_id } = useParams();
    if (data_schema_id) {
        return (
            <ErrorBoundary FallbackComponent={DataSchemaError}>
                <Suspense fallback={<Loading />}>
                    <DataSchema data_schema_id={data_schema_id} />
                </Suspense>
            </ErrorBoundary>
        );
    } else {
        navigate("/");
        return null;
    }
}

function DataSchemaError({ error, resetErrorBoundary }: FallbackProps) {
    const err = error as types.AppError;

    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="pt-4 text-center"
        >
            <div>Could not load data schema</div>
            <div>{err.Message}</div>
        </SuspenseError>
    );
}

interface DataSchemaProps {
    data_schema_id: uuid.UUIDTypes;
}
function DataSchema({ data_schema_id }: DataSchemaProps) {
    const { data: data_schema_resources } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_SCHEMA_RESOURCES, data_schema_id],
        queryFn: async () =>
            data_service.dataSchemaResourcesGet(data_schema_id),
    });
    const { data: data_schemas } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_SCHEMAS],
        queryFn: async () => data_service.dataSchemasGetAll(),
    });
    const queryClient = useQueryClient();
    const navigate = useNavigate();
    const data_schema = data_schema_resources.DataSchema;

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    function validate_label(e: ChangeEvent<HTMLInputElement>) {
        const input = e.target;
        input.setCustomValidity("");

        if (
            data_schemas
                .filter((schema) => schema.Id !== data_schema.Id)
                .map((schema) => schema.Label)
                .includes(input.value.trim())
        ) {
            input.setCustomValidity("Duplicate label, labels must be unique");
        }
    }

    async function update(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        const labelInput = document.getElementById(
            "label",
        )! as HTMLInputElement;
        labelInput.setCustomValidity("");

        const data = new FormData(e.target);
        const label = data.get("label")!.toString().trim();
        const description_str = data.get("description")!.toString().trim();

        if (!label) {
            labelInput.setCustomValidity("Label is required");
            return;
        }
        if (
            data_schemas
                .filter((schema) => schema.Id !== data_schema.Id)
                .map((schema) => schema.Label)
                .includes(label)
        ) {
            labelInput.setCustomValidity(
                "Duplicate label, labels must be unique",
            );
            return;
        }

        const description =
            description_str.length === 0 ? undefined : description_str;
        const update = {
            Id: data_schema_id,
            Label: label,
            Description: description,
        };
        await dataService.dataSchemaUpdate(update).then((resp) => {
            if (resp.status === StatusCodes.OK) {
                queryClient.invalidateQueries({
                    queryKey: [common.QUERY_KEY_DATA_SCHEMAS],
                });
                queryClient.invalidateQueries({
                    queryKey: [common.QUERY_KEY_DATA_SCHEMA, data_schema_id],
                });

                navigate(-1);
            }
        });
    }

    return (
        <div>
            <div className="pt-2 px-4 flex justify-between">
                <h1 className="text-xl">Data schema</h1>
                <div className="flex gap-2">
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
                <form onSubmit={update} className="flex flex-col gap-2 px-4">
                    <div className="flex flex-col gap-2">
                        <div>
                            <label>
                                <span className="sr-only">Label</span>
                                <input
                                    type="text"
                                    id="label"
                                    name="label"
                                    className="input-basic"
                                    defaultValue={data_schema.Label}
                                    placeholder="Label"
                                    onChange={validate_label}
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
                                    className="input-basic"
                                    placeholder="Description"
                                    defaultValue={data_schema.Description}
                                ></textarea>
                            </label>
                        </div>
                    </div>
                    <div>
                        <div>
                            <button type="submit" className="btn-submit">
                                Save
                            </button>
                        </div>
                    </div>
                </form>
            </div>
            <div className="pt-4 px-4">
                <h2>Cardinality</h2>
                <div>{data_schema.Cardinality}</div>
            </div>
            <div className="pt-2 px-4">
                <h2 className="text-lg">Schema</h2>
                <div className="flex gap-2">
                    {data_schema.Schema.map((col, idx) => (
                        <>
                            {idx !== 0 ? <div>|</div> : null}
                            <div
                                key={col.Label}
                                className="flex gap-1"
                                title={col.Description}
                            >
                                <div>{col.Label}</div>
                                <div>({value_type_to_string(col.DType)})</div>
                            </div>
                        </>
                    ))}
                </div>
            </div>
        </div>
    );
}

function value_type_to_string(data_type: types.ValueType): string {
    switch (data_type) {
        case types.PropertyTypeString:
            return "string";
        case types.PropertyTypeInt:
            return "int";
        case types.PropertyTypeUint:
            return "uint";
        case types.PropertyTypeFloat:
            return "float";
        case types.ValueTypeBoolean:
            return "boolean";
        case types.PropertyTypeTimestamp:
            return "timestamp";
    }
}
