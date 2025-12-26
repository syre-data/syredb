import { useContext, FormEvent, useState, Suspense } from "react";
import * as appStateCtx from "./AppStateContext";
import { useSuspenseQuery } from "@tanstack/react-query";
import * as app from "../bindings/syredb/app";

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
            app.AppService.GetAppConfig().catch(() => new app.AppConfig()),
    });

    function onsubmit(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
        const btn_submit = document.getElementById(
            "submit"
        )! as HTMLButtonElement;
        btn_submit.disabled = true;

        const data = new FormData(e.target as HTMLFormElement);
        const urlData = data.get("url")!;
        const usernameData = data.get("username")!;
        const passwordData = data.get("password")!;
        const dbNameData = data.get("db-name")!;
        const url = urlData.toString();
        const username = usernameData.toString();
        const password = passwordData.toString();
        const dbName = dbNameData.toString();

        const config = new app.AppConfig({
            DbUrl: url,
            DbUsername: username,
            DbPassword: password,
            DbName: dbName,
        });
        app.AppService.SaveConfig(config)
            .then(async () => {
                try {
                    await app.AppService.LoadAppConfig();
                    btn_submit.disabled = true;
                    onsuccess();
                } catch (err) {
                    setError(JSON.stringify(err));
                    btn_submit.disabled = true;
                }
            })
            .catch((err) => {
                setError(err);
                btn_submit.disabled = false;
            });
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
                        id="submit"
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
