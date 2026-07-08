import { ErrorBoundary } from "react-error-boundary";
import { useSuspenseQuery, useQueryClient } from "@tanstack/react-query";
import { Suspense, type MouseEvent, useContext } from "react";
import { useNavigate, useParams, Link, Navigate } from "react-router";
import Loading from "../components/Loading";
import { SuspenseError } from "@/components";
import type { FallbackProps } from "react-error-boundary";
import * as types from "@/types";
import * as uuid from "uuid";
import data_service from "@/service/data.service";
import icon from "@/icon";
import * as common from "@/common";
import { Context } from "@/AppStateContext";

export default function () {
    const { data_schema_id } = useParams();
    if (!data_schema_id) {
        return <Navigate to="/" replace />;
    }

    return (
        <ErrorBoundary FallbackComponent={DataSchemaError}>
            <Suspense fallback={<Loading />}>
                <DataSchema data_schema_id={data_schema_id} />
            </Suspense>
        </ErrorBoundary>
    );
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
    const ctx = useContext(Context);
    const user = ctx.user;

    const data_schema = data_schema_resources.DataSchema;

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    const canModifySchema = common.hasDbPermission(
        types.DbPermissionDataSchemaModify,
        user.DbPermissions,
    );
    return (
        <div>
            <div className="pt-2 px-4 flex items-center justify-between">
                <div className="flex gap-2 items-center">
                    <h1 className="title">{data_schema.Label}</h1>
                    <div>
                        {canModifySchema ? (
                            <Link to={`/data-schema/${data_schema_id}/edit`}>
                                <button type="button" className="btn-cmd">
                                    <icon.Pen />
                                </button>
                            </Link>
                        ) : null}
                    </div>
                </div>
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
            <div className="pt-2 px-4">{data_schema.Description}</div>
            <div className="pt-4 px-4 flex gap-2">
                <h2>Cardinality</h2>
                <div>{data_schema.Cardinality}</div>
            </div>
            <div className="pt-2 px-4">
                <h2 className="text-lg">Schema</h2>
                <div className="flex gap-2">
                    {data_schema.Fields.map((col, idx) => (
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
