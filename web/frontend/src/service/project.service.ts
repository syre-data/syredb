import { AppError } from "@/errors";
import * as types from "@/types";
import type { UUIDTypes } from "uuid";

function getUserProjects(): Promise<types.Project[]> {
    return fetch("/api/projects", {
        credentials: "same-origin",
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return (await res.json()) as types.Project[];
    });
}

function createProject(project: types.ProjectCreate): Promise<UUIDTypes> {
    return fetch("/api/project", {
        credentials: "same-origin",
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(project),
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return (await res.json()) as UUIDTypes;
    });
}

function getProjectResources(
    project: UUIDTypes,
): Promise<types.ProjectResources> {
    const params = new URLSearchParams();
    params.set("project", project.toString());
    return fetch(`/api/project/resources?${params}`, {
        credentials: "same-origin",
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return (await res.json()) as types.ProjectResources;
    });
}

function getProjectWithUserPermission(
    project: UUIDTypes,
): Promise<types.ProjectWithUserPermission> {
    const params = new URLSearchParams();
    params.set("id", project.toString());
    return fetch(`/api/project?${params}`, {
        credentials: "same-origin",
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return (await res.json()) as types.ProjectWithUserPermission;
    });
}

function dataMembershipCreate(
    project: UUIDTypes,
    data: UUIDTypes,
    label?: string,
): Promise<null> {
    return fetch("/api/project/data", {
        credentials: "same-origin",
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ project, data, label }),
    }).then(async (res) => {
        if (!res.ok) {
            const data = (await res.json()) as types.AppError;
            throw new AppError(data);
        }

        return null;
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
    hasPermission,
    dataMembershipCreate,
};
