import * as app from "../../wailsjs/go/app/App";
import { Suspense, use, useContext } from "react";
import {
    ErrorBoundary,
    FallbackProps as ErrorBoundaryProps,
} from "react-error-boundary";
import * as models from "../../wailsjs/go/models";
import { Link } from "react-router";
import icon from "../icon";
import * as appStateCtx from "../AppStateContext";
import { useSuspenseQuery } from "@tanstack/react-query";

export default function Dashboard() {
    const appState = useContext(appStateCtx.Context);

    const isOwnerRole = appState.user.Role === "owner";

    return (
        <div>
            <div className="flex gap-2 text-xl">
                <h2 className="px-4 text-lg font-bold grow">Dashboard</h2>
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
            </div>
            <main>
                <UserProjects />
            </main>
        </div>
    );
}

function UserProjects() {
    return (
        <div>
            <div className="flex gap-2 px-4">
                <h3 className="grow pb-2 text-lg font-bold">Projects</h3>
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
        queryFn: app.GetUserProjects,
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
    projects: models.app.Project[];
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
    project: models.app.Project;
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
