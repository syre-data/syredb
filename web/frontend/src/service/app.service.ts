import * as types from "@/types";

function getDbPermissions(): Promise<types.DbPermissionRecord[]> {
    return fetch("/api/app/db-permissions", {
        credentials: "same-origin",
    }).then(async (res) => await res.json());
}

export default {
    getDbPermissions,
};
