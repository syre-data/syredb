import { uuidToString } from "@/common";
import * as types from "@/types";
import * as uuid from "uuid";

function dataTypesGetAll(): Promise<types.DataType[]> {
    return fetch("/api/data-types", {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataType[]);
}

function dataTypeCreateInternal(
    label: string,
    description: string,
    schema: uuid.UUIDTypes,
): Promise<Response> {
    const params = new URLSearchParams();
    params.set("storage", types.DataStorageInternal);
    const data = {
        label,
        description,
        schema: uuidToString(schema),
    };

    return fetch(`/api/data-type?${params}`, {
        credentials: "same-origin",
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
}

function dataTypeCreateExternal(
    label: string,
    description: string,
    sources: types.ExternalSourceCreate[],
): Promise<Response> {
    const params = new URLSearchParams();
    params.set("storage", types.DataStorageExternal);
    const data = {
        label,
        description,
        sources,
    };

    return fetch(`/api/data-type?${params}`, {
        credentials: "same-origin",
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
}

function dataTypeGet(
    data_type: uuid.UUIDTypes,
): Promise<types.DataTypeInternal | types.DataTypeExternal> {
    const params = new URLSearchParams();
    params.append("id", data_type.toString());
    return fetch(`/api/data-type?${params}`, {
        credentials: "same-origin",
    }).then(async (resp) => await resp.json());
}

function dataTypeUpdate(update: types.DataTypeUpdate): Promise<Response> {
    return fetch(`/api/data-type`, {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(update),
    });
}

function dataSchemasGetAll(): Promise<types.DataSchema[]> {
    return fetch("/api/data-schemas", {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataSchema[]);
}

function downloadProjectDataAll(
    project: uuid.UUIDTypes,
    hierarchy: types.SaveDataHierarchy,
): Promise<Response> {
    const params = new URLSearchParams();
    params.append("id", project.toString());
    params.append("hierarchy", hierarchy);
    return fetch(`/api/sample-data/project?${params}`, {
        credentials: "same-origin",
    });
}

function downloadDataSingle(data: uuid.UUIDTypes): Promise<Response> {
    const params = new URLSearchParams();
    params.append("id", uuidToString(data));
    return fetch(`/resource/data?${params}`, {
        credentials: "same-origin",
    });
}

function downloadDataMultiple(
    data: uuid.UUIDTypes[],
    project: uuid.UUIDTypes,
    hierarchy: types.SaveDataHierarchy[],
) {
    throw new Error("not yet implemented");
}

function downloadDataSource(
    data: uuid.UUIDTypes,
    source: string,
): Promise<Response> {
    const params = new URLSearchParams();
    params.append("data", uuidToString(data));
    params.append("source", source);
    return fetch(`/resource/data/source?${params}`, {
        credentials: "same-origin",
    });
}

function downloadDataSourceIndex(
    data: uuid.UUIDTypes,
    source: string,
    index: number,
): Promise<Response> {
    const params = new URLSearchParams();
    params.append("data", uuidToString(data));
    params.append("source", source);
    params.append("index", index.toString());
    return fetch(`/resource/data/source?${params}`, {
        credentials: "same-origin",
    });
}

function dataSchemaCreate(schema: types.DataSchemaCreate): Promise<Response> {
    return fetch("/api/data-schema", {
        credentials: "same-origin",
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(schema),
    });
}

function dataSchemaUpdate(update: types.DataSchemaUpdate): Promise<Response> {
    return fetch("/api/data-schema", {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(update),
    });
}

function parseDataFileToSchema(
    file: File,
    schema: types.DataSchemaField[],
): Promise<any> {
    switch (file.type) {
        case "text/csv":
            return parseDataFileToSchemaCsv(file, schema);
        default:
            throw new Error("invalid file type");
    }
}

async function parseDataFileToSchemaCsv(
    file: File,
    schema: types.DataSchemaField[],
) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();

        reader.onload = (e) => {
            let content;
            if (reader.result === null) {
                reject("NO_CONTENT");
                return;
            }

            if (reader.result instanceof ArrayBuffer) {
                content = reader.result.toString();
            } else {
                content = reader.result;
            }
            try {
                const parsed = parse_data_file_to_schema_csv(content, schema);
                resolve(parsed);
            } catch (err) {
                reject(err);
            }
        };
        reader.onerror = (e) => {
            console.error(e);
            throw new Error(`${e}`);
        };
        reader.readAsText(file);
    });
}

function parse_data_file_to_schema_csv(
    content: string,
    schema: types.DataSchemaField[],
) {
    if (content.length === 0) {
        throw new Error("NO_DATA");
    }

    const cols: string[][] = [];
    for (let idx = 0; idx < schema.length; idx++) {
        cols.push([]);
    }

    let lines = content.split("\n");
    for (let lidx = 0; lidx < lines.length; lidx++) {
        const line = lines[lidx]!;
        let values = line.split(",");
        if (values.length != schema.length) {
            throw new Error(
                `INVALID_DATA: line ${lidx}, expected ${schema.length} values, found ${values.length}`,
            );
        }

        values = values.map((value) => value.trim());
        for (let vidx = 0; vidx < values.length; vidx++) {
            cols[vidx]!.push(values[vidx]!);
        }
    }

    const parsed: any[][] = [];
    for (let idx = 0; idx < schema.length; idx++) {
        const values = cols[idx]!.map((value) =>
            parse_data_file_parse_string_to_value(value, schema[idx]!.DType),
        );

        parsed.push(values);
    }

    return parsed;
}

function parse_data_file_parse_string_to_value(
    value: string,
    dtype: types.ValueType,
): any {
    switch (dtype) {
        case types.ValueTypeString:
            return value;
        case types.ValueTypeInt:
            return parseInt(value);
        case types.ValueTypeUint:
            const parsed = parseInt(value);
            if (parsed < 0) {
                throw new Error(
                    `INVALID_DATA expected usigned integer got ${parsed}`,
                );
            }
            return parsed;
        case types.ValueTypeFloat:
            return parseFloat(value);
        case types.ValueTypeBoolean:
            return value === "true";
        case types.ValueTypeTimestamp:
            return new Date(value);
    }
}

function dataSchemaResourcesGet(
    data_schema_id: uuid.UUIDTypes,
): Promise<types.DataSchemaResources> {
    const params = new URLSearchParams();
    params.append("id", uuidToString(data_schema_id));
    return fetch(`/api/data-schema?${params}`, {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataSchemaResources);
}

function dataTypeTransformsGetAll(): Promise<types.DataTypeTransform[]> {
    return fetch("/api/data-type-transforms", {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataTypeTransform[]);
}

function dataTypeTransformCreate(data: FormData): Promise<Response> {
    return fetch("/api/data-type-transform", {
        credentials: "same-origin",
        method: "post",
        body: data,
    });
}

// # Notes
// + `Sources` keys should be UUIDs
export interface DataIngest {
    Type: uuid.UUIDTypes;
    Timestamp: Date;
    Visibility: types.Visibility;
    Properties: types.Property[];
    Notes: types.DataNoteCreate[];
    Values?: Record<string, any>;
    Sources?: Record<string, any>;
}

function projectDataCreate(
    project: uuid.UUIDTypes,
    data: DataIngest[],
    labels: string[],
    sourceFiles: [string, File | FileList][],
) {
    const params = new URLSearchParams();
    params.set("project", uuidToString(project));

    const body = new FormData();
    body.set("data", JSON.stringify(data));
    body.set("project_labels", JSON.stringify(labels));
    for (const [key, file] of sourceFiles) {
        if (file instanceof FileList) {
            for (const f of file) {
                body.append(key, f);
            }
        } else {
            body.set(key, file);
        }
    }

    return fetch(`/api/data?${params}`, {
        credentials: "same-origin",
        method: "post",
        body,
    });
}

function orphanedData(): Promise<types.OrphanedDataResources> {
    return fetch("/api/data/orphaned", {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.OrphanedDataResources);
}

// TODO: Should be a `handler.DataResources` but it isn't being created by `tygo`.
function dataGet(data_id: uuid.UUIDTypes): Promise<any> {
    const params = new URLSearchParams();
    params.set("id", uuidToString(data_id));
    return fetch(`/api/data?${params}`, {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as any);
}

function dataUpdate(update: types.DataUpdate): Promise<Response> {
    return fetch("/api/data", {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(update),
    });
}

function dataOrigins(): Promise<types.DataOriginRx[]> {
    return fetch("/api/data-origins", {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataOriginRx[]);
}

function dataOriginById(
    data_origin_id: uuid.UUIDTypes,
): Promise<types.DataOriginRx> {
    const params = new URLSearchParams();
    params.set("id", uuidToString(data_origin_id));
    return fetch(`/api/data-origin?${params}`, {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataOriginRx);
}

function dataOriginCreate(origin: types.DataOriginCreate): Promise<Response> {
    return fetch("/api/data-origin", {
        credentials: "same-origin",
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(origin),
    });
}

function dataOriginUpdate(update: types.DataOriginRx): Promise<Response> {
    return fetch("/api/data-origin", {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(update),
    });
}

function dataValues(data: uuid.UUIDTypes): Promise<any> {
    const params = new URLSearchParams();
    params.set("id", uuidToString(data));
    return fetch(`/api/data/values?${params}`, {
        credentials: "same-origin",
    }).then(async (resp) => await resp.json());
}

function dataPreview(data: uuid.UUIDTypes): Promise<types.DataValuesPreview> {
    const params = new URLSearchParams();
    params.set("id", uuidToString(data));
    return fetch(`/api/data/values/preview?${params}`, {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataValuesPreview);
}

function propertiesUpdate(update: types.DataPropertiesUpdate) {
    return fetch(`/api/data/properties`, {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(update),
    });
}

function notesCreate(data: uuid.UUIDTypes, notes: types.DataNoteCreate[]) {
    return fetch(`/api/data/notes`, {
        credentials: "same-origin",
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ Id: uuidToString(data), Notes: notes }),
    });
}

export default {
    dataTypesGetAll,
    dataTypeCreateInternal,
    dataTypeCreateExternal,
    dataTypeGet,
    dataTypeUpdate,
    dataSchemaCreate,
    dataSchemaUpdate,
    dataSchemasGetAll,
    dataSchemaResourcesGet,
    downloadProjectDataAll,
    downloadDataSingle,
    downloadDataMultiple,
    downloadDataSource,
    downloadDataSourceIndex,
    parseDataFileToSchema,
    dataTypeTransformCreate,
    dataTypeTransformsGetAll,
    projectDataCreate,
    orphanedData,
    dataGet,
    dataUpdate,
    dataOrigins,
    dataOriginById,
    dataOriginCreate,
    dataOriginUpdate,
    dataValues,
    dataPreview,
    propertiesUpdate,
    notesCreate,
};
