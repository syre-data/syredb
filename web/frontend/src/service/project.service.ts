import * as model from "../../model";

export async function getUserProjects(): Promise<model.Project[]> {
    return fetch("/api/projects", {
        credentials: "same-origin",
    }).then(async (resp) => {
        return (await resp.json()) as model.Project[];
    });
}
