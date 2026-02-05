import * as model from "../model";

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
            return model.ProjectUserPermission.PROJECT_USER_PERMISSION_OWNER;
        case "admin":
            return model.ProjectUserPermission.PROJECT_USER_PERMISSION_ADMIN;
        case "read_write":
            return model.ProjectUserPermission
                .PROJECT_USER_PERMISSION_READ_WRITE;
        case "read":
            return model.ProjectUserPermission.PROJECT_USER_PERMISSION_READ;
        default:
            return undefined;
    }
}

export function is_admin_or_owner(
    user_permission: model.ProjectUserPermission,
): boolean {
    return (
        user_permission ===
            model.ProjectUserPermission.PROJECT_USER_PERMISSION_ADMIN ||
        user_permission ===
            model.ProjectUserPermission.PROJECT_USER_PERMISSION_OWNER
    );
}

export function user_role_string_to_variant(
    value: string,
): model.UserRole | undefined {
    switch (value) {
        case "owner":
            return model.UserRole.USER_ROLE_OWNER;
        case "admin":
            return model.UserRole.USER_ROLE_ADMIN;
        case "user":
            return model.UserRole.USER_ROLE_USER;
        default:
            return undefined;
    }
}

export function data_type_string_to_variant(
    value: string,
): model.DataType | undefined {
    switch (value) {
        case "string":
            return model.DataType.DATA_TYPE_STRING;
        case "int":
            return model.DataType.DATA_TYPE_INT;
        case "uint":
            return model.DataType.DATA_TYPE_UINT;
        case "float":
            return model.DataType.DATA_TYPE_FLOAT;
        case "boolean":
            return model.DataType.DATA_TYPE_BOOLEAN;
        case "timestamp":
            return model.DataType.DATA_TYPE_TIMESTAMP;
        default:
            return undefined;
    }
}

export function visibility_string_to_variant(
    value: string,
): model.Visibility | undefined {
    switch (value) {
        case "public":
            return model.Visibility.VISIBILITY_PUBLIC;
        case "private":
            return model.Visibility.VISIBILITY_PRIVATE;
        default:
            return undefined;
    }
}
