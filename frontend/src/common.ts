import * as app from "../bindings/syredb/app";

export const USER_NOT_AUTHENTICATED_ERROR = "USER_NOT_AUTHENTICATED";
export const INSUFFICIENT_PERMISSIONS_ERROR = "INSUFFICEINT_PERMISSIONS;";
export const QUERY_KEY_DATA_SCHEMA = "data_schema";

export enum MouseButton {
    Primary = 0,
    Secondary = 2,
}

export function project_user_permission_string_to_variant(
    value: string
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
    user_permission: app.ProjectUserPermission
): boolean {
    return (
        user_permission ===
            app.ProjectUserPermission.PROJECT_USER_PERMISSION_ADMIN ||
        user_permission ===
            app.ProjectUserPermission.PROJECT_USER_PERMISSION_OWNER
    );
}

export interface QuntityValue {
    magnitude: number;
    unit: string;
}

export function property_type_string_to_variant(
    value: string
): app.PropertyType | undefined {
    switch (value) {
        case "string":
            return app.PropertyType.PROPERTY_TYPE_STRING;
        case "int":
            return app.PropertyType.PROPERTY_TYPE_INT;
        case "uint":
            return app.PropertyType.PROPERTY_TYPE_UINT;
        case "float":
            return app.PropertyType.PROPERTY_TYPE_FLOAT;
        case "boolean":
            return app.PropertyType.PROPERTY_TYPE_BOOL;
        case "quantity":
            return app.PropertyType.PROPERTY_TYPE_QUANTITY;
        case "timestamp":
            return app.PropertyType.PROPERTY_TYPE_TIMESTAMP;
        default:
            return undefined;
    }
}

export function value_is_compatible_with_property_type(
    value: any,
    property_type: app.PropertyType
): boolean {
    switch (property_type) {
        case app.PropertyType.PROPERTY_TYPE_BOOL:
            return typeof value === "boolean";
        case app.PropertyType.PROPERTY_TYPE_FLOAT:
            return typeof value === "number";
        case app.PropertyType.PROPERTY_TYPE_INT:
            return Number.isInteger(value);
        case app.PropertyType.PROPERTY_TYPE_UINT:
            return Number.isInteger(value) && value >= 0;
        case app.PropertyType.PROPERTY_TYPE_STRING:
            return typeof value === "string";
        case app.PropertyType.PROPERTY_TYPE_QUANTITY:
            return (
                typeof value === "object" &&
                "magnitude" in value &&
                "unit" in value
            );
        case app.PropertyType.PROPERTY_TYPE_TIMESTAMP:
            if (value instanceof Date) {
                return true;
            }

            const date = new Date(value);
            return !isNaN(date.getTime());
        default:
            return false;
    }
}

class IncompatiblePropertyValueError extends Error {
    #message: string | undefined;
    #expected: app.PropertyType;
    #value: any;

    constructor(expected: app.PropertyType, value: any, message?: string) {
        super();
        this.#expected = expected;
        this.#value = value;
        this.#message = message;
    }

    toString() {
        let out = `expected ${this.#expected}, found ${this.#value}`;
        if (this.#message !== undefined) {
            out += `: ${this.#message}`;
        }
        return out;
    }
}

export class Property {
    #key: string;
    #type: app.PropertyType;
    #value: any;

    constructor(key: string, type: app.PropertyType, value?: any) {
        this.#key = key;
        this.#type = type;
        this.setValue(value);
    }

    key(): string {
        return this.#key;
    }

    type(): app.PropertyType {
        return this.#type;
    }

    value(): any {
        return this.#value;
    }

    setValue(value: any) {
        if (!value_is_compatible_with_property_type(value, this.#type)) {
            throw new IncompatiblePropertyValueError(this.#type, value);
        }

        this.#value = value;
    }
}

export function user_role_string_to_variant(
    value: string
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
    value: string
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

export function project_visibility_string_to_variant(
    value: string
): app.ProjectVisibility | undefined {
    switch (value) {
        case "public":
            return app.ProjectVisibility.PROJECT_PUBLIC;
        case "private":
            return app.ProjectVisibility.PROJECT_PRIVATE;
        default:
            return undefined;
    }
}
