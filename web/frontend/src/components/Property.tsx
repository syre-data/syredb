import { useRef, useLayoutEffect } from "react";
import type { ComponentPropsWithoutRef } from "react";
import * as types from "@/types";
import { parseFloatWithRemainder, parseIntWithRemainder } from "@/common";

export interface QuantityValue {
    magnitude: number;
    unit: string;
}

export interface QuantityProperty {
    MagnitudeString: string;
    MagnitudeValue: number;
    Unit: string;
}

export function type_string_to_variant(
    value: string,
): types.PropertyType | undefined {
    switch (value) {
        case "string":
            return types.PropertyTypeString;
        case "int":
            return types.PropertyTypeInt;
        case "uint":
            return types.PropertyTypeUint;
        case "float":
            return types.PropertyTypeFloat;
        case "boolean":
            return types.PropertyTypeBool;
        case "quantity":
            return types.PropertyTypeQuantity;
        case "timestamp":
            return types.PropertyTypeTimestamp;
        default:
            return undefined;
    }
}

export function value_is_compatible_with_type(
    value: any,
    property_type: types.PropertyType,
): boolean {
    switch (property_type) {
        case types.PropertyTypeBool:
            return typeof value === "boolean";
        case types.PropertyTypeFloat:
            return typeof value === "number";
        case types.PropertyTypeInt:
            return Number.isInteger(value);
        case types.PropertyTypeUint:
            return Number.isInteger(value) && value >= 0;
        case types.PropertyTypeString:
            return typeof value === "string";
        case types.PropertyTypeQuantity:
            if (typeof value !== "object") {
                return false;
            }

            return "magnitude" in value && "unit" in value;
        case types.PropertyTypeTimestamp:
            if (value instanceof Date) {
                return true;
            }

            const date = new Date(value);
            return !isNaN(date.getTime());
        default:
            return false;
    }
}

// Parses a string as the given type. If parsing succeeds, validates the value conforms to the type.
// e.g. Validates value is non-negative for uint.
//
// # Notes
// + Trims string type value.
// + For numeric and date types, throws an error if string could not be parsed as value.
// + For boolean type, returns `true` if value is in ["true", "on"], `false` otherwise.
// + For quantity type, parses first part of string as float assigned to magnitude, remainder is trimmed and assigned to unit.
//
// @throws Could not parse value.
// @throws Value is invalid.
export function stringToPropertyValue(
    value: string,
    type: types.PropertyType,
): any {
    const PARSE_ERR = "could not parse value as type";

    let parsed;
    switch (type) {
        case types.PropertyTypeString:
            return value.trim();
        case types.PropertyTypeBool:
            const trueVals = ["true", "on"];
            return trueVals.includes(value);
        case types.PropertyTypeInt:
            parsed = parseIntWithRemainder(value);
            if (
                Number.isNaN(parsed.value) ||
                parsed.remaining.trim().length > 0
            ) {
                throw new Error(PARSE_ERR);
            }
            return parsed.value;
        case types.PropertyTypeUint:
            parsed = parseIntWithRemainder(value);
            if (
                Number.isNaN(parsed.value) ||
                parsed.value < 0 ||
                parsed.remaining.trim().length > 0
            ) {
                throw new Error(PARSE_ERR);
            }
            return parsed.value;
        case types.PropertyTypeFloat:
            parsed = parseFloatWithRemainder(value);
            if (
                Number.isNaN(parsed.value) ||
                parsed.remaining.trim().length > 0
            ) {
                throw new Error(PARSE_ERR);
            }
            return parsed.value;
        case types.PropertyTypeTimestamp:
            parsed = Date.parse(value);
            if (Number.isNaN(parsed)) {
                throw new Error(PARSE_ERR);
            }
            return parsed;
        case types.PropertyTypeQuantity:
            parsed = parseFloatWithRemainder(value);
            if (
                Number.isNaN(parsed.value) ||
                parsed.remaining.trim().length === 0
            ) {
                throw new Error(PARSE_ERR);
            }
            return {
                magnitude: parsed.value,
                unit: parsed.remaining.trim(),
            };
    }
}

export function value_to_string(property: types.Property): string {
    var value: any;
    switch (property.Type) {
        case types.PropertyTypeBool:
            if (property.Value === true) {
                return "true";
            } else if (property.Value === false) {
                return "false";
            } else {
                console.error("incompatible property value", property);
                return "";
            }
        case types.PropertyTypeString:
            return property.Value;
        case types.PropertyTypeUint:
            return (property.Value as number).toString();
        case types.PropertyTypeInt:
            return (property.Value as number).toString();
        case types.PropertyTypeFloat:
            return (property.Value as number).toString();
        case types.PropertyTypeQuantity:
            value = property.Value as QuantityValue;
            if ("magnitude" in value && "unit" in value) {
                return `${value.magnitude} ${value.unit}`;
            } else if ("Magnitude" in value && "Unit" in value) {
                return `${value.Magnitude} ${value.Unit}`;
            } else {
                throw new Error(`invalid property ${property}`);
            }
        case types.PropertyTypeTimestamp:
            value = property.Value as Date;
            return value.toLocaleString();
    }
}

class IncompatiblePropertyValueError extends Error {
    #message: string | undefined;
    #expected: types.PropertyType;
    #value: any;

    constructor(expected: types.PropertyType, value: any, message?: string) {
        super();
        this.#expected = expected;
        this.#value = value;
        this.#message = message;
    }

    override toString() {
        let out = `expected ${this.#expected}, found ${this.#value}`;
        if (this.#message !== undefined) {
            out += `: ${this.#message}`;
        }
        return out;
    }
}

export class Property {
    #key: string;
    #type: types.PropertyType;
    #value: any;

    constructor(key: string, type: types.PropertyType, value?: any) {
        this.#key = key;
        this.#type = type;
        this.setValue(value);
    }

    key(): string {
        return this.#key;
    }

    type(): types.PropertyType {
        return this.#type;
    }

    value(): any {
        return this.#value;
    }

    setValue(value: any) {
        if (!value_is_compatible_with_type(value, this.#type)) {
            throw new IncompatiblePropertyValueError(this.#type, value);
        }

        this.#value = value;
    }
}

type SelectPropertyTypeProps = ComponentPropsWithoutRef<"select"> & {
    ref?: React.Ref<HTMLSelectElement>;
};
export function SelectPropertyType({ ref, ...props }: SelectPropertyTypeProps) {
    return (
        <select ref={ref} {...props}>
            <option value="string" title="Text">
                String
            </option>
            <option value="int" title="Integer">
                Int
            </option>
            <option value="uint" title="Unsigned integer (counting numbers)">
                UInt
            </option>
            <option value="float" title="Decimal number">
                Float
            </option>
            <option value="boolean" title="True/False">
                Boolean
            </option>
            <option value="quantity" title="Measured value (e.g. 10.0 cm)">
                Quantity
            </option>
            <option value="timestamp" title="A timestamp">
                Timestamp
            </option>
        </select>
    );
}

type InputPropertyValueProps = ComponentPropsWithoutRef<"input"> & {
    ref?: React.Ref<HTMLSelectElement>;
    type: types.PropertyType;
    value?: any;
    defaultValue?: any;
};
export function InputPropertyValue({
    ref,
    type,
    value,
    defaultValue,
    ...props
}: InputPropertyValueProps) {
    const inputNode = useRef(null);
    switch (type) {
        case types.PropertyTypeString:
            return (
                <input
                    ref={inputNode}
                    type="text"
                    value={value}
                    defaultValue={defaultValue}
                    {...props}
                />
            );
        case types.PropertyTypeInt:
            return (
                <input
                    ref={inputNode}
                    type="number"
                    value={value}
                    defaultValue={defaultValue}
                    {...props}
                />
            );
        case types.PropertyTypeUint:
            return (
                <input
                    ref={inputNode}
                    type="number"
                    value={value}
                    defaultValue={defaultValue}
                    min={0}
                    {...props}
                />
            );
        case types.PropertyTypeFloat:
            return (
                <input
                    ref={inputNode}
                    type="text"
                    value={value}
                    defaultValue={defaultValue}
                    {...props}
                />
            );
        case types.PropertyTypeBool:
            return (
                <input
                    ref={inputNode}
                    type="checkbox"
                    checked={value}
                    defaultChecked={defaultValue}
                    {...props}
                />
            );
        case types.PropertyTypeTimestamp:
            return (
                <input
                    ref={inputNode}
                    type="datetime-local"
                    value={value}
                    defaultValue={defaultValue}
                    {...props}
                />
            );
        case types.PropertyTypeQuantity:
            return (
                <input
                    ref={inputNode}
                    type="text"
                    value={value}
                    defaultValue={defaultValue}
                    {...props}
                />
            );
    }
}
