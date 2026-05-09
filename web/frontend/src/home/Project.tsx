import {
    MouseButton,
    QUERY_KEY_DATA_TYPE_TRANSFORMS,
    QUERY_KEY_DATA_TYPES,
    QUERY_KEY_PROJECT_RESOURCES,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import projectService from "@/service/project.service";
import type {
    DataType,
    DataTypeTransform,
    DataTypeTransformRx,
    ProjectData,
    ProjectDataInfo,
} from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Suspense, type MouseEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, redirect, useNavigate, useParams } from "react-router";
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

    const navigate = useNavigate();

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
                <ProjectDataList data={data} types={types} />
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
    data: ProjectData[];
    types: DataType[];
}
function ProjectDataList({ data, types }: ProjectDataListProps) {
    return (
        <ul className="grid gap-2 grid-cols-[repeat(3,min-content)]">
            {data.map((datum, idx) => {
                const type = types.find((type) => type.Id === datum.Type);
                if (!type) {
                    console.error(`could not get data type ${datum.Type}`);
                }

                return (
                    <ProjectDataItem
                        key={datum.Id.toString()}
                        index={idx}
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
    data: ProjectData;
    type: DataType;
}
function ProjectDataItem({ index, data, type }: ProjectDataItemProps) {
    return (
        <li className="px-4 grid grid-cols-subgrid col-span-full">
            <div className="grid grid-cols-subgrid col-span-full">
                <div className="col-1">{index + 1}.</div>
                <div className="col-2">{type.Label}</div>
                <div className="col-3">
                    {data.Label ? (
                        <>
                            data.Label
                            <span className="text-gray-500">
                                ({data.Timestamp.toString()})
                            </span>
                        </>
                    ) : (
                        data.Timestamp.toString()
                    )}
                </div>
            </div>
        </li>
    );
}
