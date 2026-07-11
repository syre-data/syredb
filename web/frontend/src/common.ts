import * as types from "@/types";
import { stringify, type UUIDTypes } from "uuid";

export const QUERY_KEY_DB_PERMISSIONS = "db_permissions";
export const QUERY_KEY_USER = "user";
export const QUERY_KEY_USER_LIST = "get_users_list";
export const QUERY_KEY_DATA_SCHEMAS = "data_schemas";
export const QUERY_KEY_DATA_SCHEMA = "data_schema";
export const QUERY_KEY_DATA_SCHEMA_RESOURCES = "data_schema_resources";
export const QUERY_KEY_DATA_TYPES = "data_types";
export const QUERY_KEY_DATA_TYPE = "data_type";
export const QUERY_KEY_DATA_TYPE_TRANSFORMS = "data_type_transforms";
export const QUERY_KEY_USER_PROJECTS = "user_projects";
export const QUERY_KEY_PROJECT_RESOURCES = "project_resources";
export const QUERY_KEY_PROJECT_SAMPLE_RESOURCES = "project_sample_resources";
export const QUERY_KEY_PROJECT = "project";
export const QUERY_KEY_ORPHANED_DATA = "orphaned_data";
export const QUERY_KEY_DATA = "data";
export const QUERY_KEY_DATA_ORIGINS = "data_origins";
export const QUERY_KEY_DATA_ORIGIN = "data_origin";
export const QUERY_KEY_DATA_VALUES = "data_values";
export const QUERY_KEY_DATA_PREVIEW = "data_preview";

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
        case "transform_create":
            return types.DbPermissionTransformCreate;
        case "transform_modify":
            return types.DbPermissionTransformModify;
        case "project_create":
            return types.DbPermissionProjectCreate;
        case "data_create":
            return types.DbPermissionDataCreate;
        case "data_read_all":
            return types.DbPermissionDataReadAll;
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

export function uuidToString(id: UUIDTypes): string {
    if (typeof id === "string") {
        return id;
    } else {
        return stringify(id);
    }
}

export function timestampToString(date: Date): string {
    const y = String(date.getUTCFullYear()).padStart(4, "0");
    const mo = String(date.getMonth()).padStart(2, "0");
    const d = String(date.getDate()).padStart(2, "0");
    const h = String(date.getHours()).padStart(2, "0");
    const m = String(date.getMinutes()).padStart(2, "0");
    const s = String(date.getSeconds()).padStart(2, "0");
    return `${y}-${mo}-${d} ${h}:${m}:${s}`;
}
