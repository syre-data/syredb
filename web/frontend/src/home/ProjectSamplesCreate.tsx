import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import {
    useParams,
    useNavigate,
    useSearchParams,
    redirect,
} from "react-router";
import * as types from "@/types";
import { Loading, SuspenseError } from "@/components/index";
import * as common from "@/common";
import { ErrorBoundary } from "react-error-boundary";
import type { FallbackProps } from "react-error-boundary";
import { useImmerReducer } from "use-immer";
import {
    createContext,
    Suspense,
    useContext,
    useLayoutEffect,
    useReducer,
    useRef,
    useState,
} from "react";
import type {
    SubmitEvent,
    ActionDispatch,
    ChangeEvent,
    KeyboardEvent,
    MouseEvent,
    InputEvent,
} from "react";
import { immerable } from "immer";
import icon from "../icon";
import classNames from "classnames";
import * as property from "@/components/Property";
import { SelectPropertyType, InputPropertyValue } from "@/components/Property";
import project_service from "@/service/project.service";
import data_service from "@/service/data.service";
import { StatusCodes } from "http-status-codes";

// TODO: Don't allow data to be uploaded if no raw data types are available.

export default function () {
    const { project_id } = useParams();
    if (project_id === undefined) {
        redirect("/");
        return;
    }

    return (
        <ErrorBoundary FallbackComponent={ProjectSamplesCreateError}>
            <Suspense fallback={<Loading />}>
                <ProjectSamplesCreate project_id={project_id} />
            </Suspense>
        </ErrorBoundary>
    );
}

function ProjectSamplesCreateError({
    error,
    resetErrorBoundary,
}: FallbackProps) {
    const err = error as common.BackendError;
    console.error(err);
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="pt-4 text-center"
        >
            <div>Could not load project data</div>
            <div>{err.message}</div>
        </SuspenseError>
    );
}

interface SampleInfo {
    id: number;
    label: string;
}

interface PropertyInfo {
    key: string;
    type: types.PropertyType;
}

class ColumnsState {
    [immerable] = true;
    columns: string[];
    // list item controls | list item label | label | tags | <properties> | list item row controls
    static readonly EXTRA_PRE_COLS = 3;
    static readonly EXTRA_POST_COLS = 1;

    constructor() {
        this.columns = ["sample.tags"];
    }

    asTemplate(): string {
        return `repeat(${this.numColumns()}, min-content)`;
    }

    numColumns(): number {
        if (this.columns.length === 1) {
            return (
                1 + ColumnsState.EXTRA_PRE_COLS + ColumnsState.EXTRA_POST_COLS
            );
        } else {
            return (
                this.columns.length +
                ColumnsState.EXTRA_PRE_COLS +
                ColumnsState.EXTRA_POST_COLS
            );
        }
    }

    addColumn(key: string) {
        this.columns.push(key);
    }

    /**
     * @param key Column key.
     * @returns `false` if the key did not exist.
     */
    removeColumn(key: string): boolean {
        const idx = this.columns.findIndex((col) => col === key);
        if (idx < 0) {
            return false;
        }
        this.columns.splice(idx, 1);
        return true;
    }

    findColumnIndex(key: string): number {
        const idx = this.columns.findIndex((col) => col === key);
        if (idx < 0) {
            return idx;
        }

        return idx + ColumnsState.EXTRA_PRE_COLS;
    }
}

interface SampleRowState {
    sample_id: number;
    expanded: boolean;
}
class SampleRowsState {
    [immerable] = true;
    rows: SampleRowState[];

    constructor(rows: SampleRowState[] = []) {
        this.rows = rows;
    }

    asTemplate(): string {
        const rows = this.rows.map((row) => {
            let template = "auto";
            if (row.expanded) {
                template += " auto auto";
            } else {
                template += " 0px 0px";
            }

            return template;
        });

        return rows.join(" ");
    }

    addRow(sample_id: number) {
        this.rows.push({ sample_id, expanded: false });
    }

    /**
     * @param sample_id Sample id.
     * @returns `false` if the row did not exist.
     */
    removeRow(sample_id: number): boolean {
        const idx = this.findIndex(sample_id);
        if (idx < 0) {
            return false;
        }

        this.rows.splice(idx, 1);
        return true;
    }

    findIndex(sample_id: number): number {
        return this.rows.findIndex((row) => row.sample_id === sample_id);
    }

    setExpanded(sample_id: number, expanded: boolean): boolean {
        const idx = this.findIndex(sample_id);
        if (idx < 0) {
            return false;
        }

        this.rows[idx]!.expanded = expanded;
        return true;
    }
}

interface DisplayState {
    columns: ColumnsState;
    rows: SampleRowsState;
}

class State {
    [immerable] = true;
    _next_sample_id: number;
    _project_sample_labels: string[];
    samples: SampleInfo[];
    properties: PropertyInfo[];
    display: DisplayState;

    constructor(project_sample_labels: string[]) {
        this._project_sample_labels = project_sample_labels;
        this.samples = [
            {
                id: 0,
                label: "",
            },
        ];
        this._next_sample_id = 1;
        this.properties = [];
        this.display = {
            columns: new ColumnsState(),
            rows: new SampleRowsState([{ sample_id: 0, expanded: false }]),
        };
    }
}

type StateAction =
    | { type: "add_sample" }
    | { type: "remove_sample"; payload: { id: number } }
    | { type: "set_sample_label"; payload: { id: number; label: string } }
    | {
          type: "add_property";
          payload: { property: PropertyInfo };
      }
    | { type: "remove_property"; payload: { key: string } }
    | {
          type: "expand_sample_row";
          payload: { sample_id: number; expand: boolean };
      };

function stateReducer(draft: State, action: StateAction) {
    let sample: SampleInfo;
    let maybe_sample: SampleInfo | undefined;
    let property;
    let idx: number;
    let key: string;
    switch (action.type) {
        case "add_sample":
            if (draft.samples.length < 1) {
                console.error("invalid draft, no samples");
            }
            sample = {
                id: draft._next_sample_id,
                label: "",
            };
            draft.samples.push(sample);

            draft.display.rows.addRow(draft._next_sample_id);

            draft._next_sample_id += 1;
            break;
        case "remove_sample":
            idx = draft.samples.findIndex(
                (sample) => sample.id === action.payload.id,
            );
            if (idx > -1) {
                draft.samples.splice(idx, 1);
            }
            break;
        case "set_sample_label":
            maybe_sample = draft.samples.find(
                (sample) => sample.id === action.payload.id,
            );
            if (maybe_sample === undefined) {
                console.error(`invalid sample id: ${action.payload.id}`);
                return;
            }
            sample = maybe_sample;
            sample.label = action.payload.label.trim();
            break;
        case "add_property":
            if (
                draft.properties.find(
                    (property) => property.key === action.payload.property.key,
                )
            ) {
                console.error(
                    `property ${action.payload.property.key} already exists`,
                );
                return;
            }
            draft.properties.push(action.payload.property);

            key = `property.${action.payload.property.key}`;
            draft.display.columns.addColumn(key);
            break;
        case "remove_property":
            idx = draft.properties.findIndex(
                (property) => property.key === action.payload.key,
            );
            if (idx < 0) {
                console.debug(`property ${action.payload.key} not found`);
                return;
            }
            draft.properties.splice(idx, 1);

            key = `property.${action.payload.key}`;
            const removed = draft.display.columns.removeColumn(key);
            if (!removed) {
                console.debug(`property ${action.payload.key} not found`);
            }
            return;
        case "expand_sample_row":
            const ok = draft.display.rows.setExpanded(
                action.payload.sample_id,
                action.payload.expand,
            );
            if (!ok) {
                console.error(
                    `sample ${action.payload.sample_id} not found in display state rows`,
                );
            }
            return;
    }
}

const StateCtx = createContext(new State([]));
const StateDispatchCtx = createContext<ActionDispatch<[StateAction]>>(() => {});

function project_sample_create_is_empty(
    sample: types.ProjectSampleCreate,
): boolean {
    const label_empty = !sample.Label || sample.Label.length === 0;
    const tags_empty = !sample.Tags || sample.Tags.length === 0;
    const properties_empty =
        !sample.Properties || sample.Properties.length === 0;
    const data_empty = sample.Data.length === 0;
    const notes_empty = sample.Notes.length === 0;

    return (
        label_empty &&
        tags_empty &&
        properties_empty &&
        data_empty &&
        notes_empty
    );
}

// interface SampleDataProperty {
//     id: string;
//     key?: string;
//     type?: types.PropertyType;
//     value: any;
// }

// interface SampleDataCreate {
//     id: string;
//     sample_id: string;
//     schema?: string;
//     file?: File;
//     timestamp?: Date;
//     properties: Map<string, SampleDataProperty>;
// }

interface SampleNoteCreate {
    id: string;
    sample_id: string;
    timestamp?: Date;
    content?: string;
}

const SAMPLE_LIST_ITEM_CONTROLS_COL = 1;
const SAMPLE_LIST_ITEM_LABEL_COL = 2;
const SAMPLE_LABEL_COL = 3;
const SAMPLE_TAGS_COL = 4;
const PROPERTIES_START_COL = 5;
const SAMPLE_LIST_ROW_CONTROLS_COL = -1;
const NEW_PROPERTY_TYPE_DEFAULT_VALUE = "string";

interface ProjectSamplesCreateProps {
    project_id: string;
}
function ProjectSamplesCreate({ project_id }: ProjectSamplesCreateProps) {
    const queryClient = useQueryClient();
    const navigate = useNavigate();
    const [searchParams, _] = useSearchParams();
    const { data: project } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_PROJECT, project_id],
        queryFn: () => project_service.getProjectWithUserPermission(project_id),
    });
    const { data: project_resources } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_PROJECT_RESOURCES, project_id],
        queryFn: async () => project_service.getProjectResources(project_id),
    });
    const { data: data_schemas } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_SCHEMA],
        queryFn: data_service.getDataSchemasRaw,
    });

    const [error, setError] = useState("");
    if (
        !project_service.hasPermission(
            types.ProjectPermissionCreateSample,
            project.Permissions,
        )
    ) {
        console.error(`invalid user permission: ${project.Permissions}`);
        navigate("/");
        return;
    }

    const project_sample_labels = project_resources.Samples.map(
        (sample) => sample.Label,
    );
    const [state, stateDispatch] = useImmerReducer(
        stateReducer,
        new State(project_sample_labels),
    );

    function cancel(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        const form = document.getElementById(
            "create-samples-form",
        )! as HTMLFormElement;
        const data = new FormData(form);
        let empty = true;
        for (const [key, value] of data.entries()) {
            const value_str = value.toString();
            let input = document.getElementById(key)!;
            let default_value = null;
            if (input.tagName === "INPUT") {
                default_value = (input as HTMLInputElement).defaultValue;
            }
            if (default_value !== null && value_str !== default_value) {
                empty = false;
                break;
            }
            if (default_value === null && value_str) {
                empty = false;
                break;
            }
        }
        if (!empty) {
            const proceed = window.confirm(
                "You have entered data. Are you sure you want to cancel?",
            );
            if (!proceed) {
                return;
            }
        }

        navigate(-1);
    }

    function submit_create_samples_form(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }
        e.stopPropagation();
        e.preventDefault();

        const form = document.getElementById(
            "create-samples-form",
        )! as HTMLFormElement;
        create_samples(form);
    }

    async function trigger_create_samples(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        const form = e.target as HTMLFormElement;
        await create_samples(form);
    }

    async function create_samples(form: HTMLFormElement) {
        setError("");

        const submitBtn = document.getElementById(
            "btn-submit",
        )! as HTMLButtonElement;
        submitBtn.disabled = true;

        const data = new FormData(form);
        for (const [key, _] of data.entries()) {
            const input = document.getElementById(key)! as HTMLInputElement;
            input.setCustomValidity("");
        }

        try {
            var [sample_info, sample_data, sample_notes] =
                project_samples_create_parse_form_data(data, state.properties);
        } catch (err) {
            console.error(err);
            submitBtn.disabled = false;
            return;
        }

        const sample_labels_dup: string[] = [];
        for (const [key, info] of sample_info.entries()) {
            if (sample_labels_dup.includes(info.Label)) {
                const input = document.getElementById(
                    `sample[${key}][label]`,
                )! as HTMLInputElement;
                input.setCustomValidity(
                    "A sample with this label already exists in this project",
                );
                input.reportValidity();
                submitBtn.disabled = false;
                return;
            }

            sample_labels_dup.push(info.Label);
        }

        for (const [id, info] of sample_info.entries()) {
            if (state._project_sample_labels.includes(info.Label)) {
                const input = document.getElementById(
                    `sample[${id}][label]`,
                )! as HTMLInputElement;
                input.setCustomValidity(
                    "A sample with this label already exists in this project",
                );
                input.reportValidity();
                submitBtn.disabled = false;
                return;
            }
        }

        // for (const data of sample_data.values()) {
        //     if (data.file === undefined || !data.file.name) {
        //         continue;
        //     }
        //     if (!data.schema) {
        //         console.debug(data.file);
        //         const input = document.getElementById(
        //             `sample[${data.sample_id}][data][${data.id}][schema]`,
        //         )! as HTMLInputElement;
        //         input.setCustomValidity("Data schema must be set");
        //         continue;
        //     }

        //     const data_properties = [];
        //     for (const property of data.properties.values()) {
        //         if (!property.key || !property.type) {
        //             console.error(`invalid sample data property`, property);
        //             submitBtn.disabled = false;
        //             return;
        //         }

        //         if (!property.value) {
        //             continue;
        //         }

        //         let value;
        //         switch (property.type) {
        //             case types.PropertyTypeString:
        //                 value = property.value;
        //                 break;
        //             case types.PropertyTypeBool:
        //                 switch (property.value) {
        //                     case "true":
        //                         value = true;
        //                         break;
        //                     case "false":
        //                         value = false;
        //                         break;
        //                     default:
        //                         console.error(
        //                             `invalid sample data property value: ${property.value}`,
        //                         );
        //                         submitBtn.disabled = false;
        //                         return;
        //                 }
        //                 break;
        //             case types.PropertyTypeInt:
        //                 value = parseInt(property.value);
        //                 break;
        //             case types.PropertyTypeUint:
        //                 value = parseInt(property.value);
        //                 if (value < 0) {
        //                     const input = document.getElementById(
        //                         `sample[${data.sample_id}][data][${data.id}][property][${property.id}][value]`,
        //                     )! as HTMLInputElement;
        //                     input.setCustomValidity(
        //                         "invalid value, must be non-negative",
        //                     );
        //                 }
        //                 break;
        //             case types.PropertyTypeFloat:
        //                 value = parseFloat(property.value);
        //                 break;
        //             case types.PropertyTypeTimestamp:
        //                 // TODO
        //                 break;
        //             case types.PropertyTypeQuantity:
        //                 if (!property.value.magnitude && !property.value.unit) {
        //                     continue;
        //                 }
        //                 if (!property.value.magnitude) {
        //                     const input = document.getElementById(
        //                         `sample[${data.sample_id}][data][${data.id}][property][${property.id}][value][magnitude]`,
        //                     )! as HTMLInputElement;
        //                     input.setCustomValidity(
        //                         "magnitude can not be empty",
        //                     );
        //                 } else if (!property.value.unit) {
        //                     const input = document.getElementById(
        //                         `sample[${data.sample_id}][data][${data.id}][property][${property.id}][value][magnitude]`,
        //                     )! as HTMLInputElement;
        //                     input.setCustomValidity("unit can not be empty");
        //                 }

        //                 const magnitude_value = parseFloat(
        //                     property.value.magnitude,
        //                 );
        //                 value = {
        //                     MagnitudeString: property.value.magnitude,
        //                     MagnitudeValue: magnitude_value,
        //                     Unit: property.value.unit,
        //                 } as property.QuantityProperty;
        //                 break;
        //             default:
        //                 throw new Error(
        //                     `invalid property type: ${property.type}`,
        //                 );
        //         }

        //         data_properties.push({
        //             Key: property.key,
        //             Type: property.type,
        //             Value: value,
        //         } satisfies types.ProjectSampleDataPropertyCreate);
        //     }

        //     const sample = sample_info.get(data.sample_id)!;
        //     const file_info = {
        //         Name: data.file.name,
        //         Size: data.file.size,
        //         File: data.file,
        //     } satisfies types.FileInfo;
        //     sample.Data.push({
        //         Schema: data.schema,
        //         File: file_info,
        //         Timestamp: data.timestamp ?? new Date(),
        //         Properties: data_properties,
        //     } satisfies types.ProjectSampleDataCreate);
        // }

        for (const note of sample_notes.values()) {
            if (!note.content) {
                continue;
            }
            if (!note.timestamp) {
                const input = document.getElementById(
                    `sample[${note.sample_id}][note][${note.id}][timestamp]`,
                )! as HTMLInputElement;
                input.setCustomValidity("timestamp must be set");
                continue;
            }

            const sample = sample_info.get(note.sample_id)!;
            sample.Notes.push({
                Timestamp: note.timestamp,
                Content: note.content,
            } satisfies types.ProjectSampleNoteCreate);
        }
        if (!form.reportValidity()) {
            submitBtn.disabled = false;
            return;
        }

        const sample_data_filtered = [
            ...sample_info
                .entries()
                .filter(
                    ([_, sample]) => !project_sample_create_is_empty(sample),
                ),
        ];
        if (sample_data_filtered.length === 0) {
            setError("No samples have information");
            submitBtn.disabled = false;
            return;
        }

        for (const [sample_id, sample] of sample_data_filtered) {
            if (sample.Label.length === 0) {
                const input = document.getElementById(
                    `sample[${sample_id}][label]`,
                )! as HTMLInputElement;
                input.setCustomValidity("label can not be empty");
            }
        }
        if (!form.reportValidity()) {
            submitBtn.disabled = false;
            return;
        }

        const samples = [...sample_data_filtered.map(([_, sample]) => sample)];
        await project_service
            .createProjectSamples(project_id, samples)
            .then((resp) => {
                queryClient.invalidateQueries({
                    queryKey: [common.QUERY_KEY_PROJECT_RESOURCES, project_id],
                });
                if (resp.ok) {
                    navigate(-1);
                } else if (resp.status === StatusCodes.UNAUTHORIZED) {
                    setError(
                        "You do not have permission to create samples for this project",
                    );
                } else {
                    console.error(resp);
                    setError("Something went wrong.");
                }
                submitBtn.disabled = false;
            })
            .catch((err: common.BackendError) => {
                setError(err.message);
                submitBtn.disabled = false;
            });
    }

    return (
        <div>
            <div className="pt-1 px-4">
                <h2 className="font-bold">
                    Create samples for {project.Label}
                </h2>
            </div>
            <StateCtx value={state}>
                <StateDispatchCtx value={stateDispatch}>
                    <div
                        className="grid gap-2"
                        style={{
                            gridTemplateColumns:
                                state.display.columns.asTemplate(),
                        }}
                    >
                        <ProjectSamplesFormHeader className="row-1 col-span-full grid-cols-subgrid" />
                        <form
                            id="create-samples-form"
                            onSubmit={trigger_create_samples}
                            className="row-2 grid col-span-full grid-cols-subgrid
                            grid-rows-[repeat(2,min-content)] pt-2"
                        >
                            <ProjectSamplesFormList
                                data_schemas={data_schemas}
                                className="row-1 col-span-full grid-cols-subgrid"
                            />
                            <div className="row-2 col-span-full flex gap-2 justify-center px-4">
                                <div>
                                    <button
                                        type="submit"
                                        id="btn-submit"
                                        className="btn-submit"
                                        onMouseDown={submit_create_samples_form}
                                    >
                                        Create samples
                                    </button>
                                </div>
                                <div>
                                    <button
                                        type="button"
                                        className="btn-submit"
                                        onMouseDown={cancel}
                                    >
                                        Cancel
                                    </button>
                                </div>
                            </div>
                        </form>
                    </div>
                </StateDispatchCtx>
            </StateCtx>
            <div className="pt-2 text-center text-red-600">{error}</div>
        </div>
    );
}

/**
 * @param data Form data
 * @returns Parsed form data
 * @throws Error if data is invalid.
 */
function project_samples_create_parse_form_data(
    data: FormData,
    properties: PropertyInfo[],
): [
    Map<string, types.ProjectSampleCreate>,
    // Map<string, SampleDataCreate>,
    Map<string, SampleNoteCreate>,
] {
    const SAMPLE_FORM_KEY_PATTERN = /sample\[(\d+?)\]\[(\w+?)\]/;
    const SAMPLE_PROPERTY_PATTERN = /sample\[\d+?\]\[property\]\[(\w+?)\]/;
    // const SAMPLE_DATA_PATTERN = /sample\[\d+?\]\[data\]\[(\d+?)\]\[(\w+?)\]/;
    // const SAMPLE_DATA_PROPERTY_PATTERN =
    //     /sample\[\d+?\]\[data\]\[\d+?\]\[property\]\[(\d+)\]\[(\w+?)\]/;
    const SAMPLE_NOTE_PATTERN = /sample\[\d+?\]\[note\]\[(\d+?)\]\[(\w+?)\]/;

    const sample_info = new Map<string, types.ProjectSampleCreate>();
    // const sample_data = new Map<string, SampleDataCreate>();
    const sample_notes = new Map<string, SampleNoteCreate>();

    for (const [key, value] of data.entries()) {
        const matches = key.match(SAMPLE_FORM_KEY_PATTERN);
        if (matches === null) {
            throw new Error(`invalid sample form key: ${key}`);
        }
        const sample_id = matches[1]!;
        const value_key = matches[2]!;
        if (!sample_info.has(sample_id)) {
            sample_info.set(sample_id, {
                Label: "",
                Tags: [],
                Properties: [],
                Data: [],
                Notes: [],
            } satisfies types.ProjectSampleCreate);
        }
        const sample = sample_info.get(sample_id)!;

        switch (value_key) {
            case "label":
                sample.Label = value.toString();
                break;
            case "tags":
                const tags_maybe_dup = value
                    .toString()
                    .split(",")
                    .map((tag) => tag.trim())
                    .filter((tag) => tag.length > 0);
                const tags = new Set(tags_maybe_dup);
                sample.Tags = [...tags];
                break;
            case "property":
                const property_matches = key.match(SAMPLE_PROPERTY_PATTERN);
                if (property_matches === null) {
                    throw new Error(`invalid sample property form key: ${key}`);
                }
                const property_key = property_matches[1]!;

                const property_data = properties.find(
                    (property) => property.key === property_key,
                );
                if (property_data === undefined) {
                    throw new Error(`invalid property key: ${property_key}`);
                }

                let property_value;
                switch (property_data.type) {
                    case types.PropertyTypeBool:
                        property_value = value.toString() === "true";
                        break;
                    case types.PropertyTypeString:
                        property_value = value.toString().trim();
                        if (property_value.length === 0) {
                            continue;
                        }
                        break;
                    case types.PropertyTypeInt:
                        property_value = value.toString().trim();
                        if (property_value.length === 0) {
                            continue;
                        }

                        property_value = parseInt(property_value);
                        if (isNaN(property_value)) {
                            const input = document.getElementById(
                                key,
                            )! as HTMLInputElement;
                            input.setCustomValidity("invalid integer");
                            continue;
                        }
                        break;
                    case types.PropertyTypeUint:
                        property_value = value.toString().trim();
                        if (property_value.length === 0) {
                            continue;
                        }

                        property_value = parseInt(property_value);
                        if (isNaN(property_value) || property_value < 0) {
                            const input = document.getElementById(
                                key,
                            )! as HTMLInputElement;
                            input.setCustomValidity("invalid unsigned integer");
                            continue;
                        }
                        break;
                    case types.PropertyTypeFloat:
                        property_value = value.toString().trim();
                        if (property_value.length === 0) {
                            continue;
                        }

                        property_value = parseFloat(property_value);
                        if (isNaN(property_value)) {
                            const input = document.getElementById(
                                key,
                            )! as HTMLInputElement;
                            input.setCustomValidity(
                                "could not parse as a number",
                            );
                            continue;
                        }
                        break;
                    case types.PropertyTypeQuantity:
                        const magnitude_key = `sample[${sample_id}][property][${property_key}][magnitude]`;
                        const unit_key = `sample[${sample_id}][property][${property_key}][unit]`;
                        if (key === unit_key) {
                            continue;
                        }
                        if (key !== magnitude_key) {
                            throw new Error(
                                `invalid sample property quantity form key: ${key}`,
                            );
                        }

                        const unit_data = data.get(unit_key);
                        if (unit_data === null) {
                            throw new Error(`quantity missing unit: ${key}`);
                        }

                        const magnitude_string = value.toString().trim();
                        const unit = unit_data.toString().trim();
                        if (
                            magnitude_string.length === 0 &&
                            unit.length === 0
                        ) {
                            continue;
                        }
                        if (
                            magnitude_string.length === 0 &&
                            unit.length !== 0
                        ) {
                            const input = document.getElementById(
                                magnitude_key,
                            )! as HTMLInputElement;

                            input.setCustomValidity(
                                "magnitude can not be empty",
                            );
                            continue;
                        }

                        const magnitude_value = parseFloat(magnitude_string);
                        if (isNaN(magnitude_value)) {
                            const input = document.getElementById(
                                magnitude_key,
                            )! as HTMLInputElement;

                            input.setCustomValidity(
                                "could not parse as a number",
                            );
                            continue;
                        }

                        if (
                            unit.length === 0 &&
                            magnitude_string.length !== 0
                        ) {
                            const input = document.getElementById(
                                unit_key,
                            )! as HTMLInputElement;

                            input.setCustomValidity("unit can not be empty");
                            continue;
                        }

                        property_value = {
                            MagnitudeString: magnitude_string,
                            MagnitudeValue: magnitude_value,
                            Unit: unit,
                        };

                        break;
                    case types.PropertyTypeTimestamp:
                        throw new Error("TODO");
                    default:
                        throw new Error(
                            `invalid sample property type key: ${key}`,
                        );
                }

                const prop = {
                    Key: property_key,
                    Type: property_data.type,
                    Value: property_value,
                } satisfies types.Property;
                sample.Properties.push(prop);
                break;
            // case "data":
            //     const data_matches = key.match(SAMPLE_DATA_PATTERN);
            //     if (data_matches === null) {
            //         throw new Error(`invalid sample data form key: ${key}`);
            //     }

            //     const data_id = data_matches[1]!;
            //     const data_key = data_matches[2]!;
            //     const data_map_key = `${sample_id}/${data_id}`;
            //     if (!sample_data.has(data_map_key)) {
            //         sample_data.set(data_map_key, {
            //             id: data_id,
            //             sample_id: sample_id,
            //             file: undefined,
            //             timestamp: undefined,
            //             properties: new Map(),
            //         });
            //     }
            //     const sample_datum = sample_data.get(data_map_key)!;
            //     switch (data_key) {
            //         case "schema":
            //             sample_datum.schema = value.toString().trim();
            //             break;
            //         case "file":
            //             const file = value as File;
            //             sample_datum.file = file;
            //             break;
            //         case "timestamp":
            //             sample_datum.timestamp = new Date(value.toString());
            //             break;
            //         case "property":
            //             const data_property_matches = key.match(
            //                 SAMPLE_DATA_PROPERTY_PATTERN,
            //             );
            //             if (data_property_matches === null) {
            //                 throw new Error(
            //                     `invalid sample data form key: ${key}`,
            //                 );
            //             }
            //             const data_property_id = data_property_matches[1]!;
            //             const data_property_key = data_property_matches[2]!;
            //             if (!sample_datum.properties.has(data_property_id)) {
            //                 sample_datum.properties.set(data_property_id, {
            //                     id: data_property_id,
            //                     key: undefined,
            //                     type: undefined,
            //                     value: undefined,
            //                 });
            //             }

            //             const datum_property =
            //                 sample_datum.properties.get(data_property_id)!;
            //             switch (data_property_key) {
            //                 case "key":
            //                     datum_property.key = value.toString();
            //                     break;
            //                 case "type":
            //                     datum_property.type =
            //                         property.type_string_to_variant(
            //                             value.toString(),
            //                         );
            //                     break;
            //                 case "value":
            //                     if (key.endsWith("[magnitude]")) {
            //                         if (datum_property.value === undefined) {
            //                             datum_property.value = {};
            //                         }
            //                         datum_property.value.magnitude = value
            //                             .toString()
            //                             .trim();
            //                     } else if (key.endsWith("[unit]")) {
            //                         if (datum_property.value === undefined) {
            //                             datum_property.value = {};
            //                         }
            //                         datum_property.value.unit = value
            //                             .toString()
            //                             .trim();
            //                     } else {
            //                         datum_property.value = value
            //                             .toString()
            //                             .trim();
            //                     }
            //                     break;
            //                 default:
            //                     throw new Error(
            //                         `invalid sample data property form key: ${key}`,
            //                     );
            //             }
            //             break;
            //         default:
            //             throw new Error(`invalid sample data key: ${key}`);
            //     }
            //     break;
            case "note":
                const note_matches = key.match(SAMPLE_NOTE_PATTERN);
                if (note_matches === null) {
                    throw new Error(`invalid sample note form key: ${key}`);
                }
                const note_id = note_matches[1]!;
                const note_key = note_matches[2]!;
                const note_map_key = `${sample_id}/${note_id}`;
                if (!sample_notes.has(note_map_key)) {
                    sample_notes.set(note_map_key, {
                        id: note_id,
                        sample_id: sample_id,
                        timestamp: undefined,
                        content: undefined,
                    });
                }
                const sample_note = sample_notes.get(note_map_key)!;
                switch (note_key) {
                    case "timestamp":
                        sample_note.timestamp = new Date(value.toString());
                        break;
                    case "content":
                        sample_note.content = value.toString().trim();
                        break;
                    default:
                        throw new Error(`invalid sample note key: ${key}`);
                }
                break;
            default:
                throw new Error(`invalid sample form property key: ${key}`);
        }
    }

    return [sample_info, sample_data, sample_notes];
}

interface ProjectSamplesFormHeaderProps {
    className?: string;
}
function ProjectSamplesFormHeader({
    className,
}: ProjectSamplesFormHeaderProps) {
    const state = useContext(StateCtx);
    const stateDispatch = useContext(StateDispatchCtx);
    const gridWrapperNode = useRef<HTMLDivElement | null>(null);
    const newPropertyKeyNode = useRef<HTMLInputElement | null>(null);
    const newPropertyTypeNode = useRef<HTMLSelectElement | null>(null);

    function on_change_property_key(e: ChangeEvent<HTMLInputElement>) {
        const keyInput = e.target;
        if (!keyInput.validity.customError) {
            return;
        }
        keyInput.setCustomValidity("");

        const key = keyInput.value.trim();
        if (
            state.properties.findIndex((property) => property.key === key) > -1
        ) {
            keyInput.setCustomValidity("key already exists");
            keyInput.reportValidity();
            return;
        }
    }

    function add_property(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        if (
            newPropertyKeyNode.current === null ||
            newPropertyTypeNode.current === null
        ) {
            return;
        }
        const keyInput = newPropertyKeyNode.current;
        keyInput.setCustomValidity("");

        const key = keyInput.value.trim();
        if (key.length === 0) {
            keyInput.setCustomValidity("key must be present");
            keyInput.reportValidity();
            return;
        }
        if (
            state.properties.findIndex((property) => property.key === key) > -1
        ) {
            keyInput.setCustomValidity("key already exists");
            keyInput.reportValidity();
            return;
        }

        const type = property.type_string_to_variant(
            newPropertyTypeNode.current.value,
        );
        if (type === undefined) {
            console.error(`invalid property type ${type}`);
            return;
        }

        stateDispatch({
            type: "add_property",
            payload: { property: { key, type } },
        });

        keyInput.value = "";
        newPropertyTypeNode.current.value = NEW_PROPERTY_TYPE_DEFAULT_VALUE;
    }

    return (
        <div
            ref={gridWrapperNode}
            className={`grid gap-2 sticky top-0 bg-white dark:bg-black ${className ?? ""}`}
        >
            <div
                className="row-1 -col-end-1 flex gap-2 sticky left-0"
                style={{ gridColumnStart: PROPERTIES_START_COL }}
            >
                <h3>Properties</h3>
                <form className="flex gap-1" onSubmit={add_property}>
                    <div>
                        <label>
                            <span className="sr-only">Property key</span>
                            <input
                                ref={newPropertyKeyNode}
                                type="text"
                                id="new-property-key"
                                name="new-property-key"
                                placeholder="Key"
                                className="input-basic"
                                onChange={on_change_property_key}
                            />
                        </label>
                    </div>
                    <div>
                        <span className="sr-only">Property type</span>
                        <SelectPropertyType
                            ref={newPropertyTypeNode}
                            id="new-property-type"
                            name="new-property-type"
                            className="input-basic"
                            defaultValue={NEW_PROPERTY_TYPE_DEFAULT_VALUE}
                        />
                    </div>
                    <button type="submit" className="btn-cmd">
                        <icon.Plus />
                    </button>
                </form>
            </div>
            <div className="row-2" style={{ gridColumn: SAMPLE_LABEL_COL }}>
                Label
            </div>
            <div className="row-2" style={{ gridColumn: SAMPLE_TAGS_COL }}>
                Tags
            </div>
            {state.properties.length === 0 ? (
                <div
                    className="row-2 -col-end-2"
                    style={{ gridColumnStart: PROPERTIES_START_COL }}
                >
                    <small>no properties (add one above)</small>
                </div>
            ) : (
                state.properties.map((property, index) => (
                    <ProjectSamplesFormHeaderProperty
                        key={property.key}
                        index={index}
                        property={property}
                    />
                ))
            )}
        </div>
    );
}

interface ProjectSamplesFormHeaderPropertyProps {
    index: number;
    property: PropertyInfo;
}
function ProjectSamplesFormHeaderProperty({
    index,
    property,
}: ProjectSamplesFormHeaderPropertyProps) {
    const state = useContext(StateCtx);
    const stateDispatch = useContext(StateDispatchCtx);

    function remove(e: MouseEvent<HTMLButtonElement>) {
        stateDispatch({
            type: "remove_property",
            payload: { key: property.key },
        });
    }

    return (
        <div
            className="row-2 flex gap-1 group/header-property"
            style={{ gridColumn: index + PROPERTIES_START_COL }}
            title={`Property ${property.key} (${property.type})`}
        >
            <div>{property.key}</div>
            <div className="invisible group-hover/header-property:visible">
                <button type="button" className="btn-cmd" onMouseDown={remove}>
                    <icon.Trash />
                </button>
            </div>
        </div>
    );
}

interface ProjectSamplesFormListProps {
    data_schemas: types.DataSchema[];
    className?: string;
}
function ProjectSamplesFormList({
    data_schemas,
    className,
}: ProjectSamplesFormListProps) {
    const SAMPLE_ROW_SPAN = 3;

    const state = useContext(StateCtx);
    const stateDispatch = useContext(StateDispatchCtx);

    function add_sample(e: MouseEvent<HTMLButtonElement>) {
        e.preventDefault();
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        stateDispatch({ type: "add_sample" });
    }

    return (
        <ol
            className={`grid gap-y-2 ${className ?? ""}`}
            style={{
                gridTemplateRows: state.display.rows.asTemplate() + " auto",
            }}
        >
            {state.samples.map((sample, idx) => {
                const expanded = state.display.rows.rows.find(
                    (row) => row.sample_id === sample.id,
                )?.expanded;

                return (
                    <li
                        key={sample.id.toString()}
                        id={`sample[${sample.id}]`}
                        className={classNames({
                            "px-4 col-span-full grid grid-cols-subgrid grid-rows-subgrid border-l \
                            group/sample-row": true,
                            "border-l-transparent": !expanded,
                            "pb-2": expanded,
                        })}
                        style={{
                            gridRowStart: idx * SAMPLE_ROW_SPAN + 1,
                            gridRowEnd: `span ${SAMPLE_ROW_SPAN}`,
                        }}
                        data-file-drop-target
                    >
                        <SamplesFormListItem
                            sample={sample}
                            index={idx}
                            data_schemas={data_schemas}
                        />
                    </li>
                );
            })}
            <li
                className=" -row-2 pl-1"
                style={{ gridColumn: SAMPLE_LIST_ITEM_LABEL_COL }}
            >
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={add_sample}
                    >
                        <icon.Plus />
                    </button>
                </div>
            </li>
        </ol>
    );
}

interface SampleNoteInfo {
    id: number;
}

// interface SampleDataInfo {
//     id: number;
// }

interface SamplesFormListItemProps {
    sample: SampleInfo;
    index: number;
    data_schemas: types.DataSchema[];
}
function SamplesFormListItem({
    sample,
    index,
    data_schemas,
}: SamplesFormListItemProps) {
    const state = useContext(StateCtx);
    const stateDispatch = useContext(StateDispatchCtx);
    const [notes, setNotes] = useState<SampleNoteInfo[]>([{ id: 0 }]);
    // const [data, setData] = useState<SampleDataInfo[]>([{ id: 0 }]);

    function update_label(e: ChangeEvent<HTMLInputElement>) {
        const input = e.target as HTMLInputElement;
        input.setCustomValidity("");

        const value = input.value.trim();
        if (value.length === 0) {
            return;
        }

        const matching_label_idx = state.samples.findIndex(
            (sample, idx) => sample.label === value && idx !== index,
        );
        if (matching_label_idx > -1) {
            input.setCustomValidity(
                `Labels must be unique, matches sample ${matching_label_idx + 1}`,
            );
            input.reportValidity();
            return;
        }
        if (state._project_sample_labels.includes(value)) {
            input.setCustomValidity(
                "A sample with this label already exists in this project",
            );
            input.reportValidity();
            return;
        }

        stateDispatch({
            type: "set_sample_label",
            payload: { id: sample.id, label: value },
        });
    }

    function add_new_sample_if_needed() {
        if (index + 1 === state.samples.length && sample.label) {
            stateDispatch({ type: "add_sample" });
        }
    }

    function remove(e: MouseEvent<HTMLButtonElement>) {
        e.preventDefault();
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        stateDispatch({ type: "remove_sample", payload: { id: sample.id } });
    }

    const expanded = state.display.rows.rows.find(
        (row) => row.sample_id === sample.id,
    )?.expanded;

    function toggle_expand(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        stateDispatch({
            type: "expand_sample_row",
            payload: { sample_id: sample.id, expand: !expanded },
        });
    }

    // function add_data(e: MouseEvent<HTMLButtonElement>) {
    //     if (e.button != common.MouseButton.Primary) {
    //         return;
    //     }

    //     const id = data.length === 0 ? 0 : data[data.length - 1]!.id + 1;
    //     setData([...data, { id }]);
    // }

    // function remove_data(id: number) {
    //     setData(data.filter((data) => data.id !== id));
    // }

    function add_note(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        const id = notes.length === 0 ? 0 : notes[notes.length - 1]!.id + 1;
        setNotes([...notes, { id }]);
    }

    function remove_note(id: number) {
        setNotes(notes.filter((note) => note.id !== id));
    }

    const ROW_MAIN_IDX = 1;
    // const ROW_DATA_IDX = 2;
    const ROW_NOTES_IDX = 3;
    return (
        <div className="contents">
            <div
                style={{
                    gridColumn: SAMPLE_LIST_ITEM_CONTROLS_COL,
                    gridRow: ROW_MAIN_IDX,
                }}
            >
                <button
                    type="button"
                    className="cursor-pointer"
                    onMouseDown={toggle_expand}
                >
                    <div
                        className={classNames({
                            "group-hover/sample-row:visible": true,
                            invisible: !expanded,
                            visible: expanded,
                            "-rotate-90": !expanded,
                        })}
                    >
                        <icon.CaretDown />
                    </div>
                </button>
            </div>
            <div
                className="text-right whitespace-nowrap"
                style={{
                    gridColumn: SAMPLE_LIST_ITEM_LABEL_COL,
                    gridRow: ROW_MAIN_IDX,
                }}
            >
                {index + 1}.
            </div>
            <div
                style={{
                    gridColumn: SAMPLE_LABEL_COL,
                    gridRow: ROW_MAIN_IDX,
                }}
            >
                <label>
                    <span className="sr-only">Label</span>
                    <input
                        type="text"
                        id={`sample[${sample.id}][label]`}
                        name={`sample[${sample.id}][label]`}
                        placeholder="Label"
                        title="Sample label"
                        className="input-basic invalid:ring-red-600"
                        onChange={update_label}
                        onBlur={add_new_sample_if_needed}
                    />
                </label>
            </div>
            <div
                className="col-3"
                style={{ gridRow: ROW_MAIN_IDX, gridColumn: SAMPLE_TAGS_COL }}
            >
                <label>
                    <span className="sr-only">Tags</span>
                    <input
                        type="text"
                        id={`sample[${sample.id}][tags]`}
                        name={`sample[${sample.id}][tags]`}
                        placeholder="Tags"
                        className="input-basic"
                        title="Comma separated list of sample tags"
                    />
                </label>
            </div>
            {state.properties.map((property) => (
                <SampleProperty
                    key={property.key}
                    sample={sample}
                    gridRow={ROW_MAIN_IDX}
                    property={property}
                />
            ))}
            <div
                className={classNames({
                    "invisible group-hover/sample-row:visible \
                    flex gap-1 items-center \
                    sticky right-0 bg-white dark:bg-black":
                        state.samples.length > 1,
                })}
                style={{
                    gridColumn: state.display.columns.numColumns(),
                    gridRow: ROW_MAIN_IDX,
                }}
            >
                {state.samples.length <= 1 ? null : (
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            title="Remove sample"
                            onMouseDown={remove}
                        >
                            <icon.Trash />
                        </button>
                    </div>
                )}
            </div>
            {/* <div
                className="-col-end-1 overflow-hidden"
                style={{
                    gridRow: ROW_DATA_IDX,
                    gridColumnStart: SAMPLE_LABEL_COL,
                }}
            >
                <div className="flex gap-2 pb-2">
                    <h3>Data</h3>
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            onMouseDown={add_data}
                        >
                            <icon.Plus />
                        </button>
                    </div>
                </div>
                <ol className="grid grid-cols-[repeat(3,min-content)] gap-1 pb-2">
                    {data.map((data, index) => (
                        <SampleDataListItem
                            key={data.id}
                            index={index}
                            sample={sample}
                            data={data}
                            data_schemas={data_schemas}
                            onRemove={remove_data}
                        />
                    ))}
                </ol>
            </div> */}
            <div
                className="-col-end-1 overflow-hidden"
                style={{
                    gridRow: ROW_NOTES_IDX,
                    gridColumnStart: SAMPLE_LABEL_COL,
                }}
            >
                <div className="flex gap-2 pb-2">
                    <h3>Notes</h3>
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            onMouseDown={add_note}
                        >
                            <icon.Plus />
                        </button>
                    </div>
                </div>
                <ol className="grid grid-cols-[1.5rem_auto]">
                    {notes.map((note, index) => (
                        <li key={note.id} className="contents">
                            <div className="col-1">{index + 1}.</div>
                            <SampleNote
                                index={index}
                                id={note.id}
                                sample={sample}
                                onRemove={remove_note}
                                className="col-2"
                            />
                        </li>
                    ))}
                </ol>
            </div>
        </div>
    );
}

interface SamplePropertyProps {
    gridRow: number;
    sample: SampleInfo;
    property: PropertyInfo;
}
function SampleProperty({ sample, gridRow, property }: SamplePropertyProps) {
    const state = useContext(StateCtx);
    const inputNode = useRef(null);

    useLayoutEffect(() => {
        if (
            inputNode.current !== null &&
            property.type === types.PropertyTypeBool
        ) {
            const input = inputNode.current as HTMLInputElement;
            input.indeterminate = true;
        }
    }, [inputNode]);

    const colIdx = state.display.columns.findColumnIndex(
        `property.${property.key}`,
    );
    if (colIdx < 0) {
        throw new Error("invalid column index");
    }

    return (
        <div style={{ gridRow, gridColumn: colIdx + 1 }}>
            <InputPropertyValue
                type={property.type}
                id={`sample[${sample.id}][property][${property.key}]`}
                name={`sample[${sample.id}][property][${property.key}]`}
                className="input-basic"
                quantity_magnitude_props={{
                    className: "input-basic min-w-0 shrink",
                }}
                quantity_unit_props={{
                    className: "input-basic min-w-0 shrink-2",
                }}
            />
        </div>
    );
}

// interface SampleDataListItemProps {
//     index: number;
//     sample: SampleInfo;
//     data: SampleDataInfo;
//     data_schemas: types.DataSchema[];
//     onRemove: (id: number) => void;
// }
// function SampleDataListItem({
//     index,
//     sample,
//     data,
//     data_schemas,
//     onRemove,
// }: SampleDataListItemProps) {
//     const [expanded, setExpanded] = useState(false);

//     function toggle_expand(e: MouseEvent<HTMLButtonElement>) {
//         if (e.button != common.MouseButton.Primary) {
//             return;
//         }

//         setExpanded(!expanded);
//     }

//     return (
//         <li
//             key={data.id}
//             className={classNames({
//                 "col-span-full grid grid-cols-subgrid group/data-list-item border-l": true,
//                 "border-l-transparent": !expanded,
//             })}
//         >
//             <div className="col-1">
//                 <button
//                     type="button"
//                     className={classNames({
//                         "group-hover/data-list-item:visible btn-cmd": true,
//                         "invisible -rotate-90": !expanded,
//                     })}
//                     onMouseDown={toggle_expand}
//                 >
//                     <icon.CaretDown />
//                 </button>
//             </div>
//             <div className="col-2">{index + 1}.</div>
//             <SampleData
//                 index={index}
//                 id={data.id}
//                 sample={sample}
//                 data_schemas={data_schemas}
//                 onRemove={onRemove}
//                 expanded={expanded}
//                 className="col-3"
//             />
//         </li>
//     );
// }

// interface FilePreview {
//     state: "empty" | "loading" | "ok" | "error";
//     error?: string;
//     filename?: string;
//     data?: any[][];
// }

// type FilePreviewAction =
//     | { type: "set_empty" }
//     | { type: "set_filename"; payload: { filename: string } }
//     | { type: "set_loading" }
//     | { type: "set_ok"; payload: { data: any[][] } }
//     | { type: "set_error"; payload: { error: string } };
// function filePreviewReducer(
//     state: FilePreview,
//     action: FilePreviewAction,
// ): FilePreview {
//     switch (action.type) {
//         case "set_empty":
//             return { state: "empty" };
//         case "set_filename":
//             return { ...state, filename: action.payload.filename };
//         case "set_loading":
//             return { state: "loading", filename: state.filename };
//         case "set_ok":
//             // TODO: set data preview
//             return {
//                 state: "ok",
//                 filename: state.filename,
//                 data: action.payload.data,
//             };
//         case "set_error":
//             return {
//                 state: "error",
//                 filename: state.filename,
//                 error: action.payload.error,
//             };
//     }
// }

// interface SampleDataProps {
//     index: number;
//     id: number;
//     sample: SampleInfo;
//     data_schemas: types.DataSchema[];
//     onRemove: (id: number) => void;
//     expanded: boolean;
//     className?: string;
// }
// function SampleData({
//     index,
//     id,
//     sample,
//     data_schemas,
//     onRemove,
//     expanded,
//     className,
// }: SampleDataProps) {
//     const file_input = useRef<HTMLInputElement>(null);
//     const [properties, setProperties] = useState<DataPropertyInfo[]>([]);
//     const [filePreview, dispatchFilePreview] = useReducer(filePreviewReducer, {
//         state: "empty",
//     });

//     function remove(e: MouseEvent<HTMLButtonElement>) {
//         if (e.button != common.MouseButton.Primary) {
//             return;
//         }

//         onRemove(id);
//     }

//     async function try_parse_data() {
//         const file_input = document.getElementById(
//             `sample[${sample.id}][data][${id}][file]`,
//         )! as HTMLInputElement;

//         const schema_input = document.getElementById(
//             `sample[${sample.id}][data][${id}][schema]`,
//         )! as HTMLSelectElement;

//         if (!file_input.files || file_input.files.length === 0) {
//             dispatchFilePreview({ type: "set_empty" });
//             return;
//         }
//         if (file_input.files.length > 1) {
//             throw new Error("multiple files selected");
//         }
//         const file = file_input.files[0]!;
//         dispatchFilePreview({
//             type: "set_filename",
//             payload: { filename: file.name },
//         });

//         const schema_id = schema_input.value.trim();
//         if (schema_id.length === 0) {
//             return;
//         }
//         const schema = data_schemas.find((schema) => schema.Id === schema_id);
//         if (schema === undefined) {
//             throw new Error(`data schema ${schema_id} not found`);
//         }

//         await data_service
//             .parseDataFileToSchema(file, schema)
//             .then((parsed) => {
//                 dispatchFilePreview({
//                     type: "set_ok",
//                     payload: { data: parsed },
//                 });
//             })
//             .catch((err) => {
//                 console.error(err);
//                 dispatchFilePreview({
//                     type: "set_error",
//                     payload: { error: `${err}` },
//                 });
//                 file_input.setCustomValidity(
//                     "Could not parse data according to schema",
//                 );
//             });
//     }

//     function add_property(key: string, type: types.PropertyType) {
//         const id =
//             properties.length === 0
//                 ? 0
//                 : properties[properties.length - 1]!.id + 1;
//         const new_prop = { id, key, type };
//         setProperties([...properties, new_prop]);
//     }

//     function remove_property(id: number) {
//         const remaining = properties.filter((property) => property.id !== id);
//         setProperties(remaining);
//     }

//     const now = nowLocalISOString();
//     return (
//         <div
//             id={`sample[${sample.id}][data][${id}]`}
//             className={`px-0.5 ${className}`}
//             data-wails-dropzone
//         >
//             <div className="flex gap-2">
//                 <div>
//                     <label>
//                         <span className="sr-only">Data schema</span>
//                         <select
//                             id={`sample[${sample.id}][data][${id}][schema]`}
//                             name={`sample[${sample.id}][data][${id}][schema]`}
//                             className="px-1 py-0.5 min-w-32 w-full min-h-4 input-basic"
//                             title={`Data schema`}
//                             onChange={try_parse_data}
//                             defaultValue={""}
//                         >
//                             <option value="" disabled>
//                                 Data schema
//                             </option>
//                             {data_schemas.map((schema) => (
//                                 <option
//                                     key={schema.Id.toString()}
//                                     value={schema.Id.toString()}
//                                     title={schema.Description}
//                                 >
//                                     {schema.Label}
//                                 </option>
//                             ))}
//                         </select>
//                     </label>
//                 </div>
//                 <div className="flex gap-1">
//                     <label>
//                         <span className="sr-only">File path</span>
//                         <input
//                             ref={file_input}
//                             type="file"
//                             id={`sample[${sample.id}][data][${id}][file]`}
//                             name={`sample[${sample.id}][data][${id}][file]`}
//                             className="input-basic"
//                             title={`Data file`}
//                             placeholder="Data file"
//                             onChange={try_parse_data}
//                         />
//                     </label>
//                     <div>
//                         <span
//                             className={classNames({
//                                 "p-0.5 align-middle aspect-square rounded-full text-red-600 cursor-pointer": true,
//                                 hidden: filePreview.state !== "error",
//                                 "inline-block": filePreview.state === "error",
//                             })}
//                             title={filePreview.error}
//                         >
//                             <icon.CircleX />
//                         </span>
//                         <span
//                             className={classNames({
//                                 "p-0.5 align-middle aspect-square rounded-full cursor-pointer": true,
//                                 hidden: filePreview.state === "error",
//                                 "inline-block": filePreview.state !== "error",
//                                 "bg-gray-600": filePreview.state === "empty",
//                                 "bg-green-600": filePreview.state === "ok",
//                             })}
//                             title={
//                                 filePreview.state === "empty"
//                                     ? "No data to preview"
//                                     : "Data parsed correctly"
//                             }
//                         >
//                             <icon.Eye />
//                         </span>
//                     </div>
//                 </div>
//                 <div>
//                     <label>
//                         <span className="sr-only">Timestamp</span>
//                         <input
//                             type="datetime-local"
//                             id={`sample[${sample.id}][data][${id}][timestamp]`}
//                             name={`sample[${sample.id}][data][${id}][timestamp]`}
//                             className="input-basic"
//                             title={`Data timestamp`}
//                             placeholder="Timestamp"
//                             defaultValue={now}
//                             max={now}
//                         />
//                     </label>
//                 </div>
//                 <div>
//                     <button
//                         type="button"
//                         className="btn-cmd"
//                         title={`Remove data ${index + 1}`}
//                         onMouseDown={remove}
//                     >
//                         <icon.Trash />
//                     </button>
//                 </div>
//             </div>
//             <SampleDataProperties
//                 sample_id={sample.id}
//                 data_id={id}
//                 properties={properties}
//                 onAddProperty={add_property}
//                 onRemoveProperty={remove_property}
//                 className={classNames({
//                     "overflow-hidden": true,
//                     "h-0": !expanded,
//                     "py-2": expanded,
//                 })}
//             />
//         </div>
//     );
// }

// interface DataPropertyInfo {
//     id: number;
//     key: string;
//     type: types.PropertyType;
// }

// interface SampleDataPropertiesProps {
//     sample_id: number;
//     data_id: number;
//     properties: DataPropertyInfo[];
//     onAddProperty: (key: string, type: types.PropertyType) => void;
//     onRemoveProperty: (id: number) => void;
//     className?: string;
// }
// function SampleDataProperties({
//     sample_id,
//     data_id,
//     properties,
//     onAddProperty,
//     onRemoveProperty,
//     className,
// }: SampleDataPropertiesProps) {
//     const newPropertyKeyNode = useRef<HTMLInputElement | null>(null);
//     const newPropertyTypeNode = useRef<HTMLSelectElement | null>(null);

//     function on_change_property_key(e: ChangeEvent<HTMLInputElement>) {
//         const key_input = e.target;
//         if (!key_input.validity.customError) {
//             return;
//         }
//         key_input.setCustomValidity("");

//         const key = key_input.value.trim();
//         if (properties.findIndex((property) => property.key === key) > -1) {
//             key_input.setCustomValidity("key already exists");
//             key_input.reportValidity();
//         }
//     }

//     function add_property() {
//         const key_input = newPropertyKeyNode.current! as HTMLInputElement;
//         const type_input = newPropertyTypeNode.current! as HTMLSelectElement;
//         key_input.setCustomValidity("");
//         type_input.setCustomValidity("");

//         const key = key_input.value.trim();
//         if (key.length === 0) {
//             key_input.setCustomValidity("key can not be empty");
//             key_input.reportValidity();
//             return;
//         }
//         if (properties.findIndex((property) => property.key === key) > -1) {
//             key_input.setCustomValidity("key already exists");
//             key_input.reportValidity();
//             return;
//         }

//         const type = property.type_string_to_variant(type_input.value);
//         if (type === undefined) {
//             type_input.setCustomValidity("invalid type");
//             type_input.reportValidity();
//             return;
//         }

//         onAddProperty(key, type);
//         key_input.value = "";
//         type_input.value = NEW_PROPERTY_TYPE_DEFAULT_VALUE;
//     }

//     function add_property_btn(e: MouseEvent<HTMLButtonElement>) {
//         if (e.button != common.MouseButton.Primary) {
//             return;
//         }

//         add_property();
//     }

//     function capture_enter(e: KeyboardEvent<HTMLInputElement>) {
//         if (e.key === "Enter") {
//             e.preventDefault();
//             e.stopPropagation();
//             add_property();
//         }
//     }

//     return (
//         <div className={className}>
//             <div className="flex gap-2 pb-2">
//                 <h4>Properties</h4>
//                 <div>
//                     <label>
//                         <span className="sr-only">Property key</span>
//                         <input
//                             ref={newPropertyKeyNode}
//                             type="text"
//                             placeholder="Key"
//                             className="input-basic"
//                             onChange={on_change_property_key}
//                             onKeyDown={capture_enter}
//                         />
//                     </label>
//                 </div>
//                 <div>
//                     <span className="sr-only">Property type</span>
//                     <SelectPropertyType
//                         ref={newPropertyTypeNode}
//                         className="input-basic"
//                         defaultValue={NEW_PROPERTY_TYPE_DEFAULT_VALUE}
//                     />
//                 </div>
//                 <div>
//                     <button
//                         type="button"
//                         className="btn-cmd"
//                         onMouseDown={add_property_btn}
//                     >
//                         <icon.Plus />
//                     </button>
//                 </div>
//             </div>
//             <ol
//                 className="grid gap-1"
//                 style={{
//                     gridTemplateColumns: `repeat(3, min-content)`,
//                 }}
//             >
//                 {properties.map((property) => {
//                     return (
//                         <SampleDataProperty
//                             key={property.id}
//                             sample_id={sample_id}
//                             data_id={data_id}
//                             property={property}
//                             onRemove={onRemoveProperty}
//                         />
//                     );
//                 })}
//             </ol>
//         </div>
//     );
// }

// interface SampleDataPropertyProps {
//     sample_id: number;
//     data_id: number;
//     property: DataPropertyInfo;
//     onRemove: (id: number) => void;
// }
// function SampleDataProperty({
//     sample_id,
//     data_id,
//     property,
//     onRemove,
// }: SampleDataPropertyProps) {
//     function remove(e: MouseEvent<HTMLButtonElement>) {
//         if (e.button != common.MouseButton.Primary) {
//             return;
//         }

//         onRemove(property.id);
//     }

//     return (
//         <div className="col-span-full grid grid-cols-subgrid group/sample-data-property">
//             <div className="col-1 whitespace-nowrap">
//                 <label>
//                     <input
//                         type="hidden"
//                         id={`sample[${sample_id}][data][${data_id}][property][${property.id}][key]`}
//                         name={`sample[${sample_id}][data][${data_id}][property][${property.id}][key]`}
//                         value={property.key}
//                         readOnly
//                     />
//                 </label>
//                 <label>
//                     <input
//                         type="hidden"
//                         id={`sample[${sample_id}][data][${data_id}][property][${property.id}][type]`}
//                         name={`sample[${sample_id}][data][${data_id}][property][${property.id}][type]`}
//                         value={property.type}
//                         readOnly
//                     />
//                 </label>
//                 {property.key}{" "}
//                 <span className="text-gray-500">({property.type})</span>
//             </div>
//             <SampleDataPropertyValue
//                 sample_id={sample_id}
//                 data_id={data_id}
//                 property={property}
//                 className="col-2"
//             />
//             <div className="col-3">
//                 <button
//                     type="button"
//                     className="btn-cmd invisible group-hover/sample-data-property:visible"
//                     onMouseDown={remove}
//                 >
//                     <icon.Trash />
//                 </button>
//             </div>
//         </div>
//     );
// }

// interface SampleDataPropertyValueProps {
//     sample_id: number;
//     data_id: number;
//     property: DataPropertyInfo;
//     className?: string;
// }
// function SampleDataPropertyValue({
//     sample_id,
//     data_id,
//     property,
//     className,
// }: SampleDataPropertyValueProps) {
//     const inputNode = useRef(null);

//     useLayoutEffect(() => {
//         if (
//             inputNode.current !== null &&
//             property.type === types.PropertyTypeBool
//         ) {
//             const input = inputNode.current as HTMLInputElement;
//             input.indeterminate = true;
//         }
//     }, [inputNode]);

//     switch (property.type) {
//         case types.PropertyTypeString:
//             return (
//                 <div className={className}>
//                     <input
//                         type="text"
//                         id={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         name={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         className="input-basic"
//                     />
//                 </div>
//             );
//         case types.PropertyTypeInt:
//             return (
//                 <div className={className}>
//                     <input
//                         type="number"
//                         id={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         name={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         className="input-basic"
//                     />
//                 </div>
//             );
//         case types.PropertyTypeUint:
//             return (
//                 <div className={className}>
//                     <input
//                         type="number"
//                         id={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         name={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         min={0}
//                         className="input-basic"
//                     />
//                 </div>
//             );
//         case types.PropertyTypeFloat:
//             return (
//                 <div className={className}>
//                     <input
//                         type="text"
//                         id={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         name={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         className="input-basic"
//                     />
//                 </div>
//             );
//         case types.PropertyTypeBool:
//             return (
//                 <div className={className}>
//                     <input
//                         ref={inputNode}
//                         type="checkbox"
//                         id={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         name={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         className="input-basic"
//                     />
//                 </div>
//             );
//         case types.PropertyTypeTimestamp:
//             return (
//                 <div className={className}>
//                     <input
//                         ref={inputNode}
//                         type="datetime-local"
//                         id={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         name={`sample[${sample_id}][data][${data_id}][property][${property.id}][value]`}
//                         className="input-basic"
//                     />
//                 </div>
//             );
//         case types.PropertyTypeQuantity:
//             return (
//                 <div className={`flex gap-1 ${className}`}>
//                     <input
//                         type="text"
//                         id={`sample[${sample_id}][data][${data_id}][property][${property.id}][value][magnitude]`}
//                         name={`sample[${sample_id}][data][${data_id}][property][${property.id}][value][magnitude]`}
//                         className="input-basic min-w-0 shrink"
//                         placeholder="Magnitude"
//                     />
//                     <input
//                         type="text"
//                         id={`sample[${sample_id}][data][${data_id}][property][${property.id}][value][unit]`}
//                         name={`sample[${sample_id}][data][${data_id}][property][${property.id}][value][unit]`}
//                         className="input-basic min-w-0 shrink-2"
//                         placeholder="Units"
//                     />
//                 </div>
//             );
//     }
// }

interface SampleNoteProps {
    index: number;
    id: number;
    sample: SampleInfo;
    onRemove: (id: number) => void;
    className?: string;
}
function SampleNote({
    index,
    id,
    sample,
    onRemove,
    className,
}: SampleNoteProps) {
    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        onRemove(id);
    }

    const now = nowLocalISOString();
    return (
        <div className={`px-0.5 ${className}`}>
            <div className="pb-2 flex gap-2">
                <div>
                    <label>
                        <span className="sr-only">Date</span>
                        <input
                            type="datetime-local"
                            id={`sample[${sample.id}][note][${id}][timestamp]`}
                            name={`sample[${sample.id}][note][${id}][timestamp]`}
                            defaultValue={now}
                            max={now}
                            className="input-basic pb-2"
                        />
                    </label>
                </div>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        title={`Remove note ${index + 1}`}
                        onMouseDown={remove}
                    >
                        <icon.Trash />
                    </button>
                </div>
            </div>
            <div>
                <label>
                    <span className="sr-only">Note</span>
                    <textarea
                        id={`sample[${sample.id}][note][${id}][content]`}
                        name={`sample[${sample.id}][note][${id}][content]`}
                        placeholder="Note"
                        className="px-1 py-0.5 w-full min-h-4 input-basic"
                    ></textarea>
                </label>
            </div>
        </div>
    );
}

function nowLocalISOString() {
    let now = new Date();
    now = new Date(Date.now() - now.getTimezoneOffset() * 60 * 1000);
    now.setSeconds(0);
    now.setMilliseconds(0);
    return now.toISOString().slice(0, -8); // YYYY-MM-DDTHH:mm
}
