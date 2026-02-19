import { useRef, useLayoutEffect } from "react";
import type { ComponentPropsWithoutRef } from "react";
import * as types from "@/types";

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
            return (
                typeof value === "object" &&
                "magnitude" in value &&
                "unit" in value
            );
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

export function value_to_string(property: types.Property): string {
    var value;
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
            return `${value.magnitude} ${value.unit}`;
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
    defaultValue?: any;
    quantity_magnitude_props?: ComponentPropsWithoutRef<"input">;
    quantity_unit_props?: ComponentPropsWithoutRef<"input">;
};
export function InputPropertyValue({
    ref,
    type,
    defaultValue,
    quantity_magnitude_props,
    quantity_unit_props,
    ...props
}: InputPropertyValueProps) {
    const inputNode = useRef(null);
    if (defaultValue && !value_is_compatible_with_type(defaultValue, type)) {
        console.error(
            "incompatible property default value for type",
            type,
            defaultValue,
        );
        defaultValue = undefined;
    }

    useLayoutEffect(() => {
        if (inputNode.current !== null && type === types.PropertyTypeBool) {
            const input = inputNode.current as HTMLInputElement;
            input.indeterminate = true;
        }
    }, [inputNode]);

    switch (type) {
        case types.PropertyTypeString:
            return (
                <input type="string" defaultValue={defaultValue} {...props} />
            );
        case types.PropertyTypeInt:
            return (
                <input type="number" defaultValue={defaultValue} {...props} />
            );
        case types.PropertyTypeUint:
            return (
                <input
                    type="number"
                    defaultValue={defaultValue}
                    min={0}
                    {...props}
                />
            );
        case types.PropertyTypeFloat:
            return <input type="text" defaultValue={defaultValue} {...props} />;
        case types.PropertyTypeBool:
            return (
                <input
                    ref={inputNode}
                    type="checkbox"
                    defaultChecked={defaultValue}
                    {...props}
                />
            );
        case types.PropertyTypeTimestamp:
            return (
                <input
                    ref={inputNode}
                    type="datetime-local"
                    defaultValue={defaultValue}
                    {...props}
                />
            );
        case types.PropertyTypeQuantity:
            const mag_id = props.id ? `${props.id}[magnitude]` : "";
            const mag_name = props.name ? `${props.name}[magnitude]` : "";
            const unit_id = props.id ? `${props.id}[unit]` : "";
            const unit_name = props.name ? `${props.name}[unit]` : "";
            const quant_props = { ...props };
            delete quant_props.id;
            delete quant_props.name;
            let default_magnitude;
            let default_unit;
            if (defaultValue) {
                default_magnitude = (defaultValue as QuantityValue).magnitude;
                default_unit = (defaultValue as QuantityValue).unit;
            }
            return (
                <>
                    <input
                        type="text"
                        id={mag_id}
                        name={mag_name}
                        defaultValue={default_magnitude}
                        placeholder="Magnitude"
                        {...quant_props}
                        {...quantity_magnitude_props}
                    />
                    <input
                        type="text"
                        id={unit_id}
                        name={unit_name}
                        defaultValue={default_unit}
                        placeholder="Unit"
                        {...quant_props}
                        {...quantity_unit_props}
                    />
                </>
            );
    }
}
