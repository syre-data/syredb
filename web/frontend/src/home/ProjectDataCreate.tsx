import {
    MouseButton,
    QUERY_KEY_DATA_SCHEMA_RESOURCES,
    QUERY_KEY_DATA_TYPES,
    uuidToString,
} from "@/common";
import {
    InputPropertyValue,
    Loading,
    SelectPropertyType,
    SuspenseError,
} from "@/components";
import {
    type_string_to_variant,
    type QuantityProperty,
} from "@/components/Property";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import {
    DataCreatorTypeUser,
    DataSchemaCardinalityMultiple,
    DataSchemaCardinalitySingle,
    DataSourceCardinalityMultiple,
    DataSourceCardinalitySingle,
    DataStorageExternal,
    DataStorageInternal,
    PropertyTypeBool,
    PropertyTypeFloat,
    PropertyTypeInt,
    PropertyTypeQuantity,
    PropertyTypeString,
    PropertyTypeTimestamp,
    PropertyTypeUint,
    ValueTypeBoolean,
    ValueTypeFloat,
    ValueTypeInt,
    ValueTypeString,
    ValueTypeTimestamp,
    ValueTypeUint,
    VisibilityPrivate,
    type DataSchemaField,
    type DataSchemaResources,
    type DataStorage,
    type DataType,
    type DataTypeExternal,
    type DataTypeExternalSourceRx,
    type DataTypeInternal,
    type Note,
    type Property,
    type PropertyType,
} from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";
import {
    Suspense,
    useEffect,
    useState,
    type ChangeEvent,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { redirect, useNavigate, useParams } from "react-router";
import { type UUIDTypes } from "uuid";

function datetime_for_input(datetime: Date): string {
    const yyyy = datetime.getFullYear();
    const mo = datetime.getMonth() + 1; // getMonth is 0 based
    const mm = mo.toString().padStart(2, "0");
    const dd = datetime.getDate().toString().padStart(2, "0");
    const hh = datetime.getHours().toString().padStart(2, "0");
    const jj = datetime.getMinutes().toString().padStart(2, "0");
    return `${yyyy}-${mm}-${dd}T${hh}:${jj}`;
}

export default function () {
    const { project_id } = useParams();
    if (project_id === undefined) {
        redirect("/");
        return;
    }

    return (
        <ErrorBoundary FallbackComponent={ProjectDataCreateError}>
            <Suspense fallback={<Loading />}>
                <ProjectDataCreate projectId={project_id} />
            </Suspense>
        </ErrorBoundary>
    );
}

function ProjectDataCreateError({ resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="pt-4 text-center"
        >
            <div>Could not load project data</div>
        </SuspenseError>
    );
}

interface ProjectDataCreateProps {
    projectId: UUIDTypes;
}
function ProjectDataCreate({ projectId }: ProjectDataCreateProps) {
    const { data: data_types } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPES],
        queryFn: dataService.dataTypesGetAll,
    });
    const navigate = useNavigate();

    function cancel(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <h2>Create project data</h2>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={cancel}
                    >
                        <Icon.Close />
                    </button>
                </div>
            </div>
            <ProjectData project={projectId} dataTypes={data_types} />
        </div>
    );
}

interface Datum {
    id: number;
}

function partition_form_data(
    form: FormData,
): Map<string, [string, FormDataEntryValue][]> {
    const data_pattern = /^data\[(\d+)\]/;
    const data_fields = new Map<string, [string, FormDataEntryValue][]>();
    for (const [key, field] of form.entries()) {
        const match = data_pattern.exec(key);
        if (match === null) {
            continue;
        }

        const id = match[1]!;
        const fields = data_fields.getOrInsert(id, []);
        fields.push([key, field]);
    }

    return data_fields;
}

function string_to_property_value_type(
    type: PropertyType,
    value: string | { magnitude: number; unit: string },
): any {
    switch (type) {
        case PropertyTypeString:
        case PropertyTypeInt:
        case PropertyTypeUint:
        case PropertyTypeFloat:
        case PropertyTypeTimestamp:
            if (typeof value !== "string") {
                throw new Error(`invalid value for type ${type}: ${value}`);
            }
    }

    switch (type) {
        case PropertyTypeString:
            return value;
        case PropertyTypeBool:
            return !!value;
        case PropertyTypeInt:
            return parseInt(value as string);
        case PropertyTypeUint:
            const vint = parseInt(value as string);
            if (vint < 0) {
                // TODO: don't throw, show user error
                throw new Error("invalid value");
            }
            return vint;
        case PropertyTypeFloat:
            return parseFloat(value as string);
        case PropertyTypeTimestamp:
            return new Date(value as string);
        case PropertyTypeQuantity:
            return value;
        default:
            throw new Error(`invalid property type for property ${type}`);
    }
}

function parse_form_data_properties(
    fields: [string, FormDataEntryValue][],
    data_id: string,
): Property[] {
    const property_pattern = new RegExp(
        `^data\\[${data_id}\\]\\[property\\]\\[(\\d+)\\]\\[(\\w+)\\](?:\\[(magnitude|unit)\\])?`,
    );
    const prop_fields = new Map<string, Partial<Property>>();
    for (const [key, field] of fields) {
        const prop_match = property_pattern.exec(key);
        if (!prop_match) {
            continue;
        }

        const prop_id = prop_match[1]!;
        const prop_field = prop_match[2]!;
        const prop_quantity_field = prop_match[3];
        const property = prop_fields.getOrInsert(prop_id, {
            Key: undefined,
            Type: undefined,
            Value: undefined,
        } as Partial<Property>);
        switch (prop_field) {
            case "key":
                property.Key = field.toString();
                break;
            case "type":
                property.Type = field.toString() as PropertyType;
                break;
            case "value":
                if (prop_quantity_field === undefined) {
                    property.Value = field.toString();
                } else {
                    if (property.Value == undefined) {
                        property.Value = {
                            MagnitudeString: undefined,
                            MagnitudeValue: undefined,
                            Unit: undefined,
                        } as Partial<QuantityProperty>;
                    }

                    switch (prop_quantity_field) {
                        case "magnitude":
                            const value = field.toString();
                            const value_num = parseFloat(value);
                            property.Value.MagnitudeString = value;
                            property.Value.MagnitudeValue = value_num;
                            break;
                        case "unit":
                            property.Value.Unit = field.toString();
                            break;
                        default:
                            throw new Error(
                                `invalid quantity property value key: ${prop_quantity_field}`,
                            );
                    }
                }
                break;
            default:
                throw new Error(`invalid property key ${key}`);
        }
    }

    const properties = new Array<Property>();
    for (const [id, property] of prop_fields.entries()) {
        if (!property.Key && !property.Value) {
            continue;
        }
        if (!property.Key) {
            throw new Error(`missing key for property ${id}`);
        }
        if (!property.Type) {
            throw new Error(`invalid property type for property ${id}`);
        }

        const value = string_to_property_value_type(
            property.Type,
            property.Value,
        );
        properties.push({
            Key: property.Key,
            Type: property.Type,
            Value: value,
        } satisfies Property);
    }

    return properties;
}

function parse_form_data_notes(
    fields: [string, FormDataEntryValue][],
    data_id: string,
): Note[] {
    const note_pattern = new RegExp(
        `^data\\[${data_id}\\]\\[note\\]\\[(\\d+)\\]\\[(\\w+)\\]`,
    );

    const note_fields = new Map<string, Partial<Note>>();
    for (const [key, field] of fields) {
        const note_match = note_pattern.exec(key);
        if (note_match) {
            const note_id = note_match[1]!;
            const note_field = note_match[2]!;
            const note = note_fields.getOrInsert(note_id, {
                Timestamp: undefined,
                Visibility: VisibilityPrivate,
                Content: undefined,
            });
            switch (note_field) {
                case "timestamp":
                    note.Timestamp = new Date(field.toString());
                    break;
                case "content":
                    note.Content = field.toString();
                    break;
                default:
                    throw new Error(`invalid note key ${key}`);
            }
        }
    }

    const notes = new Array<NoteCreate>();
    for (const [id, note] of note_fields.entries()) {
        if (!note.Content) {
            continue;
        }
        if (!note.Timestamp) {
            throw new Error(`missing timetamp for note ${id}`);
        }

        notes.push({
            Timestamp: note.Timestamp,
            Visibility: note.Visibility!,
            Content: note.Content,
        } satisfies NoteCreate);
    }

    return notes;
}

/**
 *
 * @param data_id
 * @param fields
 * @param datum
 * @returns `undefined` if the data should be ignored due to empty fields.
 */
function parse_form_data(
    fields: [string, FormDataEntryValue][],
    data_id: string,
    dataTypes: DataType[],
    dataTypeSchemas: Record<string, DataSchemaResources>,
): [string, DataIngest] | undefined {
    const e_label = fields.find(
        ([key, _]) => key === `data[${data_id}][label]`,
    );
    const e_type = fields.find(([key, _]) => key === `data[${data_id}][type]`);
    const e_timestamp = fields.find(
        ([key, _]) => key === `data[${data_id}][timestamp]`,
    );
    if (!e_label) {
        throw new Error("invalid data form input: missing label");
    }
    if (!e_timestamp) {
        throw new Error("invalid data form input: missing timestamp");
    }

    const label = e_label[1].toString();
    if (!label && !e_type) {
        return undefined;
    }
    if (!e_type) {
        throw new Error(`invalid data type: ${e_type}`);
    }

    const type = e_type[1].toString();
    const timestamp = new Date(e_timestamp[1].toString());
    const datum_info = {
        Type: type,
        CreatorType: DataCreatorTypeUser,
        Timestamp: timestamp,
        Visibility: VisibilityPrivate,
        Properties: new Array<Property>(),
        Notes: new Array<Note>(),
        Values: {},
        Sources: {},
    } satisfies DataIngest;

    datum_info.Properties = parse_form_data_properties(fields, data_id);
    datum_info.Notes = parse_form_data_notes(fields, data_id);

    const data_type = dataTypes.find((dtype) => dtype.Id === type);
    switch (data_type.Storage as DataStorage) {
        case DataStorageInternal:
            const schema = dataTypeSchemas[datum_info.Type];
            if (!schema) {
                throw new Error(
                    `data schema for type ${datum_info.Type} not found`,
                );
            }
            const values = parse_form_data_internal(
                fields,
                data_id,
                schema.DataSchema.Fields,
            );
            datum_info.Values = values;
            break;
        case DataStorageExternal:
            const dtype = data_type as DataTypeExternal;
            const sources = parse_form_data_external(
                fields,
                data_id,
                dtype.Sources,
            );
            datum_info.Sources = sources;
            break;
    }

    return [label, datum_info];
}

function parse_form_data_internal(
    fields: [string, FormDataEntryValue][],
    data_id: string,
    schema: DataSchemaField[],
) {
    let rx_idx_max = -1;
    const data_store = new Map<string, [number, string][]>();
    const data_pattern = new RegExp(
        `^data\\[${data_id}\\]\\[value\\]\\[(\\d+)\\]\\[(\\w+)\\]$`,
    );
    for (const [key, field] of fields) {
        const match = data_pattern.exec(key);
        if (!match) {
            continue;
        }

        const rx_idx = parseInt(match[1]!);
        const rx_col = match[2]!;
        const value = field.toString();
        const col_data = data_store.getOrInsert(rx_col, []);
        col_data.push([rx_idx, value]);
        if (rx_idx > rx_idx_max) {
            rx_idx_max = rx_idx;
        }
    }

    let remove_last_value = true;
    for (const values of data_store.values()) {
        for (const [idx, value] of values) {
            if (idx === rx_idx_max && value !== "") {
                remove_last_value = false;
                break;
            }
        }
    }
    if (remove_last_value) {
        for (const values of data_store.values()) {
            values.length -= 1;
        }
    }

    let values = {};
    for (const [key, values_store] of data_store.entries()) {
        const field = schema.find((field) => field.Label === key);
        if (!field) {
            throw new Error(`invalid schema field ${key}`);
        }

        const field_values = new Array<any>(values_store.length);
        switch (field.DType) {
            case ValueTypeUint:
                for (const [idx, value] of values_store) {
                    const v = parseInt(value);
                    if (v < 0) {
                        throw new Error(
                            `invalid data value ${v} of type ${field.DType}`,
                        );
                    }

                    field_values[idx] = v;
                }
                break;
            case ValueTypeInt:
                for (const [idx, value] of values_store) {
                    field_values[idx] = parseInt(value);
                }
                break;
            case ValueTypeFloat:
                for (const [idx, value] of values_store) {
                    field_values[idx] = parseFloat(value);
                }
                break;
            case ValueTypeString:
                for (const [idx, value] of values_store) {
                    field_values[idx] = value;
                }
                break;

            case ValueTypeBoolean:
            case ValueTypeTimestamp:
                throw new Error(`TODO: parse data of type ${field?.DType}`);
            default:
                throw new Error(`invalid data type ${field.DType}`);
        }

        values = { ...values, [key]: field_values };
    }

    return values;
}

// # Returns
// Object keyed by source id with values of field name(s).
function parse_form_data_external(
    fields: [string, FormDataEntryValue][],
    data_id: string,
    sources: DataTypeExternalSourceRx[],
): { [key: string]: string | string[] } {
    let values = {};
    for (const source of sources) {
        const field_key = `data[${data_id}][source][${source.Id}]`;
        const field = fields.find(([key, _]) => key === field_key);
        if (source.Required && !field) {
            throw new Error(`missing data source ${source.Label}`);
        }
        if (!field) {
            console.debug(`field ${field_key} ignored`);
            continue;
        }

        const [name, _] = field;
        values = { ...values, [uuidToString(source.Id)]: name };
    }

    return values;
}

interface ProjectDataProps {
    project: UUIDTypes;
    dataTypes: DataType[];
}
function ProjectData({ project, dataTypes }: ProjectDataProps) {
    const navigate = useNavigate();
    const [data, setData] = useState<Datum[]>([{ id: 0 }]);
    const [dataTypeSchemas, setDataTypeSchemas] = useState<
        Record<string, DataSchemaResources>
    >({});

    function create_new_datum(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        const id = Math.max(...data.map((datum) => datum.id)) + 1;
        const datum = {
            id,
        };
        setData([...data, datum]);
    }

    function remove_datum(id: number) {
        setData(data.filter((datum) => datum.id !== id));
    }

    function cache_data_type_schema(
        data_type: UUIDTypes,
        schema: DataSchemaResources,
    ) {
        setDataTypeSchemas({
            ...dataTypeSchemas,
            [data_type.toString()]: schema,
        });
    }

    async function create_project_data(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const form = new FormData(e.target);
        const data_fields = partition_form_data(form);

        const info = new Array<DataIngest>();
        const labels = new Array<string>();
        const files = new Array<[string, File | File[]]>();
        for (const [id, fields] of data_fields.entries()) {
            const idx = parseInt(id);
            const datum_values = parse_form_data(
                fields,
                id,
                dataTypes,
                dataTypeSchemas,
            );
            if (datum_values === undefined) {
                continue;
            }

            const [label, datum_info] = datum_values;
            const data_type = dataTypes.find(
                (dtype) => dtype.Id === datum_info.Type,
            )!;
            if (data_type.Storage === DataStorageExternal) {
                const dtype = data_type as DataTypeExternal;
                for (const source of dtype.Sources) {
                    const srcs = datum_info.Sources[uuidToString(source.Id)]!;
                    switch (source.Cardinality) {
                        case DataSourceCardinalitySingle:
                            files.push([srcs, form.get(srcs)! as File]);
                            break;
                        case DataSourceCardinalityMultiple:
                            files.push([srcs, form.getAll(srcs)! as File[]]);
                            break;
                    }
                }
            }
            info.push(datum_info);
            labels.push(label);
        }

        if (info.length === 0) {
            console.debug("no data");
            return;
        }

        await dataService
            .projectDataCreate(project, info, labels, files)
            .then((resp) => {
                if (resp.ok) {
                    navigate(-1);
                }
            });
    }

    return (
        <form
            className="pt-2 flex flex-col gap-2"
            onSubmit={create_project_data}
        >
            <div className="flex flex-col gap-2">
                <ol className="list-decimal">
                    {data.map((datum) => (
                        <ProjectDataItem
                            key={datum.id}
                            datum={datum}
                            dataTypes={dataTypes}
                            canRemove={data.length > 1}
                            onRemove={remove_datum}
                            cacheDataTypeSchema={cache_data_type_schema}
                        />
                    ))}
                </ol>
                <div className="px-4">
                    <button
                        type="button"
                        className="btn-cmd"
                        title="Add another data"
                        onMouseDown={create_new_datum}
                    >
                        <Icon.Plus />
                    </button>
                </div>
            </div>
            <div className="px-4">
                <button type="submit" className="btn-submit">
                    Create data
                </button>
            </div>
        </form>
    );
}

interface ProjectDataItemProps {
    datum: Datum;
    dataTypes: DataType[];
    canRemove: boolean;
    onRemove: (id: number) => void;
    cacheDataTypeSchema: (
        data_type: UUIDTypes,
        schema: DataSchemaResources,
    ) => void;
}
function ProjectDataItem({
    datum,
    dataTypes,
    canRemove,
    onRemove,
    cacheDataTypeSchema,
}: ProjectDataItemProps) {
    const [dataType, setDataType] = useState<undefined | DataType>(undefined);
    const datum_name = `data[${datum.id}]`;

    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        onRemove(datum.id);
    }

    function set_data_type(e: ChangeEvent<HTMLSelectElement>) {
        const data_type = dataTypes.find((type) => type.Id === e.target.value);
        if (!data_type) {
            console.error("invalid data type");
            return;
        }

        setDataType(data_type);
    }

    const now = datetime_for_input(new Date());
    return (
        <li className="px-4 pb-2">
            <div className="flex flex-col gap-2">
                <div className="flex gap-2">
                    <div>
                        <label>
                            <span className="sr-only">Label</span>
                            <input
                                type="text"
                                id={`${datum_name}[label]`}
                                name={`${datum_name}[label]`}
                                placeholder="Label"
                                className="input-basic"
                            />
                        </label>
                    </div>
                    <div className={classNames({ hidden: !canRemove })}>
                        <button
                            type="button"
                            className="btn-cmd"
                            onMouseDown={remove}
                            title="Remove"
                            disabled={!canRemove}
                        >
                            <Icon.Minus />
                        </button>
                    </div>
                </div>
                <div>
                    <label>
                        <span className="sr-only">Timestamp</span>
                        <input
                            type="datetime-local"
                            id={`${datum_name}[timestamp]`}
                            name={`${datum_name}[timestamp]`}
                            defaultValue={now}
                            max={now}
                            placeholder="Timestamp"
                            className="input-basic"
                        />
                    </label>
                </div>
                <div>
                    <label>
                        <span className="sr-only">Data type</span>
                        <select
                            id={`${datum_name}[type]`}
                            name={`${datum_name}[type]`}
                            className="input-basic"
                            onChange={set_data_type}
                            defaultValue=""
                            required
                        >
                            <option value="" hidden disabled>
                                Data type
                            </option>
                            {dataTypes.map((type) => (
                                <option key={type.Id} value={type.Id}>
                                    {type.Label}
                                </option>
                            ))}
                        </select>
                    </label>
                </div>
                {dataType === undefined ? null : (
                    <>
                        <DatumStorage
                            datum={datum}
                            dataType={dataType}
                            cacheDataTypeSchema={cacheDataTypeSchema}
                        />
                        <DatumProperties datum={datum} />
                        <DatumNotes datum={datum} />
                    </>
                )}
            </div>
        </li>
    );
}

interface DatumStorageProps {
    datum: Datum;
    dataType: DataType;
    cacheDataTypeSchema: (
        data_type: UUIDTypes,
        schema: DataSchemaResources,
    ) => void;
}
function DatumStorage({
    datum,
    dataType,
    cacheDataTypeSchema,
}: DatumStorageProps) {
    switch (dataType.Storage) {
        case DataStorageInternal:
            return (
                <ErrorBoundary FallbackComponent={DatumStorageError}>
                    <Suspense>
                        <DatumStorageInternal
                            datum={datum}
                            dataType={dataType}
                            cacheDataTypeSchema={cacheDataTypeSchema}
                        />
                    </Suspense>
                </ErrorBoundary>
            );
        case DataStorageExternal:
            return <DatumStorageExternal datum={datum} dataType={dataType} />;
    }
}

function DatumStorageError({ error }: FallbackProps) {
    console.error(error);
    return <div>Could not load data type resources</div>;
}

interface DatumStorageInternalProps {
    datum: Datum;
    dataType: DataTypeInternal;
    cacheDataTypeSchema: (
        data_type: UUIDTypes,
        schema: DataSchemaResources,
    ) => void;
}
function DatumStorageInternal({
    datum,
    dataType,
    cacheDataTypeSchema,
}: DatumStorageInternalProps) {
    const { data: schema } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_SCHEMA_RESOURCES, dataType.Schema],
        queryFn: async () =>
            await dataService.dataSchemaResourcesGet(dataType.Schema),
    });
    useEffect(() => {
        cacheDataTypeSchema(dataType.Id, schema);
    }, []);

    switch (schema.DataSchema.Cardinality) {
        case DataSchemaCardinalitySingle:
            return;
            <DatumStorageInternalSingle schema={schema.DataSchema.Fields} />;
            break;
        case DataSchemaCardinalityMultiple:
            return (
                <DatumStorageInternalMultipleManual
                    datum={datum}
                    schema={schema.DataSchema.Fields}
                />
            );
            break;
        default:
            console.error("invalid data schema cardinality", schema);
    }
}

interface DatumStorageInternalSingleProps {
    schema: DataSchemaField[];
}
function DatumStorageInternalSingle({
    schema,
}: DatumStorageInternalSingleProps) {
    return <div>single int</div>;
}

interface DataField {
    id: number;
    values: any[];
}

interface DatumStorageInternalMultipleProps {
    datum: Datum;
    schema: DataSchemaField[];
}
function DatumStorageInternalMultipleManual({
    datum,
    schema,
}: DatumStorageInternalMultipleProps) {
    const [fields, setFields] = useState<DataField[]>([]);

    function insert_row(index: number = -1) {
        const id = Math.max(-1, ...fields.map((rx) => rx.id)) + 1;
        const values = new Array(schema.length);
        for (let idx = 0; idx < values.length; idx++) {
            values[idx] = undefined;
        }
        const rx = {
            id,
            values,
        };

        if (index === -1) {
            index = fields.length;
        }
        setFields(fields.toSpliced(index, 0, rx));
    }

    function maybe_add_row(e: ChangeEvent<HTMLInputElement>, id: number) {
        const last = fields.at(-1);
        if (!last) {
            throw new Error("invalid fields");
        }
        if (id !== last.id) {
            return;
        }

        insert_row();
    }

    function remove_row(e: MouseEvent<HTMLButtonElement>, idx: number) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        setFields(fields.toSpliced(idx, 1));
    }

    function insert_row_mouse(e: MouseEvent<HTMLButtonElement>, idx: number) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        insert_row(idx);
    }

    useEffect(insert_row, []);
    return (
        <div>
            <table>
                <thead>
                    <tr>
                        {schema.map((field) => {
                            return (
                                <th key={field.Label} className="">
                                    {field.Label}
                                </th>
                            );
                        })}
                        <th></th>
                    </tr>
                    <tr>
                        {schema.map((field) => {
                            return (
                                <th key={field.Label} className="">
                                    ({field.DType})
                                </th>
                            );
                        })}
                        <th></th>
                    </tr>
                </thead>
                <tbody>
                    {fields.map((rx, rx_idx) => (
                        <tr key={rx.id} className="group">
                            {rx.values.map((value, c_idx) => {
                                const field = schema[c_idx]!;
                                return (
                                    <td key={field.Label}>
                                        <InputFieldManual
                                            datum={datum}
                                            idx={rx_idx}
                                            field={field}
                                            onChange={(e) =>
                                                maybe_add_row(e, rx.id)
                                            }
                                        />
                                    </td>
                                );
                            })}
                            <td>
                                <div className="flex gap-2 invisible group-hover:visible">
                                    <div>
                                        <button
                                            type="button"
                                            className="btn-cmd"
                                            onMouseDown={(e) =>
                                                remove_row(e, rx_idx)
                                            }
                                            disabled={fields.length === 1}
                                        >
                                            <Icon.Minus />
                                        </button>
                                    </div>
                                    <div>
                                        <button
                                            type="button"
                                            className="btn-cmd"
                                            onMouseDown={(e) =>
                                                insert_row_mouse(e, rx_idx + 1)
                                            }
                                        >
                                            <Icon.Plus />
                                        </button>
                                    </div>
                                </div>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

interface InputFieldManualProps {
    datum: Datum;
    idx: number;
    field: DataSchemaField;
    onChange: (e: ChangeEvent<HTMLInputElement>) => void;
}
function InputFieldManual({
    datum,
    idx,
    field,
    onChange,
}: InputFieldManualProps) {
    return (
        <input
            type="text"
            id={`data[${datum.id}][value][${idx}][${field.Label}]`}
            name={`data[${datum.id}][value][${idx}][${field.Label}]`}
            onChange={(e) => onChange(e)}
        />
    );
}

interface DataStorageExternalProps {
    datum: Datum;
    dataType: DataTypeExternal;
}
function DatumStorageExternal({ datum, dataType }: DataStorageExternalProps) {
    return (
        <div>
            <fieldset>
                <legend>Sources</legend>
                <div className="flex flex-col gap-2">
                    {dataType.Sources.map((source) => (
                        <DataSource
                            key={source.Id.toString()}
                            datum={datum}
                            source={source}
                        />
                    ))}
                </div>
            </fieldset>
        </div>
    );
}

interface DataSourceProps {
    datum: Datum;
    source: DataTypeExternalSourceRx;
}
function DataSource({ datum, source }: DataSourceProps) {
    let cardinality_icon;
    switch (source.Cardinality) {
        case DataSchemaCardinalitySingle:
            cardinality_icon = <Icon.File />;
            break;
        case DataSchemaCardinalityMultiple:
            cardinality_icon = <Icon.Files />;
            break;
    }

    return (
        <div>
            <label
                className="flex gap-2 items-center"
                title={source.Description}
            >
                <span>{source.Label}</span>
                <input
                    type="file"
                    id={`data[${datum.id}][source][${source.Id}]`}
                    name={`data[${datum.id}][source][${source.Id}]`}
                    className="input-basic"
                    multiple={
                        source.Cardinality === DataSourceCardinalityMultiple
                    }
                    accept={source.MediaTypes?.join(", ")}
                    required={source.Required}
                />
                <span>{cardinality_icon}</span>
            </label>
        </div>
    );
}

interface PropertyRx {
    id: number;
    key: string;
    type: PropertyType;
}
interface DatumPropertiesProps {
    datum: Datum;
}
function DatumProperties({ datum }: DatumPropertiesProps) {
    const [properties, setProperties] = useState<PropertyRx[]>([
        { id: 0, key: "", type: PropertyTypeString },
    ]);
    function add_property(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        let id = 0;
        if (properties.length > 0) {
            id = properties.at(-1)!.id + 1;
        }
        setProperties([
            ...properties,
            { id, key: "", type: PropertyTypeString },
        ]);
    }

    function remove(e: MouseEvent<HTMLButtonElement>, id: number) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        setProperties(properties.filter((prop) => prop.id !== id));
    }

    function validate_key(
        e: ChangeEvent<HTMLInputElement>,
        property_id: number,
    ) {
        const input = e.target;
        input.setCustomValidity("");

        const value = input.value.trim();
        const rx = properties.find((rx) => rx.id === property_id)!;
        let old_dup = [];
        for (const prop of properties) {
            if (prop.id === property_id) {
                continue;
            }
            if (value !== "" && prop.key === value) {
                input.setCustomValidity("Key already exists");
            } else if (prop.key === rx.key) {
                old_dup.push(prop.id);
            }
        }

        if (old_dup.length === 1) {
            const old_dup_input = document.getElementById(
                `data[${datum.id}][property][${old_dup[0]}][key]`,
            )! as HTMLInputElement;
            old_dup_input.setCustomValidity("");
        }

        const update = properties.map((prop) => {
            if (prop.id === property_id) {
                prop.key = value;
            }
            return prop;
        });
        setProperties(update);
    }

    function set_property_type(e: ChangeEvent<HTMLSelectElement>, id: number) {
        const property = properties.find((prop) => prop.id === id);
        if (property === undefined) {
            throw new Error("invalid property");
        }

        const type = type_string_to_variant(e.target.value);
        if (type === undefined) {
            throw new Error("invalid property");
        }

        property.type = type;
        setProperties([...properties]);
    }

    return (
        <details>
            <summary>
                <div className="inline-flex gap-2">
                    <h3>Properties</h3>
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            onMouseDown={add_property}
                        >
                            <Icon.Plus />
                        </button>
                    </div>
                </div>
            </summary>

            <ul className="grid grid-cols-[repeat(4,min-content)] gap-2">
                {properties.map((property) => {
                    return (
                        <li
                            key={property.id}
                            className="col-span-full grid-cols-subgrid grid group"
                        >
                            <div className="contents">
                                <div className="col-1">
                                    <label>
                                        <span className="sr-only">Key</span>
                                        <input
                                            type="text"
                                            id={`data[${datum.id}][property][${property.id}][key]`}
                                            name={`data[${datum.id}][property][${property.id}][key]`}
                                            placeholder="Label"
                                            className="input-basic"
                                            onChange={(e) =>
                                                validate_key(e, property.id)
                                            }
                                        />
                                    </label>
                                </div>
                                <div className="col-2">
                                    <SelectPropertyType
                                        id={`data[${datum.id}][property][${property.id}][type]`}
                                        name={`data[${datum.id}][property][${property.id}][type]`}
                                        className="input-basic"
                                        onChange={(e) =>
                                            set_property_type(e, property.id)
                                        }
                                    />
                                </div>
                                <div className="col-3 flex gap-2">
                                    <InputPropertyValue
                                        type={property.type}
                                        id={`data[${datum.id}][property][${property.id}][value]`}
                                        name={`data[${datum.id}][property][${property.id}][value]`}
                                        className="input-basic"
                                        placeholder="Value"
                                    />
                                </div>
                                <div className="col-4">
                                    <div
                                        className={classNames({
                                            invisible: true,
                                            "group-hover:visible":
                                                properties.length > 1,
                                        })}
                                    >
                                        <button
                                            type="button"
                                            className="btn-cmd"
                                            title="Remove property"
                                            disabled={properties.length < 2}
                                            onMouseDown={(e) =>
                                                remove(e, property.id)
                                            }
                                        >
                                            <Icon.Minus />
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </li>
                    );
                })}
            </ul>
        </details>
    );
}

interface NoteRx {
    id: number;
}
interface DatumNotes {
    datum: Datum;
}
function DatumNotes({ datum }: DatumNotes) {
    const [notes, setNotes] = useState<NoteRx[]>([{ id: 0 }]);
    function add_note(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        let id = 0;
        if (notes.length > 0) {
            id = notes.at(-1)!.id + 1;
        }
        setNotes([...notes, { id }]);
    }

    function remove(e: MouseEvent<HTMLButtonElement>, id: number) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        setNotes(notes.filter((prop) => prop.id !== id));
    }

    const now_str = datetime_for_input(new Date());
    return (
        <details>
            <summary>
                <div className="inline-flex gap-2">
                    <h3>Notes</h3>
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            onMouseDown={add_note}
                        >
                            <Icon.Plus />
                        </button>
                    </div>
                </div>
            </summary>

            <ul className="list-decimal px-4">
                {notes.map((note) => {
                    return (
                        <li key={note.id} className="group">
                            <div>
                                <div className="flex gap-2">
                                    <div>
                                        <label>
                                            <span className="sr-only">
                                                Timestamp
                                            </span>
                                            <input
                                                type="datetime-local"
                                                id={`data[${datum.id}][note][${note.id}][timestamp]`}
                                                name={`data[${datum.id}][note][${note.id}][timestamp]`}
                                                defaultValue={now_str}
                                                max={now_str}
                                                required
                                            />
                                        </label>
                                    </div>
                                    <div
                                        className={classNames({
                                            invisible: true,
                                            "group-hover:visible":
                                                notes.length > 1,
                                        })}
                                    >
                                        <button
                                            type="button"
                                            className="btn-cmd"
                                            disabled={notes.length < 2}
                                            onMouseDown={(e) =>
                                                remove(e, note.id)
                                            }
                                        >
                                            <Icon.Minus />
                                        </button>
                                    </div>
                                </div>
                                <div>
                                    <label>
                                        <span className="sr-only">Content</span>
                                        <textarea
                                            id={`data[${datum.id}][note][${note.id}][content]`}
                                            name={`data[${datum.id}][note][${note.id}][content]`}
                                            placeholder="Note"
                                            className="input-basic"
                                        ></textarea>
                                    </label>
                                </div>
                            </div>
                        </li>
                    );
                })}
            </ul>
        </details>
    );
}
