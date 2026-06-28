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
    DbPermissionDataTypeModify,
    type DataType,
    type DataTypeExternal,
    type DataTypeExternalSourceRx,
    type DataTypeInternal,
} from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Suspense, useContext, type MouseEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, useNavigate, useParams } from "react-router";

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
    const { data: data_types } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPES],
        queryFn: dataService.dataTypesGetAll,
    });

    switch (data_type.Storage) {
        case DataStorageInternal:
            return (
                <div>
                    <DataTypeCommon dataType={data_type} />
                    <DataTypeInternal
                        dataType={data_type as DataTypeInternal}
                    />
                </div>
            );
        case DataStorageExternal:
            return (
                <div>
                    <DataTypeCommon dataType={data_type} />
                    <DataTypeExternal
                        dataType={data_type as DataTypeExternal}
                    />
                </div>
            );
        default:
            console.error(`invalid data type storage: ${data_type}`);
            throw new Error("invalid data type storage");
    }
}

interface DataTypeCommonProps {
    dataType: DataTypeInternal | DataTypeExternal;
}
function DataTypeCommon({ dataType }: DataTypeCommonProps) {
    const navigate = useNavigate();
    const ctx = useContext(Context);
    const user = ctx.user;

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    const canModifyType = hasDbPermission(
        DbPermissionDataTypeModify,
        user.DbPermissions,
    );

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <div className="flex gap-2 items-center">
                    <h1>{dataType.Label}</h1>
                    <div>
                        {dataType.Active ? (
                            <Icon.CircleCheck />
                        ) : (
                            <Icon.CircleX />
                        )}
                    </div>
                    <div>
                        {canModifyType ? (
                            <Link to={`/data-type/${dataType.Id}/edit`}>
                                <button type="button" className="btn-cmd">
                                    <Icon.Pen />
                                </button>
                            </Link>
                        ) : null}
                    </div>
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
            <div className="px-4 pt-2">{dataType.Description}</div>
        </div>
    );
}

interface DataTypeInternalProps {
    dataType: DataTypeInternal;
}
function DataTypeInternal({ dataType }: DataTypeInternalProps) {
    const { data: data_schema } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_SCHEMA, dataType.Schema],
        queryFn: async () =>
            await dataService.dataSchemaResourcesGet(dataType.Schema),
    });

    return (
        <div>
            <div className="px-4 pt-2">
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
    );
}

interface DataTypeExternalProps {
    dataType: DataTypeExternal;
}
function DataTypeExternal({ dataType }: DataTypeExternalProps) {
    return (
        <div className="px-4">
            <div>
                <h2>Sources</h2>
            </div>
            <ol className="list-decimal px-4">
                {dataType.Sources.map((source) => (
                    <DataTypeSource
                        key={source.Id.toString()}
                        source={source}
                    />
                ))}
            </ol>
        </div>
    );
}

interface DataTypeSourceProps {
    source: DataTypeExternalSourceRx;
}
function DataTypeSource({ source }: DataTypeSourceProps) {
    return (
        <li>
            <div>
                <div className="flex items-center">
                    <div className="flex">
                        <div title="Label">{source.Label}</div>
                        <div>
                            {source.Required ? (
                                <span title="Required">*</span>
                            ) : null}
                        </div>
                    </div>
                    <div className="pl-2">
                        {source.Cardinality === DataSourceCardinalitySingle ? (
                            <Icon.File title="Single file source" />
                        ) : null}
                        {source.Cardinality ===
                        DataSourceCardinalityMultiple ? (
                            <Icon.Files title="Multiple file source" />
                        ) : null}
                    </div>
                </div>
                <div>{source.Description ?? "(no description)"}</div>
                <div>
                    Media types
                    <span className="pl-2">
                        {source.MediaTypes
                            ? source.MediaTypes.join(", ")
                            : "(no media type filter)"}
                    </span>
                </div>
            </div>
        </li>
    );
}
