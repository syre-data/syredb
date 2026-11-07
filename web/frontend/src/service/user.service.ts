import { uuidToString } from "@/common";
import { AppError } from "@/errors";
import * as types from "@/types";
import { type UUIDTypes } from "uuid";

function get_user_by_jwt_token(): Promise<types.User> {
    return fetch("/api/user", {
        credentials: "same-origin",
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return (await res.json()) as types.User;
    });
}

function get_users(): Promise<types.User[]> {
    return fetch("/api/users", {
        credentials: "same-origin",
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return (await res.json()) as types.User[];
    });
}

function update_user(user: types.User): Promise<null> {
    return fetch("/api/user", {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(user),
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return null;
    });
}

function deactivate_user(user: UUIDTypes): Promise<null> {
    return fetch("/api/user/deactivate", {
        credentials: "same-origin",
        method: "put",
        body: user.toString(),
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return null;
    });
}

function userCreate(user: types.UserCreate): Promise<UUIDTypes> {
    return fetch("/api/user/create", {
        credentials: "same-origin",
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(user),
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        const body = (await res.json) as types.ResourceCreated;
        return body.id;
    });
}

function user(user: UUIDTypes): Promise<types.User> {
    const id_str = uuidToString(user);
    const params = new URLSearchParams();
    params.append("id", id_str);
    return fetch(`/api/user?${params}`).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return (await res.json()) as types.User;
    });
}

function userUpdate(user: types.User): Promise<null> {
    return fetch("/api/user", {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(user),
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return null;
    });
}

function passwordReset(user: UUIDTypes): Promise<null> {
    return fetch("/api/user/password/reset", {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user }),
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return null;
    });
}

function passwordUpdate(current: string, update: string): Promise<null> {
    return fetch("/api/user/password/update", {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ current, update }),
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return null;
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
    passwordReset,
    passwordUpdate,
};
