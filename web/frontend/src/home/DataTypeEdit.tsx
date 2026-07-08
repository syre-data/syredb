import { Context } from "@/AppStateContext";
import {
    hasDbPermission,
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
    DataStorageExternal,
    DataStorageInternal,
    DbPermissionDataTypeCreate,
    DbPermissionDataTypeModify,
    type DataType,
    type DataTypeExternal,
    type DataTypeInternal,
    type DataTypeSourceUpdate,
    type DataTypeUpdate,
} from "@/types";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { StatusCodes } from "http-status-codes";
import {
    Suspense,
    useContext,
    type ChangeEvent,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, Navigate, useNavigate, useParams } from "react-router";
import * as uuid from "uuid";

export default function () {
    const { data_type_id } = useParams();
    if (!data_type_id) {
        return <Navigate to="/" replace />;
    }

    const ctx = useContext(Context);
    const user = ctx.user;
    const canEditType = hasDbPermission(
        DbPermissionDataTypeModify,
        user.DbPermissions,
    );
    if (!canEditType) {
        console.debug("insufficient permissions to modify data type");
        return <Navigate to="/" replace />;
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

    let content;
    switch (data_type.Storage) {
        case DataStorageInternal:
            content = (
                <DataTypeInternal
                    dataType={data_type as DataTypeInternal}
                    dataTypes={data_types}
                />
            );
            break;
        case DataStorageExternal:
            content = (
                <DataTypeExternal
                    dataType={data_type as DataTypeExternal}
                    dataTypes={data_types}
                />
            );
            break;
        default:
            console.error(`invalid data type storage: ${data_type}`);
            throw new Error("invalid data type storage");
    }

    return (
        <div>
            <div className="flex justify-between px-4 pt-2">
                <div>
                    <h1 className="title">Edit data type</h1>
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
            {content}
        </div>
    );
}

interface DataTypeInternalProps {
    dataType: DataTypeInternal;
    dataTypes: DataType[];
}
function DataTypeInternal({ dataType, dataTypes }: DataTypeInternalProps) {
    const { data: data_schema } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_SCHEMA, dataType.Schema],
        queryFn: async () =>
            await dataService.dataSchemaResourcesGet(dataType.Schema),
    });
    const queryClient = useQueryClient();
    const navigate = useNavigate();

    function validate_label(e: ChangeEvent<HTMLInputElement>) {
        e.target.setCustomValidity("");
        if (
            dataTypes
                .filter((type) => type.Id !== dataType.Id)
                .map((type) => type.Label)
                .includes(e.target.value.trim())
        ) {
            e.target.setCustomValidity("Label already exists");
        }
    }

    async function update(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const data = new FormData(e.target);
        const label = data.get("label")!.toString().trim();
        const active = !!data.get("active");
        const description = data.get("description")!.toString().trim();

        if (!label) {
            console.error("label can not be empty");
            return;
        }

        const update = {
            Id: dataType.Id,
            Label: label,
            Active: active,
            Description: description,
        } satisfies DataTypeUpdate;
        await dataService.dataTypeUpdate(update).then((resp) => {
            if (resp.status === StatusCodes.OK) {
                queryClient.invalidateQueries({
                    queryKey: [QUERY_KEY_DATA_TYPES],
                });

                queryClient.invalidateQueries({
                    queryKey: [QUERY_KEY_DATA_TYPE, dataType.Id],
                });
                navigate(-1);
            }

            console.error(resp);
        });
    }

    return (
        <div>
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
                                        defaultValue={dataType.Label}
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
                                        defaultChecked={dataType.Active}
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
                                            dataType.Description ?? ""
                                        }
                                    ></textarea>
                                </label>
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
                    <div>
                        <Link
                            to={`/data-schema/${dataType.Schema}`}
                            title="To data schema"
                        >
                            <button
                                type="button"
                                className="btn-cmd flex gap-1 items-center"
                            >
                                <Icon.DataSchema />
                                {data_schema?.DataSchema.Label}
                            </button>
                        </Link>
                    </div>
                </div>
            </div>
        </div>
    );
}

interface DataTypeExternalProps {
    dataType: DataTypeExternal;
    dataTypes: DataType[];
}
function DataTypeExternal({ dataType, dataTypes }: DataTypeExternalProps) {
    return <div>TODO</div>;
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
                                <label title="Comma separated list of accepted media types (e.g. '.png, .pdf, text/csv')">
                                    <span className="sr-only">Media types</span>
                                    <input
                                        type="text"
                                        id={`source[${source.Id}][media_types]`}
                                        name={`source[${source.Id}][media_types]`}
                                        className="input-basic"
                                        placeholder="Media type filter"
                                        defaultValue={
                                            source.MediaTypes
                                                ? source.MediaTypes.join(", ")
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
