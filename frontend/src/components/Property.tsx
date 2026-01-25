import { ComponentPropsWithoutRef, useRef, useLayoutEffect } from "react";
import * as app from "../../bindings/syredb/app";

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

export function value_is_compatible_with_type(
    value: any,
    property_type: app.PropertyType,
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
    type: app.PropertyType;
    quantity_magnitude_props?: ComponentPropsWithoutRef<"input">;
    quantity_unit_props?: ComponentPropsWithoutRef<"input">;
};
export function InputPropertyValue({
    ref,
    type,
    quantity_magnitude_props,
    quantity_unit_props,
    ...props
}: InputPropertyValueProps) {
    const inputNode = useRef(null);

    useLayoutEffect(() => {
        if (
            inputNode.current !== null &&
            type === app.PropertyType.PROPERTY_TYPE_BOOL
        ) {
            const input = inputNode.current as HTMLInputElement;
            input.indeterminate = true;
        }
    }, [inputNode]);

    switch (type) {
        case app.PropertyType.PROPERTY_TYPE_STRING:
            return <input type="string" {...props} />;
        case app.PropertyType.PROPERTY_TYPE_INT:
            return <input type="number" {...props} />;
        case app.PropertyType.PROPERTY_TYPE_UINT:
            return <input type="number" min={0} {...props} />;
        case app.PropertyType.PROPERTY_TYPE_FLOAT:
            return <input type="text" {...props} />;
        case app.PropertyType.PROPERTY_TYPE_BOOL:
            return <input ref={inputNode} type="checkbox" {...props} />;
        case app.PropertyType.PROPERTY_TYPE_TIMESTAMP:
            return <input ref={inputNode} type="datetime-local" {...props} />;
        case app.PropertyType.PROPERTY_TYPE_QUANTITY:
            const mag_id = props.id ? `${props.id}[magnitude]` : "";
            const mag_name = props.name ? `${props.name}[magnitude]` : "";
            const unit_id = props.id ? `${props.id}[unit]` : "";
            const unit_name = props.name ? `${props.name}[unit]` : "";
            const quant_props = { ...props };
            delete quant_props.id;
            delete quant_props.name;
            return (
                <>
                    <input
                        type="text"
                        id={mag_id}
                        name={mag_name}
                        placeholder="Magnitude"
                        {...quant_props}
                        {...quantity_magnitude_props}
                    />
                    <input
                        type="text"
                        id={unit_id}
                        name={unit_name}
                        placeholder="Unit"
                        {...quant_props}
                        {...quantity_unit_props}
                    />
                </>
            );
    }
}
