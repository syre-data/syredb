import * as types from "@/types";

export const QUERY_KEY_DB_PERMISSIONS = "db_permissions";
export const QUERY_KEY_USER = "user";
export const QUERY_KEY_USER_LIST = "get_users_list";
export const QUERY_KEY_DATA_SCHEMAS = "data_schemas";
export const QUERY_KEY_DATA_SCHEMA = "data_schema";
export const QUERY_KEY_DATA_SCHEMA_RESOURCES = "data_schema_resources";
export const QUERY_KEY_DATA_TYPES = "data_types";
export const QUERY_KEY_DATA_TYPE = "data_type";
export const QUERY_KEY_INGESTION_SCRIPTS = "ingestion_scripts";
export const QUERY_KEY_INGESTION_SCRIPTS_FOR_DATA_TYPE =
    "ingestion_scripts_for_data_type";
export const QUERY_KEY_DATA_TYPE_TRANSFORMS = "data_type_transforms";
export const QUERY_KEY_USER_PROJECTS = "user_projects";
export const QUERY_KEY_PROJECT_RESOURCES = "project_resources";
export const QUERY_KEY_PROJECT_SAMPLE_RESOURCES = "project_sample_resources";
export const QUERY_KEY_PROJECT = "project";
export const QUERY_KEY_ORPHANED_DATA = "orphaned_data";

export interface BackendError {
    message: string;
    cause: object;
    kind: string;
}

export enum MouseButton {
    Primary = 0,
    Secondary = 2,
}

export function hasDbPermission(
    needle: types.DbPermission,
    haystack: types.DbPermission[],
): boolean {
    return (
        haystack.includes(types.DbPermissionOwner) || haystack.includes(needle)
    );
}

export function db_permission_id_string_to_variant(
    value: string,
): types.DbPermission | undefined {
    switch (value) {
        case "owner":
            return types.DbPermissionOwner;
        case "user_create":
            return types.DbPermissionUserCreate;
        case "user_modify":
            return types.DbPermissionUserModify;
        case "data_schema_create":
            return types.DbPermissionDataSchemaCreate;
        case "data_schema_modify":
            return types.DbPermissionDataSchemaModify;
        case "data_type_create":
            return types.DbPermissionDataTypeCreate;
        case "data_type_modify":
            return types.DbPermissionDataTypeModify;
        case "ingestion_script_create":
            return types.DbPermissionIngestionScriptCreate;
        case "ingestion_script_modify":
            return types.DbPermissionIngestionScriptModify;
        case "transform_create":
            return types.DbPermissionTransformCreate;
        case "transform_modify":
            return types.DbPermissionTransformModify;
        case "project_create":
            return types.DbPermissionProjectCreate;
        case "data_create":
            return types.DbPermissionDataCreate;
        default:
            return undefined;
    }
}

export function data_schema_cardinality_string_to_variant(
    value: string,
): types.DataSchemaCardinality | undefined {
    switch (value) {
        case "single":
            return types.DataSchemaCardinalitySingle;
        case "multiple":
            return types.DataSchemaCardinalityMultiple;
        default:
            return undefined;
    }
}

export function data_type_string_to_variant(
    value: string,
): types.ValueType | undefined {
    switch (value) {
        case "string":
            return types.ValueTypeString;
        case "int":
            return types.ValueTypeInt;
        case "uint":
            return types.ValueTypeUint;
        case "float":
            return types.ValueTypeFloat;
        case "boolean":
            return types.ValueTypeBoolean;
        case "timestamp":
            return types.ValueTypeTimestamp;
        default:
            return undefined;
    }
}

export function visibility_string_to_variant(
    value: string,
): types.Visibility | undefined {
    switch (value) {
        case "public":
            return types.VisibilityPublic;
        case "private":
            return types.VisibilityPrivate;
        default:
            return undefined;
    }
}

export function data_storage_string_to_variant(
    value: string,
): types.DataStorage | undefined {
    switch (value) {
        case "internal":
            return types.DataStorageInternal;
        case "file":
            return types.DataStorageExternal;
        default:
            return undefined;
    }
}
