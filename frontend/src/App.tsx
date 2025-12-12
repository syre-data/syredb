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
import * as uuid from "uuid";
import {
    ErrorBoundary,
    FallbackProps as ErrorBoundaryProps,
} from "react-error-boundary";
import * as appStateCtx from "./AppStateContext";
import Home from "./home/Home";
import AppConfig from "./AppConfig";
import isEmail from "validator/es/lib/isEmail";
import {
    useSuspenseQuery,
    QueryClient,
    QueryClientProvider,
} from "@tanstack/react-query";

const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            retry: 0,
        },
    },
});

export default function App() {
    return (
        <QueryClientProvider client={queryClient}>
            <LoadAppState>
                <ConnectToDatabase>
                    <LoadUser>
                        <Home />
                    </LoadUser>
                </ConnectToDatabase>
            </LoadAppState>
        </QueryClientProvider>
    );
}

interface ChildrenProps {
    children: any;
}
function LoadAppState({ children }: ChildrenProps) {
    return (
        <ErrorBoundary FallbackComponent={ConfigError}>
            <Suspense fallback={<Loading />}>
                <LoadAppStateInner>{children}</LoadAppStateInner>
            </Suspense>
        </ErrorBoundary>
    );
}

interface LoadAppStateProps {
    children: any;
}
function LoadAppStateInner({ children }: LoadAppStateProps) {
    const { data: config } = useSuspenseQuery({
        queryKey: ["_init_load_app_config"],
        queryFn: app.GetConfig,
    });
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

function ConfigError({ error, resetErrorBoundary }: ErrorBoundaryProps) {
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
        const data = new FormData(target);
        const urlData = data.get("url")!;
        const usernameData = data.get("username")!;
        const passwordData = data.get("password")!;
        const dbNameData = data.get("db-name")!;
        const url = urlData.toString();
        if (!url.length) {
            setConfigError("invalid url");
            return;
        }
        const username = usernameData.toString();
        if (!username.length) {
            setConfigError("invalid username");
            return;
        }
        const password = passwordData.toString();
        if (!password.length) {
            setConfigError("invalid password");
            return;
        }
        const dbName = dbNameData.toString();
        if (!dbName.length) {
            setConfigError("invalid database name");
            return;
        }

        let update = appState.config;
        update.DbUrl = url;
        update.DbUsername = username;
        update.DbPassword = password;
        update.DbName = dbName;
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
                    <ConnectToDatabaseInner>{children}</ConnectToDatabaseInner>
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
}
function ConnectToDatabaseInner({ children }: ConnectToDatabaseInnerProps) {
    useSuspenseQuery({
        queryKey: ["_init_connect_to_db"],
        queryFn: app.ConnectToDatabase,
    });

    return children;
}

function ConnectingToDatabase() {
    return <div className="py-4 text-center">Connecting to database</div>;
}

function DatabaseConnectionError({
    error,
    resetErrorBoundary,
}: ErrorBoundaryProps) {
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
        const dbNameData = data.get("db-name");
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
        if (dbNameData === null) {
            setConfigError("invalid database name");
            return;
        }
        const dbName = dbNameData.toString();
        if (!dbName.length) {
            setConfigError("invalid database name");
            return;
        }

        let update = appState.config;
        update.DbUrl = url;
        update.DbUsername = username;
        update.DbPassword = password;
        update.DbName = dbName;
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

interface LoadUserProps {
    children: any;
}
function LoadUser({ children }: LoadUserProps) {
    const appState = useContext(appStateCtx.Context);
    if (appState.user.Id.toString() === uuid.NIL) {
        return (
            <ErrorBoundary FallbackComponent={LoadUserError}>
                <Suspense fallback={<LoadUserLoading />}>
                    <LoadUserInner>{children}</LoadUserInner>
                </Suspense>
            </ErrorBoundary>
        );
    } else {
        return children;
    }
}

function LoadUserError({ error, resetErrorBoundary }: ErrorBoundaryProps) {
    console.error("could not load user:", error);
    return <Login />;
}

function LoadUserLoading() {
    return <div>Loading user</div>;
}

interface LoadUserInnerProps {
    children: any;
}
function LoadUserInner({ children }: LoadUserInnerProps) {
    const appStateDispatch = useContext(appStateCtx.Dispatch);
    const { data: user } = useSuspenseQuery({
        queryKey: ["_init_load_user"],
        queryFn: app.LoadUser,
    });
    console.debug(user);

    if (user.Id.toString() === uuid.NIL) {
        return <Login />;
    } else {
        appStateDispatch({ type: "set_user", payload: user });
        console.debug("user");
        return children;
    }
}

function Login() {
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
        const data = new FormData(target);
        const emailData = data.get("email")!;
        const passwordData = data.get("password")!;
        const email = emailData.toString();
        const password = passwordData.toString();
        const remember = data.get("remember") !== null;

        if (!isEmail(email)) {
            const input = document.getElementById("email")! as HTMLInputElement;
            input.setCustomValidity("invalid email");
            return;
        }
        if (password.length < 1) {
            const input = document.getElementById(
                "password"
            )! as HTMLInputElement;
            input.setCustomValidity("invalid password");
            return;
        }

        const credentials = new models.main.UserCredentials({
            Email: email,
            Password: password,
        });
        app.AuthenticateAndGetUser(credentials, remember)
            .then((user) => {
                if (user.Id.toString() === uuid.NIL) {
                    setError("invalid user credentials");
                } else {
                    appStateDispatch({ type: "set_user", payload: user });
                }
            })
            .catch((err) => setError(err));
    }

    return (
        <div>
            <h2 className="text-center pt-4">Log in</h2>
            <form onSubmit={onsubmit}>
                <div>
                    <div>
                        <label>
                            Email
                            <input
                                id="email"
                                name="email"
                                type="email"
                                required
                                autoComplete="email"
                                className="input-basic user-invalid:ring-red-600"
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            Password
                            <input
                                id="password"
                                name="password"
                                type="password"
                                required
                                autoComplete="current-password"
                                className="input-basic user-invalid:ring-red-600"
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            Remember me
                            <input
                                id="remember"
                                name="remember"
                                type="checkbox"
                            />
                        </label>
                    </div>
                </div>
                <div>
                    <button type="submit" className="btn-submit">
                        Login
                    </button>
                </div>
            </form>
            <div>{error}</div>
        </div>
    );
}
