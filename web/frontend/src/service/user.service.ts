import * as model from "@/../model";
export async function get_user_by_jwt_token(): Promise<model.User> {
    return fetch("/api/user", {
        credentials: "same-origin",
    }).then(async (res) => (await res.json()) as model.User);
}
