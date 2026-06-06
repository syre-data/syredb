import { Context } from "@/AppStateContext";
import {
    hasDbPermission,
    MouseButton,
    QUERY_KEY_INGESTION_SCRIPTS,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import {
    DbPermissionIngestionScriptCreate,
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
                <IngestionScriptsList scripts={scripts} />
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
}
function IngestionScriptsList({ scripts }: IngestionScriptsListProps) {
    return (
        <ul className="px-4">
            {scripts.map((script) => (
                <li key={script.Id.toString()}>
                    <IngestionScriptItem script={script} />
                </li>
            ))}
        </ul>
    );
}

interface IngestionScriptItemProps {
    script: IngestionScript;
}
function IngestionScriptItem({ script }: IngestionScriptItemProps) {
    return <div>{script.Label}</div>;
}
