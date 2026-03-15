import * as types from "@/types";
import type { UUIDTypes } from "uuid";
function get_user_by_jwt_token(): Promise<types.User> {
    return fetch("/api/user", {
        credentials: "same-origin",
    }).then(async (res) => (await res.json()) as types.User);
}

function get_users(): Promise<types.User[]> {
    return fetch("/api/users", {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.User[]);
}

function update_user(user: types.User): Promise<Response> {
    return fetch("/api/user", {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(user),
    });
}

function deactivate_user(user: UUIDTypes): Promise<Response> {
    return fetch("/api/user/deactivate", {
        credentials: "same-origin",
        method: "put",
        body: user.toString(),
    });
}

function userCreate(user: types.UserCreate): Promise<Response> {
    return fetch("/api/user/create", {
        credentials: "same-origin",
        method: "post",
        body: JSON.stringify(user),
    });
}

// function userResources(user: UUIDTypes) : Promise<types.UserResources> {
//     const params = new URLSearchParams();
//     params.append("id", user.toString())
//     return fetch (`/api/user/resources?${params}`).then(async resp=> {
//         await resp.json() as types.UserResources
//     })
// }

function user(user: UUIDTypes): Promise<types.User> {
    const params = new URLSearchParams();
    params.append("id", user.toString());
    return fetch(`/api/user?${params}`).then(
        async (resp) => (await resp.json()) as types.User,
    );
}

function userUpdate(user: types.User): Promise<Response> {
    return fetch("/api/user", {
        credentials: "same-origin",
        method: "put",
        body: JSON.stringify(user),
    });
}

export default {
    get_user_by_jwt_token,
    get_users,
    update_user,
    deactivate_user,
    userCreate,
    user,
    userUpdate,
    //  userResources,
};
