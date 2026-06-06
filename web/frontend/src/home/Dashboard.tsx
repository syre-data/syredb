import * as types from "@/types";
import { Suspense } from "react";
import { ErrorBoundary } from "react-error-boundary";
import type { FallbackProps as ErrorBoundaryProps } from "react-error-boundary";
import { Link } from "react-router";
import icon from "../icon";
import * as common from "../common";
import { useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";
import project_service from "@/service/project.service";
import data_service from "@/service/data.service";
import { Loading, SuspenseError } from "@/components";

export default function () {
    return (
        <div>
            <div className="px-4 pt-2 flex gap-2 text-xl">
                <h2 className="text-lg font-bold grow">Dashboard</h2>
                <Nav />
            </div>
            <main>
                <UserProjects />
                <OrphanedData />
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
                    <Suspense fallback={<UserProjectsLoading />}>
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

function UserProjectsLoading() {
    return <div className="text-center">Loading projects</div>;
}

function UserProjectsInner() {
    const { data: projects } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_USER_PROJECTS],
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

function OrphanedData() {
    return (
        <ErrorBoundary FallbackComponent={OrphanedDataError}>
            <Suspense fallback={<Loading />}>
                <OrphanedDataContent />
            </Suspense>
        </ErrorBoundary>
    );
}

function OrphanedDataError({ error, resetErrorBoundary }: ErrorBoundaryProps) {
    const err = error as common.BackendError;
    console.error(err);

    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center"
        >
            <div>Could not get orphaned data</div>
            <div>{err.message}</div>
        </SuspenseError>
    );
}

function OrphanedDataContent() {
    const { data } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_ORPHANED_DATA],
        queryFn: data_service.orphanedData,
    });

    return data.Data.length > 0 ? (
        <div className="px-4 pt-2">
            <h3 className="pb-2 text-lg font-bold">Orphaned data</h3>
            <OrphanedDataList
                data={data.Data}
                origins={data.Origins}
                dataTypes={data.DataTypes}
            />
        </div>
    ) : null;
}

interface OrphanedDataListProps {
    data: types.DataWithOrigin[];
    origins: types.DataOriginRx[];
    dataTypes: types.DataType[];
}
function OrphanedDataList({ data, origins, dataTypes }: OrphanedDataListProps) {
    return (
        <table className="text-left [&_td]:px-2 [&_td]:py-0.5">
            <thead className="[&_th]:px-2 [&_th]:py-0.5">
                <tr>
                    <th>Type</th>
                    <th>Origin</th>
                    <th>Timestamp</th>
                </tr>
            </thead>
            <tbody>
                {data.map((data) => {
                    const origin = origins.find((o) => o.Id === data.Origin)!;
                    const type = dataTypes.find((t) => data.Data.Type == t.Id)!;
                    return (
                        <OrphanedDataItem
                            key={data.Data.Id.toString()}
                            data={data.Data}
                            origin={origin}
                            dataType={type}
                        />
                    );
                })}
            </tbody>
        </table>
    );
}

interface OrphanedDataItemProps {
    data: types.DataRx;
    origin: types.DataOriginRx;
    dataType: types.DataType;
}
function OrphanedDataItem({ data, origin, dataType }: OrphanedDataItemProps) {
    console.debug(data.Timestamp);
    return (
        <tr>
            <td>{dataType.Label}</td>
            <td>{origin.Label}</td>
            <td>{data.Timestamp}</td>
        </tr>
    );
}
