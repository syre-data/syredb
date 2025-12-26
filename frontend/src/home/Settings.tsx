import { MouseEvent, useContext } from "react";
import icon from "../icon";
import { MouseButton } from "../common";
import * as appStateCtx from "../AppStateContext";
import * as app from "../../bindings/syredb/app";
import { useNavigate } from "react-router";

export default function Settings() {
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
    const appStateDispatch = useContext(appStateCtx.Dispatch);
    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    async function logout(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        await app.AppService.Logout()
            .then(() => {
                appStateDispatch({ type: "signout" });
            })
            .catch((err) => {
                console.error("could not remove user auth file", err);
                // TODO: alert user
            });
    }

    return (
        <div className="flex flex-col gap-4 text-2xl border-r">
            <div className="grow">
                <button
                    type="button"
                    onMouseDown={close}
                    title="Close"
                    className="btn-cmd px-2"
                >
                    <icon.Close />
                </button>
            </div>
            <div>
                <button
                    type="button"
                    onMouseDown={logout}
                    title="Log out"
                    className="btn-cmd px-2"
                >
                    <icon.Logout />
                </button>
            </div>
        </div>
    );
}
