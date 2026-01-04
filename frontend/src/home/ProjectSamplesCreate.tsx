import { useSuspenseQuery } from "@tanstack/react-query";
import { useParams, useNavigate } from "react-router";
import * as app from "../../bindings/syredb/app";
import * as common from "../common";
import { ErrorBoundary, FallbackProps } from "react-error-boundary";
import { useImmerReducer } from "use-immer";
import {
    ActionDispatch,
    ChangeEvent,
    createContext,
    Dispatch,
    FormEvent,
    InputEvent,
    MouseEvent,
    SetStateAction,
    Suspense,
    useContext,
    useEffect,
    useLayoutEffect,
    useReducer,
    useRef,
    useState,
} from "react";
import { immerable } from "immer";
import icon from "../icon";
import classNames from "classnames";

export default function () {
    const { id: project_id } = useParams();
    const navigate = useNavigate();
    if (project_id === undefined) {
        navigate("/");
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

function Loading() {
    return <div>Loading project data</div>;
}

function ProjectSamplesCreateError({
    error,
    resetErrorBoundary,
}: FallbackProps) {
    console.error(error);
    return (
        <div>
            <div>Could not load project data</div>
            <div>{error}</div>
        </div>
    );
}

const DEFAULT_COLUMN_WIDTH = 200;
const DEFAULT_COLUMN_WIDTH_SM = 50;
const DEFAULT_ROW_HEIGHT = 24;

interface SampleInfo {
    id: number;
    label: string;
}

interface PropertyData {
    key: string;
    type: app.PropertyType;
}

interface ColumnState {
    key: string;
    width: number;
}

class ColumnsState {
    [immerable] = true;
    listMarker: number;
    sampleLabel: number;
    columns: ColumnState[];
    sampleRemove: number;

    constructor() {
        this.listMarker = DEFAULT_COLUMN_WIDTH_SM;
        this.sampleLabel = DEFAULT_COLUMN_WIDTH;
        this.columns = [{ key: "sample.tags", width: DEFAULT_COLUMN_WIDTH }];
        this.sampleRemove = DEFAULT_COLUMN_WIDTH_SM;
    }

    asTemplate(): string {
        const cols = this.columns.map((col) => `${col.width}px`).join(" ");
        if (this.columns.length === 1) {
            return `${this.listMarker}px ${this.sampleLabel}px ${cols} ${DEFAULT_COLUMN_WIDTH}px`;
        } else {
            return `${this.listMarker}px ${this.sampleLabel}px ${cols} ${this.sampleRemove}px`;
        }
    }

    numColumns(): number {
        return this.columns.length + 3;
    }

    addColumn(key: string, width: number = 200) {
        this.columns.push({ key, width });
    }

    /**
     * @param key Column key.
     * @returns `false` if the key did not exist.
     */
    removeColumn(key: string): boolean {
        const idx = this.columns.findIndex((col) => col.key === key);
        if (idx < 0) {
            return false;
        }
        this.columns.splice(idx, 1);
        return true;
    }

    findColumnIndex(key: string): number {
        const idx = this.columns.findIndex((col) => col.key === key);
        if (idx < 0) {
            return idx;
        }

        return idx + 2;
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

        this.rows[idx].expanded = expanded;
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
    properties: PropertyData[];
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
          payload: { property: PropertyData; width?: number };
      }
    | { type: "remove_property"; payload: { key: string } }
    | {
          type: "expand_sample_row";
          payload: { sample_id: number; expand: boolean };
      }
    | { type: "set_width"; payload: { column: number; width: number } };

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
                (sample) => sample.id === action.payload.id
            );
            if (idx > -1) {
                draft.samples.splice(idx, 1);
            }
            break;
        case "set_sample_label":
            maybe_sample = draft.samples.find(
                (sample) => sample.id === action.payload.id
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
                    (property) => property.key === action.payload.property.key
                )
            ) {
                console.error(
                    `property ${action.payload.property.key} already exists`
                );
                return;
            }
            draft.properties.push(action.payload.property);

            const width = action.payload.width ?? DEFAULT_COLUMN_WIDTH;
            key = `property.${action.payload.property.key}`;
            draft.display.columns.addColumn(key, width);
            break;
        case "remove_property":
            idx = draft.properties.findIndex(
                (property) => property.key === action.payload.key
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
                action.payload.expand
            );
            if (!ok) {
                console.error(
                    `sample ${action.payload.sample_id} not found in display state rows`
                );
            }
            return;
        case "set_width":
            draft.display.columns.columns[action.payload.column].width =
                action.payload.width;
            return;
    }
}

const StateCtx = createContext(new State([]));
const StateDispatchCtx = createContext<ActionDispatch<[StateAction]>>(() => {});

function project_sample_create_is_empty(
    sample: app.ProjectSampleCreate
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

interface SampleDataCreate {
    id: string;
    sample_id: string;
    schema?: string;
    file?: string;
    timestamp?: Date;
}

interface SampleNoteCreate {
    id: string;
    sample_id: string;
    timestamp?: Date;
    content?: string;
}

interface ProjectSamplesCreateProps {
    project_id: string;
}
function ProjectSamplesCreate({ project_id }: ProjectSamplesCreateProps) {
    const navigate = useNavigate();
    const { data: project } = useSuspenseQuery({
        queryKey: ["project", project_id],
        queryFn: () =>
            app.ProjectService.GetProjectWithUserPermission(project_id),
    });
    const { data: project_resources } = useSuspenseQuery({
        queryKey: ["project_resources", project_id],
        queryFn: async () => app.ProjectService.GetProjectResources(project_id),
    });
    const { data: data_schemas } = useSuspenseQuery({
        queryKey: [common.QUERY_KEY_DATA_SCHEMA],
        queryFn: app.DataService.GetDataSchemasAll,
    });

    const [error, setError] = useState("");
    const user_permission = common.project_user_permission_string_to_variant(
        project.UserPermission
    );
    if (user_permission === undefined) {
        console.error(`invalid user permission: ${project.UserPermission}`);
        navigate("/");
        return;
    }
    if (!common.is_admin_or_owner(user_permission)) {
        console.error(`insufficient permission: ${project.UserPermission}`);
        navigate("/");
        return;
    }

    const project_sample_labels = project_resources.Samples.map(
        (sample) => sample.Label
    );
    const [state, stateDispatch] = useImmerReducer(
        stateReducer,
        new State(project_sample_labels)
    );

    function cancel(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    async function create_samples(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
        setError("");

        const submitBtn = document.getElementById(
            "btn-submit"
        )! as HTMLButtonElement;
        submitBtn.disabled = true;

        const data = new FormData(e.target as HTMLFormElement);
        for (const [key, _] of data.entries()) {
            const input = document.getElementById(key)! as HTMLInputElement;
            input.setCustomValidity("");
        }

        let sample_info, sample_data, sample_notes;
        try {
            [sample_info, sample_data, sample_notes] =
                project_samples_create_parse_form_data(data, state.properties);
        } catch (err) {
            console.error(err);
            return;
        }

        const form = e.target as HTMLFormElement;
        for (const data of sample_data.values()) {
            if (!data.file) {
                continue;
            }
            if (!data.schema) {
                const input = document.getElementById(
                    `sample[${data.sample_id}][data][${data.id}][schema]`
                )! as HTMLInputElement;
                input.setCustomValidity("data schema must be set");
                continue;
            }

            const sample = sample_info.get(data.sample_id)!;
            sample.Data.push(
                new app.ProjectSampleDataCreate({
                    Schema: data.schema!,
                    FilePath: data.file!,
                    Timestamp: data.timestamp?.toISOString(),
                })
            );
        }

        for (const note of sample_notes.values()) {
            if (!note.content) {
                continue;
            }
            if (!note.timestamp) {
                const input = document.getElementById(
                    `sample[${note.sample_id}][note][${note.id}][timestamp]`
                )! as HTMLInputElement;
                input.setCustomValidity("timestamp must be set");
                continue;
            }

            const sample = sample_info.get(note.sample_id)!;
            sample.Notes.push(
                new app.ProjectSampleNoteCreate({
                    Timestamp: note.timestamp!.toISOString(),
                    Content: note.content,
                })
            );
        }
        if (!form.reportValidity()) {
            return;
        }

        const sample_data_filtered = [
            ...sample_info
                .entries()
                .filter(
                    ([_, sample]) => !project_sample_create_is_empty(sample)
                ),
        ];
        if (sample_data_filtered.length === 0) {
            setError("All samples are empty");
            return;
        }

        for (const [sample_id, sample] of sample_data_filtered) {
            if (sample.Label.length === 0) {
                const input = document.getElementById(
                    `sample[${sample_id}][label]`
                )! as HTMLInputElement;
                input.setCustomValidity("label can not be empty");
            }
        }
        if (!form.reportValidity()) {
            return;
        }

        const samples = [...sample_data_filtered.map(([_, sample]) => sample)];
        await app.ProjectService.CreateProjectSamples(project_id, samples)
            .then(() => {
                navigate(-1);
            })
            .catch((err) => {
                setError(err);
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
                    <ProjectSamplesFormHeader />
                    <form
                        onSubmit={create_samples}
                        className="flex flex-col gap-2 pt-2"
                    >
                        <ProjectSamplesFormList data_schemas={data_schemas} />
                        <div className="flex gap-2 justify-center px-4">
                            <div>
                                <button
                                    type="submit"
                                    id="btn-submit"
                                    className="btn-submit"
                                >
                                    Save
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
                </StateDispatchCtx>
            </StateCtx>
            <div>{error}</div>
        </div>
    );
}

/**
 *
 * @param data Form data
 * @returns Parsed form data
 * @throws Error if data is invalid.
 */
function project_samples_create_parse_form_data(
    data: FormData,
    properties: PropertyData[]
): [
    Map<string, app.ProjectSampleCreate>,
    Map<string, SampleDataCreate>,
    Map<string, SampleNoteCreate>
] {
    const sample_form_key_pattern = /sample\[(\d+?)\]\[(\w+?)\]/;
    const sample_property_pattern = /sample\[\d+?\]\[property\]\[(\w+?)\]/;
    const sample_data_pattern = /sample\[\d+?\]\[data\]\[(\d+?)\]\[(\w+?)\]/;
    const sample_note_pattern = /sample\[\d+?\]\[note\]\[(\d+?)\]\[(\w+?)\]/;

    const sample_info = new Map<string, app.ProjectSampleCreate>();
    const sample_data = new Map<string, SampleDataCreate>();
    const sample_notes = new Map<string, SampleNoteCreate>();

    for (const [key, value] of data.entries()) {
        const matches = key.match(sample_form_key_pattern);
        if (matches === null) {
            throw new Error(`invalid sample form key: ${key}`);
        }
        const sample_id = matches[1];
        const value_key = matches[2];
        if (!sample_info.has(sample_id)) {
            sample_info.set(
                sample_id,
                new app.ProjectSampleCreate({
                    Label: "",
                    Tags: [],
                    Properties: [],
                    Notes: [],
                })
            );
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
                const property_matches = key.match(sample_property_pattern);
                if (property_matches === null) {
                    throw new Error(`invalid sample property form key: ${key}`);
                }
                const property_key = property_matches[1];

                const property_data = properties.find(
                    (property) => property.key === property_key
                );
                if (property_data === undefined) {
                    throw new Error(`invalid property key: ${property_key}`);
                }

                let property_value;
                switch (property_data.type) {
                    case app.PropertyType.PROPERTY_TYPE_BOOL:
                        property_value = value.toString() === "true";
                        break;
                    case app.PropertyType.PROPERTY_TYPE_STRING:
                        property_value = value.toString().trim();
                        if (property_value.length === 0) {
                            continue;
                        }
                        break;
                    case app.PropertyType.PROPERTY_TYPE_INT:
                        property_value = value.toString().trim();
                        if (property_value.length === 0) {
                            continue;
                        }

                        property_value = parseInt(property_value);
                        if (isNaN(property_value)) {
                            const input = document.getElementById(
                                key
                            )! as HTMLInputElement;
                            input.setCustomValidity("invalid integer");
                            continue;
                        }
                        break;
                    case app.PropertyType.PROPERTY_TYPE_UINT:
                        property_value = value.toString().trim();
                        if (property_value.length === 0) {
                            continue;
                        }

                        property_value = parseInt(property_value);
                        if (isNaN(property_value) || property_value < 0) {
                            const input = document.getElementById(
                                key
                            )! as HTMLInputElement;
                            input.setCustomValidity("invalid unsigned integer");
                            continue;
                        }
                        break;
                    case app.PropertyType.PROPERTY_TYPE_FLOAT:
                        property_value = value.toString().trim();
                        if (property_value.length === 0) {
                            continue;
                        }

                        property_value = parseFloat(property_value);
                        if (isNaN(property_value)) {
                            const input = document.getElementById(
                                key
                            )! as HTMLInputElement;
                            input.setCustomValidity(
                                "could not parse as a number"
                            );
                            continue;
                        }
                        break;
                    case app.PropertyType.PROPERTY_TYPE_QUANTITY:
                        const magnitude_key = `sample[${sample_id}][property][${property_key}][magnitude]`;
                        const unit_key = `sample[${sample_id}][property][${property_key}][unit]`;
                        if (key === unit_key) {
                            continue;
                        }
                        if (key !== magnitude_key) {
                            throw new Error(
                                `invalid sample property quantity form key: ${key}`
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
                                magnitude_key
                            )! as HTMLInputElement;

                            input.setCustomValidity(
                                "magnitude can not be empty"
                            );
                            continue;
                        }

                        const magnitude_value = parseFloat(magnitude_string);
                        if (isNaN(magnitude_value)) {
                            const input = document.getElementById(
                                magnitude_key
                            )! as HTMLInputElement;

                            input.setCustomValidity(
                                "could not parse as a number"
                            );
                            continue;
                        }

                        if (
                            unit.length === 0 &&
                            magnitude_string.length !== 0
                        ) {
                            const input = document.getElementById(
                                unit_key
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
                    case app.PropertyType.PROPERTY_TYPE_TIMESTAMP:
                        throw new Error("TODO");
                    default:
                        throw new Error(
                            `invalid sample property type key: ${key}`
                        );
                }

                const property = new app.Property({
                    Key: property_key,
                    Type: property_data.type,
                    Value: property_value,
                });
                sample.Properties.push(property);
                break;
            case "data":
                const data_matches = key.match(sample_data_pattern);
                if (data_matches === null) {
                    throw new Error(`invalid sample data form key: ${key}`);
                }

                const data_id = data_matches[1];
                const data_key = data_matches[2];
                const data_map_key = `${sample_id}/${data_id}`;
                if (!sample_data.has(data_map_key)) {
                    sample_data.set(data_map_key, {
                        id: data_id,
                        sample_id: sample_id,
                        file: undefined,
                        timestamp: undefined,
                    });
                }
                const sample_datum = sample_data.get(data_map_key)!;
                switch (data_key) {
                    case "schema":
                        sample_datum.schema = value.toString().trim();
                        break;
                    case "file":
                        sample_datum.file = value.toString().trim();
                        break;
                    case "timestamp":
                        sample_datum.timestamp = new Date(value.toString());
                        break;
                    default:
                        throw new Error(`invalid sample data key: ${key}`);
                }
                break;
            case "note":
                const note_matches = key.match(sample_note_pattern);
                if (note_matches === null) {
                    throw new Error(`invalid sample note form key: ${key}`);
                }
                const note_id = note_matches[1];
                const note_key = note_matches[2];
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

const LIST_MARKER_COL = 1;
const SAMPLE_LABEL_COL = 2;
const SAMPLE_TAGS_COL = 3;
const PROPERTIES_START_COL = 4;
const NUM_HEADER_ROWS = 2;

function ProjectSamplesFormHeader() {
    const NEW_PROPERTY_TYPE_DEFAULT_VALUE = "string";
    const state = useContext(StateCtx);
    const stateDispatch = useContext(StateDispatchCtx);
    const gridWrapperNode = useRef<HTMLDivElement | null>(null);
    const newPropertyKeyNode = useRef<HTMLInputElement | null>(null);
    const newPropertyTypeNode = useRef<HTMLSelectElement | null>(null);

    useLayoutEffect(() => {
        if (gridWrapperNode.current === null) {
            return;
        }
        gridWrapperNode.current.style.gridTemplateColumns =
            state.display.columns.asTemplate();
    }, [state.display, gridWrapperNode]);

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

    function add_property(e: FormEvent<HTMLFormElement>) {
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

        const type = common.property_type_string_to_variant(
            newPropertyTypeNode.current.value
        );
        if (type === undefined) {
            console.error(`invalid property type ${type}`);
            return;
        }

        stateDispatch({
            type: "add_property",
            payload: { property: { key, type }, width: DEFAULT_COLUMN_WIDTH },
        });

        keyInput.value = "";
        newPropertyTypeNode.current.value = NEW_PROPERTY_TYPE_DEFAULT_VALUE;
    }

    return (
        <div ref={gridWrapperNode} className="grid gap-2 sticky top-0">
            <div
                className="row-1 -col-end-1 flex gap-2 sticky left-0"
                style={{ gridColumnStart: PROPERTIES_START_COL }}
            >
                <div>Properties</div>
                <form className="flex gap-1" onSubmit={add_property}>
                    <div>
                        <label>
                            <span className="hidden">Property key</span>
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
                        <span className="hidden">Property type</span>
                        <select
                            ref={newPropertyTypeNode}
                            id="new-property-type"
                            name="new-property-type"
                            className="input-basic"
                            defaultValue={NEW_PROPERTY_TYPE_DEFAULT_VALUE}
                        >
                            <option value="string" title="Text">
                                String
                            </option>
                            <option value="int" title="Integer">
                                Int
                            </option>
                            <option
                                value="uint"
                                title="Unsigned integer (counting numbers)"
                            >
                                UInt
                            </option>
                            <option value="float" title="Decimal number">
                                Float
                            </option>
                            <option value="boolean" title="True/False">
                                Boolean
                            </option>
                            <option
                                value="quantity"
                                title="Measured value (e.g. 10.0 cm)"
                            >
                                Quantity
                            </option>
                        </select>
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
    property: PropertyData;
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
    data_schemas: app.DataSchema[];
}
function ProjectSamplesFormList({ data_schemas }: ProjectSamplesFormListProps) {
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
            className="grid gap-y-2"
            style={{
                gridTemplateColumns: state.display.columns.asTemplate(),
                gridTemplateRows: state.display.rows.asTemplate() + " auto",
            }}
        >
            {state.samples.map((sample, idx) => {
                const expanded = state.display.rows.rows.find(
                    (row) => row.sample_id === sample.id
                )?.expanded;

                return (
                    <li
                        key={sample.id.toString()}
                        id={`sample[${sample.id}]`}
                        className={classNames({
                            "px-4 col-span-full grid grid-cols-subgrid grid-rows-subgrid border-l":
                                true,
                            "border-l-transparent": !expanded,
                            "pb-2": expanded,
                        })}
                        style={{
                            gridRowStart: idx * SAMPLE_ROW_SPAN + 1,
                            gridRowEnd: `span ${SAMPLE_ROW_SPAN}`,
                        }}
                        data-wails-dropzone
                    >
                        <SamplesFormListItem
                            sample={sample}
                            index={idx}
                            data_schemas={data_schemas}
                        />
                    </li>
                );
            })}
            <li className="px-4 col-1 -row-2 pl-1">
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

interface SampleDataInfo {
    id: number;
}

interface SamplesFormListItemProps {
    sample: SampleInfo;
    index: number;
    data_schemas: app.DataSchema[];
}
function SamplesFormListItem({
    sample,
    index,
    data_schemas,
}: SamplesFormListItemProps) {
    const state = useContext(StateCtx);
    const stateDispatch = useContext(StateDispatchCtx);
    const [notes, setNotes] = useState<SampleNoteInfo[]>([{ id: 0 }]);
    const [data, setData] = useState<SampleDataInfo[]>([{ id: 0 }]);

    function update_label(e: ChangeEvent<HTMLInputElement>) {
        const input = e.target as HTMLInputElement;
        input.setCustomValidity("");

        const value = input.value.trim();
        if (value.length === 0) {
            return;
        }

        const matching_label_idx = state.samples.findIndex(
            (sample, idx) => sample.label === value && idx !== index
        );
        if (matching_label_idx > -1) {
            input.setCustomValidity(
                `Labels must be unique, matches sample ${
                    matching_label_idx + 1
                }`
            );
            input.reportValidity();
            return;
        }
        if (state._project_sample_labels.includes(value)) {
            input.setCustomValidity(
                "A sample with this label already exists in this project"
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
        (row) => row.sample_id === sample.id
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

    function add_data(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        const id = data.length === 0 ? 0 : data[data.length - 1].id + 1;
        setData([...data, { id }]);
    }

    function remove_data(id: number) {
        setData(data.filter((data) => data.id !== id));
    }

    function add_note(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        const id = notes.length === 0 ? 0 : notes[notes.length - 1].id + 1;
        setNotes([...notes, { id }]);
    }

    function remove_note(id: number) {
        setNotes(notes.filter((note) => note.id !== id));
    }

    const ROW_MAIN_IDX = 1;
    const ROW_DATA_IDX = 2;
    const ROW_NOTES_IDX = 3;
    return (
        <div className="group/sample-row contents">
            <div style={{ gridColumn: LIST_MARKER_COL, gridRow: ROW_MAIN_IDX }}>
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
                {index + 1}.
            </div>
            <div className="col-2" style={{ gridRow: ROW_MAIN_IDX }}>
                <label>
                    <span className="hidden">Label</span>
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
            <div className="col-3" style={{ gridRow: ROW_MAIN_IDX }}>
                <label>
                    <span className="hidden">Tags</span>
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
                    sticky right-0 bg-white dark:bg-black": true,
                    "justify-center": state.display.columns.numColumns() > 4,
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
            <div
                className="col-start-2 -col-end-1 overflow-hidden"
                style={{ gridRow: ROW_DATA_IDX }}
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
                <ol className="grid grid-cols-[1.5rem_auto] gap-1 pb-2">
                    {data.map((data, index) => (
                        <li key={data.id} className="contents">
                            <div className="col-1">{index + 1}.</div>
                            <SampleData
                                index={index}
                                id={data.id}
                                sample={sample}
                                data_schemas={data_schemas}
                                onRemove={remove_data}
                                className="col-2"
                            />
                        </li>
                    ))}
                </ol>
            </div>
            <div
                className="col-start-2 -col-end-1 overflow-hidden"
                style={{ gridRow: ROW_NOTES_IDX }}
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
    property: PropertyData;
}
function SampleProperty({ sample, gridRow, property }: SamplePropertyProps) {
    const state = useContext(StateCtx);
    const inputNode = useRef(null);

    useLayoutEffect(() => {
        if (
            inputNode.current !== null &&
            property.type === app.PropertyType.PROPERTY_TYPE_BOOL
        ) {
            const input = inputNode.current as HTMLInputElement;
            input.indeterminate = true;
        }
    }, [inputNode]);

    const colIdx = state.display.columns.findColumnIndex(
        `property.${property.key}`
    );
    if (colIdx < 0) {
        console.error("invalid column index");
        return null;
    }

    const style = { gridRow, gridColumn: colIdx + 1 };
    switch (property.type) {
        case app.PropertyType.PROPERTY_TYPE_STRING:
            return (
                <div style={style}>
                    <input
                        type="string"
                        id={`sample[${sample.id}][property][${property.key}]`}
                        name={`sample[${sample.id}][property][${property.key}]`}
                        className="input-basic"
                    />
                </div>
            );
        case app.PropertyType.PROPERTY_TYPE_INT:
            return (
                <div style={style}>
                    <input
                        type="number"
                        id={`sample[${sample.id}][property][${property.key}]`}
                        name={`sample[${sample.id}][property][${property.key}]`}
                        className="input-basic"
                    />
                </div>
            );
        case app.PropertyType.PROPERTY_TYPE_UINT:
            return (
                <div style={style}>
                    <input
                        type="number"
                        id={`sample[${sample.id}][property][${property.key}]`}
                        name={`sample[${sample.id}][property][${property.key}]`}
                        min={0}
                        className="input-basic"
                    />
                </div>
            );
        case app.PropertyType.PROPERTY_TYPE_FLOAT:
            return (
                <div style={style}>
                    <input
                        type="number"
                        id={`sample[${sample.id}][property][${property.key}]`}
                        name={`sample[${sample.id}][property][${property.key}]`}
                        className="input-basic"
                    />
                </div>
            );
        case app.PropertyType.PROPERTY_TYPE_BOOL:
            return (
                <div style={style}>
                    <input
                        ref={inputNode}
                        type="checkbox"
                        id={`sample[${sample.id}][property][${property.key}]`}
                        name={`sample[${sample.id}][property][${property.key}]`}
                        className="input-basic"
                    />
                </div>
            );
        case app.PropertyType.PROPERTY_TYPE_TIMESTAMP:
            return (
                <div style={style}>
                    <input
                        ref={inputNode}
                        type="datetime-local"
                        id={`sample[${sample.id}][property][${property.key}]`}
                        name={`sample[${sample.id}][property][${property.key}]`}
                        className="input-basic"
                    />
                </div>
            );
        case app.PropertyType.PROPERTY_TYPE_QUANTITY:
            return (
                <div className="flex gap-1" style={style}>
                    <input
                        type="text"
                        id={`sample[${sample.id}][property][${property.key}][magnitude]`}
                        name={`sample[${sample.id}][property][${property.key}][magnitude]`}
                        className="input-basic min-w-0 shrink"
                        placeholder="Magnitude"
                    />
                    <input
                        type="string"
                        id={`sample[${sample.id}][property][${property.key}][unit]`}
                        name={`sample[${sample.id}][property][${property.key}][unit]`}
                        className="input-basic min-w-0 shrink-2"
                        placeholder="Units"
                    />
                </div>
            );
    }
}

interface SampleDataProps {
    index: number;
    id: number;
    sample: SampleInfo;
    data_schemas: app.DataSchema[];
    onRemove: (id: number) => void;
    className?: string;
}
function SampleData({
    index,
    id,
    sample,
    data_schemas,
    onRemove,
    className,
}: SampleDataProps) {
    const file_input = useRef<HTMLInputElement>(null);

    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        onRemove(id);
    }

    async function open_file_select(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        await app.FsService.OpenFileDialogSingle("Select a data file", [
            new app.FileFilter({
                DisplayName: "CSV",
                Pattern: "*.csv;*.tsv",
            }),
            new app.FileFilter({
                DisplayName: "Spreadsheets",
                Pattern: "*.xlsx;*.xls;*.ods",
            }),
            new app.FileFilter({ DisplayName: "All files", Pattern: "*.*" }),
        ])
            .then((path) => {
                const input = file_input.current!;
                input.value = path;
                try_parse_data();
            })
            .catch((err) => {
                if (err === "CANCELLED_BY_USER") {
                    return;
                }

                console.error(err);
                const input = file_input.current;
                input?.setCustomValidity(err.message);
            });
    }

    async function try_parse_data() {
        const file_input = document.getElementById(
            `sample[${sample.id}][data][${id}][file]`
        )! as HTMLInputElement;

        const schema_input = document.getElementById(
            `sample[${sample.id}][data][${id}][schema]`
        )! as HTMLSelectElement;

        const file = file_input.value.trim();
        const schema = schema_input.value.trim();
        if (file.length === 0 || schema.length === 0) {
            return;
        }

        await app.DataService.ParseDataFileToSchema(file, schema)
            .then((data) => {
                // TODO: data preview
                console.debug(data);
            })
            .catch((err) => {
                console.error(err);
                switch (err.message) {
                    case "INVALID_FILE_EXTENSION":
                        file_input.setCustomValidity("invalid file extension");
                    case "INCOMAPTIBLE_DATA_SIZE":
                        file_input.setCustomValidity(
                            "data file column length does not match schema"
                        );
                    default:
                        file_input.setCustomValidity(
                            "data incompatible with schema"
                        );
                }
                file_input.reportValidity();
            });
    }

    const now = nowLocalISOString();
    return (
        <div
            id={`sample[${sample.id}][data][${id}]`}
            className={`px-0.5 flex gap-2 ${className}`}
            data-wails-dropzone
        >
            <div>
                <label>
                    <span className="hidden">Data schema</span>
                    <select
                        id={`sample[${sample.id}][data][${id}][schema]`}
                        name={`sample[${sample.id}][data][${id}][schema]`}
                        className="px-1 py-0.5 w-full min-h-4 input-basic"
                        title={`Data schema`}
                        onChange={try_parse_data}
                        defaultValue={""}
                    >
                        <option value="" disabled>
                            Data schema
                        </option>
                        {data_schemas.map((schema) => (
                            <option
                                key={schema.Id}
                                value={schema.Id}
                                title={schema.Description}
                            >
                                {schema.Label}
                            </option>
                        ))}
                    </select>
                </label>
            </div>
            <div className="flex gap-1">
                <label>
                    <span className="hidden">File path</span>
                    <input
                        ref={file_input}
                        type="text"
                        id={`sample[${sample.id}][data][${id}][file]`}
                        name={`sample[${sample.id}][data][${id}][file]`}
                        className="input-basic"
                        title={`Data file`}
                        placeholder="Data file"
                        onChange={try_parse_data}
                    />
                </label>
                <div>
                    <button
                        type="button"
                        title="Select a data file"
                        className="btn-cmd"
                        onMouseDown={open_file_select}
                    >
                        <icon.FileUpload />
                    </button>
                </div>
            </div>
            <div>
                <label>
                    <span className="hidden">Timestamp</span>
                    <input
                        type="datetime-local"
                        id={`sample[${sample.id}][data][${id}][timestamp]`}
                        name={`sample[${sample.id}][data][${id}][timestamp]`}
                        className="input-basic"
                        title={`Data timestamp`}
                        placeholder="Timestamp"
                        defaultValue={now}
                        max={now}
                    />
                </label>
            </div>
            <div>
                <button
                    type="button"
                    className="btn-cmd"
                    title={`Remove data ${index + 1}`}
                    onMouseDown={remove}
                >
                    <icon.Trash />
                </button>
            </div>
        </div>
    );
}

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
                        <span className="hidden">Date</span>
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
                    <span className="hidden">Note</span>
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
    return now.toISOString().slice(0, -1);
}
