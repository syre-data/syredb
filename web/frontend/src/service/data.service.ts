import * as types from "@/types";
import type { UUIDTypes } from "uuid";

function getDataSchemasAll(): Promise<types.DataSchema[]> {
    return fetch("/api/data-schemas", {
        credentials: "same-origin",
    }).then(async (resp) => (await resp.json()) as types.DataSchema[]);
}

function saveProjectDataAll(
    project: UUIDTypes,
    hierarchy: types.SaveDataHierarchy[],
): Promise<Response> {
    throw new Error("not yet implemented");
}

function saveSampleDataSingle(sample_data: UUIDTypes): Promise<Response> {
    throw new Error("not yet implemented");
}

function saveSampleDataMultiple(
    sample_data: UUIDTypes[],
    project: UUIDTypes,
    hierarchy: types.SaveDataHierarchy[],
) {
    throw new Error("not yet implemented");
}

function saveDataSchemaSampleDataAll(
    data_schema: UUIDTypes,
    project: UUIDTypes,
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

function parseDataFileToSchema(
    file: File,
    schema: types.DataSchema,
): Promise<any> {
    switch (file.type) {
        case "text/csv":
            return parseDataFileToSchemaCsv(file, schema);
        default:
            throw new Error("invalid file type");
    }
}

async function parseDataFileToSchemaCsv(file: File, schema: types.DataSchema) {
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
    schema: types.DataSchema,
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
    dtype: types.DataType,
): any {
    switch (dtype) {
        case types.DataTypeString:
            return value;
        case types.DataTypeInt:
            return parseInt(value);
        case types.DataTypeUint:
            const parsed = parseInt(value);
            if (parsed < 0) {
                throw new Error(
                    `INVALID_DATA expected usigned integer got ${parsed}`,
                );
            }
            return parsed;
        case types.DataTypeFloat:
            return parseFloat(value);
        case types.DataTypeBoolean:
            return value === "true";
        case types.DataTypeTimestamp:
            return new Date(value);
    }
}

export default {
    getDataSchemasAll,
    saveProjectDataAll,
    saveSampleDataSingle,
    saveSampleDataMultiple,
    saveDataSchemaSampleDataAll,
    dataSchemaCreate,
    parseDataFileToSchema,
};
