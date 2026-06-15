import { MouseButton, QUERY_KEY_DATA, QUERY_KEY_USER_PROJECTS } from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import projectService from "@/service/project.service";
import type {
    DataProjectResources,
    DataRx,
    DataType,
    Note,
    ProjectResources,
    Property,
    User,
} from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";
import { StatusCodes } from "http-status-codes";
import {
    Suspense,
    useState,
    type Dispatch,
    type MouseEvent,
    type SetStateAction,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, Navigate, useNavigate, useParams } from "react-router";
import type { UUIDTypes } from "uuid";

export default function () {
    const { data_id } = useParams();
    if (!data_id) {
        return <Navigate to="/" replace />;
    }

    return (
        <ErrorBoundary FallbackComponent={LoadError}>
            <Suspense fallback={<Loading />}>
                <Data data_id={data_id} />
            </Suspense>
        </ErrorBoundary>
    );
}

function LoadError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError resetErrorBoundary={resetErrorBoundary}>
            Could not load data
        </SuspenseError>
    );
}

interface DataProps {
    data_id: UUIDTypes;
}
function Data({ data_id }: DataProps) {
    const { data: resources } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA, data_id],
        queryFn: async () => await dataService.dataGet(data_id),
    });
    const [showAddProject, setShowAddProject] = useState(false);
    const navigate = useNavigate();

    function showAddProjectFn(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        setShowAddProject(true);
    }

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    const data = resources.Data as DataRx;
    const data_type = resources.DataType as DataType;
    const properties = resources.Properties as Property[];
    const notes = resources.Notes as Note[];
    const project_resources =
        resources.ProjectResources as DataProjectResources[];
    const users = resources.Users as User[];
    const currentProjects = project_resources.map(
        (resource) => resource.Project.Id,
    );
    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <div>
                    <h1>Data</h1>
                </div>
                <div>
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            onMouseDown={close}
                        >
                            <Icon.Close />
                        </button>
                    </div>
                </div>
            </div>
            <div className="px-4 pt-2">
                <div>{data.Timestamp}</div>
                <div>{data.Visibility}</div>
                <div>{data_type.Label}</div>
            </div>
            <Properties properties={properties} />
            <Notes notes={notes} />
            <div className="pt-2">
                <div className="px-4 flex gap-2">
                    <h2 className="text-lg">Projects</h2>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={showAddProjectFn}
                    >
                        <Icon.Plus />
                    </button>
                </div>
                <div
                    className={classNames({
                        "px-4 py-2": true,
                        hidden: !showAddProject,
                    })}
                >
                    <AddProject
                        data_id={data_id}
                        currentProjects={currentProjects}
                        setShowAddProject={setShowAddProject}
                    />
                </div>
                {project_resources.length > 0 ? (
                    <ProjectList projects={project_resources} users={users} />
                ) : (
                    <ProjectsEmpty />
                )}
            </div>
        </div>
    );
}

interface PropertiesProps {
    properties: Property[];
}
function Properties({ properties }: PropertiesProps) {
    return (
        <div>
            <div className="px-4 pb-2">
                <h3>Properties</h3>
            </div>
            <div className="px-4">
                <table>
                    <tbody>
                        {properties.map((property) => (
                            <tr key={property.Key}>
                                <td>{property.Key}</td>
                                <td>{property.Type}</td>
                                <td>{property.Value}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}

interface NotesProps {
    notes: Note[];
}
function Notes({ notes }: NotesProps) {
    return (
        <div>
            <div className="px-4 pb-2">
                <h3>Notes</h3>
            </div>
            <ol>
                {notes.map((note) => (
                    <li key={note.Id.toString()}>
                        <div className="px-4 pb-2">
                            <h4>{note.Timestamp}</h4>
                            <div>{note.Content}</div>
                        </div>
                    </li>
                ))}
            </ol>
        </div>
    );
}

interface AddProjectProps {
    data_id: UUIDTypes;
    currentProjects: UUIDTypes[];
    setShowAddProject: Dispatch<SetStateAction<boolean>>;
}
function AddProject({
    data_id,
    currentProjects,
    setShowAddProject,
}: AddProjectProps) {
    const { data: projects } = useSuspenseQuery({
        queryKey: [QUERY_KEY_USER_PROJECTS],
        queryFn: projectService.getUserProjects,
    });

    async function addProject(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const data = new FormData(e.target);
        const project = data.get("add-project")!.toString();
        await projectService
            .dataMembershipCreate(project, data_id)
            .then((res) => {
                if (res.status === StatusCodes.OK) {
                    setShowAddProject(false);
                }
            });
    }

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        setShowAddProject(false);
    }

    return (
        <form onSubmit={addProject}>
            <div className="pb-2 flex gap-2">
                <div>
                    <select
                        id="add-project"
                        name="add-project"
                        className="input-basic"
                    >
                        <option disabled hidden selected>
                            Choose a project
                        </option>
                        {projects
                            .filter(
                                (project) =>
                                    !currentProjects.includes(project.Id),
                            )
                            .map((project) => (
                                <option
                                    key={project.Id.toString()}
                                    value={project.Id.toString()}
                                >
                                    {project.Label}
                                </option>
                            ))}
                    </select>
                </div>
                <div className="flex gap-2">
                    <div>
                        <button type="submit" className="btn-submit">
                            <Icon.Plus />
                        </button>
                    </div>
                    <div>
                        <button
                            type="button"
                            className="btn-submit"
                            onMouseDown={close}
                        >
                            <Icon.Close />
                        </button>
                    </div>
                </div>
            </div>
        </form>
    );
}

function ProjectsEmpty() {
    return (
        <div className="px-4">
            <p>This data does not belong to any projects, yet.</p>
            <p>
                Add it to one by clicking the <Icon.Plus className="inline" />{" "}
                above.
            </p>
        </div>
    );
}

interface ProjectListProps {
    projects: DataProjectResources[];
    users: User[];
}
function ProjectList({ projects, users }: ProjectListProps) {
    return (
        <table className="px-4">
            <tbody>
                {projects.map((project) => (
                    <ProjectItem
                        key={project.Project.Id.toString()}
                        resources={project}
                        users={users}
                    />
                ))}
            </tbody>
        </table>
    );
}

interface ProjectItemProps {
    resources: DataProjectResources;
    users: User[];
}
function ProjectItem({ resources }: ProjectItemProps) {
    return (
        <tr className="group pb-1">
            <td className="font-bold pl-4 pr-2">{resources.Project.Label}</td>
            <td className="pr-2">{resources.Project.Description}</td>
            <td className="pr-2">
                <div className="invisible group-hover:visible">
                    <Link
                        to={`/project/${resources.Project.Id}`}
                        className="align-middle"
                    >
                        <button type="button" className="btn-cmd">
                            <Icon.Eye />
                        </button>
                    </Link>
                </div>
            </td>
        </tr>
    );
}
