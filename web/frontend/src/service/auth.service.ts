import * as types from "@/types";
import { AppError } from "@/errors";

function logout(): Promise<null> {
    return fetch("/logout", {
        credentials: "same-origin",
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return null;
    });
}

export default {
    logout,
};
