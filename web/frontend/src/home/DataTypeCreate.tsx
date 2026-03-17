import type { ChangeEvent, MouseEvent, SubmitEvent } from "react";
import { Suspense, useRef, useState } from "react";
import { useNavigate } from "react-router";
import * as common from "@/common";
import data_service from "@/service/data.service";
import icon from "@/icon";
import * as types from "@/types";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Loading, SuspenseError } from "@/components";
import dataService from "@/service/data.service";
import * as uuid from "uuid";
import { StatusCodes } from "http-status-codes";

export default function () {
    return (
        <ErrorBoundary FallbackComponent={DataTypeCreateError}>
            <Suspense fallback={<Loading />}>
                <DataTypeCreate />
            </Suspense>
        </ErrorBoundary>
    );
}

function DataTypeCreateError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            Could not load resources
        </SuspenseError>
    );
}

function DataTypeCreate() {
    const { data: data_types } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_TYPES],
        queryFn: dataService.dataTypesGetAll,
    });
    const { data: data_schemas } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_SCHEMAS],
        queryFn: dataService.dataSchemasGetAll,
    });
    const navigate = useNavigate();

    function cancel(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    function create_data_type(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const labelInput = document.getElementById(
            "label",
        )! as HTMLInputElement;
        const dataSchemaInput = document.getElementById(
            "data_schema",
        )! as HTMLSelectElement;
        const recipeInput = document.getElementById(
            "recipe",
        ) as HTMLInputElement;
        labelInput.setCustomValidity("");
        dataSchemaInput.setCustomValidity("");

        let data = new FormData(e.target);
        let label = data.get("label")!.toString().trim();
        let description = data.get("description")!.toString().trim();
        let schema_str = data.get("data_schema")!.toString();
        let recipe = recipeInput.files ? recipeInput.files[0] : undefined;

        if (label.length === 0) {
            labelInput.setCustomValidity("Label must be set");
        }
        if (data_types.findIndex((type) => type.Label === label) > -1) {
            labelInput.setCustomValidity("Duplicate label");
        }

        const data_schema =
            schema_str === "" ? undefined : uuid.parse(schema_str);
        if (
            data_schema &&
            data_schemas.findIndex((s) => s.Id === data_schema) < 0
        ) {
            dataSchemaInput.setCustomValidity("Data schema does not exist");
        }

        dataService
            .dataTypeCreate(label, description, data_schema, recipe)
            .then((resp) => {
                if (resp.status === StatusCodes.OK) {
                    navigate(-1);
                    return;
                }

                console.error(resp);
            });
    }

    return (
        <div>
            <div>
                <h2>Create data type</h2>
            </div>
            <form className="flex flex-col gap-4" onSubmit={create_data_type}>
                <div className="px-4 pt-2 flex flex-col gap-2">
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
                        <label>
                            <span className="sr-only">Description</span>
                            <textarea
                                id="description"
                                name="description"
                                className="input-basic"
                                placeholder="Description"
                            ></textarea>
                        </label>
                    </div>
                    <div>
                        <label className="flex gap-2">
                            <span>Output schema</span>
                            <select
                                id="data_schema"
                                name="data_schema"
                                className="input-basic"
                            >
                                <option value="">(none)</option>
                                {data_schemas.map((schema) => (
                                    <option
                                        key={schema.Id.toString()}
                                        value={schema.Id.toString()}
                                    >
                                        {schema.Label}
                                    </option>
                                ))}
                            </select>
                        </label>
                    </div>
                    <div>
                        <label className="flex gap-2">
                            <span>Recipe</span>
                            <input
                                type="file"
                                id="recipe"
                                name="recipe"
                                className="input-basic"
                                accept=".py"
                            />
                        </label>
                    </div>
                </div>
                <div className="px-4 flex gap-2">
                    <button type="submit" className="btn-submit">
                        Create data type
                    </button>
                    <button
                        type="button"
                        className="btn-submit"
                        onMouseDown={cancel}
                    >
                        Cancel
                    </button>
                </div>
            </form>
        </div>
    );
}
