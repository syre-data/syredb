import {
    MouseButton,
    QUERY_KEY_INGESTION_SCRIPT,
    uuidToString,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import {
    DataSourceCardinalityMultiple,
    DataSourceCardinalitySingle,
    type DataType,
    type IngestionScript,
} from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import type { types } from "node:util";
import { Suspense, type MouseEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, Navigate, useNavigate, useParams } from "react-router";
import type { UUIDTypes } from "uuid";

export default function () {
    const { id } = useParams();
    if (!id) {
        return <Navigate to="/" />;
    }

    return (
        <ErrorBoundary FallbackComponent={IngestionScriptsError}>
            <Suspense fallback={<Loading />}>
                <IngestionScript id={id} />
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

interface IngestionScriptProps {
    id: UUIDTypes;
}
function IngestionScript({ id }: IngestionScriptProps) {
    const { data: resources } = useSuspenseQuery({
        queryKey: [QUERY_KEY_INGESTION_SCRIPT, id],
        queryFn: async () => await dataService.ingestionScriptResources(id),
    });

    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    const script: IngestionScript = resources.Script;
    const data_type: DataType = resources.DataType;
    const script_download_params = new URLSearchParams();
    script_download_params.set("id", uuidToString(id));
    const script_download = `/resource/ingestion-script?${script_download_params}`;
    return (
        <div>
            <div className="flex justify-between px-4 pt-2">
                <h1 className="text-lg font-bold">{script.Label}</h1>
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
            <div className="pt-2">
                <div className="px-4">Data type {data_type.Label}</div>
                <div className="px-4">
                    {script.Description
                        ? script.Description
                        : "(no description)"}
                </div>
                <div className="px-4 pt-2">
                    <h2>Sources</h2>
                    <table>
                        <tbody>
                            {script.Sources.map((source) => {
                                let cardinality;
                                switch (source.Cardinality) {
                                    case DataSourceCardinalitySingle:
                                        cardinality = (
                                            <Icon.File title="Single file only" />
                                        );
                                        break;
                                    case DataSourceCardinalityMultiple:
                                        cardinality = (
                                            <Icon.Files title="Multiple files allowed" />
                                        );
                                        break;
                                }
                                return (
                                    <tr key={source.Id.toString()}>
                                        <td className="pr-2">{source.Label}</td>
                                        <td className="pr-2">
                                            {source.Description
                                                ? source.Description
                                                : "(no description)"}
                                        </td>
                                        <td className="pr-2">{cardinality}</td>
                                        <td className="pr-2">
                                            {source.Required ? (
                                                <Icon.Check title="Required" />
                                            ) : null}
                                        </td>
                                        <td className="pr-2">
                                            {source.ExtFilter
                                                ? source.ExtFilter
                                                : "(no filter)"}
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                </div>
                <div className="px-4 pt-2">
                    <div className="flex gap-2 group">
                        <h2>Script</h2>
                        <div className="invisible group-hover:visible">
                            <a href={script_download} title="Download">
                                <button type="button" className="btn-cmd">
                                    <Icon.Download />
                                </button>
                            </a>
                        </div>
                    </div>
                    <pre>{resources.CmdScript}</pre>
                </div>
            </div>
        </div>
    );
}
