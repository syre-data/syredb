import { useReducer, Suspense } from "react";
import "./App.css";
import * as appStateCtx from "./AppStateContext";
import Home from "./home/Home";
import * as model from "../model";
import icon from "@/icon";
import {
    QueryClient,
    QueryClientProvider,
    useSuspenseQuery,
} from "@tanstack/react-query";
import type { FallbackProps } from "react-error-boundary";
import { Loading } from "./components/Common";
import { ErrorBoundary } from "react-error-boundary";
import * as user_service from "@/service/user.service";

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
            <ErrorBoundary FallbackComponent={AppError}>
                <ProvideAppState>
                    <Home />
                </ProvideAppState>
            </ErrorBoundary>
        </QueryClientProvider>
    );
}

function AppError({ error, resetErrorBoundary }: FallbackProps) {
    console.error(error);
    return (
        <div className="flex flex-col gap-2 items-center pt-4">
            <div>An error occurred</div>
            <div className="flex gap-2 items-center">
                <div>
                    <a href="/">
                        <button type="button" className="btn-cmd">
                            <icon.Home />
                        </button>
                    </a>
                </div>
                <div>
                    <button
                        type="button"
                        onMouseDown={resetErrorBoundary}
                        className="btn-cmd"
                    >
                        <icon.Reload />
                    </button>
                </div>
            </div>
        </div>
    );
}

interface ProvideAppStateProps {
    children: any;
}
function ProvideAppState({ children }: ProvideAppStateProps) {
    const { data: user } = useSuspenseQuery({
        queryKey: ["user"],
        queryFn: user_service.get_user_by_jwt_token,
    });

    return (
        <ErrorBoundary FallbackComponent={LoadUserError}>
            <Suspense fallback={<Loading />}>
                <ProvideAppStateInner user={user}>
                    {children}
                </ProvideAppStateInner>
            </Suspense>
        </ErrorBoundary>
    );
}

function LoadUserError({ error, resetErrorBoundary }: FallbackProps) {
    console.error(error);
    return (
        <div className="flex flex-col gap-2 items-center pt-4">
            <div>Could not get user</div>
            <div className="flex gap-2 items-center">
                <div>
                    <a href="/">
                        <button type="button" className="btn-cmd">
                            <icon.Home />
                        </button>
                    </a>
                </div>
                <div>
                    <button
                        type="button"
                        onMouseDown={resetErrorBoundary}
                        className="btn-cmd"
                    >
                        <icon.Reload />
                    </button>
                </div>
            </div>
        </div>
    );
}

interface ProvideAppStateInnerProps {
    user: model.User;
    children: any;
}
function ProvideAppStateInner({ user, children }: ProvideAppStateInnerProps) {
    const [state, dispatch] = useReducer(
        appStateCtx.Reducer,
        new appStateCtx.State(user),
    );

    return (
        <appStateCtx.Context value={state}>
            <appStateCtx.Dispatch value={dispatch}>
                {children}
            </appStateCtx.Dispatch>
        </appStateCtx.Context>
    );
}
