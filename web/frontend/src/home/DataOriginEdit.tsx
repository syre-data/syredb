import { MouseButton, QUERY_KEY_DATA_ORIGIN, uuidToString } from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Suspense, useState, type MouseEvent, type SubmitEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Navigate, useNavigate, useParams } from "react-router";
import { parse, type UUIDTypes } from "uuid";

export default function () {
    const { data_origin_id: origin_id_value } = useParams();
    if (!origin_id_value) {
        return <Navigate to="/" replace />;
    }
    const originId = parse(origin_id_value);

    return (
        <ErrorBoundary FallbackComponent={DataOriginError}>
            <Suspense fallback={<Loading />}>
                <DataOrigin originId={originId} />
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

interface DataOriginProps {
    originId: UUIDTypes;
}
function DataOrigin({ originId }: DataOriginProps) {
    const { data: origin } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_ORIGIN, originId],
        queryFn: async () => dataService.dataOriginById(originId),
    });
    const navigate = useNavigate();
    const [pending, setPending] = useState(false);

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    async function update(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        setPending(true);
        const data = new FormData(e.target);
        const label = data.get("label")!.toString().trim();
        const description = data.get("description")!.toString().trim();
        const active = data.has("active");

        await dataService
            .dataOriginUpdate({
                Id: uuidToString(originId),
                Label: label,
                Description: description,
                Active: active,
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
                    <h1>Data origin</h1>
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
                <form onSubmit={update}>
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
                                        defaultValue={origin.Label}
                                        placeholder="Label"
                                        minLength={1}
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
                                        defaultValue={origin.Description}
                                        placeholder="Description"
                                        rows={5}
                                        cols={40}
                                    />
                                </label>
                            </div>
                            <div>
                                <label>
                                    <input
                                        type="checkbox"
                                        id="active"
                                        name="active"
                                        className="input-basic"
                                        defaultChecked={origin.Active}
                                    />
                                    <span className="pl-2">Active</span>
                                </label>
                            </div>
                        </div>
                        <div>
                            <button
                                type="submit"
                                className="btn-submit"
                                disabled={pending}
                            >
                                Update
                            </button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    );
}
