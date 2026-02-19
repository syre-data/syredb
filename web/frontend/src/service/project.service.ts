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

function getProjectSampleResources(
    project: UUIDTypes,
    sample: UUIDTypes,
): Promise<types.ProjectSampleResources> {
    const params = new URLSearchParams();
    params.set("project", project.toString());
    params.set("sample", sample.toString());
    return fetch(`/api/project/sample-resources?${params}`, {
        credentials: "same-origin",
    }).then(async (resp) => {
        return (await resp.json()) as types.ProjectSampleResources;
    });
}

function saveSampleDataMultiple(
    sample_data: UUIDTypes,
    project: UUIDTypes,
    hierarchy: types.SaveDataHierarchy[],
): Promise<Response> {
    throw new Error("not yet impleneted");
}

function createProjectSamples(
    project: UUIDTypes,
    samples: types.ProjectSampleCreate[],
): Promise<Response> {
    const sample_data_map = new Map<uuid.UUIDTypes, File>();
    const samples_payload = [];
    for (const sample of samples) {
        const sample_payload = {
            Label: sample.Label,
            Tags: sample.Tags,
            Properties: sample.Properties,
            Data: [] as payload.ProjectSampleDataCreate[],
            Notes: sample.Notes,
        } satisfies payload.ProjectSampleCreate;
        for (const data of sample.Data) {
            const id = uuid.v4();
            const data_payload = {
                Schema: data.Schema,
                File: id,
                Timestamp: data.Timestamp,
                Properties: data.Properties,
            } satisfies payload.ProjectSampleDataCreate;

            sample_data_map.set(id, data.File.File!);
            sample_payload.Data.push(data_payload);
        }
        samples_payload.push(sample_payload);
    }

    const formdata = new FormData();
    formdata.set("project", project.toString());
    formdata.set("samples", JSON.stringify(samples_payload));
    for (const [id, file] of sample_data_map.entries()) {
        formdata.set(`datafiles[${id}]`, file);
    }

    return fetch("/api/project/samples", {
        credentials: "same-origin",
        method: "post",
        body: formdata,
    });
}

function updateProjectSample(
    project: UUIDTypes,
    update: types.ProjectSampleUpdate,
): Promise<Response> {
    return fetch("/api/project/sample", {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ project, update }),
    });
}

export default {
    getUserProjects,
    createProject,
    getProjectResources,
    getProjectWithUserPermission,
    getProjectSampleResources,
    createProjectSamples,
    saveSampleDataMultiple,
    updateProjectSample,
};
