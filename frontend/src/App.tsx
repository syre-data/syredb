import {
    Suspense,
    useEffect,
    useState,
    use,
    useContext,
    createContext,
    useReducer,
    FormEvent,
} from "react";
import logo from "./assets/images/logo-universal.png";
import "./App.css";
import * as runtime from "../wailsjs/runtime/runtime";
import * as app from "../wailsjs/go/main/App";
import * as models from "../wailsjs/go/models";
import { ErrorBoundary } from "react-error-boundary";
import * as appStateCtx from "./AppStateContext";
import Dashboard from "./Dashboard";
import AppConfig from "./AppConfig";

export default function App() {
    runtime.EventsOn("config_err", (e) => console.debug("err", e));

    return (
        <LoadAppState>
            <ConnectToDatabase>
                <Dashboard />
            </ConnectToDatabase>
        </LoadAppState>
    );
}

interface ChildrenProps {
    children: any;
}
function LoadAppState({ children }: ChildrenProps) {
    return (
        <ErrorBoundary FallbackComponent={ConfigError}>
            <Suspense fallback={<Loading />}>
                <LoadAppStateInner configPromise={app.GetConfig()}>
                    {children}
                </LoadAppStateInner>
            </Suspense>
        </ErrorBoundary>
    );
}

interface LoadAppStateProps {
    children: any;
    configPromise: Promise<models.main.AppConfig>;
}
function LoadAppStateInner({ children, configPromise }: LoadAppStateProps) {
    let config = use(configPromise);
    let [state, dispatch] = useReducer(
        appStateCtx.Reducer,
        new appStateCtx.State(config)
    );

    return (
        <appStateCtx.Context value={state}>
            <appStateCtx.Dispatch value={dispatch}>
                {children}
            </appStateCtx.Dispatch>
        </appStateCtx.Context>
    );
}

function Loading() {
    return (
        <div>
            <h2 className="pt-4 text-center dark:text-white">Loading</h2>
        </div>
    );
}

interface ConfigErrorProps {
    error: any;
    resetErrorBoundary: any;
}
function ConfigError({ error, resetErrorBoundary }: ConfigErrorProps) {
    return (
        <div className="pt-4 text-center dark:text-white">
            <h2 className="py-2">Could not load app config.</h2>
            <div>{error}</div>
        </div>
    );
}

function ConnectToDatabase({ children }: ChildrenProps) {
    const appState = useContext(appStateCtx.Context);
    const appStateDispatch = useContext(appStateCtx.Dispatch);
    const [configError, setConfigError] = useState("");

    function configIsvalid(config: models.main.AppConfig): boolean {
        return !!config.DbUrl && !!config.DbUsername && !!config.DbPassword;
    }

    function onSubmitAppConfig(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
        setConfigError("");
        const target = e.currentTarget;
        if (target === null) {
            console.error("could not get form");
            setConfigError("form error");
            return;
        }
        const data = new FormData(target as HTMLFormElement);
        const urlData = data.get("url");
        const usernameData = data.get("username");
        const passwordData = data.get("password");
        if (urlData === null) {
            setConfigError("invalid url");
            return;
        }
        const url = urlData.toString();
        if (!url.length) {
            setConfigError("invalid url");
            return;
        }
        if (usernameData === null) {
            setConfigError("invalid username");
            return;
        }
        const username = usernameData.toString();
        if (!username.length) {
            setConfigError("invalid username");
            return;
        }
        if (passwordData === null) {
            setConfigError("invalid password");
            return;
        }
        const password = passwordData.toString();
        if (!password.length) {
            setConfigError("invalid password");
            return;
        }

        let update = appState.config;
        update.DbUrl = url;
        update.DbUsername = username;
        update.DbPassword = password;
        app.SaveConfig(update)
            .then(() => {
                appStateDispatch({ type: "set_config", payload: update });
            })
            .catch((err) => setConfigError(err));
    }

    if (configIsvalid(appState.config)) {
        return (
            <ErrorBoundary FallbackComponent={DatabaseConnectionError}>
                <Suspense fallback={<ConnectingToDatabase />}>
                    <ConnectToDatabaseInner
                        connectPromise={app.ConnectToDatabase()}
                    >
                        {children}
                    </ConnectToDatabaseInner>
                </Suspense>
            </ErrorBoundary>
        );
    } else {
        return (
            <div className="text-center">
                <h2 className="text-lg py-4">Database configuration</h2>
                <AppConfig onsubmit={onSubmitAppConfig} />
                <div>{configError}</div>
            </div>
        );
    }
}

interface ConnectToDatabaseInnerProps {
    children: any;
    connectPromise: Promise<models.main.Ok>;
}
function ConnectToDatabaseInner({
    children,
    connectPromise,
}: ConnectToDatabaseInnerProps) {
    use(connectPromise);
    return children;
}

function ConnectingToDatabase() {
    return <div className="py-4 text-center">Connecting to database</div>;
}

interface DatabaseConnectionErrorProps {
    error: string;
    resetErrorBoundary: any;
}
function DatabaseConnectionError({
    error,
    resetErrorBoundary,
}: DatabaseConnectionErrorProps) {
    const appState = useContext(appStateCtx.Context);
    const appStateDispatch = useContext(appStateCtx.Dispatch);
    const [configError, setConfigError] = useState("");

    function onSubmitAppConfig(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
        setConfigError("");
        const target = e.currentTarget;
        if (target === null) {
            console.error("could not get form");
            setConfigError("form error");
            return;
        }
        const data = new FormData(target as HTMLFormElement);
        const urlData = data.get("url");
        const usernameData = data.get("username");
        const passwordData = data.get("password");
        if (urlData === null) {
            setConfigError("invalid url");
            return;
        }
        const url = urlData.toString();
        if (!url.length) {
            setConfigError("invalid url");
            return;
        }
        if (usernameData === null) {
            setConfigError("invalid username");
            return;
        }
        const username = usernameData.toString();
        if (!username.length) {
            setConfigError("invalid username");
            return;
        }
        if (passwordData === null) {
            setConfigError("invalid password");
            return;
        }
        const password = passwordData.toString();
        if (!password.length) {
            setConfigError("invalid password");
            return;
        }

        let update = appState.config;
        update.DbUrl = url;
        update.DbUsername = username;
        update.DbPassword = password;
        app.SaveConfig(update)
            .then(() => {
                appStateDispatch({ type: "set_config", payload: update });
                resetErrorBoundary();
            })
            .catch((err) => setConfigError(err));
    }

    return (
        <div className="text-center pt-4">
            <div>
                <h2 className="py-2">Could not connect to database</h2>
                <div className="px-4">{error}</div>
            </div>
            <div className="pt-4">
                <h3 className="pb-2">Update your database connection</h3>
                <AppConfig onsubmit={onSubmitAppConfig} />
                <div>{configError}</div>
            </div>
        </div>
    );
}
