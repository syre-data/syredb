import * as types from "@/types";

export const QUERY_KEY_DB_PERMISSIONS = "db_permissions";
export const QUERY_KEY_USER = "user";
export const QUERY_KEY_USER_LIST = "get_users_list";
export const QUERY_KEY_DATA_SCHEMAS = "data_schemas";
export const QUERY_KEY_DATA_SCHEMA = "data_schema";
export const QUERY_KEY_DATA_SCHEMA_RESOURCES = "data_schema_resources";
export const QUERY_KEY_DATA_TYPES = "data_types";
export const QUERY_KEY_USER_PROJECTS = "user_projects";
export const QUERY_KEY_PROJECT_RESOURCES = "project_resources";
export const QUERY_KEY_PROJECT_SAMPLE_RESOURCES = "project_sample_resources";
export const QUERY_KEY_PROJECT = "project";

export interface BackendError {
    message: string;
    cause: object;
    kind: string;
}

export enum MouseButton {
    Primary = 0,
    Secondary = 2,
}

export function project_user_permission_string_to_variant(
    value: string,
): types.ProjectPermission | undefined {
    switch (value) {
        case "owner":
            return types.ProjectPermissionOwner;
        case "create_sample":
            return types.ProjectPermissionCreateSample;
        case "read":
            return types.ProjectPermissionRead;
        default:
            return undefined;
    }
}

export function has_db_permission(
    needle: types.DbPermissionId,
    haystack: types.DbPermissionId[],
): boolean {
    return (
        haystack.includes(types.DbPermissionIdOwner) ||
        haystack.includes(needle)
    );
}

export function db_permission_id_string_to_variant(
    value: string,
): types.DbPermissionId | undefined {
    switch (value) {
        case "owner":
            return types.DbPermissionIdOwner;
        case "user_create":
            return types.DbPermissionIdUserCreate;
        case "user_modify":
            return types.DbPermissionIdUserModify;
        case "data_schema_create":
            return types.DbPermissionIdDataSchemaCreate;
        case "data_schema_modify":
            return types.DbPermissionIdDataSchemaModify;
        case "data_type_create":
            return types.DbPermissionIdDataTypeCreate;
        case "data_type_modify":
            return types.DbPermissionIdDataTypeModify;
        case "transform_create":
            return types.DbPermissionIdTransformCreate;
        case "transform_modify":
            return types.DbPermissionIdTransformModify;
        case "project_create":
            return types.DbPermissionIdProjectCreate;
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
