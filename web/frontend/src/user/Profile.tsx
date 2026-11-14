import { Context } from "@/AppStateContext";
import { QUERY_KEY_DB_PERMISSIONS } from "@/common";
import { Loading, StandardError } from "@/components";
import { MainNav, PageUserProfile } from "@/components/Nav";
import Icon from "@/icon";
import appService from "@/service/app.service";
import userService from "@/service/user.service";
import { useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";
import { StatusCodes } from "http-status-codes";
import {
    Suspense,
    useContext,
    useState,
    type ComponentProps,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { toast } from "sonner";
import { DbPermissionBadges } from "./components";

export default function () {
    return (
        <div className="flex">
            <MainNav
                current={PageUserProfile}
                className="sticky top-0 h-screen py-2"
            />
            <ErrorBoundary FallbackComponent={ProfileError}>
                <Suspense fallback={<Loading />}>
                    <UserProfile className="grow" />
                </Suspense>
            </ErrorBoundary>
        </div>
    );
}

function ProfileError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <StandardError
            resetErrorBoundary={resetErrorBoundary}
            error={error as Error}
            title={"Could not load database permissions"}
        />
    );
}

function UserProfile(props: ComponentProps<"main">) {
    const ctx = useContext(Context);
    const user = ctx.user;

    const { data: dbPermissions } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DB_PERMISSIONS],
        queryFn: appService.getDbPermissions,
    });

    const permissions = user.DbPermissions.map((perm) => {
        const info = dbPermissions.find((p) => p.Id === perm);
        if (!info) {
            throw new Error(`invalid database permission ${perm}`);
        }

        return info;
    });

    return (
        <main {...props}>
            <div className="px-4 pt-2 flex justify-between">
                <div>
                    <h2 className="text-lg">{user.Name}</h2>
                    <div className="text-secondary">{user.Email}</div>
                </div>
            </div>
            <div className="py-4">
                <div className="px-4 flex gap-2 items-center text-lg">
                    <Icon.Shield />
                    <h2>Database permissions</h2>
                </div>
                <DbPermissionBadges
                    permissions={permissions}
                    className="px-4 pt-2 flex gap-2"
                />
            </div>
            <div className="px-4 pt-2">
                <div className="pt-4 border-t border-gray-600">
                    <h2 className="text-lg">Change password</h2>
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
        </main>
    );
}

function PasswordUpdate() {
    const [invalidNew, setInvalidNew] = useState(false);
    const [pending, setPending] = useState(false);
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
        await userService
            .passwordUpdate(current, password)
            .then((_) => {
                toast.success("Password updated");
            })
            .catch((resp) => {
                if (resp.status === StatusCodes.UNAUTHORIZED) {
                    currentInput.ariaInvalid = "true";
                    setUpdateErr("Current password is invalid");
                } else {
                    toast.error("Could not update password");
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
                            className="input-basic w-full"
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
                            className="input-basic w-full"
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
                            className="input-basic w-full"
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
                <div>{updateErr === "" ? null : updateErr}</div>
            </div>
        </form>
    );
}
