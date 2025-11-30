import {
    Suspense,
    useEffect,
    useState,
    use,
    createContext,
    useReducer,
} from "react";
import logo from "./assets/images/logo-universal.png";
import "./App.css";
import * as runtime from "../wailsjs/runtime/runtime";
import * as app from "../wailsjs/go/main/App";
import * as models from "../wailsjs/go/models";
import { ErrorBoundary } from "react-error-boundary";
import Home from "./Home";
import * as appState from "./AppStateContext";

export default function App() {
    runtime.EventsOn("config_err", (e) => console.debug("err", e));

    return (
        <ErrorBoundary FallbackComponent={ConfigError}>
            <Suspense fallback={<Loading />}>
                <LoadAppState statePromise={app.GetConfig()}>
                    <Home />
                </LoadAppState>
            </Suspense>
        </ErrorBoundary>
    );
}

interface LoadAppStateProps {
    statePromise: Promise<models.main.AppConfig>;
    children: any;
}
function LoadAppState({ statePromise, children }: LoadAppStateProps) {
    let config = use(statePromise);
    let [state, dispatch] = useReducer(
        appState.Reducer,
        new appState.State(config)
    );

    return (
        <appState.Context value={state}>
            <appState.Dispatch value={dispatch}>{children}</appState.Dispatch>
        </appState.Context>
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
    error: string;
    resetErrorBoundary: any;
}
function ConfigError({ error, resetErrorBoundary }: ConfigErrorProps) {
    return (
        <div className="pt-4 text-center dark:text-white">
            <h2>Could not load app config.</h2>
            <div>{error}</div>
        </div>
    );
}
