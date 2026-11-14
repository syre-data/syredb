import { Loading, StandardError } from "@/components";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Suspense, useContext, useState, type MouseEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import * as common from "@/common";
import dataService from "@/service/data.service";
import classNames from "classnames";
import { Link } from "react-router";
import Icon from "@/icon";
import {
    DbPermissionDataSchemaCreate,
    DbPermissionDataSchemaModify,
    DbPermissionOwner,
    type DataSchema,
} from "@/types";
import { Context } from "@/AppStateContext";
import { MainNav, PageDataSchemas } from "@/components/Nav";

export default function () {
    return (
        <div className="flex">
            <MainNav
                current={PageDataSchemas}
                className="sticky top-0 h-screen py-2"
            />
            <main className="grow">
                <ErrorBoundary FallbackComponent={DataSchemaError}>
                    <Suspense fallback={<Loading />}>
                        <DataSchemas />
                    </Suspense>
                </ErrorBoundary>
            </main>
        </div>
    );
}

function DataSchemaError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <StandardError
            resetErrorBoundary={resetErrorBoundary}
            error={error as Error}
            title={"Could not load data schemas"}
        />
    );
}

function DataSchemas() {
    const { data: dataSchemas } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_SCHEMA],
        queryFn: dataService.dataSchemasGetAll,
    });

    const ctx = useContext(Context);
    const user = ctx.user;

    const isDbOwner = common.hasDbPermission(
        DbPermissionOwner,
        user.DbPermissions,
    );
    const canCreateSchema =
        isDbOwner ||
        common.hasDbPermission(
            DbPermissionDataSchemaCreate,
            user.DbPermissions,
        );
    const canModifySchemas =
        isDbOwner ||
        common.hasDbPermission(
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
            </div>
            <div>
                {dataSchemas.length === 0 ? (
                    <DataSchemasEmpty />
                ) : (
                    <DataSchemasContent
                        canModifySchemas={canModifySchemas}
                        dataSchemas={dataSchemas}
                    />
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
    dataSchemas: DataSchema[];
    canModifySchemas: boolean;
}
function DataSchemasContent({
    dataSchemas,
    canModifySchemas,
}: DataSchemasContentProps) {
    return (
        <table className="table-std">
            <tbody>
                {dataSchemas.map((schema, index) => (
                    <DataSchemaItem
                        key={schema.Id.toString()}
                        canModifySchemas={canModifySchemas}
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
    canModifySchemas: boolean;
}
function DataSchemaItem({
    index,
    schema,
    canModifySchemas,
}: DataSchemaItemProps) {
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
                                "transition-[rotate]": true,
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
                    {canModifySchemas ? (
                        <Link to={`/data-schema/${schema.Id}`}>
                            <button
                                type="button"
                                className="btn-cmd"
                                title="Edit data schema"
                            >
                                <Icon.Eye />
                            </button>
                        </Link>
                    ) : null}
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
