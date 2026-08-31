import {
    MouseButton,
    QUERY_KEY_DATA_SCHEMA_RESOURCES,
    QUERY_KEY_DATA_TYPES,
    QUERY_KEY_PROJECT_RESOURCES,
    stringToDataValue,
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
    stringToPropertyValue,
    type_string_to_variant,
} from "@/components/Property";
import Icon from "@/icon";
import {
    default as dataService,
    type DataIngest as DataIngestServer,
} from "@/service/data.service";
import projectService from "@/service/project.service";
import {
    DataSchemaCardinalityMultiple,
    DataSchemaCardinalitySingle,
    DataSourceCardinalityMultiple,
    DataSourceCardinalitySingle,
    DataStorageExternal,
    DataStorageInternal,
    PropertyTypeBool,
    PropertyTypeString,
    ValueTypeBoolean,
    ValueTypeInt,
    ValueTypeTimestamp,
    ValueTypeUint,
    VisibilityPrivate,
    type DataNoteCreate,
    type DataSchema,
    type DataSchemaField,
    type DataSchemaResources,
    type DataSourceCardinality,
    type DataType,
    type DataTypeExternal,
    type DataTypeExternalSourceRx,
    type DataTypeInternal,
    type Property,
    type PropertyType,
    type Visibility,
} from "@/types";
import { useForm, useSelector } from "@tanstack/react-form";
import { useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";
import {
    Suspense,
    useCallback,
    useEffect,
    useRef,
    useState,
    type ChangeEvent,
    type ComponentPropsWithoutRef,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import {
    redirect,
    useNavigate,
    useParams,
    type NavigateFunction,
} from "react-router";
import { type UUIDTypes } from "uuid";

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
    const { data: project_resources } = useSuspenseQuery({
        queryKey: [QUERY_KEY_PROJECT_RESOURCES, projectId],
        queryFn: async () =>
            await projectService.getProjectResources(projectId),
    });
    const navigate = useNavigate();

    function cancel(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    const project_labels = project_resources.Data.map(
        (data) => data.Label,
    ).filter((label) => label !== undefined && label !== null);
    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <h2 className="title">Create project data</h2>
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
            <ProjectData
                project={projectId}
                dataTypes={data_types}
                projectLabels={project_labels}
            />
        </div>
    );
}

interface PropertyCreate {
    key: string;
    type: PropertyType;
    value: string;
}

interface DataSourceCreate {
    id: UUIDTypes;
    cardinality: DataSourceCardinality;
    source: null | File | FileList;
}

interface DataIngest {
    type?: UUIDTypes;
    timestamp: Date;
    visibility: Visibility;
    properties: PropertyCreate[];
    notes: DataNoteCreate[];
    valuesSingle: any[];
    valuesMultiple: any[][];
    sources: DataSourceCreate[];
    projectLabel?: string;
}

function prepareData(
    values: DataIngest[],
    data_types: DataType[],
    data_schemas: Map<string, DataSchema>,
): {
    ingest: DataIngestServer[];
    labels: string[];
    files: Map<string, File | File[]>;
} {
    const out = {
        ingest: new Array<DataIngestServer>(values.length),
        labels: new Array<string>(values.length),
        files: new Map(),
    };

    for (const idx in values) {
        const data = values[idx]!;
        if (!data.type) {
            continue;
        }

        out.labels[idx] = data.projectLabel ?? "";
        const properties = prepareDataProperties(data.properties);
        for (const note of data.notes) {
            note.Content = note.Content.trim();
        }
        data.notes = data.notes.filter((note) => note.Content.length > 0);

        let dvalues = undefined;
        let sources = undefined;
        const dtype = data_types.find((dtype) => {
            return dtype.Id === data.type;
        })!;
        const schema = data_schemas.get(dtype.Id)!;
        switch (dtype.Storage) {
            case DataStorageInternal:
                switch (schema.Cardinality) {
                    case DataSchemaCardinalitySingle:
                        dvalues = prepareDataInternalSingle(
                            data.valuesSingle,
                            schema.Fields,
                        );
                        dvalues = Object.fromEntries(dvalues);
                        break;
                    case DataSchemaCardinalityMultiple:
                        dvalues = prepareDataInternalMultiple(
                            data.valuesMultiple,
                            schema.Fields,
                        );
                        dvalues = Object.fromEntries(dvalues);
                        break;
                    default:
                        throw new Error(
                            `invalid cardinality: data type ${data.type}, ${dtype.Cardinality}`,
                        );
                }
                break;
            case DataStorageExternal:
                const { sources: dsources, files } = prepareDataExternal(
                    idx,
                    data.sources,
                );
                sources = Object.fromEntries(dsources);
                files.forEach((value, key) => out.files.set(key, value));
                break;
            default:
                throw new Error(`invalid data type: ${data.type}`);
        }

        out.ingest[idx] = {
            Type: data.type,
            Timestamp: data.timestamp,
            Visibility: data.visibility,
            Properties: properties,
            Notes: data.notes,
            Values: dvalues,
            Sources: sources,
        };
    }

    return out;
}

function prepareDataExternal(
    idx: string,
    sources: DataSourceCreate[],
): {
    sources: Map<string, string>;
    files: Map<string, null | File | FileList>;
} {
    const out = {
        sources: new Map(),
        files: new Map(),
    };
    for (const source of sources) {
        const key = `data[${idx}].source[${source.id}]`;
        out.sources.set(uuidToString(source.id), key);
        out.files.set(key, source.source);
        // switch (source.cardinality) {
        //     case DataSourceCardinalitySingle:
        //         break;
        //     case DataSourceCardinalityMultiple:
        //         const sourceList = source.source as FileList;
        //         const smap = new Array(sourceList.length);
        //         for (let sdx = 0; sdx < sourceList.length; sdx++) {
        //             const skey = key + `[${sdx}]`;
        //             smap[sdx] = skey;
        //             out.files.set(skey, sourceList.item(sdx));
        //         }
        //         out.sources.set(uuidToString(source.id), smap);
        //         break;
        //     default:
        //         throw new Error(
        //             `invalid data source cardinality ${source.cardinality}`,
        //         );
        // }
    }

    return out;
}

// @throws Property key or value is empty.
function prepareDataProperties(properties: PropertyCreate[]): Property[] {
    return properties
        .filter((property) => property.key !== "" && property.value !== "")
        .map((property) => {
            if (!property.key) {
                throw new Error("Property does not have key");
            }
            if (!property.value) {
                throw new Error("Property does not have value");
            }

            let value;
            try {
                value = stringToPropertyValue(property.value, property.type);
            } catch (err) {
                throw new Error(`property ${property.key}: ${err}`);
            }

            return {
                Key: property.key,
                Type: property.type,
                Value: value,
            };
        });
}

// @throws `data` and `schema` do not have same length.
function prepareDataInternalSingle(
    data: any[],
    schema: DataSchemaField[],
): Map<string, any> {
    if (data.length !== schema.length) {
        throw new Error(`invalid values: data ${data}, schema: ${schema}`);
    }

    const values = new Map();
    let value;
    for (let idx = 0; idx < data.length; idx++) {
        const field = schema[idx]!;
        value = stringToDataValue(data[idx]!, field.DType);
        values.set(field.Label, value);
    }

    return values;
}

// # Notes
// `data` should be in row major form.
//
// @throws If a record and `schema` do not have same length.
function prepareDataInternalMultiple(
    data: any[],
    schema: DataSchemaField[],
): Map<string, any[]> {
    for (let rx of data) {
        if (rx.length !== schema.length) {
            throw new Error(
                `invalid values:\nschema:\n${schema}\ndata\n${data}`,
            );
        }
    }

    const keys = new Array(schema.length);
    const values = new Map<string, any[]>();
    for (let idx in schema) {
        let key = schema[idx]!.Label;
        keys[idx] = key;
        values.set(key, new Array(data.length));
    }
    let rdx = 0;
    for (let idx in data) {
        const rx = data[idx];
        let empty = true;
        for (let kdx = 0; kdx < keys.length; kdx++) {
            if (rx[kdx]) {
                empty = false;
                break;
            }
        }
        if (empty) {
            continue;
        }

        for (let kdx = 0; kdx < keys.length; kdx++) {
            values.get(keys[kdx])![rdx] = rx[kdx];
        }
        rdx++;
    }

    for (const field of schema) {
        const vals = values.get(field.Label)!.slice(0, rdx);
        if (!field.Nullable) {
            for (const val of vals) {
                if (!val) {
                    throw new Error(
                        `all values must be present in ${field.Label}\n${vals}`,
                    );
                }
            }
        }

        values.set(
            field.Label,
            vals.map((val) => stringToDataValue(val, field.DType)),
        );
    }

    return values;
}

type SubmissionMeta = {
    dataSchemas: Map<string, DataSchema>;
};

function useFormProjectData(
    navigate: NavigateFunction,
    project: UUIDTypes,
    data_types: DataType[],
) {
    const defaultMeta: SubmissionMeta = {
        dataSchemas: new Map(),
    };
    return useForm({
        defaultValues: {
            data: [
                {
                    type: undefined,
                    timestamp: new Date(),
                    visibility: VisibilityPrivate,
                    properties: [],
                    notes: [],
                    valuesSingle: new Array(),
                    valuesMultiple: new Array(),
                    sources: new Array(),
                    projectLabel: undefined,
                } as DataIngest,
            ],
        },
        onSubmitMeta: defaultMeta,
        onSubmit: async ({ value, meta }) => {
            const {
                ingest,
                labels,
                files: fileMap,
            } = prepareData(value.data, data_types, meta.dataSchemas);

            const files = Array.from(fileMap.entries());
            await dataService
                .projectDataCreate(project, ingest, labels, files)
                .then((resp) => {
                    if (resp.ok) {
                        navigate(-1);
                    }
                });
        },
    });
}
type ProjectDataFormApi = ReturnType<typeof useFormProjectData>;

interface ProjectDataProps {
    project: UUIDTypes;
    dataTypes: DataType[];
    projectLabels: string[];
}
function ProjectData({ project, dataTypes, projectLabels }: ProjectDataProps) {
    const navigate = useNavigate();
    const [dataTypeSchemas, setDataTypeSchemas] = useState<
        Map<string, DataSchemaResources>
    >(new Map());
    const form = useFormProjectData(navigate, project, dataTypes);

    function create_new_datum(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.pushFieldValue("data", {
            type: undefined,
            timestamp: new Date(),
            visibility: VisibilityPrivate,
            properties: [],
            notes: [],
            valuesSingle: new Array(),
            valuesMultiple: new Array(),
            sources: new Array(),
            projectLabel: undefined,
        } as DataIngest);
    }

    function cache_data_type_schema(
        data_type: UUIDTypes,
        schema: DataSchemaResources,
    ) {
        setDataTypeSchemas(
            new Map(dataTypeSchemas.set(uuidToString(data_type), schema)),
        );
    }

    const dataSchemas = new Map();
    for (let [key, resource] of dataTypeSchemas) {
        dataSchemas.set(key, resource.DataSchema);
    }
    async function create_project_data(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        form.handleSubmit({
            dataSchemas: dataSchemas,
        });
    }

    return (
        <form
            className="pt-2 flex flex-col gap-2"
            onSubmit={create_project_data}
        >
            <div className="flex flex-col gap-2">
                <ol className="list-decimal space-y-8">
                    <form.Field name="data" mode="array">
                        {(data) => {
                            return data.state.value.map((_datum, idx) => (
                                <ProjectDataItem
                                    key={idx}
                                    form={form}
                                    idx={idx}
                                    dataTypes={dataTypes}
                                    projectLabels={projectLabels}
                                    cacheDataTypeSchema={cache_data_type_schema}
                                />
                            ));
                        }}
                    </form.Field>
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
                <button
                    type="submit"
                    className="btn-submit"
                    disabled={!form.state.canSubmit}
                >
                    Create data
                </button>
            </div>
        </form>
    );
}

interface ProjectDataItemProps {
    form: ProjectDataFormApi;
    idx: number;
    dataTypes: DataType[];
    projectLabels: string[];
    cacheDataTypeSchema: (
        data_type: UUIDTypes,
        schema: DataSchemaResources,
    ) => void;
}
function ProjectDataItem({
    form,
    idx,
    dataTypes,
    projectLabels,
    cacheDataTypeSchema,
}: ProjectDataItemProps) {
    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.removeFieldValue("data", idx);
    }

    function set_data_type(e: ChangeEvent<HTMLSelectElement>) {
        const data_type = dataTypes.find((type) => type.Id === e.target.value);
        if (!data_type) {
            console.error("invalid data type");
            return;
        }

        form.setFieldValue(`data[${idx}].type`, data_type.Id);
    }

    const datum_type = useSelector(form.store, (state) => {
        return dataTypes.find(
            (dtype) => dtype.Id === state.values.data[idx]?.type,
        );
    });
    const canRemove = form.state.values.data.length > 1;
    const now = timestampToString(new Date());
    return (
        <li className="px-4 pb-2 group">
            <div className="flex flex-col gap-2">
                <div className="flex gap-2">
                    <div>
                        <form.Field
                            name={`data[${idx}].projectLabel`}
                            validators={{
                                onChange: ({ value, fieldApi }) => {
                                    if (!value) {
                                        return undefined;
                                    }

                                    const labels =
                                        fieldApi.form.state.values.data
                                            .filter((_, ddx) => ddx !== idx)
                                            .map((data) => data.projectLabel)
                                            .filter(
                                                (label) =>
                                                    label !== undefined &&
                                                    label !== null &&
                                                    label.length > 0,
                                            );

                                    if (
                                        labels.includes(value) ||
                                        projectLabels.includes(value)
                                    ) {
                                        return "Label already exists";
                                    }
                                    return undefined;
                                },
                            }}
                        >
                            {(field) => {
                                return (
                                    <label>
                                        <span className="sr-only">Label</span>
                                        <input
                                            type="text"
                                            placeholder="Label"
                                            className="input-basic"
                                            value={field.state.value}
                                            onChange={(e) =>
                                                field.handleChange(
                                                    e.target.value.trim(),
                                                )
                                            }
                                        />
                                        {field.state.meta.isValid ? null : (
                                            <div className="text-sm">
                                                {field.state.meta.errors.join(
                                                    ", ",
                                                )}
                                            </div>
                                        )}
                                    </label>
                                );
                            }}
                        </form.Field>
                    </div>
                    <div
                        className={classNames({
                            hidden: !canRemove,
                            "invisible group-hover:visible": canRemove,
                        })}
                    >
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
                    <form.Field
                        name={`data[${idx}].timestamp`}
                        validators={{
                            onBlur: (state) => {
                                if (isNaN(+state.value)) {
                                    return "Timestamp is invalid";
                                }
                                if (state.value > new Date()) {
                                    return "Timestamp can not be in the future";
                                }
                                return undefined;
                            },
                        }}
                    >
                        {(field) => {
                            return (
                                <label>
                                    <span className="sr-only">Timestamp</span>
                                    <input
                                        type="datetime-local"
                                        max={now}
                                        value={timestampToString(
                                            field.state.value,
                                        )}
                                        placeholder="Timestamp"
                                        className="input-basic"
                                        onChange={(e) =>
                                            field.handleChange(
                                                new Date(e.target.value),
                                            )
                                        }
                                        onBlur={field.handleBlur}
                                    />
                                    {field.state.meta.isValid ? null : (
                                        <div className="text-sm">
                                            {field.state.meta.errors.join(", ")}
                                        </div>
                                    )}
                                </label>
                            );
                        }}
                    </form.Field>
                </div>
                <div>
                    <form.Field name={`data[${idx}].type`}>
                        {(field) => {
                            const type_id = field.state.value;
                            return (
                                <label>
                                    <span className="sr-only">Data type</span>
                                    <select
                                        className="input-basic"
                                        onChange={set_data_type}
                                        value={
                                            type_id ? uuidToString(type_id) : ""
                                        }
                                        required
                                    >
                                        <option value="" hidden disabled>
                                            Data type
                                        </option>
                                        {dataTypes.map((type) => (
                                            <option
                                                key={type.Id}
                                                value={type.Id}
                                            >
                                                {type.Label}
                                            </option>
                                        ))}
                                    </select>
                                </label>
                            );
                        }}
                    </form.Field>
                </div>
                {datum_type === undefined ? null : (
                    <>
                        <DatumStorage
                            form={form}
                            idx={idx}
                            dataType={datum_type}
                            cacheDataTypeSchema={cacheDataTypeSchema}
                        />
                        <DatumProperties form={form} idx={idx} />
                        <DatumNotes form={form} idx={idx} />
                    </>
                )}
            </div>
        </li>
    );
}

interface DatumStorageProps {
    form: ProjectDataFormApi;
    idx: number;
    dataType: DataType;
    cacheDataTypeSchema: (
        data_type: UUIDTypes,
        schema: DataSchemaResources,
    ) => void;
}
function DatumStorage({
    form,
    idx,
    dataType,
    cacheDataTypeSchema,
}: DatumStorageProps) {
    switch (dataType.Storage) {
        case DataStorageInternal:
            return (
                <ErrorBoundary FallbackComponent={DatumStorageError}>
                    <Suspense>
                        <DatumStorageInternal
                            form={form}
                            idx={idx}
                            dataType={dataType}
                            cacheDataTypeSchema={cacheDataTypeSchema}
                        />
                    </Suspense>
                </ErrorBoundary>
            );
        case DataStorageExternal:
            return (
                <DatumStorageExternal
                    idx={idx}
                    dataType={dataType}
                    form={form}
                />
            );
        default:
            console.error(dataType);
            throw new Error("invalid data type");
    }
}

function DatumStorageError({ error }: FallbackProps) {
    console.error(error);
    return <div>Could not load data type resources</div>;
}

interface DatumStorageInternalProps {
    form: ProjectDataFormApi;
    idx: number;
    dataType: DataTypeInternal;
    cacheDataTypeSchema: (
        data_type: UUIDTypes,
        schema: DataSchemaResources,
    ) => void;
}
function DatumStorageInternal({
    form,
    idx,
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
            return (
                <DatumStorageInternalSingle
                    idx={idx}
                    form={form}
                    schema={schema.DataSchema.Fields}
                />
            );
        case DataSchemaCardinalityMultiple:
            return (
                <DatumStorageInternalMultipleManual
                    form={form}
                    idx={idx}
                    schema={schema.DataSchema.Fields}
                />
            );
        default:
            console.error("invalid data schema cardinality", schema);
    }
}

interface DatumStorageInternalSingleProps {
    idx: number;
    form: ProjectDataFormApi;
    schema: DataSchemaField[];
}
function DatumStorageInternalSingle({
    idx,
    form,
    schema,
}: DatumStorageInternalSingleProps) {
    useEffect(() => {
        form.setFieldValue(
            `data[${idx}].valuesSingle`,
            new Array(schema.length).fill(""),
        );
    }, []);

    return (
        <ol>
            <form.Field name={`data[${idx}].valuesSingle`} mode="array">
                {(values) => {
                    return values.state.value.map((_val, sdx) => {
                        const field = schema[sdx]!;
                        return (
                            <li key={sdx} className="pb-2">
                                <label className="flex gap-2">
                                    <div>{field.Label}</div>
                                    <form.Field
                                        name={`data[${idx}].valuesSingle[${sdx}]`}
                                        validators={{
                                            onBlur: ({ value }) => {
                                                const field = schema[sdx]!;
                                                if (
                                                    !field.Nullable &&
                                                    value === ""
                                                ) {
                                                    return "Value is required";
                                                }
                                                try {
                                                    stringToDataValue(
                                                        value,
                                                        field.DType,
                                                    );
                                                } catch (error) {
                                                    return "Invalid value";
                                                }

                                                return undefined;
                                            },
                                        }}
                                    >
                                        {(ffield) => (
                                            <InputFieldManual
                                                field={field}
                                                value={ffield.state.value}
                                                title={
                                                    ffield.state.meta.isValid
                                                        ? undefined
                                                        : ffield.state.meta.errors.join(
                                                              ", ",
                                                          )
                                                }
                                                className={`input-basic \
                                                ${
                                                    ffield.state.meta.isValid
                                                        ? ""
                                                        : "border border-syre-red-500 bg-syre-red-50 dark:bg-syre-red-900"
                                                }`}
                                                onChange={(e) => {
                                                    ffield.handleChange(
                                                        e.target.value,
                                                    );
                                                }}
                                                onBlur={ffield.handleBlur}
                                            />
                                        )}
                                    </form.Field>
                                </label>
                            </li>
                        );
                    });
                }}
            </form.Field>
        </ol>
    );
}

enum InternalMultipleViewMode {
    Table = "table",
    Text = "text",
}

interface DatumStorageInternalMultipleProps {
    form: ProjectDataFormApi;
    idx: number;
    schema: DataSchemaField[];
}
function DatumStorageInternalMultipleManual({
    form,
    idx,
    schema,
}: DatumStorageInternalMultipleProps) {
    const [mode, setMode] = useState<InternalMultipleViewMode>(
        InternalMultipleViewMode.Table,
    );

    useEffect(() => {
        form.pushFieldValue(
            `data[${idx}].valuesMultiple`,
            new Array(schema.length).fill(""),
        );
    }, []);

    let view;
    switch (mode) {
        case InternalMultipleViewMode.Table:
            view = (
                <DatumStorageInternalMultipleManualTable
                    form={form}
                    idx={idx}
                    schema={schema}
                />
            );
            break;
        case InternalMultipleViewMode.Text:
            view = (
                <DatumStorageInternalMultipleManualTextarea
                    form={form}
                    idx={idx}
                />
            );
            break;
        default:
            throw new Error("invalid view mode");
    }

    return (
        <div>
            <div className="py-1">
                <InternalMultipleViewModeToggle
                    value={mode}
                    onChange={setMode}
                />
            </div>
            {view}
        </div>
    );
}

interface InternalMultipleViewModeToggleProps {
    value: InternalMultipleViewMode;
    onChange: (mode: InternalMultipleViewMode) => void;
}
function InternalMultipleViewModeToggle({
    value,
    onChange,
}: InternalMultipleViewModeToggleProps) {
    const iconClass =
        "text-secondary \
        peer-checked:bg-primary-700 dark:peer-checked:bg-primary-500 \
        peer-checked:text-white dark:peer-checked:text-black \
        transition-colors duration-200 ease-in-out\
        rounded-full block \
        h-full aspect-square p-1 cursor-pointer";

    return (
        <fieldset className="flex">
            <label
                className="rounded-l-full border-l border-t border-b pr-0.5 cursor-pointer"
                title="Table"
            >
                <input
                    type="radio"
                    checked={value === InternalMultipleViewMode.Table}
                    className="hidden peer"
                    onChange={(_) => onChange(InternalMultipleViewMode.Table)}
                />
                <span className="sr-only">"Table"</span>
                <span className={iconClass}>
                    <Icon.Table />
                </span>
            </label>
            <label
                className="rounded-r-full border-r border-t border-b pl-0.5 cursor-pointer"
                title="Text"
            >
                <input
                    type="radio"
                    name="visibility"
                    checked={value === InternalMultipleViewMode.Text}
                    className="hidden peer"
                    onChange={(_) => onChange(InternalMultipleViewMode.Text)}
                />
                <span className="sr-only">Text</span>
                <span className={iconClass}>
                    <Icon.Text />
                </span>
            </label>
        </fieldset>
    );
}

const classTableBorderColors = "border-gray-700 dark:border-gray-300";
const classTablePadding = "px-1 py-0.5";

interface DatumStorageInternalMultipleTableProps {
    form: ProjectDataFormApi;
    idx: number;
    schema: DataSchemaField[];
}
function DatumStorageInternalMultipleManualTable({
    form,
    idx,
    schema,
}: DatumStorageInternalMultipleTableProps) {
    return (
        <div>
            <table>
                <thead>
                    <tr>
                        {schema.map((field) => {
                            return (
                                <th
                                    key={field.Label}
                                    className={`border border-b-0 ${classTableBorderColors} ${classTablePadding}`}
                                >
                                    {field.Label}
                                </th>
                            );
                        })}
                        <th></th>
                    </tr>
                    <tr>
                        {schema.map((field) => {
                            return (
                                <th
                                    key={field.Label}
                                    className={`border border-t-0 ${classTableBorderColors} ${classTablePadding}`}
                                >
                                    ({field.DType})
                                </th>
                            );
                        })}
                        <th></th>
                    </tr>
                </thead>
                <tbody>
                    <form.Field
                        name={`data[${idx}].valuesMultiple`}
                        mode="array"
                        validators={{
                            onSubmit: ({ value }) => {
                                let empty = true;
                                for (let rx of value) {
                                    for (let val of rx) {
                                        if (val !== "") {
                                            empty = false;
                                            break;
                                        }
                                        if (!empty) {
                                            break;
                                        }
                                    }
                                }
                                if (empty) {
                                    return "Values can not be empty";
                                }
                                return undefined;
                            },
                        }}
                    >
                        {(values) =>
                            values.state.value.map((_row, rdx) => (
                                <DatumStorageInternalMultipleManualTableRow
                                    key={rdx}
                                    form={form}
                                    idx={idx}
                                    rdx={rdx}
                                    schema={schema}
                                />
                            ))
                        }
                    </form.Field>
                </tbody>
            </table>
            <form.Subscribe
                selector={(state) =>
                    state.fieldMeta[`data[${idx}].valuesMultiple`]
                }
            >
                {(meta) =>
                    meta?.isValid ? null : (
                        <div className="text-sm">{meta?.errors.join(", ")}</div>
                    )
                }
            </form.Subscribe>
        </div>
    );
}

interface DatumStorageInternalMultipleTableRowProps {
    form: ProjectDataFormApi;
    idx: number;
    rdx: number;
    schema: DataSchemaField[];
}
function DatumStorageInternalMultipleManualTableRow({
    form,
    idx,
    rdx,
    schema,
}: DatumStorageInternalMultipleTableRowProps) {
    function remove_row(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        if (form.state.values.data[idx]!.valuesMultiple.length > 1) {
            form.removeFieldValue(`data[${idx}].valuesMultiple`, rdx);
        }
    }

    function insert_row_mouse(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.insertFieldValue(
            `data[${idx}].valuesMultiple`,
            rdx + 1,
            new Array(schema.length).fill(""),
        );
    }

    function maybe_add_row() {
        const num_rx = form.state.values.data[idx]!.valuesMultiple.length;
        if (num_rx - 1 === rdx) {
            form.pushFieldValue(
                `data[${idx}].valuesMultiple`,
                new Array(schema.length).fill(""),
            );
        }
    }

    const num_rows = form.state.values.data[idx]!.valuesMultiple.length;
    return (
        <tr className="group/input-row">
            <form.Field
                name={`data[${idx}].valuesMultiple[${rdx}]`}
                mode="array"
            >
                {(values) =>
                    values.state.value.map((_val, sdx) => (
                        <td
                            key={sdx}
                            className={`border ${classTableBorderColors}`}
                        >
                            <form.Field
                                name={`data[${idx}].valuesMultiple[${rdx}][${sdx}]`}
                                validators={{
                                    onBlur: ({ value, fieldApi }) => {
                                        const field = schema[sdx]!;
                                        if (!field.Nullable && value === "") {
                                            const rx = fieldApi.form
                                                .getFieldValue(
                                                    `data[${idx}].valuesMultiple[${rdx}]`,
                                                )!
                                                .filter(
                                                    (val, tdx) =>
                                                        tdx !== sdx &&
                                                        val !== "",
                                                );
                                            if (rx.length === 0) {
                                                return undefined;
                                            } else {
                                                return "Value is required";
                                            }
                                        }
                                        try {
                                            stringToDataValue(
                                                value,
                                                field.DType,
                                            );
                                        } catch (error) {
                                            return "Invalid value";
                                        }

                                        return undefined;
                                    },
                                }}
                            >
                                {(ffield) => {
                                    const field = schema[sdx]!;
                                    return (
                                        <InputFieldManual
                                            key={sdx}
                                            field={field}
                                            value={ffield.state.value}
                                            title={
                                                ffield.state.meta.isValid
                                                    ? undefined
                                                    : ffield.state.meta.errors.join(
                                                          ", ",
                                                      )
                                            }
                                            className={`px-1 py-0.5 \
                                                ${
                                                    ffield.state.meta.isValid
                                                        ? ""
                                                        : "border border-syre-red-500 bg-syre-red-50 dark:bg-syre-red-900"
                                                }`}
                                            onChange={(e) => {
                                                maybe_add_row();
                                                ffield.handleChange(
                                                    e.target.value,
                                                );
                                            }}
                                            onBlur={ffield.handleBlur}
                                        />
                                    );
                                }}
                            </form.Field>
                        </td>
                    ))
                }
            </form.Field>
            <td>
                <div className="pl-2 flex gap-1 invisible group-hover/input-row:visible">
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            title="Remove this row"
                            onMouseDown={remove_row}
                            disabled={num_rows < 2}
                            tabIndex={-1}
                        >
                            <Icon.Minus />
                        </button>
                    </div>
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            title="Insert new row below"
                            onMouseDown={insert_row_mouse}
                            tabIndex={-1}
                        >
                            <Icon.Plus />
                        </button>
                    </div>
                </div>
            </td>
        </tr>
    );
}

type InputFieldManualProps = ComponentPropsWithoutRef<"input"> & {
    field: DataSchemaField;
};
function InputFieldManual({ field, ...props }: InputFieldManualProps) {
    let input_type = "text";
    switch (field.DType) {
        case ValueTypeUint:
            input_type = "number";
            props.min = 0;
            break;
        case ValueTypeInt:
            input_type = "number";
            break;
        case ValueTypeTimestamp:
            input_type = "datetime-locale";
            break;
        case ValueTypeBoolean:
            input_type = "checkbox";
            break;
    }

    return <input type={input_type} {...props} />;
}

interface DatumStorageInternalMultipleTextareaProps {
    form: ProjectDataFormApi;
    idx: number;
}
function DatumStorageInternalMultipleManualTextarea({
    idx,
    form,
}: DatumStorageInternalMultipleTextareaProps) {
    const [text, setText] = useState("");
    useEffect(() => {
        const values = form.state.values.data[idx]!.valuesMultiple;
        let out = "";
        for (let rx of values) {
            out += rx.join(", ");
            out += "\n";
        }
        setText(out);
    }, []);

    const parseValue = useDebouncedCallback((value: string) => {
        const out = new Array();
        const rows = value.split("\n");
        for (let rx of rows) {
            let vals = rx.split(",").map((v) => v.trim());
            out.push(vals);
        }

        form.setFieldValue(`data[${idx}].valuesMultiple`, out);
    }, 300);

    return (
        <textarea
            className="input-basic resize"
            value={text}
            onChange={(e) => {
                const value = e.target.value;
                setText(value);
                parseValue(value);
            }}
        ></textarea>
    );
}

interface DataStorageExternalProps {
    idx: number;
    form: ProjectDataFormApi;
    dataType: DataTypeExternal;
}
function DatumStorageExternal({
    idx,
    form,
    dataType,
}: DataStorageExternalProps) {
    useEffect(() => {
        const sources = dataType.Sources.map((source) => {
            return {
                id: source.Id,
                cardinality: source.Cardinality,
                source: null,
            };
        });

        form.setFieldValue(`data[${idx}].sources`, sources);
    }, []);

    return (
        <div>
            <fieldset>
                <legend>Sources</legend>
                <div className="flex flex-col gap-2">
                    <form.Field name={`data[${idx}].sources`} mode="array">
                        {(_values) =>
                            dataType.Sources.map((source, sdx) => (
                                <DataSource
                                    key={source.Id.toString()}
                                    form={form}
                                    idx={idx}
                                    sdx={sdx}
                                    source={source}
                                />
                            ))
                        }
                    </form.Field>
                </div>
            </fieldset>
        </div>
    );
}

interface DataSourceProps {
    form: ProjectDataFormApi;
    idx: number;
    sdx: number;
    source: DataTypeExternalSourceRx;
}
function DataSource({ form, idx, sdx, source }: DataSourceProps) {
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
                <form.Field name={`data[${idx}].sources[${sdx}].source`}>
                    {(field) => (
                        <input
                            type="file"
                            className="input-basic"
                            multiple={
                                source.Cardinality ===
                                DataSourceCardinalityMultiple
                            }
                            accept={source.MediaTypes?.join(", ")}
                            required={source.Required}
                            onChange={(e) => {
                                let value;
                                switch (source.Cardinality) {
                                    case DataSourceCardinalitySingle:
                                        value = e.target.files
                                            ? e.target.files[0]!
                                            : null;
                                        break;
                                    case DataSourceCardinalityMultiple:
                                        value = e.target.files;
                                        break;
                                    default:
                                        throw new Error(
                                            `invalid data source caradinality ${source.Cardinality}`,
                                        );
                                }

                                field.handleChange(value);
                            }}
                        />
                    )}
                </form.Field>
                <span>{cardinality_icon}</span>
            </label>
        </div>
    );
}

interface DatumPropertiesProps {
    form: ProjectDataFormApi;
    idx: number;
}
function DatumProperties({ form, idx }: DatumPropertiesProps) {
    const wrapper = useRef<HTMLDetailsElement>(null);

    function add_property(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.pushFieldValue(`data[${idx}].properties`, {
            key: "",
            type: PropertyTypeString,
            value: "",
        });

        wrapper.current!.open = true;
    }

    return (
        <details ref={wrapper}>
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
            <form.Subscribe
                selector={(state) => state.values.data[idx]!.properties.length}
            >
                {(plength) =>
                    plength === 0 ? (
                        <div className="text-secondary">(no properties)</div>
                    ) : (
                        <ul className="grid grid-cols-[repeat(4,min-content)] gap-2">
                            <form.Field
                                name={`data[${idx}].properties`}
                                mode="array"
                            >
                                {(properties) =>
                                    properties.state.value.map(
                                        (_property, pdx) => {
                                            return (
                                                <DatumPropertiesItem
                                                    key={pdx}
                                                    form={form}
                                                    idx={idx}
                                                    pdx={pdx}
                                                />
                                            );
                                        },
                                    )
                                }
                            </form.Field>
                        </ul>
                    )
                }
            </form.Subscribe>
        </details>
    );
}

interface DatumPropertiesItemProps {
    form: ProjectDataFormApi;
    idx: number;
    pdx: number;
}
function DatumPropertiesItem({ form, idx, pdx }: DatumPropertiesItemProps) {
    const [nativeError, setNativeError] = useState<string>("");
    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.removeFieldValue(`data[${idx}].properties`, idx);
    }

    return (
        <li className="col-span-full grid-cols-subgrid grid group">
            <div className="contents">
                <div className="col-1">
                    <form.Field
                        name={`data[${idx}].properties[${pdx}].key`}
                        validators={{
                            onChange: ({ value, fieldApi }) => {
                                if (value === "") {
                                    return undefined;
                                }

                                const keys = fieldApi.form.state.values.data[
                                    idx
                                ]!.properties.filter(
                                    (_, prop_dx) => prop_dx !== pdx,
                                ).map((prop) => prop.key);
                                if (keys.includes(value)) {
                                    return "Key already exists";
                                }
                                return undefined;
                            },
                        }}
                    >
                        {(field) => {
                            return (
                                <label>
                                    <span className="sr-only">Key</span>
                                    <input
                                        type="text"
                                        placeholder="Label"
                                        className="input-basic"
                                        value={field.state.value}
                                        onChange={(e) =>
                                            field.handleChange(
                                                e.target.value.trim(),
                                            )
                                        }
                                    />
                                    {field.state.meta.isValid ? null : (
                                        <div className="text-sm">
                                            {field.state.meta.errors.join(", ")}
                                        </div>
                                    )}
                                </label>
                            );
                        }}
                    </form.Field>
                </div>
                <div className="col-2">
                    <form.Field name={`data[${idx}].properties[${pdx}].type`}>
                        {(field) => {
                            return (
                                <SelectPropertyType
                                    className="input-basic"
                                    value={field.state.value}
                                    onChange={(e) =>
                                        field.handleChange(
                                            type_string_to_variant(
                                                e.target.value,
                                            )!,
                                        )
                                    }
                                />
                            );
                        }}
                    </form.Field>
                </div>
                <div className="col-3 flex gap-2">
                    <form.Subscribe
                        selector={(state) =>
                            state.values.data[idx]!.properties[pdx]!.type
                        }
                        children={(ptype) => (
                            <form.Field
                                name={`data[${idx}].properties[${pdx}].value`}
                                validators={{
                                    onChange: ({ value }) => {
                                        if (value.trim().length === 0) {
                                            return undefined;
                                        }

                                        try {
                                            stringToPropertyValue(value, ptype);
                                        } catch (err) {
                                            return "Value is incompatible with type";
                                        }
                                        return undefined;
                                    },
                                }}
                            >
                                {(field) => {
                                    return (
                                        <div>
                                            <InputPropertyValue
                                                type={ptype}
                                                className="input-basic"
                                                placeholder="Value"
                                                value={field.state.value}
                                                onChange={(e) => {
                                                    let value;
                                                    switch (ptype) {
                                                        case PropertyTypeBool:
                                                            value =
                                                                e.target.checked.toString();
                                                            break;
                                                        default:
                                                            value =
                                                                e.target.value;
                                                    }
                                                    field.handleChange(value);
                                                }}
                                                onInput={(e) =>
                                                    setNativeError(
                                                        e.target
                                                            .validationMessage,
                                                    )
                                                }
                                            />
                                            {field.state.meta.isValid ? (
                                                nativeError === "" ? null : (
                                                    <div className="text-sm">
                                                        {nativeError}
                                                    </div>
                                                )
                                            ) : (
                                                <div className="text-sm">
                                                    {field.state.meta.errors.join(
                                                        ", ",
                                                    )}
                                                </div>
                                            )}
                                        </div>
                                    );
                                }}
                            </form.Field>
                        )}
                    />
                </div>
                <div className="col-4">
                    <div className="invisible group-hover:visible">
                        <button
                            type="button"
                            className="btn-cmd"
                            title="Remove property"
                            onMouseDown={remove}
                        >
                            <Icon.Minus />
                        </button>
                    </div>
                </div>
            </div>
        </li>
    );
}

interface DatumNotesProps {
    form: ProjectDataFormApi;
    idx: number;
}
function DatumNotes({ form, idx }: DatumNotesProps) {
    const wrapper = useRef<HTMLDetailsElement>(null);

    function add_note(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.pushFieldValue(`data[${idx}].notes`, {
            Timestamp: new Date(),
            Visibility: VisibilityPrivate,
            Content: "",
        });

        wrapper.current!.open = true;
    }

    return (
        <details ref={wrapper}>
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
            <form.Subscribe
                selector={(state) => state.values.data[idx]!.notes.length}
            >
                {(nlength) =>
                    nlength === 0 ? (
                        <div className="text-secondary">(no notes)</div>
                    ) : (
                        <ul className="list-decimal px-4">
                            <form.Field
                                name={`data[${idx}].notes`}
                                mode="array"
                            >
                                {(notes) =>
                                    notes.state.value.map((_note, ndx) => {
                                        return (
                                            <DatumNotesItem
                                                key={ndx}
                                                form={form}
                                                idx={idx}
                                                ndx={ndx}
                                            />
                                        );
                                    })
                                }
                            </form.Field>
                        </ul>
                    )
                }
            </form.Subscribe>
        </details>
    );
}

interface DatumNotesItemProps {
    form: ProjectDataFormApi;
    idx: number;
    ndx: number;
}
function DatumNotesItem({ form, idx, ndx }: DatumNotesItemProps) {
    function remove(e: MouseEvent<HTMLButtonElement>, ndx: number) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        form.removeFieldValue(`data[${idx}].notes`, ndx);
    }

    const now_str = timestampToString(new Date());
    return (
        <li className="group">
            <div>
                <div className="flex gap-2 pb-1">
                    <div>
                        <form.Field
                            name={`data[${idx}].notes[${ndx}].Timestamp`}
                            validators={{
                                onBlur: (state) => {
                                    if (isNaN(+state.value)) {
                                        return "Timestamp is invalid";
                                    }
                                    if (state.value > new Date()) {
                                        return "Timestamp can not be in the future";
                                    }

                                    return undefined;
                                },
                            }}
                        >
                            {(field) => {
                                return (
                                    <label>
                                        <span className="sr-only">
                                            Timestamp
                                        </span>
                                        <input
                                            type="datetime-local"
                                            max={now_str}
                                            value={timestampToString(
                                                field.state.value,
                                            )}
                                            onChange={(e) =>
                                                field.handleChange(
                                                    new Date(e.target.value),
                                                )
                                            }
                                            onBlur={field.handleBlur}
                                            required
                                        />
                                        {field.state.meta.isValid ? null : (
                                            <div className="text-sm">
                                                {field.state.meta.errors.join(
                                                    ", ",
                                                )}
                                            </div>
                                        )}
                                    </label>
                                );
                            }}
                        </form.Field>
                    </div>
                    <div className="invisible group-hover:visible">
                        <button
                            type="button"
                            className="btn-cmd"
                            onMouseDown={(e) => remove(e, ndx)}
                        >
                            <Icon.Minus />
                        </button>
                    </div>
                </div>
                <div>
                    <form.Field name={`data[${idx}].notes[${ndx}].Content`}>
                        {(field) => {
                            return (
                                <label>
                                    <span className="sr-only">Content</span>
                                    <textarea
                                        placeholder="Content"
                                        className="input-basic resize"
                                        value={field.state.value}
                                        onChange={(e) =>
                                            field.handleChange(
                                                e.target.value.trimStart(),
                                            )
                                        }
                                    ></textarea>
                                </label>
                            );
                        }}
                    </form.Field>
                </div>
            </div>
        </li>
    );
}

function useDebouncedCallback<T>(callback: (value: T) => void, delay: number) {
    const timeout = useRef<ReturnType<typeof setTimeout>>(undefined);

    return useCallback(
        (value: T) => {
            clearTimeout(timeout.current);

            timeout.current = setTimeout(() => {
                callback(value);
            }, delay);
        },
        [callback, delay],
    );
}
