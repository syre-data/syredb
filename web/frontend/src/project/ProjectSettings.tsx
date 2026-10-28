import { useParams, redirect } from "react-router";
import { ErrorBoundary } from "react-error-boundary";
import type { FallbackProps } from "react-error-boundary";
import { Suspense } from "react";
import SuspenseError from "@/components/SuspenseError";

export default function () {
    const { project_id } = useParams();
    if (project_id) {
        return (
            <ErrorBoundary FallbackComponent={ProjectSettingsError}>
                <Suspense fallback={<Loading />}>
                    <ProjectSettings id={project_id} />
                </Suspense>
            </ErrorBoundary>
        );
    } else {
        throw redirect("/");
    }
}

function Loading() {
    return <div className="text-center pt-4">Loading</div>;
}

function ProjectSettingsError({ error, resetErrorBoundary }: FallbackProps) {
    console.error(error);

    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="flex flex-col gap-2 items-center pt-4"
        >
            <div>Could not load project</div>
            <div>{error.message}</div>
        </SuspenseError>
    );
}

interface ProjectSettngsProps {
    id: string;
}
function ProjectSettings({ id }: ProjectSettngsProps) {
    return <div>{id}</div>;
}
