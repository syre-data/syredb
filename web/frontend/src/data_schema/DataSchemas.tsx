import { Loading, SuspenseError } from "@/components";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Suspense, useContext, useState, type MouseEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import * as common from "@/common";
import dataService from "@/service/data.service";
import classNames from "classnames";
import { Link, useNavigate } from "react-router";
import Icon from "@/icon";
import {
    DbPermissionDataSchemaCreate,
    DbPermissionDataSchemaModify,
    type DataSchema,
} from "@/types";
import { Context } from "@/AppStateContext";

export default function () {
    return (
        <ErrorBoundary FallbackComponent={DataSchemaError}>
            <Suspense fallback={<Loading />}>
                <DataSchemas />
            </Suspense>
        </ErrorBoundary>
    );
}

function DataSchemaError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            Could not load data schemas
        </SuspenseError>
    );
}

function DataSchemas() {
    const { data: data_schemas } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_SCHEMA],
        queryFn: dataService.dataSchemasGetAll,
    });

    const navigate = useNavigate();
    const ctx = useContext(Context);
    const user = ctx.user;

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        console.log("back");
        navigate(-1);
    }

    const canCreateSchema = common.hasDbPermission(
        DbPermissionDataSchemaCreate,
        user.DbPermissions,
    );
    const canModifySchema = common.hasDbPermission(
        DbPermissionDataSchemaModify,
        user.DbPermissions,
    );
    return (
        <div>
            <div className="pt-2 px-4 flex justify-between">
                <div className="flex gap-2 items-center">
                    <h1 className="title">Data schemas</h1>
                    <div>
                        {canCreateSchema ? (
                            <Link to="/data-schema/create">
                                <button
                                    type="button"
                                    className="btn-cmd"
                                    title="Create data schema"
                                >
                                    <Icon.Plus />
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
            <div>
                {data_schemas.length === 0 ? (
                    <DataSchemasEmpty />
                ) : (
                    <DataSchemasContent data_schemas={data_schemas} />
                )}
            </div>
        </div>
    );
}

function DataSchemasEmpty() {
    return (
        <div className="px-4">
            <div>
                <div>No data schemas</div>
                <div>
                    Create one by clicking the <Icon.Plus className="inline" />{" "}
                    above.
                </div>
            </div>
        </div>
    );
}

interface DataSchemasContentProps {
    data_schemas: DataSchema[];
}
function DataSchemasContent({ data_schemas }: DataSchemasContentProps) {
    return (
        <table className="table-std">
            <tbody>
                {data_schemas.map((schema, index) => (
                    <DataSchemaItem
                        key={schema.Id.toString()}
                        index={index}
                        schema={schema}
                    />
                ))}
            </tbody>
        </table>
    );
}

interface DataSchemaItemProps {
    index: number;
    schema: DataSchema;
}
function DataSchemaItem({ index, schema }: DataSchemaItemProps) {
    const [expanded, setExpanded] = useState(false);

    function toggle_expand(e: MouseEvent<HTMLTableCellElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        setExpanded(!expanded);
    }

    const description = schema.Description ?? "(no description)";
    return (
        <>
            <tr
                className={classNames({
                    group: true,
                })}
            >
                <td
                    className={classNames({
                        "w-0": true,
                        "border-b-0!": expanded,
                    })}
                    onMouseDown={toggle_expand}
                >
                    <div
                        className={classNames({
                            "invisible group-hover:visible": !expanded,
                        })}
                    >
                        <button
                            type="button"
                            className={classNames({
                                "btn-cmd transition-[rotate]": true,
                                "-rotate-90": !expanded,
                            })}
                        >
                            <Icon.CaretDown />
                        </button>
                    </div>
                </td>
                <td
                    className={classNames({
                        "whitespace-nowrap cursor-pointer font-semibold w-0": true,
                        "border-b-0!": expanded,
                    })}
                    onMouseDown={toggle_expand}
                >
                    {schema.Label}
                </td>
                <td
                    className={classNames({
                        "border-b-0!": expanded,
                    })}
                >
                    {description}
                </td>
                <td
                    className={classNames({
                        "border-b-0!": expanded,
                    })}
                >
                    <Link to={`/data-schema/${schema.Id}`}>
                        <button
                            type="button"
                            className="btn-cmd"
                            title="Edit data schema"
                        >
                            <Icon.Eye />
                        </button>
                    </Link>
                </td>
            </tr>
            <tr
                className={classNames({
                    "overflow-hidden": true,
                    collapse: !expanded,
                })}
            >
                <td></td>
                <td colSpan={3}>
                    {schema.Fields.map((col, idx) => (
                        <div
                            key={col.Label}
                            className="inline"
                            title={col.Description}
                        >
                            <span>{col.Label}</span> <span>({col.DType})</span>
                            {idx === schema.Fields.length - 1 ? "" : " | "}
                        </div>
                    ))}
                </td>
            </tr>
        </>
    );
}
