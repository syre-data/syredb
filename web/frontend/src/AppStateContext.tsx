import { createContext } from "react";
import type { ActionDispatch } from "react";
import * as uuid from "uuid";
import { AccountStatusDeactivated, type User } from "./types";

export class State {
    user: User;

    constructor(user: User) {
        this.user = user;
    }
}

export const Context = createContext(
    new State({
        Id: uuid.NIL,
        Email: "",
        Name: "",
        AccountStatus: AccountStatusDeactivated,
        DbPermissions: [],
    }),
);
export const Dispatch = createContext<ActionDispatch<[Action]>>(() => {});
export type Action = { type: "set_user"; payload: User };

export function Reducer(prevState: State, action: Action): State {
    switch (action.type) {
        case "set_user": {
            let update = structuredClone(prevState);
            update.user = action.payload;
            return update;
        }
    }
}
