import { Loading, SuspenseError } from "@/components";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Suspense, useState, type MouseEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import * as common from "@/common";
import dataService from "@/service/data.service";
import classNames from "classnames";
import { Link } from "react-router";
import Icon from "@/icon";
import type { DataSchemaRecord } from "@/types";

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

    return (
        <div className={`group`}>
            <div className="flex gap-2 items-center px-4">
                <h3 className="text-lg font-bold">Data schemas</h3>
                <div
                    className={classNames({
                        "invisible group-hover:visible":
                            data_schemas.length > 0,
                    })}
                >
                    <Link to="/data-schema/create">
                        <button
                            type="button"
                            className="btn-cmd"
                            title="Create data schema"
                        >
                            <Icon.Plus />
                        </button>
                    </Link>
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
                    Create your first data schema by clicking the{" "}
                    <Icon.Plus className="inline" /> above.
                </div>
            </div>
        </div>
    );
}

interface DataSchemasContentProps {
    data_schemas: DataSchemaRecord[];
}
function DataSchemasContent({ data_schemas }: DataSchemasContentProps) {
    return (
        <ul className="grid gap-2 grid-cols-[repeat(4,min-content)]">
            {data_schemas.map((schema, index) => (
                <li
                    key={schema.Id.toString()}
                    className="grid grid-cols-subgrid col-span-full"
                >
                    <DataSchemaContent index={index} schema={schema} />
                </li>
            ))}
        </ul>
    );
}

interface DataSchemaContentProps {
    index: number;
    schema: DataSchemaRecord;
}
function DataSchemaContent({ index, schema }: DataSchemaContentProps) {
    const ROW_SPAN = 2;
    const [expanded, setExpanded] = useState(false);

    function toggle_expand(e: MouseEvent<HTMLDivElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        setExpanded(!expanded);
    }

    const description = schema.Description ?? "(no description)";
    return (
        <div className="grid col-span-full grid-cols-subgrid group/schema-row">
            <div
                className="grid grid-cols-subgrid col-span-2"
                onMouseDown={toggle_expand}
            >
                <div
                    className={classNames({
                        "col-1 pl-4 row-1": true,
                        "invisible hover:visible group-hover/schema-row:visible":
                            !expanded,
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
                <div className="row-1 col-2 whitespace-nowrap cursor-pointer">
                    {schema.Label}
                </div>
            </div>
            <div className="row-1 col-3 whitespace-nowrap">{description}</div>
            <div className="row-1 col-4 invisible group-hover/schema-row:visible">
                <Link to={`/data_schema/${schema.Id}`}>
                    <button
                        type="button"
                        className="btn-cmd"
                        title="Edit data schema"
                    >
                        <Icon.Pen />
                    </button>
                </Link>
            </div>
            <div
                className={classNames({
                    "row-2 col-start-2 -col-end-1 overflow-hidden whitespace-nowrap flex gap-2 transition-[height]": true,
                    "h-0": !expanded,
                })}
            >
                {schema.Schema.map((col, idx) => (
                    <div key={col.label}>
                        <span>{col.label}</span> <span>({col.dtype})</span>
                        {idx === schema.Schema.length - 1 ? "" : " | "}
                    </div>
                ))}
            </div>
        </div>
    );
}
