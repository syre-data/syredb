import * as app from "../../bindings/syredb/app";
import { MouseEvent, Suspense, use, useContext, useState } from "react";
import {
    ErrorBoundary,
    FallbackProps as ErrorBoundaryProps,
    FallbackProps,
} from "react-error-boundary";
import { Link } from "react-router";
import icon from "../icon";
import * as appStateCtx from "../AppStateContext";
import * as common from "../common";
import { useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";

export default function Dashboard() {
    const appState = useContext(appStateCtx.Context);
    const isAdminOrOwnerRole =
        appState.user.Role === "admin" || appState.user.Role === "owner";

    return (
        <div>
            <div className="flex gap-2 text-xl">
                <h2 className="px-4 text-lg font-bold grow">Dashboard</h2>
                <Nav />
            </div>
            <main>
                <UserProjects />
                {isAdminOrOwnerRole ? <DataSchemas className="pt-4" /> : null}
            </main>
        </div>
    );
}

function Nav() {
    const appState = useContext(appStateCtx.Context);
    const isOwnerRole = appState.user.Role === "owner";

    return (
        <div className="flex gap-2">
            <div>
                {isOwnerRole ? (
                    <Link to="/users" title="Users">
                        <button type="button" className="btn-cmd">
                            <icon.Users />
                        </button>
                    </Link>
                ) : null}
            </div>
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
    console.error(error);

    return (
        <div>
            <div>Could not get your projects</div>
            <div>{error}</div>
        </div>
    );
}

function LoadingUserProjects() {
    return <div className="text-center">Loading projects</div>;
}

function UserProjectsInner() {
    const { data: projects } = useSuspenseQuery({
        queryKey: ["user_projects"],
        queryFn: app.AppService.GetUserProjects,
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
    projects: app.Project[];
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
    project: app.Project;
}
function ProjectCard({ project }: ProjectCardProps) {
    return (
        <div className="border rounded-sm px-2 py-1">
            <Link to={`/project/${project.Id}`}>
                <div>
                    <h4 className="font-bold">{project.Label}</h4>
                    <div>{project.Description}</div>
                </div>
            </Link>
        </div>
    );
}

interface DataSchemasProps {
    className: string;
}
function DataSchemas({ className }: DataSchemasProps) {
    return (
        <div className={className}>
            <div className="flex gap-2 items-baseline px-4">
                <h3 className="text-lg font-bold">Data schemas</h3>
                <div className="gap-1">
                    <Link to="/data_schema/create">
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
                <ErrorBoundary FallbackComponent={DataSchemaError}>
                    <Suspense fallback={<DataSchemaLoading />}>
                        <DataSchemaInner />
                    </Suspense>
                </ErrorBoundary>
            </div>
        </div>
    );
}

function DataSchemaError({ error, resetErrorBoundary }: FallbackProps) {
    console.error(error);
    return (
        <div className="text-center">
            <div>Could not get data schemas</div>
            <div>{error}</div>
        </div>
    );
}

function DataSchemaLoading() {
    return <div className="text-center">Loading</div>;
}

function DataSchemaInner() {
    const { data: data_schemas } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_SCHEMA],
        queryFn: app.AppService.GetDataSchemas,
    });

    if (data_schemas.length === 0) {
        return (
            <div className="px-4">
                <div>
                    <div>No data schemas</div>
                    <div>
                        Create your first data schema by clicking the + above.
                    </div>
                </div>
            </div>
        );
    } else {
        return (
            <ul className="grid gap-y-2 grid-cols-[50px_1fr_5fr]">
                {data_schemas.map((schema, index) => (
                    <li key={schema.Id.toString()} className="contents">
                        <DataSchemaContent index={index} schema={schema} />
                    </li>
                ))}
            </ul>
        );
    }
}

interface DataSchemaProps {
    index: number;
    schema: app.DataSchema;
}
function DataSchemaContent({ index, schema }: DataSchemaProps) {
    const ROW_SPAN = 2;
    const [expanded, setExpanded] = useState(false);

    function toggle_expand(e: MouseEvent<HTMLDivElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        setExpanded(!expanded);
    }

    const row_idx = index * ROW_SPAN;
    const row_main = row_idx + 1;
    const row_schema = row_main + 1;
    const description = schema.Description ?? "(no description)";
    return (
        <div className="contents group/schema-row">
            <div className="contents" onMouseDown={toggle_expand}>
                <div
                    className={classNames({
                        "col-start-1 pl-4": true,
                        "invisible hover:visible group-hover/schema-row:visible":
                            !expanded,
                    })}
                    style={{
                        gridRow: row_main,
                    }}
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
                <div
                    className="col-start-2 cursor-pointer"
                    style={{
                        gridRow: row_main,
                    }}
                >
                    {schema.Label}
                </div>
            </div>
            <div className="col-start-3" style={{ gridRow: row_main }}>
                {description}
            </div>
            <div
                className={classNames({
                    "col-start-2 col-span-full overflow-hidden flex gap-2 transition-[height]":
                        true,
                    "h-0": !expanded,
                })}
                style={{ gridRow: row_schema }}
            >
                {schema.Schema.map((col, idx) => (
                    <div>
                        <span>{col.label}</span> <span>({col.dtype})</span>
                        {idx === schema.Schema.length - 1 ? "" : " | "}
                    </div>
                ))}
            </div>
        </div>
    );
}
