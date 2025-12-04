import { useContext, FormEvent, useState } from "react";
import * as models from "../wailsjs/go/models";
import * as appStateCtx from "./AppStateContext";

interface AppConfigProps {
    onsubmit: (e: FormEvent<HTMLFormElement>) => void;
}
export default function AppConfig({ onsubmit }: AppConfigProps) {
    const appState = useContext(appStateCtx.Context);

    return (
        <>
            <form onSubmit={onsubmit} className="inline-flex flex-col gap-2">
                <div className="inline-flex flex-col gap-2">
                    <div>
                        <label className="inline-flex gap-2 items-center">
                            <span className="hidden">Database URL</span>
                            <input
                                type="text"
                                name="url"
                                defaultValue={appState.config.DbUrl}
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
                                name="db-name"
                                defaultValue={appState.config.DbName}
                                placeholder="Database name"
                                className="block ring rounded-xs px-1 py-0.5"
                                minLength={1}
                            />
                        </label>
                    </div>
                    <div>
                        <label className="inline-flex gap-2 items-center">
                            <span className="hidden">Username</span>
                            <input
                                type="text"
                                name="username"
                                defaultValue={appState.config.DbUsername}
                                placeholder="Username"
                                autoComplete="username"
                                className="block ring rounded-xs px-1 py-0.5"
                                minLength={1}
                            />
                        </label>
                    </div>
                    <div>
                        <label className="inline-flex gap-2 items-center">
                            <span className="hidden">Password</span>
                            <input
                                type="password"
                                name="password"
                                defaultValue={appState.config.DbPassword}
                                placeholder="Password"
                                autoComplete="current-password"
                                className="block ring rounded-xs px-1 py-0.5"
                                minLength={1}
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
        </>
    );
}
