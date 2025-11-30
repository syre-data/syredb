import React, { ActionDispatch, createContext } from "react";
import * as models from "../wailsjs/go/models";

export class State {
    config: models.main.AppConfig;

    constructor(config: models.main.AppConfig) {
        this.config = config;
    }
}

export const Context = createContext(new State(new models.main.AppConfig()));
export const Dispatch = createContext<ActionDispatch<[Action]>>(() => {});
export type Action = { type: "set_config"; payload: models.main.AppConfig };

export function Reducer(state: State, action: Action) {
    switch (action.type) {
        case "set_config": {
            let update = structuredClone(state);
            update.config = action.payload;
            return update;
        }

        default: {
            throw Error("Unknown action: " + action.type);
        }
    }
}
