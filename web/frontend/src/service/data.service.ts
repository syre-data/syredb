import * as types from "@/types";
import * as uuid from "uuid";

function dataTypesGetAll(): Promise<types.DataType[]> {
    return fetch("/api/data-types", {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataType[]);
}

function dataTypeCreate(
    label: string,
    sources: types.DataTypeSourceCreate[],
    description?: string,
    data_schema?: Uint8Array<ArrayBufferLike>,
    recipe?: File,
): Promise<Response> {
    const data = new FormData();
    data.set("label", label);
    data.set("sources", JSON.stringify(sources));
    if (description) {
        data.set("description", description);
    }
    if (data_schema) {
        data.set("data_schema", uuid.stringify(data_schema));
    }
    if (recipe) {
        data.set("recipe", recipe);
    }

    return fetch("/api/data-type", {
        credentials: "same-origin",
        method: "post",
        body: data,
    });
}

function dataTypeGet(data_type: uuid.UUIDTypes): Promise<types.DataType> {
    const params = new URLSearchParams();
    params.append("id", data_type.toString());
    return fetch(`/api/data-type?${params}`, {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataType);
}

function dataTypeUpdate(update: types.DataTypeUpdate): Promise<Response> {
    return fetch(`/api/data-type`, {
        credentials: "same-origin",
        method: "put",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(update),
    });
}

function dataSchemasGetAll(): Promise<types.DataSchemaRecord[]> {
    return fetch("/api/data-schemas", {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataSchemaRecord[]);
}

function saveProjectDataAll(
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

function saveSampleDataSingle(sample_data: uuid.UUIDTypes): Promise<Response> {
    const params = new URLSearchParams();
    params.append("id", sample_data.toString());
    return fetch(`/api/sample-data/single?${params}`, {
        credentials: "same-origin",
    });
}

function saveSampleDataMultiple(
    sample_data: uuid.UUIDTypes[],
    project: uuid.UUIDTypes,
    hierarchy: types.SaveDataHierarchy[],
) {
    throw new Error("not yet implemented");
}

function saveDataSchemaSampleDataAll(
    data_schema: uuid.UUIDTypes,
    project: uuid.UUIDTypes,
    hierarchy: types.SaveDataHierarchy[],
): Promise<Response> {
    throw new Error("not yet implemented");
}

function dataSchemaCreate(
    data_schema: types.DataSchemaCreate,
): Promise<Response> {
    return fetch("/api/data-schema", {
        credentials: "same-origin",
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data_schema),
    });
}

function transformCreate(transform: types.TransformCreate): Promise<Response> {
    const data = new FormData();
    data.append("input", transform.SourceSchema.toString());
    data.append("output", transform.DestinationSchema.toString());
    data.append("label", transform.Label);
    data.append("description", transform.Description);
    data.append("script", transform.Script);
    return fetch("/api/transform", {
        credentials: "same-origin",
        method: "post",
        body: data,
    });
}

function parseDataFileToSchema(
    file: File,
    schema: types.DataSchemaRecord,
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
    schema: types.DataSchemaRecord,
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
    schema: types.DataSchemaRecord,
) {
    if (content.length === 0) {
        throw new Error("NO_DATA");
    }

    const cols: string[][] = [];
    for (let idx = 0; idx < schema.Schema.length; idx++) {
        cols.push([]);
    }

    let lines = content.split("\n");
    for (let lidx = 0; lidx < lines.length; lidx++) {
        const line = lines[lidx]!;
        let values = line.split(",");
        if (values.length != schema.Schema.length) {
            throw new Error(
                `INVALID_DATA: line ${lidx}, expected ${schema.Schema.length} values, found ${values.length}`,
            );
        }

        values = values.map((value) => value.trim());
        for (let vidx = 0; vidx < values.length; vidx++) {
            cols[vidx]!.push(values[vidx]!);
        }
    }

    const parsed: any[][] = [];
    for (let idx = 0; idx < schema.Schema.length; idx++) {
        const values = cols[idx]!.map((value) =>
            parse_data_file_parse_string_to_value(
                value,
                schema.Schema[idx]!.dtype,
            ),
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
    params.append("id", data_schema_id.toString());
    return fetch(`/api/data-schema?${params}`, {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataSchemaResources);
}

function transformSchemasGet(): Promise<types.TransformResources> {
    const params = new URLSearchParams();
    params.append("type", "transform");
    return fetch(`/api/data-schemas?${params}`, {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.TransformResources);
}

export default {
    dataTypesGetAll,
    dataTypeCreate,
    dataTypeGet,
    dataTypeUpdate,
    dataSchemasGetAll,
    saveProjectDataAll,
    saveSampleDataSingle,
    saveSampleDataMultiple,
    saveDataSchemaSampleDataAll,
    dataSchemaCreate,
    parseDataFileToSchema,
    dataSchemaResourcesGet,
    transformSchemasGet,
    transformCreate,
};
