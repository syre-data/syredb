import { Suspense, useContext } from "react";
import type { MouseEvent } from "react";
import icon from "../icon";
import { MouseButton, QUERY_KEY_USER } from "../common";
import * as appStateCtx from "../AppStateContext";
import { useNavigate, Link } from "react-router";
import auth_service from "@/service/auth.service";
import { ErrorBoundary } from "react-error-boundary";
import { useSuspenseQuery } from "@tanstack/react-query";
import type { DbPermission } from "@/types";

export default function () {
    return (
        <div className="flex">
            <Nav />
            <div>
                <h2 className="pt-2 px-2 font-bold text-2xl">Settings</h2>
            </div>
        </div>
    );
}

function Nav() {
    return (
        <div className="flex flex-col gap-4 text-2xl border-r">
            <div className="grow">
                <Link to="/" title="Dashboard">
                    <button type="button" className="btn-cmd">
                        <icon.Home />
                    </button>
                </Link>
            </div>
            <div>
                <div>
                    <Link to="/profile" title="Profile">
                        <button type="button" className="btn-cmd">
                            <icon.User />
                        </button>
                    </Link>
                </div>
                <div>
                    <Link to="/users" title="Users">
                        <button type="button" className="btn-cmd">
                            <icon.Users />
                        </button>
                    </Link>
                </div>
                <div>
                    <Link to="/data-schemas" title="Data schemas">
                        <button type="button" className="btn-cmd">
                            <icon.DataSchema />
                        </button>
                    </Link>
                </div>
                <div>
                    <Link to="/data-types" title="Data types">
                        <button type="button" className="btn-cmd">
                            <icon.DataType />
                        </button>
                    </Link>
                </div>
                <div>
                    <Link to="/data-origins" title="Data origins">
                        <button type="button" className="btn-cmd">
                            <icon.LocationPin />
                        </button>
                    </Link>
                </div>
                <div>
                    <Link
                        to="/data-type-transforms"
                        title="Data type transforms"
                    >
                        <button type="button" className="btn-cmd">
                            <icon.Function />
                        </button>
                    </Link>
                </div>
            </div>
            <div>
                <Link to="/logout" title="Log out">
                    <button type="button" className="btn-cmd">
                        <icon.Logout />
                    </button>
                </Link>
            </div>
        </div>
    );
}
