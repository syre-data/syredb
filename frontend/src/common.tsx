export const USER_NOT_AUTHENTICATED_ERROR = "USER_NOT_AUTHENTICATED";
export const INSUFFICIENT_PERMISSIONS_ERROR = "INSUFFICEINT_PERMISSIONS;";

export enum MouseButton {
    Primary = 0,
    Secondary = 2,
}

export enum UserPermission {
    Owner = "owner",
    Admin = "admin",
    ReadWrite = "read_write",
    Read = "read",
}

export function user_permission_from_string(
    user_permission_string: string
): UserPermission | null {
    switch (user_permission_string) {
        case UserPermission.Owner:
            return UserPermission.Owner;
        case UserPermission.Admin:
            return UserPermission.Admin;
        case UserPermission.ReadWrite:
            return UserPermission.ReadWrite;
        case UserPermission.Read:
            return UserPermission.Read;
        default:
            return null;
    }
}

export function is_admin_or_owner(user_permission: UserPermission): boolean {
    return (
        user_permission === UserPermission.Admin ||
        user_permission === UserPermission.Owner
    );
}

export enum PropertyType {
    String = "string",
    Int = "int",
    UInt = "uint",
    Float = "float",
    Boolean = "boolean",
    Quantity = "quantity",
}

export interface QuntityValue {
    magnitude: number;
    unit: string;
}

export function property_type_string_to_variant(
    value: string
): PropertyType | null {
    switch (value) {
        case "string":
            return PropertyType.String;
        case "int":
            return PropertyType.Int;
        case "uint":
            return PropertyType.UInt;
        case "float":
            return PropertyType.Float;
        case "boolean":
            return PropertyType.Boolean;
        case "quantity":
            return PropertyType.Quantity;
        default:
            return null;
    }
}

export function value_is_compatible_with_property_type(
    value: any,
    property_type: PropertyType
): boolean {
    switch (property_type) {
        case PropertyType.Boolean:
            return typeof value === "boolean";
        case PropertyType.Float:
            return typeof value === "number";
        case PropertyType.Int:
            return Number.isInteger(value);
        case PropertyType.UInt:
            return Number.isInteger(value) && value >= 0;
        case PropertyType.String:
            return typeof value === "string";
        case PropertyType.Quantity:
            return (
                typeof value === "object" &&
                "magnitude" in value &&
                "unit" in value
            );
    }
}

class IncompatiblePropertyValueError extends Error {
    #message: string | undefined;
    #expected: PropertyType;
    #value: any;

    constructor(expected: PropertyType, value: any, message?: string) {
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
    #type: PropertyType;
    #value: any;

    constructor(key: string, type: PropertyType, value?: any) {
        this.#key = key;
        this.#type = type;
        this.setValue(value);
    }

    key(): string {
        return this.#key;
    }

    type(): PropertyType {
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
