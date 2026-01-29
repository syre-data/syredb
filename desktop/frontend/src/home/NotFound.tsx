import { Link, useNavigate } from "react-router";
import icon from "../icon";
import { MouseEvent } from "react";
import * as common from "../common";

export default function () {
    const navigate = useNavigate();

    function back(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    return (
        <div className="text-center text-xl pt-4">
            <div className="pb-2">Oops, looks like this page doesn't exist</div>
            <div className="flex justify-center gap-2">
                <div>
                    <Link to="/" className="cursor-pointer">
                        <icon.Home className="" />
                    </Link>
                </div>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={back}
                    >
                        <icon.BackArrow />
                    </button>
                </div>
            </div>
        </div>
    );
}
