import { Suspense, MouseEvent } from "react";
import { ErrorBoundary, FallbackProps } from "react-error-boundary";
import { Link, useNavigate, useParams } from "react-router";
import * as common from "../common";
import { useSuspenseQuery } from "@tanstack/react-query";
import * as app from "../../bindings/syredb/app";
import icon from "../icon";
import { UUID } from "../../bindings/github.com/google/uuid";

export default function () {
    const navigate = useNavigate();
    const { project_id, sample_id } = useParams();
    if (project_id && sample_id) {
        return (
            <ErrorBoundary FallbackComponent={SampleError}>
                <Suspense fallback={<Loading />}>
                    <Sample project_id={project_id} sample_id={sample_id} />
                </Suspense>
            </ErrorBoundary>
        );
    } else {
        navigate("/");
        return null;
    }
}

function Loading() {
    return <div className="text-center pt-4">Loading</div>;
}

function SampleError({ error, resetErrorBoundary }: FallbackProps) {
    const err = error as common.BackendError;
    const navigate = useNavigate();

    if (err.message === common.USER_NOT_AUTHENTICATED_ERROR) {
        console.error(common.USER_NOT_AUTHENTICATED_ERROR);
        navigate("/");
        return null;
    } else {
        console.error(err);
    }

    function reload(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        resetErrorBoundary();
    }

    return (
        <div className="flex flex-col gap-2 items-center pt-4">
            <div>Could not load project</div>
            <div>{err.message}</div>
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

interface SampleProps {
    project_id: UUID;
    sample_id: UUID;
}
function Sample({ project_id, sample_id }: SampleProps) {
    const navigate = useNavigate();
    const { data: sample_resources } = useSuspenseQuery({
        queryKey: ["project_sample_resources", project_id, sample_id],
        queryFn: async () =>
            app.ProjectService.GetProjectSampleResources(project_id, sample_id),
    });

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    return (
        <div>
            <div className="flex px-4 pt-2">
                <div className="grow">Sample</div>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={close}
                    >
                        <icon.Close />
                    </button>
                </div>
            </div>
        </div>
    );
}
