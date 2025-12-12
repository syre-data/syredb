import * as app from "../../wailsjs/go/main/App";
import { Suspense, use } from "react";
import {
    ErrorBoundary,
    FallbackProps as ErrorBoundaryProps,
} from "react-error-boundary";
import * as models from "../../wailsjs/go/models";
import { Link } from "react-router";
import icons from "../icons";

export default function Dashboard() {
    return (
        <div>
            <div className="flex gap-2">
                <h2 className="px-4 text-lg font-bold grow">Dashboard</h2>
                <div>
                    <Link to="/settings">
                        <button type="button" className="btn-cmd">
                            <icons.Gear />
                        </button>
                    </Link>
                </div>
            </div>
            <main>
                <UserRecentSampleGroups />
            </main>
        </div>
    );
}

function UserRecentSampleGroups() {
    return (
        <div>
            <div className="flex gap-2 px-4">
                <h3 className="grow">Samples</h3>
                <div className="flex gap-1">
                    <Link to="/sample_group/create">
                        <button
                            type="button"
                            className="block cursor-pointer"
                            title="Add new samples"
                        >
                            <icons.Plus />
                        </button>
                    </Link>
                </div>
            </div>
            <div>
                <ErrorBoundary FallbackComponent={RecentSampleGroupsError}>
                    <Suspense>
                        <UserRecentSampleGroupsInner
                            sampleGroupsPromise={app.GetRecentSampleGroupsWithSamples()}
                        />
                    </Suspense>
                </ErrorBoundary>
            </div>
        </div>
    );
}

function RecentSampleGroupsError({
    error,
    resetErrorBoundary,
}: ErrorBoundaryProps) {
    return (
        <div>
            <div>Could not get recent samples</div>
            <div>{error}</div>
        </div>
    );
}

function LoadingRecentSampleGroups() {
    return <div className="text-center">Loading samples</div>;
}

interface UserRecentSampleGroupsInnerProps {
    sampleGroupsPromise: Promise<models.main.Sample[]>;
}
function UserRecentSampleGroupsInner({
    sampleGroupsPromise,
}: UserRecentSampleGroupsInnerProps) {
    const sampleGroups = use(sampleGroupsPromise);

    return <div>sample groups</div>;
}
