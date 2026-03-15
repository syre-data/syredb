import { useReducer, Suspense } from "react";
import "./App.css";
import * as appStateCtx from "./AppStateContext";
import Home from "./home/Home";
import * as types from "@/types";
import {
    QueryClient,
    QueryClientProvider,
    useSuspenseQuery,
} from "@tanstack/react-query";
import type { FallbackProps } from "react-error-boundary";
import Loading from "./components/Loading";
import { ErrorBoundary } from "react-error-boundary";
import * as common from "@/common";
import user_service from "@/service/user.service";
import { SuspenseError } from "./components";

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
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="pt-4 text-center"
        >
            <div>An error occurred</div>
        </SuspenseError>
    );
}

interface ProvideAppStateProps {
    children: any;
}
function ProvideAppState({ children }: ProvideAppStateProps) {
    const { data: user } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_USER],
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
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="pt-4 text-center"
        >
            <div>Could not get user</div>
        </SuspenseError>
    );
}

interface ProvideAppStateInnerProps {
    user: types.User;
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
