import { MouseButton, QUERY_KEY_PROJECT_RESOURCES } from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import projectService from "@/service/project.service";
import {
    DataStorageExternal,
    DataStorageInternal,
    type DataType,
    type ProjectData,
} from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Suspense, type MouseEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, redirect, useParams } from "react-router";
import * as uuid from "uuid";

export default function () {
    const { project_id } = useParams();
    if (!project_id) {
        console.error("project id not present");
        redirect("/");
        return;
    }

    return (
        <ErrorBoundary FallbackComponent={ProjectError}>
            <Suspense fallback={<Loading />}>
                <Project projectId={project_id} />
            </Suspense>
        </ErrorBoundary>
    );
}

function ProjectError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            <div>Could not load project</div>
        </SuspenseError>
    );
}

interface ProjectProps {
    projectId: uuid.UUIDTypes;
}
function Project({ projectId }: ProjectProps) {
    const { data: resources } = useSuspenseQuery({
        queryKey: [QUERY_KEY_PROJECT_RESOURCES, projectId],
        queryFn: async () =>
            await projectService.getProjectResources(projectId),
    });

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <h2 className="text-lg">{resources.Project.Label}</h2>
                <div>
                    <div>
                        <Link to="/">
                            <button type="button" className="btn-cmd">
                                <Icon.Home />
                            </button>
                        </Link>
                    </div>
                </div>
            </div>
            <ProjectData
                projectId={projectId}
                data={resources.Data}
                types={resources.DataTypes}
            />
        </div>
    );
}

interface ProjectDataProps {
    projectId: uuid.UUIDTypes;
    data: ProjectData[];
    types: DataType[];
}
function ProjectData({ projectId, data, types }: ProjectDataProps) {
    return (
        <div>
            <div className="px-4 flex gap-2">
                <h3 className="text-lg">Data</h3>
                <div>
                    <Link to={`/project/${projectId}/data/create`}>
                        <button type="button" className="btn-cmd">
                            <Icon.Plus />
                        </button>
                    </Link>
                </div>
            </div>
            {data.length === 0 ? (
                <ProjectDataEmpty />
            ) : (
                <ProjectDataList
                    projectId={projectId}
                    data={data}
                    types={types}
                />
            )}
        </div>
    );
}

function ProjectDataEmpty() {
    return (
        <div className="px-4">
            <p>No data yet.</p>
            <p>
                Click the <Icon.Plus className="inline" /> above to create some.
            </p>
        </div>
    );
}

interface ProjectDataListProps {
    projectId: uuid.UUIDTypes;
    data: ProjectData[];
    types: DataType[];
}
function ProjectDataList({ projectId, data, types }: ProjectDataListProps) {
    return (
        <ul className="grid gap-2 grid-cols-[repeat(4,min-content)]">
            {data.map((datum, idx) => {
                const type = types.find((type) => type.Id === datum.Type);
                if (!type) {
                    console.error(`could not get data type ${datum.Type}`);
                }

                return (
                    <ProjectDataItem
                        key={datum.Id.toString()}
                        index={idx}
                        projectId={projectId}
                        data={datum}
                        type={type}
                    />
                );
            })}
        </ul>
    );
}

interface ProjectDataItemProps {
    index: number;
    projectId: uuid.UUIDTypes;
    data: ProjectData;
    type: DataType;
}
function ProjectDataItem({
    index,
    projectId,
    data,
    type,
}: ProjectDataItemProps) {
    function open_preview(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        console.error("TODO");
    }

    const date = new Date(data.Timestamp);
    const y = date.getUTCFullYear();
    const mo = date.getMonth();
    const d = date.getDate();
    const h = date.getHours();
    const m = date.getMinutes();
    const s = date.getSeconds();
    const timestamp = `${y}-${mo}-${d}T${h}-${m}-${s}`;
    let filename = data.Label ?? `data-${timestamp}`;
    switch (type.Storage) {
        case DataStorageExternal:
            // TODO: If source is a single file, use that extension.
            filename += ".zip";
            break;
        case DataStorageInternal:
            filename += ".csv";
            break;
        default:
            console.error("invalid data storage");
    }

    const params = new URLSearchParams();
    params.append("id", data.Id.toString());
    const download = `/resource/data?${params}`;
    return (
        <li className="px-4 grid grid-cols-subgrid col-span-full group">
            <div className="grid grid-cols-subgrid col-span-full">
                <div className="col-1">{index + 1}.</div>
                <div className="col-2 text-nowrap">{type.Label}</div>
                <div className="col-3">
                    {data.Label ? (
                        <>
                            <span className="text-nowrap">data.Label</span>
                            <span className="text-gray-500">
                                ({data.Timestamp.toString()})
                            </span>
                        </>
                    ) : (
                        data.Timestamp.toString()
                    )}
                </div>
                <div className="flex gap-1 invisible group-hover:visible">
                    <div>
                        {/* TODO: Check permissions */}
                        <Link to={`/data/${data.Id}?project=${projectId}`}>
                            <button
                                type="button"
                                className="btn-cmd"
                                title="Edit"
                            >
                                <Icon.Pen />
                            </button>
                        </Link>
                    </div>
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            title="Preview"
                            onMouseDown={open_preview}
                        >
                            <Icon.Eye />
                        </button>
                    </div>
                    <div>
                        <a href={download}>
                            <button
                                type="button"
                                className="btn-cmd"
                                title="Download"
                            >
                                <Icon.Download />
                            </button>
                        </a>
                    </div>
                </div>
            </div>
        </li>
    );
}
