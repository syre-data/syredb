import {
    MouseButton,
    QUERY_KEY_DATA_TYPES,
    QUERY_KEY_INGESTION_SCRIPTS,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import type { IngestionScript } from "@/types";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import {
    Suspense,
    type ChangeEvent,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, useNavigate } from "react-router";

export default function () {
    return (
        <ErrorBoundary FallbackComponent={IngestionScriptsError}>
            <Suspense fallback={<Loading />}>
                <IngestionScripts />
            </Suspense>
        </ErrorBoundary>
    );
}

function IngestionScriptsError({ resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            Could not load ingestion scripts
        </SuspenseError>
    );
}

function IngestionScripts() {
    const { data: scripts } = useSuspenseQuery({
        queryKey: [QUERY_KEY_INGESTION_SCRIPTS],
        queryFn: dataService.ingestionScriptsGetAll,
    });
    const { data: data_types } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPES],
        queryFn: dataService.dataTypesGetAll,
    });
    const navigate = useNavigate();
    const queryClient = useQueryClient();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    function validate_label(e: ChangeEvent<HTMLInputElement>) {
        const input = e.target;
        input.setCustomValidity("");

        const label = input.value;
        if (label.length === 0) {
            input.setCustomValidity("Label is required");
            return;
        }
        if (scripts.findIndex((script) => script.Label === label) > -1) {
            input.setCustomValidity("Label already exists");
            return;
        }
    }

    async function create_ingestion_script(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        const data = new FormData(e.target);
        await dataService.ingestionScriptCreate(data).then((resp) => {
            if (resp.ok) {
                queryClient.invalidateQueries({
                    queryKey: [QUERY_KEY_INGESTION_SCRIPTS],
                });
                navigate(-1);
            }
        });
    }

    return (
        <div>
            <div className="flex justify-between px-4 pt-2">
                <div className="flex gap-2">
                    <h2 className="text-xl">Create ingestion script</h2>
                </div>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={close}
                        title="Close"
                    >
                        <Icon.Close />
                    </button>
                </div>
            </div>
            <form
                className="px-4 pt-2 flex flex-col gap-2"
                onSubmit={create_ingestion_script}
            >
                <div className="flex flex-col gap-2">
                    <div>
                        <label>
                            <span className="sr-only">Label</span>
                            <input
                                type="text"
                                id="label"
                                name="label"
                                placeholder="Label"
                                className="input-basic"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="sr-only">Data type</span>
                            <select
                                id="data_type"
                                name="data_type"
                                className="input-basic"
                                required
                            >
                                <option disabled selected hidden>
                                    Data type
                                </option>
                                {data_types.map((type) => (
                                    <option
                                        key={type.Id.toString()}
                                        value={type.Id.toString()}
                                    >
                                        {type.Label}
                                    </option>
                                ))}
                            </select>
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="pr-2">Script</span>
                            <input
                                type="file"
                                id="script"
                                name="script"
                                accept=".py"
                                className="input-basic"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="sr-only">Description</span>
                            <textarea
                                id="description"
                                name="descripton"
                                className="input-basic"
                                placeholder="Description"
                            ></textarea>
                        </label>
                    </div>
                </div>
                <div>
                    <button type="submit" className="btn-submit">
                        Create
                    </button>
                </div>
            </form>
        </div>
    );
}
