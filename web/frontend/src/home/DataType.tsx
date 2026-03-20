import {
    MouseButton,
    QUERY_KEY_DATA_SCHEMA,
    QUERY_KEY_DATA_TYPE,
    QUERY_KEY_DATA_TYPES,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import {
    DataSourceCardinalityMultiple,
    DataSourceCardinalitySingle,
    type DataTypeSourceRecord,
    type DataTypeSourceUpdate,
    type DataTypeUpdate,
} from "@/types";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { StatusCodes } from "http-status-codes";
import {
    Suspense,
    type ChangeEvent,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, useNavigate, useParams } from "react-router";
import * as uuid from "uuid";

export default function () {
    const navigate = useNavigate();
    const { data_type_id } = useParams();
    if (data_type_id === undefined) {
        console.log("data type id not present");
        navigate(-1);
        return;
    }

    return (
        <ErrorBoundary FallbackComponent={DataTypeError}>
            <Suspense fallback={<Loading />}>
                <DataType data_type_id={data_type_id} />
            </Suspense>
        </ErrorBoundary>
    );
}

function DataTypeError({ resetErrorBoundary, error }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            Could not load data type resources
        </SuspenseError>
    );
}

interface DataTypeProps {
    data_type_id: string;
}
function DataType({ data_type_id }: DataTypeProps) {
    const { data: data_type } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPE, data_type_id],
        queryFn: async () => dataService.dataTypeGet(data_type_id),
    });
    const { data: data_schema } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_SCHEMA, data_type.Schema],
        queryFn: async () =>
            data_type.Schema === uuid.NIL
                ? null
                : await dataService.dataSchemaResourcesGet(data_type.Schema),
    });
    const { data: data_types } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPES],
        queryFn: dataService.dataTypesGetAll,
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
        e.target.setCustomValidity("");
        if (
            data_types
                .filter((type) => type.Id !== data_type_id)
                .map((type) => type.Label)
                .includes(e.target.value.trim())
        ) {
            e.target.setCustomValidity("Label already exists");
        }
    }

    function download_recipe(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        console.debug("TODO: Download recipe");
    }

    async function update(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const data = new FormData(e.target);
        const label = data.get("label")!.toString().trim();
        const active = !!data.get("active");
        const description = data.get("description")!.toString().trim();
        const sources = data_type.Sources.map((source) => source.Id).map(
            (id) => {
                const description = data
                    .get(`source[${id}][description]`)!
                    .toString()
                    .trim();
                const extension_filter = data
                    .get(`source[${id}][extension_filter]`)!
                    .toString()
                    .split(",")
                    .map((ext) => ext.trim())
                    .filter((ext) => ext.length > 0);
                return {
                    Id: id,
                    Description: description,
                    ExtensionFilter: extension_filter,
                } satisfies DataTypeSourceUpdate;
            },
        );

        if (!label) {
            console.error("label can not be empty");
            return;
        }

        const update = {
            Id: data_type.Id,
            Label: label,
            Active: active,
            Description: description,
            Sources: sources,
        } satisfies DataTypeUpdate;
        await dataService.dataTypeUpdate(update).then((resp) => {
            if (resp.status === StatusCodes.OK) {
                queryClient.invalidateQueries({
                    queryKey: [QUERY_KEY_DATA_TYPES],
                });

                queryClient.invalidateQueries({
                    queryKey: [QUERY_KEY_DATA_TYPE, data_type_id],
                });
                navigate(-1);
            }

            console.error(resp);
        });
    }

    return (
        <div>
            <div className="flex justify-between px-4 pt-2">
                <div>
                    <h2 className="text-lg">Edit data type</h2>
                </div>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        title="Close"
                        onMouseDown={close}
                    >
                        <Icon.Close />
                    </button>
                </div>
            </div>
            <div className="pt-4 px-4 flex gap-2">
                <div>
                    <form className="flex flex-col gap-2" onSubmit={update}>
                        <div className="flex flex-col gap-2">
                            <div>
                                <label>
                                    <span className="sr-only">Label</span>
                                    <input
                                        type="text"
                                        id="label"
                                        name="label"
                                        placeholder="Label"
                                        className="input-basic"
                                        defaultValue={data_type.Label}
                                        onChange={validate_label}
                                        required
                                    />
                                </label>
                            </div>
                            <div>
                                <label title="Allow users to assign this data type to new data">
                                    <input
                                        type="checkbox"
                                        id="active"
                                        name="active"
                                        placeholder="Active"
                                        className="input-basic"
                                        defaultChecked={data_type.Active}
                                    />
                                    <span className="pl-2">Active</span>
                                </label>
                            </div>
                            <div>
                                <label>
                                    <textarea
                                        id="description"
                                        name="description"
                                        placeholder="Description"
                                        className="input-basic"
                                        defaultValue={
                                            data_type.Description ?? ""
                                        }
                                    ></textarea>
                                </label>
                            </div>
                            <div>
                                {data_type.Sources ? (
                                    <DataTypeSources
                                        sources={data_type.Sources}
                                    />
                                ) : (
                                    "(no sources)"
                                )}
                            </div>
                        </div>
                        <div>
                            <button type="submit" className="btn-submit">
                                Save
                            </button>
                        </div>
                    </form>
                </div>
                <div className="flex gap-2">
                    {data_type.Recipe === uuid.NIL ? null : (
                        <div>
                            <button
                                type="button"
                                className="btn-cmd"
                                title="Download recipe"
                                onMouseDown={download_recipe}
                            >
                                <Icon.Gear />
                            </button>
                        </div>
                    )}
                    {data_type.Schema === uuid.NIL ? null : (
                        <div>
                            <Link
                                to={`/data-schema/${data_type.Schema}`}
                                title="To data schema"
                            >
                                <button type="button" className="btn-cmd">
                                    <Icon.DataSchema />
                                </button>
                            </Link>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

interface DataTypeSourcesProps {
    sources: DataTypeSourceRecord[];
}
function DataTypeSources({ sources }: DataTypeSourcesProps) {
    return (
        <fieldset className="flex flex-col gap-2">
            <legend>Sources</legend>

            <ol className="list-decimal px-4">
                {sources.map((source) => (
                    <li key={source.Id.toString()}>
                        <div className="flex flex-col gap-2">
                            <div title="Label">{source.Label}</div>
                            <div className="flex gap-2">
                                <div>
                                    {source.Required ? (
                                        <span title="Required">*</span>
                                    ) : null}
                                </div>
                                <div>
                                    {source.Cardinality ===
                                    DataSourceCardinalitySingle ? (
                                        <Icon.File title="Single file source" />
                                    ) : null}
                                    {source.Cardinality ===
                                    DataSourceCardinalityMultiple ? (
                                        <Icon.Files title="Multiple file source" />
                                    ) : null}
                                </div>
                            </div>
                            <div>
                                <label>
                                    <span className="sr-only">Description</span>
                                    <textarea
                                        id={`source[${source.Id}][description]`}
                                        name={`source[${source.Id}][description]`}
                                        className="input-basic"
                                        placeholder="Description"
                                        defaultValue={source.Description ?? ""}
                                    ></textarea>
                                </label>
                            </div>
                            <div>
                                <label title="Comma separated list of accepted extensions (e.g. 'png, jpg')">
                                    <span className="sr-only">
                                        Extension filter
                                    </span>
                                    <input
                                        type="text"
                                        id={`source[${source.Id}][extension_filter]`}
                                        name={`source[${source.Id}][extension_filter]`}
                                        className="input-basic"
                                        placeholder="Extension filter"
                                        defaultValue={
                                            source.ExtensionFilter
                                                ? source.ExtensionFilter.join(
                                                      ", ",
                                                  )
                                                : ""
                                        }
                                    />
                                </label>
                            </div>
                        </div>
                    </li>
                ))}
            </ol>
        </fieldset>
    );
}
