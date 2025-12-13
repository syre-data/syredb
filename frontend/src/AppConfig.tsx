import { useContext, FormEvent, useState, Suspense } from "react";
import * as models from "../wailsjs/go/models";
import * as appStateCtx from "./AppStateContext";
import { useSuspenseQuery } from "@tanstack/react-query";
import * as app from "../wailsjs/go/main/App";

interface AppConfigProps {
    onsuccess?: () => void;
}

export default function AppConfig({ onsuccess = () => {} }: AppConfigProps) {
    return (
        <Suspense fallback={<Loading />}>
            <AppConfigInner onsuccess={onsuccess} />
        </Suspense>
    );
}

function Loading() {
    return <div className="pt-4 text-center">Loading app config</div>;
}

function AppConfigInner({ onsuccess = () => {} }: AppConfigProps) {
    const [error, setError] = useState("");
    const { data: config } = useSuspenseQuery({
        queryKey: ["app_config"],
        queryFn: () =>
            app.GetAppConfig().catch(() => new models.main.AppConfig()),
    });

    function onsubmit(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
        const target = e.currentTarget;
        if (target === null) {
            console.error("could not get form");
            return;
        }
        const data = new FormData(target as HTMLFormElement);
        const urlData = data.get("url")!;
        const usernameData = data.get("username")!;
        const passwordData = data.get("password")!;
        const dbNameData = data.get("db-name")!;
        const url = urlData.toString();
        const username = usernameData.toString();
        const password = passwordData.toString();
        const dbName = dbNameData.toString();

        const config = new models.main.AppConfig({
            DbUrl: url,
            DbUsername: username,
            DbPassword: password,
            DbName: dbName,
        });
        app.SaveConfig(config)
            .then(async () => {
                try {
                    await app.LoadAppConfig();
                    onsuccess();
                } catch (err) {
                    setError(JSON.stringify(err));
                }
            })
            .catch((err) => setError(err));
    }

    return (
        <div>
            <form onSubmit={onsubmit} className="inline-flex flex-col gap-2">
                <div className="inline-flex flex-col gap-2">
                    <div>
                        <label className="inline-flex gap-2 items-center">
                            <span className="hidden">Database URL</span>
                            <input
                                type="text"
                                name="url"
                                defaultValue={config.DbUrl}
                                placeholder="Database URL"
                                className="block ring rounded-xs px-1 py-0.5"
                                minLength={1}
                            />
                            {/* TODO: Include instructions that URL should take the form of host[:port] */}
                        </label>
                    </div>
                    <div>
                        <label className="inline-flex gap-2 items-center">
                            <span className="hidden">Database name</span>
                            <input
                                type="text"
                                id="db-name"
                                name="db-name"
                                defaultValue={config.DbName}
                                placeholder="Database name"
                                className="block ring rounded-xs px-1 py-0.5"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label className="inline-flex gap-2 items-center">
                            <span className="hidden">Username</span>
                            <input
                                type="text"
                                id="username"
                                name="username"
                                defaultValue={config.DbUsername}
                                placeholder="Username"
                                autoComplete="username"
                                className="block ring rounded-xs px-1 py-0.5"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label className="inline-flex gap-2 items-center">
                            <span className="hidden">Password</span>
                            <input
                                type="password"
                                id="password"
                                name="password"
                                defaultValue={config.DbPassword}
                                placeholder="Password"
                                autoComplete="current-password"
                                className="block ring rounded-xs px-1 py-0.5"
                                required
                            />
                        </label>
                    </div>
                </div>
                <div>
                    <button
                        type="submit"
                        className="cursor-pointer border rounded-sm px-1 py-0.5"
                    >
                        Save & Connect
                    </button>
                </div>
            </form>
            <div>{error}</div>
        </div>
    );
}
