import { Context } from "@/AppStateContext";
import {
    hasDbPermission,
    MouseButton,
    QUERY_KEY_DATA_TYPES,
    QUERY_KEY_INGESTION_SCRIPTS,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import {
    DbPermissionIngestionScriptCreate,
    type DataType,
    type IngestionScript,
} from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Suspense, useContext, type MouseEvent } from "react";
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
    const { data: types } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPES],
        queryFn: dataService.dataTypesGetAll,
    });

    const navigate = useNavigate();

    const ctx = useContext(Context);
    const user = ctx.user;

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    const canCreateScript = hasDbPermission(
        DbPermissionIngestionScriptCreate,
        user.DbPermissions,
    );
    return (
        <div>
            <div className="flex justify-between px-4 pt-2">
                <div className="flex gap-2">
                    <h2 className="text-xl">Ingestion scripts</h2>
                    <div>
                        {canCreateScript ? (
                            <Link to="/ingestion-script/create">
                                <button className="btn-cmd align-middle">
                                    <Icon.Plus />
                                </button>
                            </Link>
                        ) : null}
                    </div>
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
            {scripts.length === 0 ? (
                <IngestionScriptsEmpty />
            ) : (
                <IngestionScriptsList scripts={scripts} types={types} />
            )}
        </div>
    );
}

function IngestionScriptsEmpty() {
    return (
        <div className="px-4">
            <div>You don't have any ingestion scripts yet.</div>
            <div>
                Click the <Icon.Plus className="inline" /> above to create your
                first one.
            </div>
        </div>
    );
}

interface IngestionScriptsListProps {
    scripts: IngestionScript[];
    types: DataType[];
}
function IngestionScriptsList({ scripts, types }: IngestionScriptsListProps) {
    return (
        <div className="px-4">
            <table>
                <tbody>
                    {scripts.map((script) => {
                        const type = types.find(
                            (type) => type.Id === script.Type,
                        )!;

                        return (
                            <IngestionScriptItem
                                key={script.Id.toString()}
                                script={script}
                                type={type}
                            />
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
}

interface IngestionScriptItemProps {
    script: IngestionScript;
    type: DataType;
}
function IngestionScriptItem({ script, type }: IngestionScriptItemProps) {
    return (
        <tr className="group">
            <td className="pr-2">{script.Label}</td>
            <td className="pr-2">{type.Label}</td>
            <td className="pr-2">
                <div className="invisible group-hover:visible">
                    <Link to={`/ingestion-script/${script.Id}`}>
                        <button type="button" className="btn-cmd">
                            <Icon.Eye />
                        </button>
                    </Link>
                </div>
            </td>
        </tr>
    );
}
