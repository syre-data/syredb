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
} from "@/components/Property";
import { VisibilityIcon } from "@/components/Visibility";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import {
    PropertyTypeBool,
    PropertyTypeFloat,
    PropertyTypeInt,
    PropertyTypeQuantity,
    PropertyTypeString,
    PropertyTypeTimestamp,
    PropertyTypeUint,
    VisibilityPrivate,
    VisibilityPublic,
    type DataNoteCreate,
    type DataProjectResources,
    type DataPropertiesUpdate,
    type DataRx,
    type DataType,
    type DataUpdate,
    type Note,
    type Property,
    type PropertyType,
    type User,
    type Visibility,
} from "@/types";
import {
    FieldApi,
    useForm,
    type ReactFormExtendedApi,
} from "@tanstack/react-form";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
    Suspense,
    useState,
    type ChangeEvent,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { data, useParams } from "react-router";
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
                <div>
                    <DataEditForm data={data} />
                </div>
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
    function update(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const fdata = new FormData(e.target);
        const visibility = fdata.get("visibility")!.toString() as Visibility;

        const update = {
            Id: uuidToString(data.Id),
            Visibility: visibility,
        } as DataUpdate;
        dataService.dataUpdate(update);
    }

    return (
        <form className="px-4 flex flex-col gap-2" onSubmit={update}>
            <div>
                <div>
                    <fieldset className="flex gap-2">
                        <div>
                            <label>
                                <input
                                    type="radio"
                                    name="visibility"
                                    value={VisibilityPrivate}
                                    defaultChecked={
                                        data.Visibility === VisibilityPrivate
                                    }
                                />
                                <span className="pl-2">Private</span>
                            </label>
                        </div>
                        <div>
                            <label>
                                <input
                                    type="radio"
                                    name="visibility"
                                    value={VisibilityPublic}
                                    defaultChecked={
                                        data.Visibility === VisibilityPublic
                                    }
                                />
                                <span className="pl-2">Public</span>
                            </label>
                        </div>
                    </fieldset>
                </div>
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
function DataProperties({ data, properties }: DataPropertiesProps) {
    const [newProperties, setNewProperties] = useState<NewProperty[]>([]);

    function create_property(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        const max_id = newProperties.at(-1)?.id || 0;
        const newProp = {
            id: max_id + 1,
            key: undefined,
            type: PropertyTypeString,
            value: undefined,
        } as NewProperty;

        setNewProperties([...newProperties, newProp]);
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
            {properties.length + newProperties.length ? (
                <DataPropertiesContent
                    data={data}
                    properties={properties}
                    newProperties={newProperties}
                    setNewProperties={setNewProperties}
                />
            ) : (
                <DataPropertiesEmpty />
            )}
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

interface NewProperty {
    id: number;
    key?: string;
    type: PropertyType;
    value: any;
}

interface DataPropertiesContentProps {
    data: UUIDTypes;
    properties: Property[];
    newProperties: NewProperty[];
    setNewProperties: React.Dispatch<React.SetStateAction<NewProperty[]>>;
}
function DataPropertiesContent({
    data,
    properties,
    newProperties,
    setNewProperties,
}: DataPropertiesContentProps) {
    const [removed, setRemoved] = useState<Array<string>>([]);

    function update_key(e: ChangeEvent<HTMLInputElement>, id: number) {
        const property = newProperties.find((property) => property.id === id);
        if (property === undefined) {
            throw new Error("invalid property");
        }

        const value = e.target.value.trim();
        property.key = value;

        if (value === "") {
            return;
        }

        let idx = properties.findIndex((prop) => prop.Key == value);
        if (idx > -1) {
            e.target.setCustomValidity("Key already exists");
            return;
        }

        idx = newProperties.findIndex(
            (prop) => prop.id !== id && prop.key === value,
        );
        if (idx > -1) {
            console.debug("match");
            e.target.setCustomValidity("Key already exists");
            return;
        }
    }

    function set_property_type(e: ChangeEvent<HTMLSelectElement>, id: number) {
        const property = newProperties.find((prop) => prop.id === id);
        if (property === undefined) {
            throw new Error("invalid property");
        }

        const type = type_string_to_variant(e.target.value);
        if (type === undefined) {
            throw new Error("invalid property");
        }

        property.type = type;
        setNewProperties([...newProperties]);
    }

    function remove(e: MouseEvent<HTMLButtonElement>, key: string) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        setRemoved([...removed, key]);
    }

    function removeNew(e: MouseEvent<HTMLButtonElement>, id: number) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        const idx = newProperties.findIndex((prop) => prop.id === id);
        if (idx < 0) {
            throw new Error("invalid property");
        }

        newProperties.splice(idx, 1);
        setNewProperties([...newProperties]);
    }

    async function update(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const fdata = new FormData(e.target);
        const update = {
            Id: uuidToString(data),
            Remove: removed,
            Create: [],
            ValuesUpdate: [],
        } as DataPropertiesUpdate;

        const propPattern = new RegExp(/^property\[(.+)\]\[value\]/);
        for (const [name, field] of fdata.entries()) {
            const match = propPattern.exec(name);
            if (!match) {
                continue;
            }

            const key = match[1]!;
            const property = properties.find((property) => property.Key == key);
            if (property === undefined) {
                throw new Error("property key does not exist");
            }

            let value;
            switch (property.Type) {
                case PropertyTypeString:
                    value = field.toString().trim();
                    if (value !== property.Value) {
                        update.ValuesUpdate.push({ Key: key, Value: value });
                    }
                    break;
                case PropertyTypeBool:
                    // handled above
                    break;
                case PropertyTypeInt:
                    value = parseInt(field.toString());
                    if (value !== property.Value) {
                        update.ValuesUpdate.push({ Key: key, Value: value });
                    }
                    break;
                case PropertyTypeUint:
                    value = parseInt(field.toString());
                    if (value < 0) {
                        throw new Error("invalid property value");
                    }
                    if (value !== property.Value) {
                        update.ValuesUpdate.push({ Key: key, Value: value });
                    }
                    break;
                case PropertyTypeFloat:
                    value = parseFloat(field.toString());
                    if (value !== property.Value) {
                        update.ValuesUpdate.push({ Key: key, Value: value });
                    }
                    break;
                case PropertyTypeTimestamp:
                    value = new Date(field.toString());
                    if (value !== property.Value) {
                        update.ValuesUpdate.push({ Key: key, Value: value });
                    }
                    break;
                case PropertyTypeQuantity:
                    if (name.endsWith("[unit]")) {
                        continue;
                    }

                    const unitName = name.replace("[magnitude]", "[unit]");
                    const unit = fdata.get(unitName)!.toString().trim();
                    const magnitude = parseFloat(field.toString());
                    if (
                        magnitude !== property.Value.Magnitude ||
                        unit !== property.Value.Unit
                    ) {
                        update.ValuesUpdate.push({
                            Key: key,
                            Value: { Magnitude: magnitude, Unit: unit },
                        });
                    }
                    break;
            }
        }

        const newPropsFields = new Map<
            string,
            [string, FormDataEntryValue][]
        >();
        const newPropPattern = new RegExp(
            /^new\[(\d+)\]\[(.+?)\](?:\[(magnitude|unit)\])?/,
        );
        for (const [name, field] of fdata.entries()) {
            const match = newPropPattern.exec(name);
            if (!match) {
                continue;
            }

            const id = match[1]!;
            let label = match[2]!;
            if (match[3] != undefined) {
                label += ":" + match[3];
            }
            const vals = newPropsFields.get(id) ?? [];
            vals.push([label, field]);
            newPropsFields.set(id, vals);
        }

        for (const [id, fields] of newPropsFields.entries()) {
            const prop = {
                Key: undefined,
                Type: undefined,
                Value: undefined,
            } as Partial<Property>;
            for (const [label, field] of fields) {
                switch (label) {
                    case "key":
                        prop.Key = field.toString().trim();
                        break;
                    case "type":
                        prop.Type = type_string_to_variant(
                            field.toString().trim(),
                        );
                        break;
                    case "value":
                        prop.Value = field;
                        break;
                    case "value:magnitude":
                        if (prop.Value === undefined) {
                            prop.Value = {
                                Magnitude: field.toString().trim(),
                                Unit: undefined,
                            };
                        } else {
                            prop.Value.Magnitude = field.toString().trim();
                        }
                        break;
                    case "value:unit":
                        if (prop.Value === undefined) {
                            prop.Value = {
                                Magnitude: undefined,
                                Unit: field.toString().trim(),
                            };
                        } else {
                            prop.Value.Unit = field.toString().trim();
                        }
                        break;
                    default:
                        throw new Error(`invalid property label ${label}`);
                }
            }
            if (prop.Key === undefined || prop.Type === undefined) {
                console.debug(prop);
                throw new Error("invalid property fields");
            }

            if (prop.Type === PropertyTypeBool && prop.Value === undefined) {
                prop.Value = false;
            }

            if (prop.Value === undefined) {
                console.debug(prop);
                throw new Error("invalid property value");
            }
            if (prop.Key === "" && prop.Value.length === 0) {
                continue;
            }
            if (prop.Key === "") {
                const el = document.getElementById(
                    `new[${id}][key]`,
                )! as HTMLInputElement;
                el.setCustomValidity("Key can not be empty");
                return;
            }

            switch (prop.Type) {
                case PropertyTypeString:
                    prop.Value = prop.Value.trim();
                    break;
                case PropertyTypeBool:
                    if (typeof prop.Value === "string") {
                        prop.Value = true;
                    }
                    break;
                case PropertyTypeInt:
                    prop.Value = parseInt(prop.Value);
                    break;
                case PropertyTypeUint:
                    prop.Value = parseInt(prop.Value);
                    if (prop.Value < 0) {
                        throw new Error("invalid property value");
                    }
                    break;
                case PropertyTypeFloat:
                    prop.Value = parseFloat(prop.Value);
                    break;
                case PropertyTypeTimestamp:
                    Date.parse(prop.Value);
                    break;
                case PropertyTypeQuantity:
                    prop.Value.Magnitude = parseFloat(prop.Value.Magnitude);
                    prop.Value.Unit = prop.Value.Unit.trim();
            }

            update.Create.push(prop as Property);
        }

        await dataService.propertiesUpdate(update).then((resp) => {
            if (resp.ok) {
                setNewProperties([]);
            }
        });
    }

    return (
        <form className="px-4 pt-2 flex flex-col gap-2" onSubmit={update}>
            <div>
                <table>
                    <tbody>
                        {properties
                            .filter(
                                (property) => !removed.includes(property.Key),
                            )
                            .map((property) => {
                                let defaultValue = property.Value;
                                if (property.Type === PropertyTypeQuantity) {
                                    defaultValue = {
                                        magnitude: property.Value.Magnitude,
                                        unit: property.Value.Unit,
                                    };
                                }

                                return (
                                    <tr
                                        key={property.Key}
                                        className="group/row"
                                    >
                                        <th className="pr-2 py-1 text-left">
                                            {property.Key}
                                        </th>
                                        <td className="px-2 py-1">
                                            {property.Type}
                                        </td>
                                        <td className="px-2 py-1">
                                            <div className="flex gap-2">
                                                <InputPropertyValue
                                                    id={`property[${property.Key}][value]`}
                                                    name={`property[${property.Key}][value]`}
                                                    type={property.Type}
                                                    className="input-basic"
                                                    defaultValue={defaultValue}
                                                />
                                            </div>
                                        </td>
                                        <td className="pl-2 py-1">
                                            <div className=" invisible group-hover/row:visible">
                                                <button
                                                    type="button"
                                                    className="btn-cmd"
                                                    onMouseDown={(e) =>
                                                        remove(e, property.Key)
                                                    }
                                                    title={`Remove property ${property.Key}`}
                                                >
                                                    <Icon.Minus />
                                                </button>
                                            </div>
                                        </td>
                                    </tr>
                                );
                            })}
                        {newProperties.map((property) => {
                            return (
                                <tr key={property.id}>
                                    <td className="pr-2 py-1">
                                        <input
                                            id={`new[${property.id}][key]`}
                                            name={`new[${property.id}][key]`}
                                            type="text"
                                            min="1"
                                            className="input-basic invalid:border-syre-red-600 dark:invalid:border-syre-red-500"
                                            placeholder="Key"
                                            required
                                            onChange={(e) =>
                                                update_key(e, property.id)
                                            }
                                        />
                                    </td>
                                    <td className="px-2 py-1">
                                        <SelectPropertyType
                                            id={`new[${property.id}][type]`}
                                            name={`new[${property.id}][type]`}
                                            className="input-basic"
                                            onChange={(e) =>
                                                set_property_type(
                                                    e,
                                                    property.id,
                                                )
                                            }
                                        />
                                    </td>
                                    <td className="px-2 py-1">
                                        <div className="flex gap-2">
                                            <InputPropertyValue
                                                id={`new[${property.id}][value]`}
                                                name={`new[${property.id}][value]`}
                                                type={property.type}
                                                className="input-basic"
                                            />
                                        </div>
                                    </td>
                                    <td className="pl-2 py-1">
                                        <button
                                            type="button"
                                            className="btn-cmd"
                                            onMouseDown={(e) =>
                                                removeNew(e, property.id)
                                            }
                                            title="Remove new property"
                                        >
                                            <Icon.Minus />
                                        </button>
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>
            <div className="float gap-2">
                <div>
                    <button type="submit" className="btn-submit">
                        Save
                    </button>
                </div>
            </div>
        </form>
    );
}

function useNotesForm(data: UUIDTypes) {
    return useForm({
        defaultValues: {
            notes: new Array<DataNoteCreate>(),
        },
        onSubmit: async ({ value }) => {
            await dataService.notesCreate(data, value.notes);
        },
    });
}

type NotesFormApi = ReturnType<typeof useNotesForm>;

interface DataNotesProps {
    data: UUIDTypes;
    notes: Note[];
}
function DataNotes({ data, notes }: DataNotesProps) {
    const form = useNotesForm(data);

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
                        {(field) => {
                            return (
                                <div className="flex flex-col gap-2">
                                    {field.state.value.map((_, idx) => {
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
                                                                        name={`notes[${idx}].Created`}
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
                    {notes.map((note) => {
                        return (
                            <li>
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
