import * as app from "../bindings/syredb/app";

export const USER_NOT_AUTHENTICATED_ERROR = "USER_NOT_AUTHENTICATED";
export const INSUFFICIENT_PERMISSIONS_ERROR = "INSUFFICEINT_PERMISSIONS;";
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
): app.ProjectUserPermission | undefined {
    switch (value) {
        case "owner":
            return app.ProjectUserPermission.PROJECT_USER_PERMISSION_OWNER;
        case "admin":
            return app.ProjectUserPermission.PROJECT_USER_PERMISSION_ADMIN;
        case "read_write":
            return app.ProjectUserPermission.PROJECT_USER_PERMISSION_READ_WRITE;
        case "read":
            return app.ProjectUserPermission.PROJECT_USER_PERMISSION_READ;
        default:
            return undefined;
    }
}

export function is_admin_or_owner(
    user_permission: app.ProjectUserPermission,
): boolean {
    return (
        user_permission ===
            app.ProjectUserPermission.PROJECT_USER_PERMISSION_ADMIN ||
        user_permission ===
            app.ProjectUserPermission.PROJECT_USER_PERMISSION_OWNER
    );
}

export function user_role_string_to_variant(
    value: string,
): app.UserRole | undefined {
    switch (value) {
        case "owner":
            return app.UserRole.USER_ROLE_OWNER;
        case "admin":
            return app.UserRole.USER_ROLE_ADMIN;
        case "user":
            return app.UserRole.USER_ROLE_USER;
        default:
            return undefined;
    }
}

export function data_type_string_to_variant(
    value: string,
): app.DataType | undefined {
    switch (value) {
        case "string":
            return app.DataType.DATA_TYPE_STRING;
        case "int":
            return app.DataType.DATA_TYPE_INT;
        case "uint":
            return app.DataType.DATA_TYPE_UINT;
        case "float":
            return app.DataType.DATA_TYPE_FLOAT;
        case "boolean":
            return app.DataType.DATA_TYPE_BOOLEAN;
        case "timestamp":
            return app.DataType.DATA_TYPE_TIMESTAMP;
        default:
            return undefined;
    }
}

export function visibility_string_to_variant(
    value: string,
): app.Visibility | undefined {
    switch (value) {
        case "public":
            return app.Visibility.VISIBILITY_PUBLIC;
        case "private":
            return app.Visibility.VISIBILITY_PRIVATE;
        default:
            return undefined;
    }
}
