import * as types from "@/types";
import * as payload from "@/types/handler";
import type { UUIDTypes } from "uuid";
import * as uuid from "uuid";

function getUserProjects(): Promise<types.Project[]> {
    return fetch("/api/projects", {
        credentials: "same-origin",
    }).then(async (resp) => {
        return (await resp.json()) as types.Project[];
    });
}

function createProject(project: types.ProjectCreate): Promise<UUIDTypes> {
    return fetch("/api/project", {
        credentials: "same-origin",
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(project),
    }).then(async (resp) => {
        return (await resp.json()) as UUIDTypes;
    });
}

function getProjectResources(
    project: UUIDTypes,
): Promise<types.ProjectResources> {
    const params = new URLSearchParams();
    params.set("project", project.toString());
    return fetch(`/api/project/resources?${params}`, {
        credentials: "same-origin",
    }).then(async (resp) => {
        return (await resp.json()) as types.ProjectResources;
    });
}

function getProjectWithUserPermission(
    project: UUIDTypes,
): Promise<types.ProjectWithUserPermission> {
    const params = new URLSearchParams();
    params.set("id", project.toString());
    return fetch(`/api/project?${params}`, {
        credentials: "same-origin",
    }).then(
        async (resp) => (await resp.json()) as types.ProjectWithUserPermission,
    );
}

function saveSampleDataMultiple(
    sample_data: UUIDTypes,
    project: UUIDTypes,
    hierarchy: types.SaveDataHierarchy[],
): Promise<Response> {
    throw new Error("not yet impleneted");
}

function dataMembershipCreate(
    project: UUIDTypes,
    data: UUIDTypes,
    label?: string,
): Promise<Response> {
    return fetch("/api/project/data", {
        credentials: "same-origin",
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ project, data, label }),
    });
}

function hasPermission(
    needle: types.ProjectPermission,
    haystack: types.ProjectPermission[],
): boolean {
    return (
        haystack.includes(types.ProjectPermissionOwner) ||
        haystack.includes(needle)
    );
}

export default {
    getUserProjects,
    createProject,
    getProjectResources,
    getProjectWithUserPermission,
    saveSampleDataMultiple,
    hasPermission,
    dataMembershipCreate,
};
