import { useSuspenseQuery } from "@tanstack/react-query";
import { useParams, Link, useNavigate } from "react-router";
import * as app from "../../wailsjs/go/app/App";
import * as models from "../../wailsjs/go/models";
import icon from "../icon";
import { ErrorBoundary, FallbackProps } from "react-error-boundary";
import {
    createContext,
    FormEvent,
    MouseEvent,
    Suspense,
    useContext,
    useEffect,
    useLayoutEffect,
    useRef,
    useState,
} from "react";
import * as common from "../common";

interface CommonProjectData {
    project_id: string;
    user_permission: common.UserPermission;
}

const CommonProjectDataCtx = createContext<CommonProjectData>({
    project_id: "",
    user_permission: common.UserPermission.Read,
});

export default function () {
    const navigate = useNavigate();
    const { id: project_id } = useParams();
    if (project_id) {
        return (
            <ErrorBoundary FallbackComponent={ProjectError}>
                <Suspense fallback={<Loading />}>
                    <Project id={project_id} />
                </Suspense>
            </ErrorBoundary>
        );
    } else {
        navigate("/");
        return <></>;
    }
}

function Loading() {
    return <div className="text-center pt-4">Loading</div>;
}

function ProjectError({ error, resetErrorBoundary }: FallbackProps) {
    const navigate = useNavigate();

    function reload(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        resetErrorBoundary();
    }

    if (error === common.USER_NOT_AUTHENTICATED_ERROR) {
        console.error(common.USER_NOT_AUTHENTICATED_ERROR);
        navigate("/");
        return <></>;
    }

    return (
        <div className="flex flex-col gap-2 items-center pt-4">
            <div>Could not load project</div>
            <div>{error}</div>
            <div className="flex gap-2 items-center">
                <div>
                    <Link to="/">
                        <button type="button" className="btn-cmd">
                            <icon.Home />
                        </button>
                    </Link>
                </div>
                <div>
                    <button
                        type="button"
                        onMouseDown={reload}
                        className="btn-cmd"
                    >
                        <icon.Reload />
                    </button>
                </div>
            </div>
        </div>
    );
}

interface ProjectProps {
    id: string;
}
function Project({ id }: ProjectProps) {
    const navigate = useNavigate();
    const { data: project_resources } = useSuspenseQuery({
        queryKey: ["project_resources", id],
        queryFn: async () => app.GetProjectResources(id),
    });

    let user_permission;
    switch (project_resources.ProjectUserPermission) {
        case common.UserPermission.Owner:
            user_permission = common.UserPermission.Owner;
            break;
        case common.UserPermission.Admin:
            user_permission = common.UserPermission.Admin;
            break;
        case common.UserPermission.ReadWrite:
            user_permission = common.UserPermission.ReadWrite;
            break;
        case common.UserPermission.Read:
            user_permission = common.UserPermission.Read;
            break;
        default:
            console.error(
                `invalid user permission: ${project_resources.ProjectUserPermission}`
            );
            navigate("/");
            return;
    }

    return (
        <CommonProjectDataCtx value={{ project_id: id, user_permission }}>
            <div>
                <ProjectHeader project={project_resources.Project} />
                <ProjectSampleList
                    samples={project_resources.Samples}
                    className="pt-2"
                />
            </div>
        </CommonProjectDataCtx>
    );
}

interface ProjectHeaderProps {
    project: models.app.Project;
}
function ProjectHeader({ project }: ProjectHeaderProps) {
    return (
        <div className="flex gap-2 pt-1 pb-2 px-2 border-b">
            <h2
                className={`group/project-header-label flex gap-2 grow font-bold`}
            >
                {project.Label}
            </h2>
            <div className="flex gap-1">
                {
                    <div>
                        <Link to={`/project/${project.Id}/settings`}>
                            <icon.Gear />
                        </Link>
                    </div>
                }
                <div>
                    <Link to="/">
                        <button type="button" className="btn-cmd">
                            <icon.Home />
                        </button>
                    </Link>
                </div>
            </div>
        </div>
    );
}

interface ProjectSampleListProps {
    samples: models.app.ProjectSample[];
    className: string;
}
function ProjectSampleList({ samples, className }: ProjectSampleListProps) {
    const project_data = useContext(CommonProjectDataCtx);

    return (
        <div className={className}>
            <div className="flex gap-2 px-4">
                <h3 className="font-bold">Samples</h3>
                {common.is_admin_or_owner(project_data.user_permission) ? (
                    <div>
                        <Link
                            to={`/project/${project_data.project_id}/samples/create`}
                        >
                            <button type="button" className="btn-cmd">
                                <icon.Plus />
                            </button>
                        </Link>
                    </div>
                ) : null}
            </div>
            {samples.length == 0 ? (
                <ProjectSampleListEmpty />
            ) : (
                <ul>
                    {samples.map((sample) => (
                        <li key={sample.Id.toString()}>
                            <ProjectSampleListItem sample={sample} />
                        </li>
                    ))}
                </ul>
            )}
        </div>
    );
}

function ProjectSampleListEmpty() {
    const project_data = useContext(CommonProjectDataCtx);
    return (
        <div className="px-4">
            <div>No samples</div>
            {common.is_admin_or_owner(project_data.user_permission) ? (
                <div>
                    <small>
                        Click the <icon.Plus className="inline" /> to add some
                    </small>
                </div>
            ) : null}
        </div>
    );
}

interface ProjectSampleListItemProps {
    sample: models.app.ProjectSample;
}
function ProjectSampleListItem({ sample }: ProjectSampleListItemProps) {
    return <div>{sample.Label}</div>;
}
