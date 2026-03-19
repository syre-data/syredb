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

interface DataTypeSourceInput {
    id: number;
    label: string;
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

    const [sources, setSources] = useState<DataTypeSourceInput[]>([]);

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

        const sources_input = [];
        for (const idx in sources) {
            const key = `source[${idx}]`;
            const label = data.get(`${key}[label]`)!.toString().trim();
            const cardinality_str = data.get(`${key}[cardinality]`)!.toString();
            const required = !!data.get(`${key}[required]`);
            const description = data
                .get(`${key}[description]`)!
                .toString()
                .trim();
            const extension_filter = data
                .get(`${key}[extension_filter]`)!
                .toString()
                .split(",")
                .map((ext) => ext.trim())
                .filter((ext) => ext.length > 0);

            let cardinality: types.DataSourceCardinality;
            switch (cardinality_str) {
                case types.DataSourceCardinalitySingle:
                    cardinality = types.DataSourceCardinalitySingle;
                    break;
                case types.DataSourceCardinalityMultiple:
                    cardinality = types.DataSourceCardinalityMultiple;
                    break;
                default:
                    const input = document.getElementById(
                        `${key}[cardinality]`,
                    )! as HTMLFieldSetElement;
                    input.setCustomValidity("Invalid value");
                    return;
            }

            sources_input.push({
                Cardinality: cardinality,
                Required: required,
                Label: label,
                Description: description,
                ExtensionFilter: extension_filter,
            } satisfies types.DataTypeSourceCreate);
        }

        dataService
            .dataTypeCreate(
                label,
                sources_input,
                description,
                data_schema,
                recipe,
            )
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
                        <SourcesList
                            sources={sources}
                            setSources={setSources}
                        />
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

interface SourcesListProps {
    sources: DataTypeSourceInput[];
    setSources: React.Dispatch<React.SetStateAction<DataTypeSourceInput[]>>;
}
function SourcesList({ sources, setSources }: SourcesListProps) {
    function add_source(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        const id =
            sources
                .map((source) => source.id)
                .reduce((a, b) => Math.max(a, b), -1) + 1;
        const source = {
            id,
            label: "",
        } satisfies DataTypeSourceInput;
        setSources([...sources, source]);
    }

    function remove_source(id: number, e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        setSources(sources.filter((source) => source.id !== id));
    }

    function update_label(idx: number, e: ChangeEvent<HTMLInputElement>) {
        const source = sources[idx]!;
        source.label = e.target.value;

        e.target.setCustomValidity("");
        const matching = sources.findIndex(
            (s) => s.id !== source.id && s.label === e.target.value,
        );
        if (source.label && matching > -1) {
            e.target.setCustomValidity(
                `Label must be unique. Duplicates ${matching + 1}.`,
            );
            e.target.reportValidity();
        }
    }

    return (
        <fieldset>
            <div className="flex gap-2">
                <legend>Sources</legend>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        title="Add source"
                        onMouseDown={add_source}
                    >
                        <icon.Plus />
                    </button>
                </div>
            </div>

            <ol className="flex flex-col gap-2 list-decimal px-4">
                {sources.map((source, idx) => {
                    return (
                        <li key={source.id}>
                            <div className="flex flex-col gap-2 pt-2 group">
                                <div>
                                    <div className="flex gap-2 whitespace-nowrap">
                                        <div>
                                            <label>
                                                <span className="sr-only">
                                                    Label
                                                </span>
                                                <input
                                                    type="text"
                                                    id={`source[${idx}][label]`}
                                                    name={`source[${idx}][label]`}
                                                    placeholder="Label"
                                                    className="input-basic"
                                                    onChange={(e) =>
                                                        update_label(idx, e)
                                                    }
                                                    required
                                                />
                                            </label>
                                        </div>
                                        <div>
                                            <button
                                                type="button"
                                                className="btn-cmd invisible group-hover:visible"
                                                onMouseDown={(e) =>
                                                    remove_source(source.id, e)
                                                }
                                            >
                                                <icon.Trash />
                                            </button>
                                        </div>
                                    </div>
                                </div>
                                <div>
                                    <fieldset>
                                        <legend className="sr-only">
                                            Cardinality
                                        </legend>
                                        <div className="flex gap-2">
                                            <label title="Accept a single file">
                                                <input
                                                    type="radio"
                                                    id={`source[${idx}][cardinality][single]`}
                                                    name={`source[${idx}][cardinality]`}
                                                    value={
                                                        types.DataSourceCardinalitySingle
                                                    }
                                                    defaultChecked={true}
                                                />
                                                <span className="pl-2">
                                                    Single
                                                </span>
                                            </label>
                                            <label title="Accepts multiple files">
                                                <input
                                                    type="radio"
                                                    id={`source[${idx}][cardinality][multiple]`}
                                                    name={`source[${idx}][cardinality]`}
                                                    value={
                                                        types.DataSourceCardinalityMultiple
                                                    }
                                                />
                                                <span className="pl-2">
                                                    Multiple
                                                </span>
                                            </label>
                                        </div>
                                    </fieldset>
                                </div>
                                <div>
                                    <label>
                                        <input
                                            type="checkbox"
                                            id={`source[${idx}][required]`}
                                            name={`source[${idx}][required]`}
                                            className="input-basic"
                                            defaultChecked={true}
                                        />
                                        <span className="pl-2">Required</span>
                                    </label>
                                </div>
                                <div>
                                    <label>
                                        <span className="sr-only">
                                            Description
                                        </span>
                                        <textarea
                                            id={`source[${idx}][description]`}
                                            name={`source[${idx}][description]`}
                                            className="input-basic"
                                            placeholder="Description"
                                        ></textarea>
                                    </label>
                                </div>
                                <div>
                                    <label title="Comma separated list of accepted file extensions (e.g. 'jpg, png, bmp')">
                                        <span className="sr-only">
                                            Extension filter
                                        </span>
                                        <input
                                            type="text"
                                            id={`source[${idx}][extension_filter]`}
                                            name={`source[${idx}][extension_filter]`}
                                            className="input-basic"
                                            placeholder="Extension filter"
                                        />
                                    </label>
                                </div>
                            </div>
                        </li>
                    );
                })}
            </ol>
        </fieldset>
    );
}
