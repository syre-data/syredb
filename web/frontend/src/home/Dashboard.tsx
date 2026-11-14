import * as types from "@/types";
import { Suspense } from "react";
import { ErrorBoundary } from "react-error-boundary";
import type {
    FallbackProps as ErrorBoundaryProps,
    FallbackProps,
} from "react-error-boundary";
import { Link } from "react-router";
import icon from "../icon";
import * as common from "../common";
import { useSuspenseQuery } from "@tanstack/react-query";
import project_service from "@/service/project.service";
import data_service from "@/service/data.service";
import { Loading, StandardError } from "@/components";
import { toast } from "sonner";
import { AppError } from "@/errors";

export default function () {
    return (
        <div className="h-screen bg-gray-100 dark:bg-gray-900">
            <div className="px-4 pt-2 flex gap-2 text-xl">
                <h2 className="text-lg font-bold grow">Dashboard</h2>
                <Nav />
            </div>
            <main className="flex flex-col gap-4">
                <UserProjects />
                <OrphanedData />
                <RecentData />
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
                <h3 className="pb-2 text-lg font-bold">Your projects</h3>
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
    return (
        <StandardError
            resetErrorBoundary={resetErrorBoundary}
            error={error as Error}
            title={"Could not load your projects"}
        />
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
                    className="cursor-pointer border px-2 py-0.5 rounded-sm"
                >
                    Create your first project
                </button>
            </Link>
        </div>
    );
}

interface UserProjectsDeckProps {
    projects: types.ProjectWithUserPermission[];
}
function UserProjectsDeck({ projects }: UserProjectsDeckProps) {
    return (
        <div className="flex gap-2 px-4">
            {projects.map((project) => (
                <ProjectCard
                    key={project.Project.Id.toString()}
                    project={project}
                />
            ))}
        </div>
    );
}

interface ProjectCardProps {
    project: types.ProjectWithUserPermission;
}
function ProjectCard({ project }: ProjectCardProps) {
    return (
        <div className="card">
            <Link to={`/project/${project.Project.Id}`}>
                <div className="px-4 py-2">
                    <div className="pb-2">
                        <h4 className="font-bold">{project.Project.Label}</h4>
                    </div>
                    <div>{project.Project.Description}</div>
                </div>
            </Link>
        </div>
    );
}

function OrphanedData() {
    return (
        <ErrorBoundary FallbackComponent={OrphanedDataError}>
            <Suspense fallback={<Loading />}>
                <OrphanedDataInner />
            </Suspense>
        </ErrorBoundary>
    );
}

function OrphanedDataError({ error }: ErrorBoundaryProps) {
    if (error instanceof AppError) {
        console.error("SyreDB error", error.toObject());
    } else {
        console.error(error);
    }

    toast.error("Could not get orphaned data", {
        cancel: { label: "Cancel", onClick: () => {} },
    });
    return null;
}

function OrphanedDataInner() {
    const { data } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_ORPHANED_DATA],
        queryFn: data_service.orphanedData,
    });

    return data.Data.length > 0 ? (
        <div className="px-4 pt-2">
            <div className="card">
                <div className="px-4 py-2 flex gap-2 items-baseline">
                    <h3 className="text-lg font-bold">Orphaned data</h3>
                    <div className="text-sm text-secondary">
                        (does not belong to a project)
                    </div>
                </div>
                <div className="pb-2">
                    <OrphanedDataList
                        data={data.Data}
                        origins={data.Origins}
                        dataTypes={data.DataTypes}
                        children={data.Children}
                    />
                </div>
            </div>
        </div>
    ) : null;
}

interface OrphanedDataListProps {
    data: types.DataWithOrigin[];
    origins: types.DataOriginRx[];
    dataTypes: types.DataType[];
    children: { [key: string]: types.DataRx[] };
}
function OrphanedDataList({
    data,
    origins,
    dataTypes,
    children,
}: OrphanedDataListProps) {
    return (
        <table className="table-compact w-full [&_th]:last:w-full [&_td]:last:w-full [&_td]:text-nowrap [&_th]:first:pl-4 [&_td]:first:pl-4">
            <thead>
                <tr>
                    <th>Type</th>
                    <th>Origin</th>
                    <th>Timestamp</th>
                    <th></th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
                {data.map((data) => {
                    const data_id = common.uuidToString(data.Data.Id);
                    const origin = origins.find((o) => o.Id === data.Origin)!;
                    const type = dataTypes.find((t) => data.Data.Type == t.Id)!;
                    const d_children = children[data_id] ?? [];
                    return (
                        <OrphanedDataItem
                            key={data_id}
                            data={data.Data}
                            origin={origin}
                            dataType={type}
                            children={d_children}
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
    children: types.DataRx[];
}
function OrphanedDataItem({
    data,
    origin,
    dataType,
    children,
}: OrphanedDataItemProps) {
    return (
        <tr className="group hover:bg-gray-200 dark:hover:bg-gray-700">
            <td>
                <span className="select-all">{dataType.Label}</span>
            </td>
            <td>
                <span className="select-all">{origin.Label}</span>
            </td>
            <td>
                <span className="select-all">
                    {common.timestampToString(data.Timestamp)}
                </span>
            </td>
            <td className="text-secondary">({children.length} children)</td>
            <td>
                <div className="invisible group-hover:visible">
                    <Link to={`/data/${data.Id}`}>
                        <button type="button" className="btn-cmd align-bottom">
                            <icon.Eye />
                        </button>
                    </Link>
                </div>
            </td>
        </tr>
    );
}

function RecentData() {
    return (
        <div className="px-4">
            <div className="card">
                <div className="pt-2">
                    <h3 className="px-4 pb-2 text-lg font-bold">Recent data</h3>
                </div>
                <div className="pb-2">
                    <ErrorBoundary FallbackComponent={RecentDataError}>
                        <Suspense fallback={<Loading />}>
                            <RecentDataInner />
                        </Suspense>
                    </ErrorBoundary>
                </div>
            </div>
        </div>
    );
}

function RecentDataError({ resetErrorBoundary, error }: FallbackProps) {
    return (
        <StandardError
            resetErrorBoundary={resetErrorBoundary}
            error={error as Error}
            title={"Could not load recent data"}
        />
    );
}

function RecentDataInner() {
    const { data } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_RECENT_DATA],
        queryFn: async () => await data_service.recentData(10),
    });

    return data.Data.length > 0 ? (
        <RecentDataList data={data} />
    ) : (
        <div>No recent data</div>
    );
}

interface RecentDataListProps {
    data: types.RecentDataResources;
}
function RecentDataList({ data: resources }: RecentDataListProps) {
    return (
        <table className="table-compact w-full [&_th]:last:w-full [&_td]:last:w-full [&_td]:text-nowrap [&_th]:first:pl-4 [&_td]:first:pl-4">
            <thead>
                <tr>
                    <th>Type</th>
                    <th>Origin</th>
                    <th>Timestamp</th>
                    <th>Creator</th>
                    <th>Projects</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
                {resources.Data.map((datum) => {
                    const creator = resources.Users.find(
                        (user) => user.Id === datum.Creator,
                    );
                    const dtype = resources.DataTypes.find(
                        (dtype) => dtype.Id === datum.Data.Type,
                    );
                    const origin = resources.Origins.find(
                        (origin) => origin.Id === datum.Origin,
                    );
                    const project_memberships =
                        resources.ProjectMemberships.filter((membership) => {
                            console.debug(
                                datum.Data.Id,
                                membership.Data,
                                datum.Id === membership.Data,
                            );
                            return membership.Data === datum.Data.Id;
                        });
                    const projects = resources.Projects.filter(
                        (project) =>
                            project_memberships.findIndex(
                                (membership) =>
                                    project.Id === membership.Project,
                            ) > -1,
                    );

                    return (
                        <RecentDataListItem
                            key={datum.Id}
                            data={datum.Data}
                            creator={creator}
                            dataType={dtype}
                            origin={origin}
                            projects={projects}
                        />
                    );
                })}
            </tbody>
        </table>
    );
}

interface RecentDataListItemProps {
    data: types.DataRx;
    creator: types.User;
    dataType: types.DataType;
    origin: types.DataOriginRx;
    projects: types.Project[];
}
function RecentDataListItem({
    data,
    creator,
    dataType,
    origin,
    projects,
}: RecentDataListItemProps) {
    return (
        <tr className="group hover:bg-gray-200 dark:hover:bg-gray-700">
            <td>
                <span className="select-all">{dataType.Label}</span>
            </td>
            <td>
                <span className="select-all">{origin.Label}</span>
            </td>
            <td>
                <span className="select-all">
                    {common.timestampToString(data.Timestamp)}
                </span>
            </td>
            <td>
                <span className="select-all">{creator.Name}</span>
            </td>
            <td>
                {projects.length ? (
                    <div className="flex gap-2">
                        {projects.map((prj) => {
                            return (
                                <div
                                    key={common.uuidToString(prj.Id)}
                                    className="px-1 py-0.5 rounded hover:bg-gray-300 dark:hover:bg-gray-800 cursor-pointer"
                                >
                                    <Link to={`/project/${prj.Id}`}>
                                        {prj.Label}
                                    </Link>
                                </div>
                            );
                        })}
                    </div>
                ) : (
                    <span className="text-secondary">(no projects)</span>
                )}
            </td>
            <td>
                <div className="invisible group-hover:visible select-all">
                    <Link to={`/data/${data.Id}`}>
                        <button type="button" className="btn-cmd align-bottom">
                            <icon.Eye />
                        </button>
                    </Link>
                </div>
            </td>
        </tr>
    );
}
