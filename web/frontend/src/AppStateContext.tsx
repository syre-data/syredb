import { createContext } from "react";
import type { ActionDispatch } from "react";
import * as types from "@/types";
import * as uuid from "uuid";

export class State {
    user: types.User;

    constructor(user: types.User) {
        this.user = user;
    }
}

export const Context = createContext(
    new State({
        Id: uuid.NIL,
        Email: "",
        Name: "",
        AccountStatus: types.AccountStatusDeactivated,
        Role: types.UserRoleUser,
    }),
);
export const Dispatch = createContext<ActionDispatch<[Action]>>(() => {});
export type Action = { type: "set_user"; payload: types.User };

export function Reducer(prevState: State, action: Action): State {
    switch (action.type) {
        case "set_user": {
            let update = structuredClone(prevState);
            update.user = action.payload;
            return update;
        }
    }
}
