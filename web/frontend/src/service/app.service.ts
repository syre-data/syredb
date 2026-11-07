import { AppError } from "@/errors";
import * as types from "@/types";

function getDbPermissions(): Promise<types.DbPermissionRecord[]> {
    return fetch("/api/app/db-permissions", {
        credentials: "same-origin",
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return (await res.json()) as types.DbPermissionRecord[];
    });
}

export default {
    getDbPermissions,
};
