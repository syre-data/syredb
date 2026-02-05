import { createContext } from "react";
import type { ActionDispatch } from "react";
import * as model from "../model";
import * as uuid from "uuid";

export class State {
    user: model.User;

    constructor(user: model.User) {
        this.user = user;
    }
}

export const Context = createContext(
    new State({
        Id: uuid.NIL,
        Email: "",
        Name: "",
        AccountStatus: model.ACCOUNT_STATUS_DISABLED,
        Role: model.USER_ROLE_USER,
    }),
);
export const Dispatch = createContext<ActionDispatch<[Action]>>(() => {});
export type Action =
    | { type: "set_user"; payload: model.User }
    | { type: "signout" };

export function Reducer(prevState: State, action: Action): State {
    switch (action.type) {
        case "set_user": {
            let update = structuredClone(prevState);
            update.user = action.payload;
            return update;
        }
    }

    throw new Error("invalid action", {
        cause: `recieved action of type ${action.type}`,
    });
}
