import * as model from "./types";

export const QUERY_KEY_DATA_SCHEMA = "data_schema";

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
): model.ProjectUserPermission | undefined {
    switch (value) {
        case "owner":
            return model.ProjectUserPermissionOwner;
        case "admin":
            return model.ProjectUserPermissionAdmin;
        case "read_write":
            return model.ProjectUserPermissionReadWrite;
        case "read":
            return model.ProjectUserPermissionRead;
        default:
            return undefined;
    }
}

export function is_admin_or_owner(
    user_permission: model.ProjectUserPermission,
): boolean {
    return (
        user_permission === model.ProjectUserPermissionAdmin ||
        user_permission === model.ProjectUserPermissionOwner
    );
}

export function user_role_string_to_variant(
    value: string,
): model.UserRole | undefined {
    switch (value) {
        case "owner":
            return model.UserRoleOwner;
        case "admin":
            return model.UserRoleAdmin;
        case "user":
            return model.UserRoleUser;
        default:
            return undefined;
    }
}

export function data_type_string_to_variant(
    value: string,
): model.DataType | undefined {
    switch (value) {
        case "string":
            return model.DataTypeString;
        case "int":
            return model.DataTypeInt;
        case "uint":
            return model.DataTypeUint;
        case "float":
            return model.DataTypeFloat;
        case "boolean":
            return model.DataTypeBoolean;
        case "timestamp":
            return model.DataTypeTimestamp;
        default:
            return undefined;
    }
}

export function visibility_string_to_variant(
    value: string,
): model.Visibility | undefined {
    switch (value) {
        case "public":
            return model.VisibilityPublic;
        case "private":
            return model.VisibilityPrivate;
        default:
            return undefined;
    }
}
