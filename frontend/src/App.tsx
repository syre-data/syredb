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
import * as app from "../wailsjs/go/app/App";
import * as models from "../wailsjs/go/models";
import * as uuid from "uuid";
import { ErrorBoundary, FallbackProps } from "react-error-boundary";
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
            <ProvideAppState>
                <ConnectToDatabase>
                    <LoadUserFromAuthFile>
                        <Home />
                    </LoadUserFromAuthFile>
                </ConnectToDatabase>
            </ProvideAppState>
        </QueryClientProvider>
    );
}

interface ChildrenProps {
    children: any;
}
function ProvideAppState({ children }: ChildrenProps) {
    const [state, dispatch] = useReducer(
        appStateCtx.Reducer,
        new appStateCtx.State()
    );

    return (
        <appStateCtx.Context value={state}>
            <appStateCtx.Dispatch value={dispatch}>
                {children}
            </appStateCtx.Dispatch>
        </appStateCtx.Context>
    );
}

function ConnectToDatabase({ children }: ChildrenProps) {
    return (
        <ErrorBoundary FallbackComponent={DatabaseConnectionError}>
            <Suspense fallback={<ConnectingToDatabase />}>
                <ConnectToDatabaseInner>{children}</ConnectToDatabaseInner>
            </Suspense>
        </ErrorBoundary>
    );
}

interface ConnectToDatabaseInnerProps {
    children: any;
}
function ConnectToDatabaseInner({ children }: ConnectToDatabaseInnerProps) {
    useSuspenseQuery({
        queryKey: ["_init_connect_to_db"],
        queryFn: app.ConnectToDatabase,
        gcTime: 0,
    });

    return children;
}

function ConnectingToDatabase() {
    return <div className="py-4 text-center">Connecting to database</div>;
}

function DatabaseConnectionError({ error, resetErrorBoundary }: FallbackProps) {
    if (error === "FILE_NOT_FOUND") {
        return (
            <div className="text-center pt-4">
                <h3 className="pb-2">Setup your database connection</h3>
                <AppConfig onsuccess={resetErrorBoundary} />
            </div>
        );
    } else {
        return (
            <div className="text-center pt-4">
                <div>
                    <h2 className="py-2">Could not connect to database</h2>
                    <div className="px-4">{error + ""}</div>
                </div>
                <div className="pt-4">
                    <h3 className="pb-2">Update your database connection</h3>
                    <AppConfig onsuccess={resetErrorBoundary} />
                </div>
            </div>
        );
    }
}

interface LoadUserProps {
    children: any;
}
function LoadUserFromAuthFile({ children }: LoadUserProps) {
    const appState = useContext(appStateCtx.Context);
    if (appState.user.Id.toString() === uuid.NIL) {
        return (
            <ErrorBoundary FallbackComponent={LoadUserFromAuthFileError}>
                <Suspense fallback={<LoadUserFromAuthFileLoading />}>
                    <LoadUserFromAuthFileInner>
                        {children}
                    </LoadUserFromAuthFileInner>
                </Suspense>
            </ErrorBoundary>
        );
    } else {
        return children;
    }
}

function LoadUserFromAuthFileError({
    error,
    resetErrorBoundary,
}: FallbackProps) {
    console.error("could not load user:", error);
    return <Login onsuccess={resetErrorBoundary} />;
}

function LoadUserFromAuthFileLoading() {
    return <div>Loading user</div>;
}

interface LoadUserFromAuthFileInnerProps {
    children: any;
}
function LoadUserFromAuthFileInner({
    children,
}: LoadUserFromAuthFileInnerProps) {
    const { data: user } = useSuspenseQuery({
        queryKey: ["_init_load_user"],
        queryFn: app.LoadUserFromAuthFile,
        gcTime: 0,
    });

    if (user.Id.toString() === uuid.NIL) {
        return <Login />;
    } else {
        return <SetUser user={user}>{children}</SetUser>;
    }
}

interface SetUserProps {
    user: models.app.User;
    children: any;
}
function SetUser({ user, children }: SetUserProps) {
    const appStateDispatch = useContext(appStateCtx.Dispatch);
    useEffect(() => {
        appStateDispatch({ type: "set_user", payload: user });
    }, [appStateDispatch, user]);
    return children;
}

interface LoginProps {
    onsuccess?: () => void;
}
function Login({ onsuccess = () => {} }: LoginProps) {
    const appStateDispatch = useContext(appStateCtx.Dispatch);
    const [error, setError] = useState("");

    function onsubmit(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
        setError("");
        const btn_submit = document.getElementById(
            "submit"
        )! as HTMLButtonElement;
        btn_submit.disabled = true;

        const data = new FormData(e.target as HTMLFormElement);
        const email = data.get("email")!.toString();
        const password = data.get("password")!.toString();
        const remember = data.get("remember") !== null;

        if (!isEmail(email)) {
            console.debug(email);
            const input = document.getElementById("email")! as HTMLInputElement;
            input.setCustomValidity("invalid email");
            btn_submit.disabled = false;
            return;
        }

        const credentials = new models.app.UserCredentials({
            Email: email,
            Password: password,
        });
        app.AuthenticateAndGetUser(credentials, remember)
            .then((user) => {
                if (user.Id.toString() === uuid.NIL) {
                    setError("invalid user credentials");
                } else {
                    appStateDispatch({ type: "set_user", payload: user });
                    btn_submit.disabled = false;
                    onsuccess();
                }
            })
            .catch((err) => {
                setError(err);
                btn_submit.disabled = false;
            });
    }

    return (
        <div className="flex flex-col gap-2 items-center pt-4">
            <h2>Log in</h2>
            <div className="text-red-600">{error}</div>
            <form
                onSubmit={onsubmit}
                className="flex flex-col gap-2 items-center"
            >
                <div className="flex flex-col gap-2 items-center">
                    <div>
                        <label>
                            <span className="hidden">Email</span>
                            <input
                                id="email"
                                name="email"
                                type="email"
                                autoComplete="email"
                                placeholder="Email"
                                className="input-basic user-invalid:ring-red-600"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="hidden">Password</span>
                            <input
                                id="password"
                                name="password"
                                type="password"
                                autoComplete="current-password"
                                placeholder="Password"
                                className="input-basic user-invalid:ring-red-600"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            <input
                                id="remember"
                                name="remember"
                                type="checkbox"
                            />
                            <span className="pl-2">Remember me</span>
                        </label>
                    </div>
                </div>
                <div>
                    <button type="submit" id="submit" className="btn-submit">
                        Login
                    </button>
                </div>
            </form>
        </div>
    );
}
