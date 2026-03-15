import * as types from "@/types";
import { Suspense, useContext, useState } from "react";
import type { MouseEvent } from "react";
import { ErrorBoundary } from "react-error-boundary";
import type {
    FallbackProps as ErrorBoundaryProps,
    FallbackProps,
} from "react-error-boundary";
import { Link } from "react-router";
import icon from "../icon";
import * as appStateCtx from "../AppStateContext";
import * as common from "../common";
import { useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";
import project_service from "@/service/project.service";
import data_service from "@/service/data.service";
import { SuspenseError } from "@/components";

export default function Dashboard() {
    const appState = useContext(appStateCtx.Context);
    const canCreateOrModifyDataSchema =
        common.has_db_permission(
            types.DbPermissionDataSchemaCreate,
            appState.user.DbPermissions,
        ) ||
        common.has_db_permission(
            types.DbPermissionDataSchemaModify,
            appState.user.DbPermissions,
        );
    const canCreateOrModifyDataTypes =
        common.has_db_permission(
            types.DbPermissionDataTypeCreate,
            appState.user.DbPermissions,
        ) ||
        common.has_db_permission(
            types.DbPermissionDataTypeModify,
            appState.user.DbPermissions,
        );

    const canModifyUsers = common.has_db_permission(
        types.DbPermissionUserModify,
        appState.user.DbPermissions,
    );

    return (
        <div>
            <div className="flex gap-2 text-xl">
                <h2 className="px-4 text-lg font-bold grow">Dashboard</h2>
                <Nav />
            </div>
            <main>
                <UserProjects />
            </main>
        </div>
    );
}

function Nav() {
    return (
        <div className="flex gap-2">
            <div>
                <Link to="/settings" title="Settings">
                    <button type="button" className="btn-cmd">
                        <icon.Gear />
                    </button>
                </Link>
            </div>
        </div>
    );
}

function UserProjects() {
    return (
        <div>
            <div className="flex gap-2 items-baseline px-4">
                <h3 className="pb-2 text-lg font-bold">Projects</h3>
                <div className="flex gap-1">
                    <Link to="/project/create">
                        <button
                            type="button"
                            className="btn-cmd block"
                            title="Create new project"
                        >
                            <icon.Plus />
                        </button>
                    </Link>
                </div>
            </div>
            <div>
                <ErrorBoundary FallbackComponent={UserProjectsError}>
                    <Suspense fallback={<LoadingUserProjects />}>
                        <UserProjectsInner />
                    </Suspense>
                </ErrorBoundary>
            </div>
        </div>
    );
}

function UserProjectsError({ error, resetErrorBoundary }: ErrorBoundaryProps) {
    const err = error as common.BackendError;
    console.error(err);

    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center"
        >
            <div>Could not get your projects</div>
            <div>{err.message}</div>
        </SuspenseError>
    );
}

function LoadingUserProjects() {
    return <div className="text-center">Loading projects</div>;
}

function UserProjectsInner() {
    const { data: projects } = useSuspenseQuery({
        queryKey: ["user_projects"],
        queryFn: project_service.getUserProjects,
    });

    return (
        <div>
            {projects.length == 0 ? (
                <UserProjectsEmpty />
            ) : (
                <UserProjectsDeck projects={projects} />
            )}
        </div>
    );
}

function UserProjectsEmpty() {
    return (
        <div className="text-center">
            <Link to="/project/create">
                <button
                    type="button"
                    className="cursor-pointer border px-1 py-0.5"
                >
                    Create your first project
                </button>
            </Link>
        </div>
    );
}

interface UserProjectsDeckProps {
    projects: types.Project[];
}
function UserProjectsDeck({ projects }: UserProjectsDeckProps) {
    return (
        <div className="flex gap-2 px-4">
            {projects.map((project) => (
                <ProjectCard key={project.Id.toString()} project={project} />
            ))}
        </div>
    );
}

interface ProjectCardProps {
    project: types.Project;
}
function ProjectCard({ project }: ProjectCardProps) {
    return (
        <div className="border rounded-sm">
            <Link to={`/project/${project.Id}`}>
                <div className="px-2 py-1">
                    <h4 className="font-bold">{project.Label}</h4>
                    <div>{project.Description}</div>
                </div>
            </Link>
        </div>
    );
}

interface DataTypesProps {
    className?: string;
}
function DataTypes({ className }: DataTypesProps) {
    return (
        <div className={className ?? ""}>
            <ErrorBoundary FallbackComponent={DataTypesError}>
                <Suspense fallback={<DataTypesLoading />}>
                    <DataTypesInner />
                </Suspense>{" "}
            </ErrorBoundary>
        </div>
    );
}

function DataTypesError({ error, resetErrorBoundary }: ErrorBoundaryProps) {
    const err = error as common.BackendError;
    console.error(err);

    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center"
        >
            <div>Could not get data types</div>
            <div>{err.message}</div>
        </SuspenseError>
    );
}

function DataTypesLoading() {
    return (
        <div>
            <div className="flex gap-2 items-center px-4">
                <h3 className="text-lg font-bold">Data types</h3>
            </div>
            <div className="text-center">Loading</div>
        </div>
    );
}

function DataTypesInner() {
    const { data: data_types } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_TYPES],
        queryFn: data_service.dataTypesGetAll,
    });

    return (
        <div className={`group`}>
            <div className="flex gap-2 items-center px-4">
                <h3 className="text-lg font-bold">Data types</h3>
                <div
                    className={classNames({
                        "invisible group-hover:visible": data_types.length > 0,
                    })}
                >
                    <Link to="/data-type">
                        <button
                            type="button"
                            className="btn-cmd"
                            title="Edit data types"
                        >
                            <icon.Pen />
                        </button>
                    </Link>
                </div>
            </div>
            {data_types.length === 0 ? (
                <DataTypesEmpty />
            ) : (
                <DataTypesContent data_types={data_types} />
            )}
        </div>
    );
}

function DataTypesEmpty() {
    return (
        <div className="px-4">
            <div>
                <div>No data types</div>
                <div>
                    Create some by clicking the <icon.Pen className="inline" />{" "}
                    above.
                </div>
            </div>
        </div>
    );
}

interface DataTypesContentProps {
    data_types: types.DataType[];
}
function DataTypesContent({ data_types }: DataTypesContentProps) {
    return (
        <ul>
            {data_types.map((data_type) => (
                <li key={data_type.Id.toString()} className="px-4">
                    {data_type.Description}
                </li>
            ))}
        </ul>
    );
}

interface DataSchemasProps {
    className?: string;
}
function DataSchemas({ className }: DataSchemasProps) {
    return (
        <div className={className ?? ""}>
            <ErrorBoundary FallbackComponent={DataSchemaError}>
                <Suspense fallback={<DataSchemaLoading />}>
                    <DataSchemaInner />
                </Suspense>
            </ErrorBoundary>
        </div>
    );
}

function DataSchemaError({ error, resetErrorBoundary }: FallbackProps) {
    const err = error as common.BackendError;
    console.error(err);
    return (
        <div className="text-center">
            <div>Could not get data schemas</div>
            <div>{err.message}</div>
        </div>
    );
}

function DataSchemaLoading() {
    return (
        <div>
            <h3 className="text-lg font-bold">Data schemas</h3>
            <div className="text-center">Loading</div>
        </div>
    );
}

function DataSchemaInner() {
    const { data: data_schemas } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_SCHEMA],
        queryFn: data_service.dataSchemasGetAll,
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
                            <icon.Plus />
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
                    <icon.Plus className="inline" /> above.
                </div>
            </div>
        </div>
    );
}

interface DataSchemasContentProps {
    data_schemas: types.DataSchemaRecord[];
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
    schema: types.DataSchemaRecord;
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
                        <icon.CaretDown />
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
                        <icon.Pen />
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
