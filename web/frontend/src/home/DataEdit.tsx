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
    SuspenseError,
} from "@/components";
import {
    parse_string_to_type,
    type_string_to_variant,
    value_is_compatible_with_type,
    value_to_string,
} from "@/components/Property";
import { VisibilityFormToggle, VisibilityIcon } from "@/components/Visibility";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import {
    PropertyTypeString,
    VisibilityPrivate,
    VisibilityPublic,
    type PropertyValueUpdate,
    type DataNoteCreate,
    type DataProjectResources,
    type DataRx,
    type DataType,
    type DataUpdate,
    type Note,
    type Property,
    type PropertyType,
    type User,
    PropertyTypeQuantity,
    PropertyTypeBool,
    type DataPropertiesUpdate,
} from "@/types";
import { useForm } from "@tanstack/react-form";
import {
    QueryClient,
    useMutation,
    useQuery,
    useQueryClient,
    useSuspenseQuery,
} from "@tanstack/react-query";
import { Suspense, type MouseEvent, type SubmitEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { useParams } from "react-router";
import { Navigate } from "react-router";
import { useNavigate } from "react-router";
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
        <SuspenseError resetErrorBoundary={resetErrorBoundary}>
            Could not load data
        </SuspenseError>
    );
}

interface DataEditProps {
    data_id: UUIDTypes;
}
function DataEdit({ data_id }: DataEditProps) {
    const { data: resources } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA, data_id],
        queryFn: async () => await dataService.dataGet(data_id),
    });
    const navigate = useNavigate();

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

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    return (
        <div>
            <div className="flex justify-between pt-2 px-4">
                <h1 className="text-lg flex gap-2">
                    <span>{data_type.Label}</span> |
                    <span>{timestampToString(new Date(data.Timestamp))}</span>
                </h1>
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
            <div>
                <DataEditForm data={data} />
                <DataProperties data={data.Id} properties={properties} />
                <DataNotes data={data.Id} notes={notes} />
            </div>
        </div>
    );
}

interface DataEditFormProps {
    data: DataRx;
}
function DataEditForm({ data }: DataEditFormProps) {
    const queryClient = useQueryClient();
    const form = useForm({
        defaultValues: {
            visibility: data.Visibility,
        },
        onSubmit: async ({ value }) => {
            const update = {
                Id: uuidToString(data.Id),
                Visibility: value.visibility,
            } as DataUpdate;

            await dataService.dataUpdate(update).then((res) => {
                if (res.ok) {
                    queryClient.invalidateQueries({
                        queryKey: [QUERY_KEY_DATA, uuidToString(data.Id)],
                    });
                }
            });
        },
    });

    function update(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        form.handleSubmit();
    }

    return (
        <form className=" pt-2 px-4 flex flex-col gap-2" onSubmit={update}>
            <div>
                <form.Field name="visibility">
                    {(field) => {
                        return (
                            <div>
                                <VisibilityFormToggle
                                    defaultValue={data.Visibility}
                                    className="text-lg"
                                    onChange={(visibility) =>
                                        field.handleChange(visibility)
                                    }
                                />
                            </div>
                        );
                    }}
                </form.Field>
            </div>
            <div>
                <div>
                    <button type="submit" className="btn-submit">
                        Save
                    </button>
                </div>
            </div>
        </form>
    );
}

interface DataPropertiesProps {
    data: UUIDTypes;
    properties: Property[];
}

interface PropertyCreate {
    key?: string;
    type: PropertyType;
    value: any;
}

function useMutationProperties(queryClient: QueryClient, data: UUIDTypes) {
    return useMutation({
        mutationFn: (update: DataPropertiesUpdate) => {
            return dataService.propertiesUpdate(update).then((_) => update);
        },
        onSettled: () => {
            queryClient.invalidateQueries({
                queryKey: [QUERY_KEY_DATA, uuidToString(data)],
            });
        },
    });
}

type PropertiesMutationApi = ReturnType<typeof useMutationProperties>;

function useFormProperties(
    data: UUIDTypes,
    properties: Property[],
    onSubmit: PropertiesMutationApi,
) {
    return useForm({
        defaultValues: {
            properties: properties,
            removed: new Array<Property>(),
            created: new Array<PropertyCreate>(),
        },
        onSubmit: async ({ value, formApi }) => {
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
                        value = parse_string_to_type(
                            property.value,
                            property.type,
                        );
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
                .filter(
                    (_, idx) =>
                        formApi.getFieldMeta(`properties[${idx}].Value`)!
                            .isDirty,
                )
                .map((property) => {
                    let value = property.Value;
                    if (typeof value === "string") {
                        value = parse_string_to_type(
                            property.Value,
                            property.Type,
                        );
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
                console.debug("no changes to form");
                return;
            }

            const update = {
                Id: data,
                Remove: removed,
                Create: created,
                ValuesUpdate: updated,
            };
            await onSubmit.mutateAsync(update).then((update) => {
                let props = [...formApi.state.values.properties];
                props = props.filter(
                    (property) => !update.Remove.includes(property.Key),
                );
                props = props.concat(update.Create);
                for (const { Key, Value } of update.ValuesUpdate) {
                    const idx = props.findIndex(
                        (property) => property.Key === Key,
                    );
                    if (idx < 0) {
                        throw new Error("invalid property");
                    }
                    props[idx]!.Value = Value;
                }

                formApi.reset({
                    properties: props,
                    removed: new Array<Property>(),
                    created: new Array<PropertyCreate>(),
                });
            });
        },
    });
}

type PropertiesFormApi = ReturnType<typeof useFormProperties>;

function DataProperties({ data, properties }: DataPropertiesProps) {
    const queryClient = useQueryClient();
    const update_mutation = useMutationProperties(queryClient, data);

    const properties_sorted = properties.toSorted((a, b) =>
        a.Key.localeCompare(b.Key),
    );
    const form = useFormProperties(data, properties_sorted, update_mutation);

    function create_property(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.pushFieldValue("created", {
            key: "",
            type: PropertyTypeString,
            value: "",
        });
    }

    return (
        <div className="pt-2 group">
            <div className="px-4 flex gap-2">
                <h2>Properties</h2>
                <div className="invisible group-hover:visible">
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
                    state.values.properties.length +
                    state.values.created.length +
                    state.values.removed.length
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
    return (
        <div className="px-4 pt-2 text-syre-grey-700 dark:text-syre-grey-300">
            (no properties)
        </div>
    );
}

interface DataPropertiesContentProps {
    form: PropertiesFormApi;
}
function DataPropertiesContent({ form }: DataPropertiesContentProps) {
    function update(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        form.handleSubmit();
    }

    return (
        <form className="px-4 pt-2 flex flex-col gap-2" onSubmit={update}>
            <div>
                <table>
                    <tbody>
                        <form.Field name="properties" mode="array">
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
                        <form.Field name="created" mode="array">
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
                        <form.Field name="removed" mode="array">
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
            <div className="float gap-2">
                <div>
                    <form.Subscribe
                        selector={(state) => [
                            state.canSubmit,
                            state.isSubmitting,
                        ]}
                        children={([canSubmit, isSubmitting]) => (
                            <button
                                type="submit"
                                className="btn-submit"
                                disabled={!canSubmit}
                            >
                                {isSubmitting ? "Saving" : "Save"}
                            </button>
                        )}
                    />
                </div>
            </div>
        </form>
    );
}

interface DataPropertyItemProps {
    form: PropertiesFormApi;
    idx: number;
    property: Property;
}
function DataPropertyItem({ form, idx, property }: DataPropertyItemProps) {
    function remove(e: MouseEvent<HTMLButtonElement>, idx: number) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        const removing = form.state.values.properties[idx]!;
        const idx_insert = form.state.values.removed.findIndex(
            (property) => property.Key.localeCompare(removing.Key) > 0,
        );
        if (idx_insert === -1) {
            form.pushFieldValue("removed", removing);
        } else {
            form.insertFieldValue("removed", idx_insert, removing);
        }

        form.removeFieldValue("properties", idx);
    }

    return (
        <tr className="group/row">
            <th className="pr-2 py-1 text-left">{property.Key}</th>
            <td className="px-2 py-1">{property.Type}</td>
            <td className="px-2 py-1">
                <form.Field name={`properties[${idx}].Value`}>
                    {(pvalue) => {
                        let value = pvalue.state.value;
                        if (typeof value === "string") {
                            value = parse_string_to_type(value, property.Type);
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
                                            update = parse_string_to_type(
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
    form: PropertiesFormApi;
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

        form.removeFieldValue("created", idx);
    }

    return (
        <tr>
            <td className="pr-2 py-1">
                <form.Field
                    name={`created[${idx}].key`}
                    validators={{
                        onChangeListenTo: ["properties"],
                        onChange: ({ value, fieldApi }) => {
                            if (value === "") {
                                return undefined;
                            }

                            let match =
                                fieldApi.form.state.values.created.findIndex(
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
                                fieldApi.form.state.values.properties.findIndex(
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
                <form.Field name={`created[${idx}].type`}>
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
                        selector={(state) => state.values.created[idx]?.type!}
                    >
                        {(property_type) => (
                            <form.Field name={`created[${idx}].value`}>
                                {(pvalue) => {
                                    let value = pvalue.state.value;
                                    if (typeof value === "string") {
                                        value = parse_string_to_type(
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
                                                        parse_string_to_type(
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
    form: PropertiesFormApi;
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

        const restoring = form.state.values.removed[idx]!;
        const idx_insert = form.state.values.properties.findIndex(
            (property) => property.Key.localeCompare(restoring.Key) > 0,
        );
        if (idx_insert === -1) {
            form.pushFieldValue("properties", restoring);
        } else {
            form.insertFieldValue("properties", idx_insert, restoring);
        }

        form.removeFieldValue("removed", idx);
    }

    return (
        <tr className="group/row">
            <th className="pr-2 py-1 text-left text-syre-grey-600 dark:text-syre-grey-300">
                {property.Key}
            </th>
            <td className="px-2 py-1 text-syre-grey-600 dark:text-syre-grey-300">
                {property.Type}
            </td>
            <td className="px-2 py-1 text-syre-grey-600 dark:text-syre-grey-300">
                {value_to_string(property)}
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

function useFormNotes(data: UUIDTypes) {
    return useForm({
        defaultValues: {
            notes: new Array<DataNoteCreate>(),
        },
        onSubmit: async ({ value }) => {
            await dataService.notesCreate(data, value.notes);
        },
    });
}

type NotesFormApi = ReturnType<typeof useFormNotes>;

interface DataNotesProps {
    data: UUIDTypes;
    notes: Note[];
}
function DataNotes({ data, notes }: DataNotesProps) {
    const form = useFormNotes(data);

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
        <div className="pt-2 group">
            <div className="px-4 flex gap-2">
                <h2>Notes</h2>
                <div className="invisible group-hover:visible">
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
                        <DataNotesContent form={form} notes={notes} />
                    )
                }
            </form.Subscribe>
        </div>
    );
}

function DataNotesEmpty() {
    return (
        <div className="px-4 pt-2 text-syre-grey-700 dark:text-syre-grey-300">
            (no notes)
        </div>
    );
}

interface DataNotesContentProps {
    form: NotesFormApi;
    notes: DataNoteCreate[];
}
function DataNotesContent({ form, notes }: DataNotesContentProps) {
    function remove(e: MouseEvent<HTMLButtonElement>, idx: number) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.removeFieldValue("notes", idx);
    }

    function update(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        form.handleSubmit();
    }

    notes.sort((a, b) => b.Timestamp.valueOf() - a.Timestamp.valueOf());
    return (
        <div>
            <form className="px-4 flex flex-col gap-2" onSubmit={update}>
                <div className="flex flex-col gap-2">
                    <form.Field name="notes" mode="array">
                        {(notes) => {
                            return (
                                <div className="flex flex-col gap-2">
                                    {notes.state.value.map((_, idx) => {
                                        return (
                                            <div
                                                key={idx}
                                                className="flex gap-2 group"
                                            >
                                                <div>
                                                    <div>
                                                        <form.Field
                                                            name={`notes[${idx}].Timestamp`}
                                                        >
                                                            {(timestamp) => (
                                                                <label>
                                                                    <span className="sr-only">
                                                                        Created
                                                                    </span>
                                                                    <input
                                                                        type="datetime-local"
                                                                        name={
                                                                            timestamp.name
                                                                        }
                                                                        value={timestampToString(
                                                                            timestamp
                                                                                .state
                                                                                .value,
                                                                        )}
                                                                        onChange={(
                                                                            e,
                                                                        ) =>
                                                                            timestamp.handleChange(
                                                                                new Date(
                                                                                    e
                                                                                        .target
                                                                                        .value,
                                                                                ),
                                                                            )
                                                                        }
                                                                        max={timestampToString(
                                                                            new Date(),
                                                                        )}
                                                                    />
                                                                </label>
                                                            )}
                                                        </form.Field>
                                                        <form.Field
                                                            name={`notes[${idx}].Visibility`}
                                                        >
                                                            {(visibility) => (
                                                                <fieldset className="flex gap-2">
                                                                    <label>
                                                                        <input
                                                                            type="radio"
                                                                            name={
                                                                                visibility.name
                                                                            }
                                                                            value={
                                                                                VisibilityPublic
                                                                            }
                                                                            onChange={(
                                                                                _,
                                                                            ) =>
                                                                                visibility.handleChange(
                                                                                    VisibilityPublic,
                                                                                )
                                                                            }
                                                                            defaultChecked={
                                                                                visibility
                                                                                    .state
                                                                                    .value ===
                                                                                VisibilityPublic
                                                                            }
                                                                        />
                                                                        <span className="pl-2">
                                                                            Public
                                                                        </span>
                                                                    </label>
                                                                    <label>
                                                                        <input
                                                                            type="radio"
                                                                            name={
                                                                                visibility.name
                                                                            }
                                                                            value={
                                                                                VisibilityPrivate
                                                                            }
                                                                            onChange={(
                                                                                _,
                                                                            ) =>
                                                                                visibility.handleChange(
                                                                                    VisibilityPrivate,
                                                                                )
                                                                            }
                                                                            defaultChecked={
                                                                                visibility
                                                                                    .state
                                                                                    .value ===
                                                                                VisibilityPrivate
                                                                            }
                                                                        />
                                                                        <span className="pl-2">
                                                                            Private
                                                                        </span>
                                                                    </label>
                                                                </fieldset>
                                                            )}
                                                        </form.Field>
                                                    </div>
                                                    <div>
                                                        <form.Field
                                                            name={`notes[${idx}].Content`}
                                                        >
                                                            {(content) => (
                                                                <textarea
                                                                    placeholder="Content..."
                                                                    className="input-basic"
                                                                    minLength={
                                                                        1
                                                                    }
                                                                    onChange={(
                                                                        e,
                                                                    ) =>
                                                                        content.handleChange(
                                                                            e
                                                                                .target
                                                                                .value,
                                                                        )
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
                                                        onMouseDown={(e) =>
                                                            remove(e, idx)
                                                        }
                                                        title="Remove new note"
                                                    >
                                                        <Icon.Minus />
                                                    </button>
                                                </div>
                                            </div>
                                        );
                                    })}
                                </div>
                            );
                        }}
                    </form.Field>
                </div>
                <ol>
                    {notes.map((note, idx) => {
                        return (
                            <li key={idx}>
                                <div className="flex gap-2 items-center">
                                    <div>
                                        {timestampToString(note.Timestamp)}
                                    </div>
                                    <div>
                                        <VisibilityIcon
                                            visibility={note.Visibility}
                                        />
                                    </div>
                                </div>
                                <div>{note.Content}</div>
                            </li>
                        );
                    })}
                </ol>
                <div>
                    <form.Subscribe
                        selector={(state) => [
                            state.canSubmit,
                            state.isSubmitting,
                        ]}
                        children={([canSubmit, isSubmitting]) => (
                            <div>
                                <button
                                    type="submit"
                                    className="btn-submit"
                                    disabled={!canSubmit}
                                >
                                    {isSubmitting ? "Saving" : "Save"}
                                </button>
                            </div>
                        )}
                    />
                </div>
            </form>
        </div>
    );
}
