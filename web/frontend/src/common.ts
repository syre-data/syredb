import * as types from "@/types";

export const QUERY_KEY_DATA_SCHEMA = "data_schema";
export const QUERY_KEY_TRANSFORM_SCHEMA = "transform_schema";

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
): types.ProjectUserPermission | undefined {
    switch (value) {
        case "owner":
            return types.ProjectUserPermissionOwner;
        case "admin":
            return types.ProjectUserPermissionAdmin;
        case "read_write":
            return types.ProjectUserPermissionReadWrite;
        case "read":
            return types.ProjectUserPermissionRead;
        default:
            return undefined;
    }
}

export function has_db_permission(
    needle: types.DbUserPermission,
    haystack: types.DbUserPermission[],
): boolean {
    return (
        haystack.includes(types.DbUserPermissionOwner) ||
        haystack.includes(needle)
    );
}

export function user_db_permission_string_to_variant(
    value: string,
): types.DbUserPermission | undefined {
    switch (value) {
        case "owner":
            return types.DbUserPermissionOwner;
        case "add_user":
            return types.DbUserPermissionAddUser;
        case "modify_user":
            return types.DbUserPermissionModifyUser;
        case "create_data_schema":
            return types.DbUserPermissionCreateDataSchema;
        case "modify_data_schema":
            return types.DbUserPermissionModifyDataSchema;
        case "create_transform":
            return types.DbUserPermissionCreateTransform;
        case "create_project":
            return types.DbUserPermissionCreateProject;
        default:
            return undefined;
    }
}

export function data_type_string_to_variant(
    value: string,
): types.DataType | undefined {
    switch (value) {
        case "string":
            return types.DataTypeString;
        case "int":
            return types.DataTypeInt;
        case "uint":
            return types.DataTypeUint;
        case "float":
            return types.DataTypeFloat;
        case "boolean":
            return types.DataTypeBoolean;
        case "timestamp":
            return types.DataTypeTimestamp;
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
