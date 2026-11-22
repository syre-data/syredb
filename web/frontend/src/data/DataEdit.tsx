import {
    MouseButton,
    QUERY_KEY_DATA,
    timestampToString,
    uuidToString,
} from "@/common";
import {
    InputPropertyValue,
    Loading,
    SelectPropertyType,
    StandardError,
} from "@/components";
import {
    stringToPropertyValue,
    type_string_to_variant,
    value_is_compatible_with_type,
    valueToString,
} from "@/components/Property";
import { VisibilityFormToggle, VisibilityBadge } from "@/components/Visibility";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import {
    PropertyTypeString,
    VisibilityPrivate,
    type PropertyValueUpdate,
    type DataNoteCreate,
    type DataProjectResources,
    type DataRx,
    type DataType,
    type Note,
    type Property,
    type PropertyType,
    type User,
    PropertyTypeQuantity,
    PropertyTypeBool,
    type DataResourcesUpdate,
} from "@/types";
import { useForm } from "@tanstack/react-form";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import {
    Suspense,
    type ComponentProps,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { useNavigate, useParams } from "react-router";
import { Navigate } from "react-router";
import { toast } from "sonner";
import type { UUIDTypes } from "uuid";

export default function () {
    const { data_id } = useParams();
    if (!data_id) {
        return <Navigate to="/" replace />;
    }

    return (
        <ErrorBoundary FallbackComponent={LoadError}>
            <Suspense fallback={<Loading />}>
                <DataEdit data_id={data_id} />
            </Suspense>
        </ErrorBoundary>
    );
}

function LoadError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <StandardError
            resetErrorBoundary={resetErrorBoundary}
            error={error as Error}
            title={"Could not load data"}
        />
    );
}

interface PropertiesUpdate {
    properties: Property[];
    removed: Array<Property>;
    created: Array<PropertyCreate>;
}

function prepareProperties(
    value: PropertiesUpdate,
    isPropertyValueDirty: (idx: number) => boolean,
) {
    const removed = value.removed.map((property) => property.Key);
    const created = value.created.map((property) => {
        let value;
        if (property.type === PropertyTypeQuantity) {
            value = {
                Magnitude: property.value.magnitude,
                Unit: property.value.unit,
            };
        } else {
            try {
                value = stringToPropertyValue(property.value, property.type);
            } catch (err) {
                throw new Error(
                    `could not convert value '${property.value}' (type ${typeof property.value}) to type ${property.type}`,
                );
            }
        }
        return {
            Key: property.key,
            Type: property.type,
            Value: value,
        } as Property;
    });
    const updated = value.properties
        .filter((_, idx) => isPropertyValueDirty(idx))
        .map((property) => {
            let value = property.Value;
            if (typeof value === "string") {
                value = stringToPropertyValue(property.Value, property.Type);
            }

            if (!value_is_compatible_with_type(value, property.Type)) {
                throw new Error(
                    `property value '${property.Value}' incompatible with type ${property.Type}`,
                );
            }

            return {
                Key: property.Key,
                Value: value,
            } as PropertyValueUpdate;
        });

    if (
        value.removed.length === 0 &&
        created.length === 0 &&
        updated.length === 0
    ) {
        console.info("no changes to properties");
        return;
    }

    return {
        Remove: removed,
        Create: created,
        ValuesUpdate: updated,
    };
}

function useFormData(data: DataRx, properties: Property[]) {
    const queryClient = useQueryClient();
    const properties_sorted = properties.toSorted((a, b) =>
        a.Key.localeCompare(b.Key),
    );

    return useForm({
        defaultValues: {
            visibility: data.Visibility,
            properties: {
                properties: properties_sorted,
                removed: new Array<Property>(),
                created: new Array<PropertyCreate>(),
            },
            notes: new Array<DataNoteCreate>(),
        },
        onSubmit: async ({ value, formApi }) => {
            const properties = prepareProperties(
                value.properties,
                (idx) =>
                    formApi.getFieldMeta(`properties.properties[${idx}].Value`)!
                        .isDirty,
            );
            const update = {
                Id: uuidToString(data.Id),
                Visibility: value.visibility,
                Properties: properties,
                Notes: value.notes,
            } as DataResourcesUpdate;

            await dataService.dataResourcesUpdate(update).then((_) => {
                queryClient.invalidateQueries({
                    queryKey: [QUERY_KEY_DATA, uuidToString(data.Id)],
                });

                let updatedProperties = structuredClone(
                    value.properties.properties,
                );
                if (update.Properties) {
                    updatedProperties = updatedProperties.filter((prop) =>
                        update.Properties?.Remove.includes(prop.Key),
                    );
                    updatedProperties = updatedProperties.concat(
                        update.Properties.Create,
                    );
                    for (const val of update.Properties.ValuesUpdate) {
                        const prop = updatedProperties.find(
                            (prop) => prop.Key === val.Key,
                        );
                        if (!prop) {
                            continue;
                        }

                        prop.Value = val.Value;
                    }
                    updatedProperties = updatedProperties.toSorted((a, b) =>
                        a.Key.localeCompare(b.Key),
                    );
                }

                const newDefaults = {
                    visibility: update.Visibility,
                    properties: {
                        properties: updatedProperties,
                        created: new Array(),
                        removed: new Array(),
                    },
                    notes: update.Notes,
                };
                formApi.reset(newDefaults);

                toast.success("Data updated");
            });
        },
    });
}
type DataFormApi = ReturnType<typeof useFormData>;

interface DataEditProps {
    data_id: UUIDTypes;
}
function DataEdit({ data_id }: DataEditProps) {
    const { data: resources } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA, data_id],
        queryFn: async () => await dataService.dataGet(data_id),
    });

    const data = resources.Data as DataRx;
    const data_type = resources.DataType as DataType;
    const properties = resources.Properties as Property[];
    const notes = resources.Notes as Note[];
    const project_resources =
        resources.ProjectResources as DataProjectResources[];
    const users = resources.Users as User[];
    const currentProjects = project_resources.map(
        (resource) => resource.Project.Id,
    );

    const form = useFormData(data, properties);
    function update(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        form.handleSubmit();
    }
    return (
        <div>
            <Header
                dtype_label={data_type.Label}
                timestamp={timestampToString(data.Timestamp)}
            />
            <form className=" pt-2 flex flex-col gap-2" onSubmit={update}>
                <DataEditVisibility form={form} />
                <div className="pt-2 flex flex-col gap-4 divide-y divide-gray-500">
                    <DataProperties form={form} className="pb-4" />
                    <DataNotes form={form} notes={notes} users={users} />
                </div>
                <form.Subscribe
                    selector={(state) => [state.canSubmit, state.isSubmitting]}
                >
                    {([canSubmit, isSubmitting]) => (
                        <div className="px-4 pt-2 flex gap-2 justify-between">
                            <div>
                                <button
                                    type="submit"
                                    className="btn-submit"
                                    disabled={!canSubmit}
                                >
                                    {isSubmitting ? "Saving" : "Save"}
                                </button>
                            </div>
                            <div>
                                <button
                                    type="reset"
                                    className="btn-cancel"
                                    disabled={!canSubmit}
                                    onClick={(_) => form.reset()}
                                >
                                    Reset
                                </button>
                            </div>
                        </div>
                    )}
                </form.Subscribe>
            </form>
        </div>
    );
}

interface HeaderProps {
    dtype_label: string;
    timestamp: string;
}
function Header({ dtype_label, timestamp }: HeaderProps) {
    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    return (
        <div className="px-4 pt-2 flex justify-between">
            <div className="group flex gap-2">
                <h1 className="text-xl font-bold flex gap-2 items-center">
                    <Icon.Data />
                    <span>{dtype_label}</span>
                    <span>created at</span>
                    <span>{timestamp}</span>
                </h1>
            </div>
            <div>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={close}
                    >
                        <Icon.Close />
                    </button>
                </div>
            </div>
        </div>
    );
}

interface DataEditFormProps {
    form: DataFormApi;
}
function DataEditVisibility({ form }: DataEditFormProps) {
    return (
        <div className="px-4">
            <form.Field name="visibility">
                {(field) => {
                    return (
                        <VisibilityFormToggle
                            defaultValue={form.state.values.visibility}
                            className="text-sm"
                            onChange={(visibility) =>
                                field.handleChange(visibility)
                            }
                        />
                    );
                }}
            </form.Field>
        </div>
    );
}

interface DataPropertiesProps extends ComponentProps<"div"> {
    form: DataFormApi;
}

interface PropertyCreate {
    key?: string;
    type: PropertyType;
    value: any;
}

function DataProperties({ form, className, ...props }: DataPropertiesProps) {
    function create_property(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.pushFieldValue("properties.created", {
            key: "",
            type: PropertyTypeString,
            value: "",
        });
    }

    return (
        <div className={`group ${className}`} {...props}>
            <div className="px-4 flex gap-2 items-baseline">
                <Icon.Tag />
                <h2 className="text-xl font-bold">Properties</h2>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={create_property}
                        title="Add a new property"
                    >
                        <Icon.Plus />
                    </button>
                </div>
            </div>
            <form.Subscribe
                selector={(state) =>
                    state.values.properties.properties.length +
                    state.values.properties.created.length +
                    state.values.properties.removed.length
                }
            >
                {(count) =>
                    count === 0 ? (
                        <DataPropertiesEmpty />
                    ) : (
                        <DataPropertiesContent form={form} />
                    )
                }
            </form.Subscribe>
        </div>
    );
}

function DataPropertiesEmpty() {
    return <div className="px-4 pt-2 text-secondary">(no properties)</div>;
}

interface DataPropertiesContentProps {
    // form: PropertiesFormApi;
    form: DataFormApi;
}
function DataPropertiesContent({ form }: DataPropertiesContentProps) {
    return (
        <div className="px-4 pt-2 flex flex-col gap-2">
            <div>
                <table>
                    <tbody>
                        <form.Field name="properties.properties" mode="array">
                            {(properties) => {
                                return (
                                    <>
                                        {properties.state.value.map(
                                            (property, idx) => (
                                                <DataPropertyItem
                                                    key={property.Key}
                                                    form={form}
                                                    idx={idx}
                                                    property={property}
                                                />
                                            ),
                                        )}
                                    </>
                                );
                            }}
                        </form.Field>
                    </tbody>
                </table>
                <table>
                    <tbody>
                        <form.Field name="properties.created" mode="array">
                            {(properties) => (
                                <>
                                    {properties.state.value.map(
                                        (property, idx) => (
                                            <DataPropertyCreateItem
                                                key={idx}
                                                form={form}
                                                idx={idx}
                                                property={property}
                                            />
                                        ),
                                    )}
                                </>
                            )}
                        </form.Field>
                    </tbody>
                </table>
                <table>
                    <tbody>
                        <form.Field name="properties.removed" mode="array">
                            {(properties) => (
                                <>
                                    {properties.state.value.map(
                                        (property, idx) => (
                                            <DataPropertyRemovedItem
                                                key={property.Key}
                                                form={form}
                                                idx={idx}
                                                property={property}
                                            />
                                        ),
                                    )}
                                </>
                            )}
                        </form.Field>
                    </tbody>
                </table>
            </div>
        </div>
    );
}

interface DataPropertyItemProps {
    // form: PropertiesFormApi;
    form: DataFormApi;
    idx: number;
    property: Property;
}
function DataPropertyItem({ form, idx, property }: DataPropertyItemProps) {
    function remove(e: MouseEvent<HTMLButtonElement>, idx: number) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        const removing = form.state.values.properties.properties[idx]!;
        const idx_insert = form.state.values.properties.removed.findIndex(
            (property) => property.Key.localeCompare(removing.Key) > 0,
        );
        if (idx_insert === -1) {
            form.pushFieldValue("properties.removed", removing);
        } else {
            form.insertFieldValue("properties.removed", idx_insert, removing);
        }

        form.removeFieldValue("properties.properties", idx);
    }

    return (
        <tr className="group/row">
            <th className="pr-2 py-1 text-left">{property.Key}</th>
            <td className="px-2 py-1">{property.Type}</td>
            <td className="px-2 py-1">
                <form.Field name={`properties.properties[${idx}].Value`}>
                    {(pvalue) => {
                        let value = pvalue.state.value;
                        if (typeof value === "string") {
                            value = stringToPropertyValue(value, property.Type);
                        }
                        if (property.Type === PropertyTypeQuantity) {
                            value = {
                                magnitude: value.Magnitude,
                                unit: value.Unit,
                            };
                        }

                        return (
                            <div className="flex gap-2">
                                <InputPropertyValue
                                    name={pvalue.name}
                                    type={property.Type}
                                    className="input-basic"
                                    value={value}
                                    onChange={(e) => {
                                        let update;
                                        if (
                                            property.Type === PropertyTypeBool
                                        ) {
                                            update = e.target.checked;
                                        } else if (
                                            property.Type ===
                                            PropertyTypeQuantity
                                        ) {
                                            if (
                                                e.target.name ===
                                                `properties[${idx}].Value.magnitude`
                                            ) {
                                                update = {
                                                    magnitude: parseFloat(
                                                        e.target.value,
                                                    ),
                                                    unit: value.unit,
                                                };
                                            } else if (
                                                e.target.name ===
                                                `properties[${idx}].Value.unit`
                                            ) {
                                                update = {
                                                    magnitude: value.magnitude,
                                                    unit: e.target.value.trim(),
                                                };
                                            } else {
                                                throw new Error(
                                                    `unknown field '${e.target.name}'`,
                                                );
                                            }
                                        } else {
                                            update = stringToPropertyValue(
                                                e.target.value,
                                                property.Type,
                                            );
                                        }

                                        pvalue.handleChange(update);
                                    }}
                                    placeholder="Value"
                                />
                            </div>
                        );
                    }}
                </form.Field>
            </td>
            <td className="pl-2 py-1">
                <div className=" invisible group-hover/row:visible">
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={(e) => remove(e, idx)}
                        title={`Remove property ${property.Key}`}
                    >
                        <Icon.Minus />
                    </button>
                </div>
            </td>
        </tr>
    );
}

interface DataPropertyCreatedItemProps {
    form: DataFormApi;
    idx: number;
    property: PropertyCreate;
}
function DataPropertyCreateItem({
    form,
    idx,
    property,
}: DataPropertyCreatedItemProps) {
    function remove(e: MouseEvent<HTMLButtonElement>, idx: number) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.removeFieldValue("properties.created", idx);
    }

    return (
        <tr>
            <td className="pr-2 py-1">
                <form.Field
                    name={`properties.created[${idx}].key`}
                    validators={{
                        onChangeListenTo: ["properties"],
                        onChange: ({ value, fieldApi }) => {
                            if (value === "") {
                                return undefined;
                            }

                            let match =
                                fieldApi.form.state.values.properties.created.findIndex(
                                    (other, odx) => {
                                        return (
                                            odx !== idx && other.key === value
                                        );
                                    },
                                );
                            if (match > -1) {
                                return `Key must be unique`;
                            }

                            match =
                                fieldApi.form.state.values.properties.properties.findIndex(
                                    (other) => {
                                        return other.Key === value;
                                    },
                                );
                            if (match > -1) {
                                return `Key must be unique`;
                            }

                            return undefined;
                        },
                    }}
                >
                    {(key) => (
                        <input
                            name={key.name}
                            type="text"
                            min="1"
                            className="input-basic invalid:border-syre-red-600 dark:invalid:border-syre-red-500"
                            placeholder="Key"
                            value={key.state.value}
                            onChange={(e) =>
                                key.handleChange(e.target.value.trim())
                            }
                            required
                        />
                    )}
                </form.Field>
            </td>
            <td className="px-2 py-1">
                <form.Field name={`properties.created[${idx}].type`}>
                    {(ptype) => (
                        <SelectPropertyType
                            name={ptype.name}
                            className="input-basic"
                            value={ptype.state.value}
                            onChange={(e) =>
                                ptype.handleChange(
                                    type_string_to_variant(e.target.value)!,
                                )
                            }
                        />
                    )}
                </form.Field>
            </td>
            <td className="px-2 py-1">
                <div className="flex gap-2">
                    <form.Subscribe
                        selector={(state) =>
                            state.values.properties.created[idx]?.type!
                        }
                    >
                        {(property_type) => (
                            <form.Field
                                name={`properties.created[${idx}].value`}
                            >
                                {(pvalue) => {
                                    let value = pvalue.state.value;
                                    if (typeof value === "string") {
                                        value = stringToPropertyValue(
                                            value,
                                            property.type,
                                        );
                                    }
                                    if (
                                        property.type === PropertyTypeQuantity
                                    ) {
                                        value = {
                                            magnitude: value.Magnitude,
                                            unit: value.Unit,
                                        };
                                    }

                                    return (
                                        <InputPropertyValue
                                            name={pvalue.name}
                                            type={property_type}
                                            className="input-basic"
                                            placeholder="Value"
                                            onChange={(e) => {
                                                let update;
                                                if (
                                                    property.type ===
                                                    PropertyTypeBool
                                                ) {
                                                    update = e.target.checked;
                                                } else if (
                                                    property_type ===
                                                    PropertyTypeQuantity
                                                ) {
                                                    if (
                                                        e.target.name ===
                                                        `created[${idx}].value.magnitude`
                                                    ) {
                                                        update = {
                                                            magnitude:
                                                                parseFloat(
                                                                    e.target
                                                                        .value,
                                                                ),
                                                            unit: pvalue.state
                                                                .value.unit,
                                                        };
                                                    } else if (
                                                        e.target.name ===
                                                        `created[${idx}].value.unit`
                                                    ) {
                                                        update = {
                                                            magnitude:
                                                                pvalue.state
                                                                    .value
                                                                    .magnitude,
                                                            unit: e.target.value.trim(),
                                                        };
                                                    } else {
                                                        throw new Error(
                                                            `unknown field '${e.target.name}'`,
                                                        );
                                                    }
                                                } else {
                                                    update =
                                                        stringToPropertyValue(
                                                            e.target.value,
                                                            property.type,
                                                        );
                                                }

                                                pvalue.handleChange(update);
                                            }}
                                        />
                                    );
                                }}
                            </form.Field>
                        )}
                    </form.Subscribe>
                </div>
            </td>
            <td className="pl-2 py-1">
                <button
                    type="button"
                    className="btn-cmd"
                    onMouseDown={(e) => remove(e, idx)}
                    title="Remove new property"
                >
                    <Icon.Minus />
                </button>
            </td>
        </tr>
    );
}

interface DataPropertyRemovedItemProps {
    // form: PropertiesFormApi;
    form: DataFormApi;
    idx: number;
    property: Property;
}
function DataPropertyRemovedItem({
    form,
    idx,
    property,
}: DataPropertyRemovedItemProps) {
    function keep(e: MouseEvent<HTMLButtonElement>, idx: number) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        const restoring = form.state.values.properties.removed[idx]!;
        const idx_insert = form.state.values.properties.properties.findIndex(
            (property) => property.Key.localeCompare(restoring.Key) > 0,
        );
        if (idx_insert === -1) {
            form.pushFieldValue("properties.properties", restoring);
        } else {
            form.insertFieldValue(
                "properties.properties",
                idx_insert,
                restoring,
            );
        }

        form.removeFieldValue("properties.removed", idx);
    }

    return (
        <tr className="group/row">
            <th className="pr-2 py-1 text-left text-secondary">
                {property.Key}
            </th>
            <td className="px-2 py-1 text-secondary">{property.Type}</td>
            <td className="px-2 py-1 text-secondary">
                {valueToString(property)}
            </td>
            <td className="pl-2 py-1">
                <div className=" invisible group-hover/row:visible">
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={(e) => keep(e, idx)}
                        title={`Keep property ${property.Key}`}
                    >
                        <Icon.Plus />
                    </button>
                </div>
            </td>
        </tr>
    );
}

interface DataNotesProps extends ComponentProps<"div"> {
    form: DataFormApi;
    notes: Note[];
    users: User[];
}
function DataNotes({
    form,
    notes,
    users,
    className,
    ...props
}: DataNotesProps) {
    function create_note(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.pushFieldValue("notes", {
            Timestamp: new Date(),
            Visibility: VisibilityPrivate,
            Content: "",
        });
    }

    return (
        <div className={`group ${className}`} {...props}>
            <div className="px-4 flex gap-2 items-baseline">
                <Icon.Notebook />
                <h2 className="text-xl font-bold">Notes</h2>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={create_note}
                        title="Create a new note"
                    >
                        <Icon.Plus />
                    </button>
                </div>
            </div>
            <form.Subscribe selector={(state) => state.values.notes.length}>
                {(count) =>
                    notes.length + count === 0 ? (
                        <DataNotesEmpty />
                    ) : (
                        <DataNotesContent
                            form={form}
                            notes={notes}
                            users={users}
                        />
                    )
                }
            </form.Subscribe>
        </div>
    );
}

function DataNotesEmpty() {
    return <div className="px-4 pt-2 text-secondary">(no notes)</div>;
}

interface DataNotesContentProps {
    form: DataFormApi;
    notes: Note[];
    users: User[];
}
function DataNotesContent({ form, notes, users }: DataNotesContentProps) {
    notes.sort((a, b) => b.Timestamp.valueOf() - a.Timestamp.valueOf());
    return (
        <div className="px-4 flex flex-col gap-4">
            <div>
                <form.Field name="notes" mode="array">
                    {(notes) => {
                        return (
                            <div className="flex flex-col gap-4 divide-y divide-gray-400 dark:divide-gray-600">
                                {notes.state.value.map((_, idx) => (
                                    <DataNoteCreate
                                        key={idx}
                                        form={form}
                                        idx={idx}
                                        className="pb-4"
                                    />
                                ))}
                            </div>
                        );
                    }}
                </form.Field>
            </div>
            <ol>
                {notes.map((note, idx) => {
                    const creator = users.find(
                        (user) => user.Id === note.Creator,
                    );
                    return <DataNote key={idx} note={note} creator={creator} />;
                })}
            </ol>
        </div>
    );
}

interface DataNoteCreateProps extends ComponentProps<"div"> {
    form: DataFormApi;
    idx: number;
}
function DataNoteCreate({
    form,
    idx,
    className,
    ...props
}: DataNoteCreateProps) {
    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.removeFieldValue("notes", idx);
    }

    return (
        <div className={`flex gap-2 group ${className}`} {...props}>
            <div>
                <div className="flex gap-2 items-baseline">
                    <form.Field name={`notes[${idx}].Timestamp`}>
                        {(timestamp) => (
                            <label>
                                <span className="sr-only">Created</span>
                                <input
                                    type="datetime-local"
                                    name={timestamp.name}
                                    value={timestampToString(
                                        timestamp.state.value,
                                    )}
                                    onChange={(e) =>
                                        timestamp.handleChange(
                                            new Date(e.target.value),
                                        )
                                    }
                                    max={timestampToString(new Date())}
                                />
                            </label>
                        )}
                    </form.Field>
                    <form.Field name={`notes[${idx}].Visibility`}>
                        {(field) => (
                            <VisibilityFormToggle
                                className="text-sm"
                                defaultValue={field.state.value}
                                onChange={field.handleChange}
                            />
                        )}
                    </form.Field>
                </div>
                <div className="pt-2">
                    <form.Field name={`notes[${idx}].Content`}>
                        {(content) => (
                            <textarea
                                placeholder="Content..."
                                className="input-basic w-full h-28"
                                minLength={1}
                                onChange={(e) =>
                                    content.handleChange(e.target.value)
                                }
                            />
                        )}
                    </form.Field>
                </div>
            </div>
            <div className="invisible group-hover:visible">
                <button
                    type="button"
                    className="btn-cmd"
                    onMouseDown={remove}
                    title="Remove new note"
                >
                    <Icon.Minus />
                </button>
            </div>
        </div>
    );
}

interface DataNoteProps {
    note: Note;
    creator?: User;
}
function DataNote({ note, creator }: DataNoteProps) {
    let creatorName = "(unknown)";
    if (creator) {
        if (creator.Name) {
            creatorName = creator.Name;
        } else if (creator.Email) {
            creatorName = creator.Email;
        } else {
            creatorName = uuidToString(creator.Id);
        }
    }

    return (
        <li className="border rounded overflow-hidden">
            <div className="flex gap-2 items-center px-4 py-2 bg-gray-200 dark:bg-gray-700">
                <div>{timestampToString(note.Timestamp)}</div>
                <div className="flex gap-2 items-baseline">
                    <Icon.User />
                    {creatorName}
                </div>
                <div>
                    <VisibilityBadge
                        className="text-sm"
                        visibility={note.Visibility}
                    />
                </div>
            </div>
            <div className="px-4 py-2">{note.Content}</div>
        </li>
    );
}
