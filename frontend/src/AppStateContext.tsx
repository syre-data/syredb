import React, { ActionDispatch, createContext } from "react";
import * as models from "../wailsjs/go/models";
import * as uuid from "uuid";

export class State {
    user: models.app.User;

    constructor() {
        this.user = new models.app.User({
            Id: uuid.NIL,
            Email: "",
            Name: "",
            PermissionRoles: [],
        });
    }
}

export const Context = createContext(new State());
export const Dispatch = createContext<ActionDispatch<[Action]>>(() => {});
export type Action =
    | { type: "set_user"; payload: models.app.User }
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
            update.user = new models.app.User({
                Id: uuid.NIL,
                Email: "",
                Name: "",
                PermissionRoles: [],
            });
            return update;
        }
    }
}
