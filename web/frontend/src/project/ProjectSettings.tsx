import { useParams, redirect } from "react-router";
import { ErrorBoundary } from "react-error-boundary";
import type { FallbackProps } from "react-error-boundary";
import { Suspense } from "react";
import { StandardError } from "@/components/SuspenseError";
import type { AppError } from "@/types";

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
    return (
        <StandardError
            resetErrorBoundary={resetErrorBoundary}
            error={error as AppError}
            title={"Could not load project"}
        />
    );
}

interface ProjectSettngsProps {
    id: string;
}
function ProjectSettings({ id }: ProjectSettngsProps) {
    return <div>{id}</div>;
}
