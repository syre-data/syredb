import {
    MouseButton,
    QUERY_KEY_DATA_ORIGIN,
    QUERY_KEY_DATA_ORIGINS,
    uuidToString,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
    Suspense,
    useState,
    type ChangeEvent,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Navigate, useNavigate, useParams } from "react-router";
import { parse, type UUIDTypes } from "uuid";

export default function () {
    return (
        <ErrorBoundary FallbackComponent={DataOriginError}>
            <Suspense fallback={<Loading />}>
                <DataOrigin />
            </Suspense>
        </ErrorBoundary>
    );
}

function DataOriginError({ resetErrorBoundary, error }: FallbackProps) {
    return (
        <SuspenseError resetErrorBoundary={resetErrorBoundary}>
            Could not load data origins
        </SuspenseError>
    );
}

function DataOrigin() {
    const { data: origins } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_ORIGINS],
        queryFn: async () => dataService.dataOrigins(),
    });
    const navigate = useNavigate();
    const [pending, setPending] = useState(false);
    const labels = origins.map((origin) => origin.Label);

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    function validateLabel(e: ChangeEvent<HTMLInputElement>) {
        e.target.setCustomValidity("");
        const label = e.target.value.toString().trim();
        if (labels.includes(label)) {
            e.target.setCustomValidity("Label already exists");
        }
    }

    async function create(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        setPending(true);
        const data = new FormData(e.target);
        const label = data.get("label")!.toString().trim();
        const description = data.get("description")!.toString().trim();

        await dataService
            .dataOriginCreate({
                Label: label,
                Description: description,
                Active: true,
            })
            .then((res) => {
                if (res.ok) {
                    navigate(-1);
                } else {
                    setPending(false);
                    console.error("error", res);
                }
            });
    }

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <div>
                    <h1>New data origin</h1>
                </div>
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
            <div className="px-4 pt-2">
                <form onSubmit={create}>
                    <div className="flex flex-col gap-2">
                        <div className="flex flex-col gap-2">
                            <div>
                                <label>
                                    <span className="sr-only">Label</span>
                                    <input
                                        type="text"
                                        id="label"
                                        name="label"
                                        className="input-basic"
                                        placeholder="Label"
                                        minLength={1}
                                        onChange={validateLabel}
                                        required
                                    />
                                </label>
                            </div>
                            <div>
                                <label>
                                    <span className="sr-only">Description</span>
                                    <textarea
                                        id="description"
                                        name="description"
                                        className="input-basic"
                                        placeholder="Description"
                                        rows={5}
                                        cols={40}
                                    />
                                </label>
                            </div>
                        </div>
                        <div>
                            <button
                                type="submit"
                                className="btn-submit"
                                disabled={pending}
                            >
                                Create
                            </button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    );
}
