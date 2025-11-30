import { useContext, FormEvent, useState } from "react";
import * as models from "../wailsjs/go/models";
import * as appStateCtx from "./AppStateContext";
import { SaveConfig } from "../wailsjs/go/main/App";

export default function Home() {
    const appConfig = useContext(appStateCtx.Context);
    if (!appConfig.config.DbUrl) {
        return <AppConfig />;
    } else {
        return <div>good to go</div>;
    }
}

function AppConfig() {
    const appState = useContext(appStateCtx.Context);
    const appStateDispatch = useContext(appStateCtx.Dispatch);
    const [error, setError] = useState("");

    function onsubmit(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
        setError("");
        const target = e.currentTarget;
        if (target === null) {
            console.error("could not get form");
            setError("form error");
            return;
        }
        const data = new FormData(target as HTMLFormElement);
        const urlData = data.get("url");
        if (urlData === null) {
            setError("invalid url");
            return;
        }
        const url = urlData.toString();
        if (!url.length) {
            setError("invalid url");
            return;
        }

        let update = appState.config;
        update.DbUrl = url;
        SaveConfig(update)
            .then(() => {
                appStateDispatch({ type: "set_config", payload: update });
            })
            .catch((err) => setError(err));
    }

    return (
        <div className="text-center">
            <h2 className="text-lg py-4">Database configuration</h2>
            <form onSubmit={onsubmit} className="inline-flex flex-col gap-2">
                <div>
                    <label className="inline-flex gap-2 items-center">
                        <span className="block">Database URL</span>
                        <input
                            type="text"
                            name="url"
                            className="block ring rounded-xs px-1 py-0.5"
                            minLength={1}
                        />
                    </label>
                </div>
                <div>
                    <button
                        type="submit"
                        className="cursor-pointer border rounded-sm px-1 py-0.5"
                    >
                        Save
                    </button>
                </div>
            </form>
            <div>{error}</div>
        </div>
    );
}
