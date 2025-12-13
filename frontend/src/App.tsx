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
            <ProvideAppState>
                <ConnectToDatabase>
                    <LoadUser>
                        <Home />
                    </LoadUser>
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

function DatabaseConnectionError({
    error,
    resetErrorBoundary,
}: ErrorBoundaryProps) {
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
    return <Login onsuccess={resetErrorBoundary} />;
}

function LoadUserLoading() {
    return <div>Loading user</div>;
}

interface LoadUserInnerProps {
    children: any;
}
function LoadUserInner({ children }: LoadUserInnerProps) {
    const { data: user } = useSuspenseQuery({
        queryKey: ["_init_load_user"],
        queryFn: app.LoadUser,
        gcTime: 0,
    });

    if (user.Id.toString() === uuid.NIL) {
        return <Login />;
    } else {
        return <SetUser user={user}>{children}</SetUser>;
    }
}

interface SetUserProps {
    user: models.main.User;
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
                    onsuccess();
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
