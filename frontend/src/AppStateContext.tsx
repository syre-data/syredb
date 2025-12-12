import React, { ActionDispatch, createContext } from "react";
import * as models from "../wailsjs/go/models";
import * as uuid from "uuid";

export class State {
    config: models.main.AppConfig;
    user: models.main.User;

    constructor(config: models.main.AppConfig) {
        this.config = config;
        this.user = new models.main.User({
            Id: uuid.NIL,
            Email: "",
            Name: "",
            PermissionRoles: null,
        });
    }
}

export const Context = createContext(new State(new models.main.AppConfig()));
export const Dispatch = createContext<ActionDispatch<[Action]>>(() => {});
export type Action =
    | { type: "set_config"; payload: models.main.AppConfig }
    | { type: "set_user"; payload: models.main.User };

export function Reducer(state: State, action: Action) {
    console.debug("reduver");
    switch (action.type) {
        case "set_config": {
            let update = structuredClone(state);
            update.config = action.payload;
            return update;
        }

        case "set_user": {
            let update = structuredClone(state);
            update.user = action.payload;
            return update;
        }

        // TODO: Include for safety?
        // default: {
        //     throw Error("Unknown action: " + action.type);
        // }
    }
}
