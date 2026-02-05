import * as model from "@/../model";

export async function GetDataSchemasAll(): Promise<model.DataSchema[]> {
    return fetch("/api/data-schemas", {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as model.DataSchema[]);
}
