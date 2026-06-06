import { Context } from "@/AppStateContext";
import { MouseButton } from "@/common";
import Icon from "@/icon";
import userService from "@/service/user.service";
import classNames from "classnames";
import { StatusCodes } from "http-status-codes";
import {
    useContext,
    useState,
    type InputEvent,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { useNavigate } from "react-router";

export default function () {
    const navigate = useNavigate();
    const ctx = useContext(Context);
    const user = ctx.user;

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <div>
                    <h2 className="text-lg">{user.Name}</h2>
                    <div>{user.Email}</div>
                </div>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={close}
                    >
                        <Icon.Close />
                    </button>
                </div>
            </div>
            <div className="px-4 pt-4">
                <h3>Permissions</h3>
                <ul>
                    {user.DbPermissions.map((perm) => {
                        return <li key={perm}>{perm}</li>;
                    })}
                </ul>
            </div>
            <div className="px-4 pt-4">
                <div>
                    <h3>Change password</h3>
                    <div>
                        <small>
                            <strong className="pr-2">
                                Forgot your password?
                            </strong>
                            Ask an administrator to reset it for you.
                        </small>
                    </div>
                </div>
                <div className="pt-2">
                    <PasswordUpdate />
                </div>
            </div>
        </div>
    );
}

function PasswordUpdate() {
    const [invalidNew, setInvalidNew] = useState(false);
    const [pending, setPending] = useState(false);
    const [success, setSuccess] = useState(false);
    const [updateErr, setUpdateErr] = useState("");

    function isValid(password: string): boolean {
        setInvalidNew(false);

        const lowerCase = new RegExp(/[a-z]/);
        const upperCase = new RegExp(/[A-Z]/);
        const numberPtrn = new RegExp(/\d/);
        const specialChar = new RegExp(/[!@#$%^&*]/);
        if (
            password.length < 8 ||
            password.length > 32 ||
            !lowerCase.test(password) ||
            !upperCase.test(password) ||
            !numberPtrn.test(password) ||
            !specialChar.test(password)
        ) {
            setInvalidNew(true);
            return false;
        }

        return true;
    }

    function validateNewPassword(): boolean {
        const newInput = document.getElementById(
            "password-new",
        )! as HTMLInputElement;
        const repeatInput = document.getElementById(
            "password-repeat",
        )! as HTMLInputElement;
        repeatInput.setCustomValidity("");

        const val = newInput.value;
        const repeat = repeatInput.value;

        if (val === "" && repeat === "") {
            return false;
        }

        const pwValid = isValid(val);
        if (!pwValid) {
            return false;
        }

        if (val !== repeat) {
            repeatInput.setCustomValidity("Does not match");
            return false;
        }

        return true;
    }

    async function updatePassword(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        const currentInput = document.getElementById(
            "password-current",
        )! as HTMLInputElement;
        currentInput.ariaInvalid = "false";

        const data = new FormData(e.target);
        if (!validateNewPassword()) {
            return;
        }

        const current = data.get("password-current")!.toString();
        const password = data.get("password-new")!.toString();
        setPending(true);
        await userService.passwordUpdate(current, password).then((resp) => {
            if (resp.status === StatusCodes.OK) {
                setSuccess(true);
            } else if (resp.status === StatusCodes.UNAUTHORIZED) {
                currentInput.ariaInvalid = "true";
                setUpdateErr("Current password is invalid");
            } else {
                //TODO: Report error
            }
        });
        setPending(false);
    }

    return (
        <form onSubmit={updatePassword}>
            <div>
                <div className="pb-2">
                    <label>
                        <span className="sr-only">Current password</span>
                        <input
                            id="password-current"
                            name="password-current"
                            type="password"
                            placeholder="Current password"
                            className="input-basic"
                        />
                    </label>
                </div>
                <div className="pb-2">
                    <label>
                        <span className="sr-only">New password</span>
                        <input
                            id="password-new"
                            name="password-new"
                            type="password"
                            placeholder="New password"
                            className="input-basic"
                        />
                    </label>
                    <div
                        className={classNames({
                            hidden: !invalidNew,
                        })}
                    >
                        <ul>
                            <li>Between 8 and 32 characters long</li>
                            <li>Include 1 number</li>
                            <li>
                                Include 1 uppercase and 1 lowercase character
                            </li>
                            <li>
                                Include 1 special character (!, @, #, $, %, ^,
                                &, *)
                            </li>
                        </ul>
                    </div>
                </div>
                <div className="pb-2">
                    <label>
                        <span className="sr-only">Repeat new password</span>
                        <input
                            id="password-repeat"
                            name="password-reapeat"
                            type="password"
                            placeholder="Repeat new password"
                            className="input-basic"
                        />
                    </label>
                </div>
            </div>
            <div className="pt-2">
                <button type="submit" className="btn-submit" disabled={pending}>
                    Change password
                </button>
            </div>
            <div>
                <div>{success ? "Password updated" : null}</div>
                <div>{updateErr === "" ? null : updateErr}</div>
            </div>
        </form>
    );
}
