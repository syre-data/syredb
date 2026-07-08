import icon from "../icon";
import { Link } from "react-router";

export default function () {
    return (
        <div className="flex">
            <Nav />
            <div>
                <h1 className="pt-2 px-4 title">Settings</h1>
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
