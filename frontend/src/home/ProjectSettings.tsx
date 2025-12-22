import { useSuspenseQuery } from "@tanstack/react-query";
import { useParams, Link, useNavigate } from "react-router";
import * as app from "../../wailsjs/go/app/App";
import * as models from "../../wailsjs/go/models";
import icon from "../icon";
import { ErrorBoundary, FallbackProps } from "react-error-boundary";
import {
    FormEvent,
    MouseEvent,
    Suspense,
    useEffect,
    useLayoutEffect,
    useRef,
    useState,
} from "react";
import * as common from "../common";

export default function () {
    const navigate = useNavigate();
    const { id: project_id } = useParams();
    if (project_id) {
        return (
            <ErrorBoundary FallbackComponent={ProjectSettingsError}>
                <Suspense fallback={<Loading />}>
                    <ProjectSettings id={project_id} />
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

function ProjectSettingsError({ error, resetErrorBoundary }: FallbackProps) {
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

interface ProjectSettngsProps {
    id: string;
}
function ProjectSettings({ id }: ProjectSettngsProps) {
    return <div>{id}</div>;
}
