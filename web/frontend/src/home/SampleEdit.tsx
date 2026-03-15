import { Suspense, useContext, useState } from "react";
import type { MouseEvent, ChangeEvent, Dispatch, SubmitEvent } from "react";
import { ErrorBoundary } from "react-error-boundary";
import type { FallbackProps } from "react-error-boundary";
import { useNavigate, useParams } from "react-router";
import * as common from "../common";
import { useSuspenseQuery } from "@tanstack/react-query";
import * as types from "@/types";
import icon from "@/icon";
import Loading from "@/components/Loading";
import type { UUIDTypes } from "uuid";
import {
    InputPropertyValue,
    value_to_string as property_value_to_string,
    SelectPropertyType,
    type_string_to_variant,
    value_is_compatible_with_type,
} from "@/components/Property";
import SuspenseError from "@/components/SuspenseError";
import { immerable } from "immer";
import { useImmerReducer } from "use-immer";
import * as appStateCtx from "@/AppStateContext";
import project_service from "@/service/project.service";

export default function () {
    const navigate = useNavigate();
    const { project_id, sample_id } = useParams();
    if (project_id && sample_id) {
        return (
            <ErrorBoundary FallbackComponent={SampleError}>
                <Suspense fallback={<Loading />}>
                    <Sample project_id={project_id} sample_id={sample_id} />
                </Suspense>
            </ErrorBoundary>
        );
    } else {
        navigate("/");
        return null;
    }
}

function SampleError({ error, resetErrorBoundary }: FallbackProps) {
    const err = error as common.BackendError;

    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="flex flex-col gap-2 items-center pt-4"
        >
            <div>Could not load project</div>
            <div>{err.message}</div>
        </SuspenseError>
    );
}

class SampleState {
    [immerable] = true;
    dirty: boolean;
    label: string;
    tags: string[];
    properties: types.Property[];
    properties_removed: string[];
    project_notes: types.ProjectSampleNote[];
    data: types.SampleData[];
    users: types.User[];
    sample_user_permissions: types.SampleUserPermissions[];

    constructor(resources: types.ProjectSampleResources) {
        this.dirty = false;
        this.label = resources.ProjectMembership.Label;
        this.tags = resources.ProjectTags;
        this.properties = resources.Properties;
        this.properties_removed = [];
        this.project_notes = resources.ProjectNotes;
        this.data = resources.Data;
        this.users = resources.Users;
        this.sample_user_permissions = resources.SampleUserPermissions;
    }
}

type SampleStateAction =
    | { type: "set_dirty" }
    | { type: "remove_property"; payload: { key: string } };

function sampleStateReducer(draft: SampleState, action: SampleStateAction) {
    switch (action.type) {
        case "set_dirty":
            draft.dirty = true;
            break;
        case "remove_property":
            if (
                draft.properties.findIndex(
                    (property) => property.Key === action.payload.key,
                ) < 0
            ) {
                return;
            }

            draft.properties = draft.properties.filter(
                (property) => property.Key !== action.payload.key,
            );
            draft.properties_removed.push(action.payload.key);
            draft.dirty = true;
            break;
    }
}

interface SampleProps {
    project_id: UUIDTypes;
    sample_id: UUIDTypes;
}
function Sample({ project_id, sample_id }: SampleProps) {
    const { data: sample_resources } = useSuspenseQuery({
        queryKey: [
            common.QUERY_KEY_PROJECT_SAMPLE_RESOURCES,
            project_id,
            sample_id,
        ],
        queryFn: async () =>
            project_service.getProjectSampleResources(project_id, sample_id),
    });
    const [sampleState, sampleStateDispatch] = useImmerReducer(
        sampleStateReducer,
        new SampleState(sample_resources),
    );
    const appState = useContext(appStateCtx.Context);
    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        if (sampleState.dirty) {
            const proceed = window.confirm(
                "You have unsaved changes. Are you sure you want to discard them?",
            );
            if (!proceed) {
                return;
            }
        }

        navigate(-1);
    }

    function validate_sample_label(e: ChangeEvent<HTMLInputElement>) {
        const value = e.target.value;
    }

    async function update_sample(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const data = new FormData(e.target);
        const label = data.get("label")!.toString().trim();
        const tags = data
            .get("tags")!
            .toString()
            .split(",")
            .map((tag) => tag.trim());

        let upsert_properties = [];
        try {
            upsert_properties = parse_form_data_new_properties(data);
        } catch (err) {
            console.error("TODO");
            throw err;
        }

        try {
            upsert_properties = [
                ...upsert_properties,
                ...parse_form_data_update_properties(
                    data,
                    sample_resources.Properties,
                ),
            ];
        } catch (err) {
            console.error("TODO");
            throw err;
        }

        const update = {
            Id: sample_id,
            Label: label,
            Tags: tags,
            PropertiesUpsert: upsert_properties,
            PropertiesRemove: sampleState.properties_removed,
            NotesNew: [],
            NotesUpdate: [],
            NotesRemove: [],
        } satisfies types.ProjectSampleUpdate;

        await project_service.updateProjectSample(project_id, update);
    }

    const user_permissions =
        sample_resources.SampleUserPermissions.find(
            (permissions) => permissions.User === appState.user.Id,
        )?.Permissions ?? [];

    const user_can_modify_properties =
        user_permissions.includes(types.SampleUserPermissionModifyProperties) ||
        user_permissions.includes(types.SampleUserPermissionOwner);

    const user_can_add_data =
        user_permissions.includes(types.SampleUserPermissionAddData) ||
        user_permissions.includes(types.SampleUserPermissionOwner);

    const FORM_ROW_CLASSNAME = "pb-2 col-span-full grid grid-cols-subgrid";
    return (
        <div>
            <div className="flex px-4 py-2">
                <h1 className="grow text-lg font-bold">
                    {sample_resources.ProjectMembership.Label}
                </h1>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={close}
                    >
                        <icon.Close />
                    </button>
                </div>
            </div>
            <form onSubmit={update_sample}>
                <div className="pb-2">
                    <section className="px-4 pb-2 grid grid-cols-[repeat(2,min-content)] gap-2">
                        <div className={FORM_ROW_CLASSNAME}>
                            <label className="contents">
                                <span className="col-1">Label</span>
                                <input
                                    type="text"
                                    id="label"
                                    name="label"
                                    defaultValue={
                                        sample_resources.ProjectMembership.Label
                                    }
                                    placeholder="Label"
                                    className="col-2 input-basic"
                                    onChange={validate_sample_label}
                                />
                            </label>
                        </div>
                        <div className={FORM_ROW_CLASSNAME}>
                            <label className="contents">
                                <span className="col-1">Tags</span>
                                <input
                                    type="text"
                                    id="tags"
                                    name="tags"
                                    defaultValue={sample_resources.ProjectTags.join(
                                        ", ",
                                    )}
                                    placeholder="Tags"
                                    className="col-2 input-basic"
                                />
                            </label>
                        </div>
                    </section>
                    {user_can_modify_properties ? (
                        <SamplePropertiesEditable
                            properties={sampleState.properties}
                            sampleStateDispatch={sampleStateDispatch}
                        />
                    ) : (
                        <SampleProperties properties={sampleState.properties} />
                    )}
                    {user_can_add_data ? (
                        <SampleDataEditable
                            data={sampleState.data}
                            derived_data={sample_resources.DerivedData}
                            data_schemas={sample_resources.DataSchemas}
                        />
                    ) : (
                        <SampleData
                            data={sample_resources.Data}
                            data_schemas={sample_resources.DataSchemas}
                        />
                    )}
                </div>
                <div className="flex gap-2 justify-center">
                    <div>
                        <button type="submit" className="btn-submit">
                            Save
                        </button>
                    </div>
                </div>
            </form>
        </div>
    );
}

/**
 * Parse form data for new properties.
 *
 * @param data Form data
 * @returns New properties extracted from the form data.
 * @throws Error if data is invalid.
 */
function parse_form_data_new_properties(data: FormData): types.Property[] {
    const SAMPLE_PROPERTY_PATTERN = /^property\[new\]\[(\d+?)\]/;

    const property_ids_list = [];
    for (const key of data.keys()) {
        const match = key.match(SAMPLE_PROPERTY_PATTERN);
        if (match) {
            property_ids_list.push(match[1]);
        }
    }
    const property_ids = new Set(property_ids_list);

    const properties = [];
    for (const id of property_ids.values()) {
        const key_id = `property[new][${id}][key]`;
        const type_id = `property[new][${id}][type]`;
        const value_id = `property[new][${id}][value]`;

        const key_data = data.get(key_id);
        const type_data = data.get(type_id);
        if (key_data === null || type_data === null) {
            continue;
        }

        const key = key_data.toString().trim();
        const type_str = type_data.toString().trim();
        const type = type_string_to_variant(type_str);
        if (type === undefined) {
            throw new Error(`invalid type ${type_str}`);
        }

        let value;
        let value_data;
        switch (type) {
            case types.PropertyTypeBool:
            case types.PropertyTypeFloat:
            case types.PropertyTypeInt:
            case types.PropertyTypeString:
            case types.PropertyTypeTimestamp:
            case types.PropertyTypeUint:
                value_data = data.get(value_id);
                if (value_data === null) {
                    throw new Error(`property ${key} does not have a value`);
                }
                const value_str = value_data.toString().trim();
                if (key.length === 0 && value_str.length === 0) {
                    continue;
                }
                if (key.length === 0) {
                    const input = document.getElementById(
                        value_id,
                    )! as HTMLInputElement;
                    input.setCustomValidity("Value can not be empty");
                }
                if (value_str.length === 0) {
                    const input = document.getElementById(
                        key_id,
                    )! as HTMLInputElement;
                    input.setCustomValidity("Key can not be empty");
                }
                break;
            case types.PropertyTypeQuantity:
                console.error("TODO");
                throw new Error("TODO");
                break;
        }

        switch (type) {
            case types.PropertyTypeBool:
                value = value_data!.toString() === "true";
                break;
            case types.PropertyTypeString:
                value = value_data!.toString().trim();
                break;
            case types.PropertyTypeInt:
                value = value_data!.toString().trim();
                if (value.length === 0) {
                    continue;
                }

                value = parseInt(value);
                if (isNaN(value)) {
                    const input = document.getElementById(
                        value_id,
                    )! as HTMLInputElement;
                    input.setCustomValidity("invalid integer");
                    continue;
                }
                break;
            case types.PropertyTypeUint:
                value = value_data!.toString().trim();
                if (value.length === 0) {
                    continue;
                }

                value = parseInt(value);
                if (isNaN(value) || value < 0) {
                    const input = document.getElementById(
                        value_id,
                    )! as HTMLInputElement;
                    input.setCustomValidity("invalid unsigned integer");
                    continue;
                }
                break;
            case types.PropertyTypeFloat:
                value = value_data!.toString().trim();
                if (value.length === 0) {
                    continue;
                }

                value = parseFloat(value);
                if (isNaN(value)) {
                    const input = document.getElementById(
                        value_id,
                    )! as HTMLInputElement;
                    input.setCustomValidity("could not parse as a number");
                    continue;
                }
                break;
            case types.PropertyTypeQuantity as types.PropertyType:
                const magnitude_key = `property[${id}][value][magnitude]`;
                const unit_key = `property[${id}][value][unit]`;

                const magnitude_data = data.get(magnitude_key);
                const unit_data = data.get(unit_key);
                if (magnitude_data === null) {
                    throw new Error(`quantity missing magnitude: ${key}`);
                }
                if (unit_data === null) {
                    throw new Error(`quantity missing unit: ${key}`);
                }

                const magnitude_string = magnitude_data.toString().trim();
                const unit = unit_data.toString().trim();
                if (magnitude_string.length === 0 && unit.length === 0) {
                    continue;
                }
                if (magnitude_string.length === 0 && unit.length !== 0) {
                    const input = document.getElementById(
                        magnitude_key,
                    )! as HTMLInputElement;

                    input.setCustomValidity("magnitude can not be empty");
                    continue;
                }

                const magnitude_value = parseFloat(magnitude_string);
                if (isNaN(magnitude_value)) {
                    const input = document.getElementById(
                        magnitude_key,
                    )! as HTMLInputElement;

                    input.setCustomValidity("could not parse as a number");
                    continue;
                }

                if (unit.length === 0 && magnitude_string.length !== 0) {
                    const input = document.getElementById(
                        unit_key,
                    )! as HTMLInputElement;

                    input.setCustomValidity("unit can not be empty");
                    continue;
                }

                value = {
                    MagnitudeString: magnitude_string,
                    MagnitudeValue: magnitude_value,
                    Unit: unit,
                };

                break;
            case types.PropertyTypeTimestamp:
                throw new Error("TODO");
            default:
                throw new Error(`invalid sample property type key: ${key}`);
        }

        if (!value_is_compatible_with_type(value, type)) {
            throw new Error(`incompatible value for type ${type}: ${value}`);
        }

        properties.push({
            Key: key,
            Type: type,
            Value: value,
        } satisfies types.Property);
    }

    return properties;
}

/**
 * Parse form data for new properties.
 *
 * @param data Form data
 * @returns New properties extracted from the form data.
 * @throws Error if data is invalid.
 */
function parse_form_data_update_properties(
    data: FormData,
    properties: types.Property[],
): types.Property[] {
    const SAMPLE_PROPERTY_PATTERN = /^property\[existing\]\[(\w+?)\]\[value\]/;

    const update = [];
    for (const [name, entry] of data.entries()) {
        const match = name.match(SAMPLE_PROPERTY_PATTERN);
        if (match === null) {
            continue;
        }

        const key = match[1]!;
        const property = properties.find((property) => property.Key === key);
        if (property === undefined) {
            throw new Error(`property not found ${key}`);
        }

        let value;
        switch (property.Type) {
            case types.PropertyTypeBool ||
                types.PropertyTypeFloat ||
                types.PropertyTypeInt ||
                types.PropertyTypeString ||
                types.PropertyTypeTimestamp ||
                types.PropertyTypeUint:
                const value_str = entry.toString().trim();
                if (key.length === 0 && value_str.length === 0) {
                    continue;
                }
                if (key.length === 0) {
                    const input = document.getElementById(
                        name,
                    )! as HTMLInputElement;
                    input.setCustomValidity("Value can not be empty");
                }
                if (value_str.length === 0) {
                    const input = document.getElementById(
                        name,
                    )! as HTMLInputElement;
                    input.setCustomValidity("Key can not be empty");
                }
                break;
            case types.PropertyTypeQuantity:
                console.error("todo");
                throw new Error("TODO");
                break;
        }

        switch (property.Type) {
            case types.PropertyTypeBool:
                value = entry!.toString() === "true";
                break;
            case types.PropertyTypeString:
                value = entry!.toString().trim();
                break;
            case types.PropertyTypeInt:
                value = entry!.toString().trim();
                if (value.length === 0) {
                    continue;
                }

                value = parseInt(value);
                if (isNaN(value)) {
                    const input = document.getElementById(
                        name,
                    )! as HTMLInputElement;
                    input.setCustomValidity("invalid integer");
                    continue;
                }
                break;
            case types.PropertyTypeUint:
                value = entry!.toString().trim();
                if (value.length === 0) {
                    continue;
                }

                value = parseInt(value);
                if (isNaN(value) || value < 0) {
                    const input = document.getElementById(
                        name,
                    )! as HTMLInputElement;
                    input.setCustomValidity("invalid unsigned integer");
                    continue;
                }
                break;
            case types.PropertyTypeFloat:
                value = entry!.toString().trim();
                if (value.length === 0) {
                    continue;
                }

                value = parseFloat(value);
                if (isNaN(value)) {
                    const input = document.getElementById(
                        name,
                    )! as HTMLInputElement;
                    input.setCustomValidity("could not parse as a number");
                    continue;
                }
                break;
            case types.PropertyTypeQuantity as types.PropertyType:
                const magnitude_key = `name[magnitude]`;
                const unit_key = `name[unit]`;

                const magnitude_data = data.get(magnitude_key);
                const unit_data = data.get(unit_key);
                if (magnitude_data === null) {
                    throw new Error(`quantity missing magnitude: ${key}`);
                }
                if (unit_data === null) {
                    throw new Error(`quantity missing unit: ${key}`);
                }

                const magnitude_string = magnitude_data.toString().trim();
                const unit = unit_data.toString().trim();
                if (magnitude_string.length === 0 && unit.length === 0) {
                    continue;
                }
                if (magnitude_string.length === 0 && unit.length !== 0) {
                    const input = document.getElementById(
                        magnitude_key,
                    )! as HTMLInputElement;

                    input.setCustomValidity("magnitude can not be empty");
                    continue;
                }

                const magnitude_value = parseFloat(magnitude_string);
                if (isNaN(magnitude_value)) {
                    const input = document.getElementById(
                        magnitude_key,
                    )! as HTMLInputElement;

                    input.setCustomValidity("could not parse as a number");
                    continue;
                }

                if (unit.length === 0 && magnitude_string.length !== 0) {
                    const input = document.getElementById(
                        unit_key,
                    )! as HTMLInputElement;

                    input.setCustomValidity("unit can not be empty");
                    continue;
                }

                value = {
                    MagnitudeString: magnitude_string,
                    MagnitudeValue: magnitude_value,
                    Unit: unit,
                };

                break;
            case types.PropertyTypeTimestamp:
                throw new Error("TODO");
            default:
                throw new Error(`invalid sample property type key: ${key}`);
        }

        if (!value_is_compatible_with_type(value, property.Type)) {
            throw new Error(
                `incompatible value for type ${property.Type}: ${value}`,
            );
        }

        if (property.Value !== value) {
            update.push({
                Key: key,
                Type: property.Type,
                Value: value,
            } satisfies types.Property);
        }
    }

    return update;
}

interface SamplePropertiesProps {
    properties: types.Property[];
}
function SampleProperties({ properties }: SamplePropertiesProps) {
    const FORM_ROW_CLASSNAME = "pb-2 col-span-full grid grid-cols-subgrid";

    return (
        <section>
            <div>
                <h2 className="text-lg font-bold">Properties</h2>
            </div>
            <div className="grid gap-2 grid-cols-[]">
                {properties.map((property) => (
                    <PropertyStatic
                        property={property}
                        className={FORM_ROW_CLASSNAME}
                    />
                ))}
            </div>
        </section>
    );
}

interface PropertyStaticProps {
    property: types.Property;
    className?: string;
}
function PropertyStatic({ property, className }: PropertyStaticProps) {
    return (
        <div className={className ?? ""}>
            <div className="col-1">{property.Key}</div>
            <div className="col-2">{property_value_to_string(property)}</div>
        </div>
    );
}

interface SamplePropertiesEditableProps {
    properties: types.Property[];
    sampleStateDispatch: Dispatch<SampleStateAction>;
}
function SamplePropertiesEditable({
    properties,
    sampleStateDispatch,
}: SamplePropertiesEditableProps) {
    const [newProperties, setNewProperties] = useState<number[]>([]);

    function add_property(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        const new_id =
            newProperties.length === 0
                ? 0
                : newProperties[newProperties.length - 1]! + 1;
        setNewProperties([...newProperties, new_id]);
    }

    function remove_new_property(id: number) {
        setNewProperties(newProperties.filter((pid) => pid !== id));
    }

    return (
        <section>
            <div className="flex gap-2 items-center px-4 pb-2">
                <h2 className="text-g font-bold">Properties</h2>
                <div>
                    <button
                        type="button"
                        className="btn-cmd align-middle"
                        onMouseDown={add_property}
                    >
                        <icon.Plus />
                    </button>
                </div>
            </div>
            <div className="pb-2 grid gap-2 grid-cols-[repeat(3,min-content)]">
                {properties.map((property) => (
                    <PropertyEditable
                        key={property.Key}
                        property={property}
                        className="px-4 col-span-full grid grid-cols-subgrid"
                        onRemove={(key) =>
                            sampleStateDispatch({
                                type: "remove_property",
                                payload: { key: key },
                            })
                        }
                    />
                ))}
            </div>
            <div className="grid gap-2 grid-cols-[repeat(4,min-content)]">
                {newProperties.map((id) => {
                    return (
                        <NewProperty
                            key={id}
                            id={id}
                            className="px-4 col-span-full grid grid-cols-subgrid"
                            onRemove={remove_new_property}
                        />
                    );
                })}
            </div>
        </section>
    );
}

interface PropertyEditableProps {
    property: types.Property;
    onRemove: (key: string) => void;
    className?: string;
}
function PropertyEditable({
    property,
    onRemove,
    className,
}: PropertyEditableProps) {
    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        onRemove(property.Key);
    }

    const classname = `group/sample-row ${className}`;
    return (
        <div className={classname}>
            <div className="col-1 whitespace-nowrap">
                <span className="inline-block pr-1">{property.Key}</span>
                <span>({property.Type})</span>
            </div>
            <InputPropertyValue
                type={property.Type}
                id={`property[existing][${property.Key}][value]`}
                name={`property[existing][${property.Key}][value]`}
                defaultValue={property.Value}
                className="col-2 input-basic"
            />
            <div className="col-3 flex gap-1">
                <button
                    className="btn-cmd invisible group-hover/sample-row:visible"
                    title="Remove property"
                    onMouseDown={remove}
                >
                    <icon.Trash />
                </button>
            </div>
        </div>
    );
}

interface NewPropertyProps {
    id: number;
    className?: string;
    onRemove: (id: number) => void;
}
function NewProperty({ id, className, onRemove }: NewPropertyProps) {
    const [type, setType] = useState(
        types.PropertyTypeString as types.PropertyType,
    );

    function set_type(e: ChangeEvent<HTMLSelectElement>) {
        const new_type = type_string_to_variant(e.target.value);
        if (new_type) {
            setType(new_type);
        }
    }

    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        onRemove(id);
    }

    const classname = `group/sample-row ${className}`;
    return (
        <div className={classname}>
            <div className="col-1">
                <label>
                    <span className="sr-only">Key</span>
                    <input
                        type="text"
                        id={`property[new][${id}][key]`}
                        name={`property[new][${id}][key]`}
                        placeholder="Key"
                        className="input-basic"
                    />
                </label>
            </div>
            <div className="col-2">
                <label>
                    <span className="sr-only">Type</span>
                    <SelectPropertyType
                        id={`property[new][${id}][type]`}
                        name={`property[new][${id}][type]`}
                        className="input-basic"
                        value={type}
                        onChange={set_type}
                    />
                </label>
            </div>
            <div className="col-3">
                <label className="flex gap-2">
                    <span className="sr-only">Value</span>
                    <InputPropertyValue
                        id={`property[new][${id}][value]`}
                        name={`property[new][${id}][value]`}
                        type={type}
                        className="input-basic"
                    />
                </label>
            </div>
            <div className="col-4 invisible group-hover/sample-row:visible">
                <button type="button" className="btn-cmd" onMouseDown={remove}>
                    <icon.Trash />
                </button>
            </div>
        </div>
    );
}

interface SampleDataProps {
    data: types.SampleData[];
    data_schemas: types.DataSchema[];
}
function SampleData({ data, data_schemas }: SampleDataProps) {
    return (
        <section>
            <div>
                <h2 className="text-lg font-bold">Data</h2>
            </div>
        </section>
    );
}

interface SampleDataEditableProps {
    data: types.SampleData[];
    derived_data: types.DerivedData[];
    data_schemas: types.DataSchema[];
}
function SampleDataEditable({
    data,
    derived_data,
    data_schemas,
}: SampleDataEditableProps) {
    console.debug(data, data_schemas);
    return (
        <section>
            <div className="flex gap-1 px-4">
                <h2 className="text-lg font-bold">Data</h2>
                <div>
                    <button type="button" className="btn-cmd">
                        <icon.Plus />
                    </button>
                </div>
            </div>
            <div>
                <ul className="grid gap-2 grid-cols-[repeat(3,min-content)]">
                    {data.map((data) => {
                        const data_schema = data_schemas.find(
                            (schema) => schema.Id === data.Schema,
                        )!;

                        const children_data = derived_data.filter(
                            (child) => child.SampleData === data.Id,
                        );

                        return (
                            <li key={data.Id.toString()} className="contents">
                                <SampleDataEditableListItem
                                    data={data}
                                    children={children_data}
                                    data_schema={data_schema}
                                />
                            </li>
                        );
                    })}
                </ul>
            </div>
        </section>
    );
}

interface SampleDataEditableListItemProps {
    data: types.SampleData;
    children: types.DerivedData[];
    data_schema: types.DataSchema;
}
function SampleDataEditableListItem({
    data,
    children,
    data_schema,
}: SampleDataEditableListItemProps) {
    return (
        <div className="grid col-span-full grid-cols-subgrid">
            <div className="col-1 pl-4">
                {data.Label ?? data.Timestamp.toLocaleString()}
            </div>
            <div className="col-2">{data_schema.Label}</div>
            <div className="col-3">{children.length} children</div>
        </div>
    );
}
