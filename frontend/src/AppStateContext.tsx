import React, { ActionDispatch, createContext } from "react";
import * as app from "../bindings/syredb/app";
import * as uuid from "uuid";

export class State {
    user: app.User;

    constructor() {
        this.user = new app.User({
            Id: uuid.NIL,
            Email: "",
            Name: "",
            Role: app.UserRole.$zero,
        });
    }
}

export const Context = createContext(new State());
export const Dispatch = createContext<ActionDispatch<[Action]>>(() => {});
export type Action =
    | { type: "set_user"; payload: app.User }
    | { type: "signout" };

export function Reducer(state: State, action: Action) {
    switch (action.type) {
        case "set_user": {
            let update = structuredClone(state);
            update.user = action.payload;
            return update;
        }

        case "signout": {
            let update = structuredClone(state);
            update.user = new app.User({
                Id: uuid.NIL,
                Email: "",
                Name: "",
                Role: app.UserRole.$zero,
            });
            return update;
        }
    }
}
