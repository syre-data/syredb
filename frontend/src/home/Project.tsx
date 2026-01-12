import { useSuspenseQuery } from "@tanstack/react-query";
import { useParams, Link, useNavigate } from "react-router";
import * as app from "../../bindings/syredb/app";
import icon from "../icon";
import { ErrorBoundary, FallbackProps } from "react-error-boundary";
import {
    ActionDispatch,
    ChangeEvent,
    createContext,
    FormEvent,
    InputEvent,
    MouseEvent,
    Suspense,
    useContext,
    useEffect,
    useLayoutEffect,
    useReducer,
    useRef,
    useState,
} from "react";
import * as common from "../common";
import classNames from "classnames";
import { useImmerReducer } from "use-immer";
import { UUID } from "../../bindings/github.com/google/uuid";
import { immerable } from "immer";
import * as appStateCtx from "../AppStateContext";

interface CommonProjectData {
    project_id: string;
    user_permission: app.ProjectUserPermission;
}

const CommonProjectDataCtx = createContext<CommonProjectData>({
    project_id: "",
    user_permission: app.ProjectUserPermission.$zero,
});

export default function () {
    const navigate = useNavigate();
    const { id: project_id } = useParams();
    if (project_id) {
        return (
            <ErrorBoundary FallbackComponent={ProjectError}>
                <Suspense fallback={<Loading />}>
                    <Project id={project_id} />
                </Suspense>
            </ErrorBoundary>
        );
    } else {
        navigate("/");
        return null;
    }
}

function Loading() {
    return <div className="text-center pt-4">Loading</div>;
}

function ProjectError({ error, resetErrorBoundary }: FallbackProps) {
    const navigate = useNavigate();

    if (error.message === common.USER_NOT_AUTHENTICATED_ERROR) {
        console.error(common.USER_NOT_AUTHENTICATED_ERROR);
        navigate("/");
        return null;
    } else {
        console.error(error);
    }

    function reload(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        resetErrorBoundary();
    }

    return (
        <div className="flex flex-col gap-2 items-center pt-4">
            <div>Could not load project</div>
            <div>{error.message}</div>
            <div className="flex gap-2 items-center">
                <div>
                    <Link to="/">
                        <button type="button" className="btn-cmd">
                            <icon.Home />
                        </button>
                    </Link>
                </div>
                <div>
                    <button
                        type="button"
                        onMouseDown={reload}
                        className="btn-cmd"
                    >
                        <icon.Reload />
                    </button>
                </div>
            </div>
        </div>
    );
}

interface ProjectProps {
    id: string;
}
function Project({ id }: ProjectProps) {
    const { data: project_resources } = useSuspenseQuery({
        queryKey: ["project_resources", id],
        queryFn: async () => app.ProjectService.GetProjectResources(id),
    });

    return (
        <CommonProjectDataCtx
            value={{
                project_id: id,
                user_permission: project_resources.ProjectUserPermission,
            }}
        >
            <div className="h-full flex flex-col">
                <ProjectHeader project={project_resources.Project} />
                <ResourceBrowser
                    project_resources={project_resources}
                    className="grow pt-2"
                />
            </div>
        </CommonProjectDataCtx>
    );
}

interface ProjectHeaderProps {
    project: app.Project;
    className?: string;
}
function ProjectHeader({ project, className }: ProjectHeaderProps) {
    const project_info = useContext(CommonProjectDataCtx);

    return (
        <div
            className={`flex gap-2 pt-1 pb-2 px-4 border-b ${className ?? ""}`}
        >
            <div className="flex gap-2 grow">
                <h2 className={`font-bold`}>{project.Label}</h2>
                {common.is_admin_or_owner(project_info.user_permission) ? (
                    <div>
                        <Link
                            to={`/project/${project_info.project_id}/samples/create`}
                        >
                            <button
                                type="button"
                                className="border rounded-full whitespace-nowrap px-2 cursor-pointer"
                            >
                                <icon.Plus className="inline" /> Add samples
                            </button>
                        </Link>
                    </div>
                ) : null}
            </div>
            <div className="flex gap-2 items-center">
                {
                    <div>
                        <Link to={`/project/${project.Id}/settings`}>
                            <icon.Gear />
                        </Link>
                    </div>
                }
                <div>
                    <Link to="/">
                        <button type="button" className="btn-cmd">
                            <icon.Home />
                        </button>
                    </Link>
                </div>
            </div>
        </div>
    );
}

enum PrimaryResourceType {
    Sample = "sample",
    Data = "data",
}

interface ResourceBrowserProps {
    project_resources: app.ProjectResources;
    className?: string;
}
function ResourceBrowser({
    project_resources,
    className,
}: ResourceBrowserProps) {
    const primaryResourceTypeNode = useRef<HTMLSelectElement>(null);
    const [primaryResourceType, setPrimaryResourceType] = useState(
        PrimaryResourceType.Sample
    );
    const [queryResults, setQueryResults] = useState<UUID[]>(
        project_resources.Samples.map((sample) => sample.Id)
    );

    function set_primary_resource_type() {
        const input = primaryResourceTypeNode.current!;
        switch (input.value) {
            case "sample":
                if (primaryResourceType !== PrimaryResourceType.Sample) {
                    setPrimaryResourceType(PrimaryResourceType.Sample);
                }
                break;
            case "data":
                if (primaryResourceType !== PrimaryResourceType.Data) {
                    setPrimaryResourceType(PrimaryResourceType.Data);
                }
                break;
            default:
                throw new Error(
                    `invalid primary resource type: ${input.value}`
                );
        }
    }

    function query_resources(e: FormEvent) {
        e.preventDefault();
    }

    return (
        <div className={`flex flex-col ${className ?? ""}`}>
            <div className="pb-2">
                <form onSubmit={query_resources} className="flex gap-2 px-4">
                    <div className="flex gap-2">
                        <div>
                            <label>
                                <span className="hidden">Primary resource</span>
                                <select
                                    ref={primaryResourceTypeNode}
                                    onChange={set_primary_resource_type}
                                    className="input-basic"
                                    defaultValue="sample"
                                >
                                    <option value="sample">Samples</option>
                                    <option value="data">Data</option>
                                </select>
                            </label>
                        </div>
                    </div>
                    <div>
                        <button type="submit" className="btn-submit">
                            Search
                        </button>
                    </div>
                </form>
            </div>
            {(() => {
                switch (primaryResourceType) {
                    case PrimaryResourceType.Sample:
                        return (
                            <ResourceBrowserSamples
                                project_resources={project_resources}
                                samples={queryResults}
                                className="grow"
                            />
                        );
                    case PrimaryResourceType.Data:
                        return (
                            <ResourceBrowserSampleData
                                data={queryResults}
                                className="grow"
                            />
                        );
                    default:
                        throw new Error("invalid primary resource type");
                }
            })()}
        </div>
    );
}

class SampleBrowserState {
    [immerable] = true;
    samples: app.ProjectSample[];
    sample_property_keys: string[];
    active_sample?: UUID;

    constructor(project_resources: app.ProjectResources, samples: UUID[]) {
        this.samples = project_resources.Samples.filter(
            (sample) => samples.findIndex((id) => sample.Id === id) > -1
        );

        this.sample_property_keys = this.samples
            .flatMap((sample) =>
                sample.Properties.map((property) => property.Key)
            )
            .sort()
            .reduce((keys, key) => {
                if (keys[keys.length - 1] !== key) {
                    keys.push(key);
                }
                return keys;
            }, [] as string[]);
    }

    grid_template_columns(): string {
        const cols = ["auto"];
        for (const col of this.sample_property_keys) {
            cols.push("auto");
        }

        return cols.join(" ");
    }
}

type SampleBrowserStateAction =
    | {
          type: "toggle_sample_active_state";
          payload: {
              sample: UUID;
              state?: boolean;
          };
      }
    | { type: "clear_active_sample" };

function sample_browser_state_reducer(
    draft: SampleBrowserState,
    action: SampleBrowserStateAction
) {
    switch (action.type) {
        case "toggle_sample_active_state":
            if (action.payload.state === true) {
                draft.active_sample = action.payload.sample;
            } else if (
                action.payload.state === false &&
                draft.active_sample === action.payload.sample
            ) {
                draft.active_sample = undefined;
            } else if (
                draft.active_sample &&
                draft.active_sample === action.payload.sample
            ) {
                draft.active_sample = undefined;
            } else {
                draft.active_sample = action.payload.sample;
            }
            break;
        case "clear_active_sample":
            draft.active_sample = undefined;
            break;
    }
}

const SampleBrowserStateCtx = createContext(
    new SampleBrowserState(new app.ProjectResources(), [])
);

const SampleBrowserStateDispatchCtx = createContext<
    ActionDispatch<[SampleBrowserStateAction]>
>(() => {});

interface ResourceBrowserSamplesProps {
    project_resources: app.ProjectResources;
    samples: UUID[];
    className?: string;
}
function ResourceBrowserSamples({
    project_resources,
    samples,
    className,
}: ResourceBrowserSamplesProps) {
    const [state, stateDispatch] = useImmerReducer(
        sample_browser_state_reducer,
        new SampleBrowserState(project_resources, samples)
    );

    return (
        <SampleBrowserStateCtx value={state}>
            <SampleBrowserStateDispatchCtx value={stateDispatch}>
                <div className={`flex ${className ?? ""}`}>
                    <div className="grow">
                        <ResourceBrowserSamplesHeader />
                        <ol
                            className="grow grid"
                            style={{
                                gridTemplateColumns:
                                    state.grid_template_columns(),
                            }}
                        >
                            {state.samples.map((sample, idx) => (
                                <ResourceBrowserSamplesListitem
                                    key={sample.Id}
                                    index={idx}
                                    sample={sample}
                                />
                            ))}
                        </ol>
                    </div>
                    {state.active_sample ? (
                        <SampleDetail
                            sample={
                                state.samples.find(
                                    (sample) =>
                                        sample.Id === state.active_sample
                                )!
                            }
                            onClose={() =>
                                stateDispatch({
                                    type: "clear_active_sample",
                                })
                            }
                            className="h-full border-l min-w-[20%]"
                        />
                    ) : null}
                </div>
            </SampleBrowserStateDispatchCtx>
        </SampleBrowserStateCtx>
    );
}

const SAMPLE_BROWSER_LABEL_COL = 1;
const SAMPLE_BROWSER_PROPERTIES_COL_OFFSET = 1;

interface ResourceBrowserSamplesHeaderProps {
    className?: string;
}
function ResourceBrowserSamplesHeader({
    className,
}: ResourceBrowserSamplesHeaderProps) {
    const state = useContext(SampleBrowserStateCtx);

    return (
        <div
            className={`grid ${className ?? ""}`}
            style={{
                gridTemplateColumns: state.grid_template_columns(),
            }}
        >
            <div
                className="grid-row-1"
                style={{
                    gridColumnStart: SAMPLE_BROWSER_PROPERTIES_COL_OFFSET + 1,
                    gridColumnEnd:
                        SAMPLE_BROWSER_PROPERTIES_COL_OFFSET +
                        state.sample_property_keys.length +
                        1,
                }}
            >
                Properties
            </div>
            {state.sample_property_keys.map((key, idx) => (
                <div
                    key={key}
                    className="grid-row-2"
                    style={{
                        gridColumn:
                            idx + SAMPLE_BROWSER_PROPERTIES_COL_OFFSET + 1,
                    }}
                >
                    {key}
                </div>
            ))}
        </div>
    );
}

interface ResourceBrowserSamplesListitem {
    index: number;
    sample: app.ProjectSample;
}
function ResourceBrowserSamplesListitem({
    index,
    sample,
}: ResourceBrowserSamplesListitem) {
    const state = useContext(SampleBrowserStateCtx);
    const stateDispatch = useContext(SampleBrowserStateDispatchCtx);
    const [isActive, setIsActive] = useState(false);

    useEffect(() => {
        setIsActive(state.active_sample === sample.Id);
    }, [state.active_sample]);

    function toggle_active_state(e: MouseEvent<HTMLLIElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        stateDispatch({
            type: "toggle_sample_active_state",
            payload: { sample: sample.Id },
        });
    }

    return (
        <li
            onMouseDown={toggle_active_state}
            className={classNames({
                "grid grid-cols-subgrid col-span-full cursor-pointer \
                hover:bg-gray-50 dark:hover:bg-gray-900 px-4": true,
                "bg-gray-50 dark:bg-gray-900": isActive,
            })}
            style={{ gridRow: index + 1 }}
        >
            <div style={{ gridColumn: SAMPLE_BROWSER_LABEL_COL }}>
                {sample.Label}
            </div>
            {state.sample_property_keys.map((property, idx) => {
                const sample_prop = sample.Properties.find(
                    (sample_prop) => sample_prop.Key === property
                );
                if (sample_prop === undefined) {
                    return <div key={property}></div>;
                } else {
                    return (
                        <div
                            key={property}
                            style={{
                                gridColumn:
                                    idx +
                                    SAMPLE_BROWSER_PROPERTIES_COL_OFFSET +
                                    1,
                            }}
                        >
                            <SamplePropertyValue
                                type={sample_prop.Type}
                                value={sample_prop.Value}
                            />
                        </div>
                    );
                }
            })}
        </li>
    );
}

interface PropertyValueProps {
    type: app.PropertyType;
    value: any;
}
function SamplePropertyValue({ type, value }: PropertyValueProps) {
    let val_typed;
    switch (type) {
        case app.PropertyType.PROPERTY_TYPE_BOOL:
            val_typed = value as boolean;
            if (val_typed) {
                return "true";
            } else {
                return "false";
            }
        case app.PropertyType.PROPERTY_TYPE_INT:
            val_typed = value as number;
            return val_typed.toString();
        case app.PropertyType.PROPERTY_TYPE_UINT:
            val_typed = value as number;
            return val_typed.toString();
        case app.PropertyType.PROPERTY_TYPE_FLOAT:
            val_typed = value as number;
            return val_typed.toString();
        case app.PropertyType.PROPERTY_TYPE_STRING:
            return value;
        case app.PropertyType.PROPERTY_TYPE_QUANTITY:
            val_typed = value as common.QuantityProperty;
            return `${val_typed.MagnitudeString} ${val_typed.Unit}`;
        case app.PropertyType.PROPERTY_TYPE_TIMESTAMP:
            return;
    }
}

interface SampleDetailProps {
    sample: app.ProjectSample;
    onClose: () => void;
    className?: string;
}
function SampleDetail({ sample, onClose, className }: SampleDetailProps) {
    return (
        <ErrorBoundary FallbackComponent={SampleDetailError}>
            <Suspense
                fallback={
                    <SampleDetailLoading
                        sample={sample}
                        className={className}
                    />
                }
            >
                <SampleDetailInner
                    sample={sample}
                    onClose={onClose}
                    className={className}
                />
            </Suspense>
        </ErrorBoundary>
    );
}

function SampleDetailError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <div>
            <div>Could not load sample details.</div>
            <div>{error.message}</div>
        </div>
    );
}

interface SampleDetailLoadingProps {
    sample: app.ProjectSample;
    className?: string;
}
function SampleDetailLoading({ sample, className }: SampleDetailLoadingProps) {
    return (
        <div className={className ?? ""}>
            <h3 className="grow text-xl font-bold">{sample.Label}</h3>
            <div className="text-center">Loading</div>
        </div>
    );
}

interface SampleDetailInnerProps {
    sample: app.ProjectSample;
    onClose: () => void;
    className?: string;
}
function SampleDetailInner({
    sample,
    onClose,
    className,
}: SampleDetailInnerProps) {
    const project_data = useContext(CommonProjectDataCtx);
    const { data: sample_resources } = useSuspenseQuery({
        queryKey: ["project_sample_resources", sample.Id],
        queryFn: async () =>
            app.ProjectService.GetProjectSampleResources(
                project_data.project_id,
                sample.Id
            ),
    });

    const [expandedProperties, setExpandedProperties] = useState(true);
    const [expandedData, setExpandedData] = useState(false);
    const [expandedNotes, setExpandedNotes] = useState(false);
    const [dataSelected, setDataSelected] = useState<UUID[]>([]);

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        onClose();
    }

    function toggle_expand_properties(e: MouseEvent<HTMLDivElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        setExpandedProperties(!expandedProperties);
    }

    function toggle_expand_data(e: MouseEvent<HTMLDivElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        setExpandedData(!expandedData);
    }

    function toggle_expand_notes(e: MouseEvent<HTMLDivElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        setExpandedNotes(!expandedNotes);
    }

    function set_selected_data(data: UUID, selected: boolean) {
        if (selected) {
            setDataSelected([...dataSelected, data]);
        } else {
            const selected = dataSelected.filter(
                (selected) => selected !== data
            );
            setDataSelected(selected);
        }
    }

    function download_selected_data(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }
        e.stopPropagation();

        console.debug("download", dataSelected);
        // TODO
    }

    return (
        <div className={`bg-white dark:bg-black ${className ?? ""}`}>
            <div className="flex gap-2 px-2">
                <h3 className="grow text-xl font-bold">{sample.Label}</h3>
                <div>
                    <button
                        type="button"
                        onMouseDown={close}
                        className="btn-cmd"
                    >
                        <icon.Close />
                    </button>
                </div>
            </div>
            <div>
                <div className="border-b">
                    <div
                        onMouseDown={toggle_expand_properties}
                        className="flex gap-2 cursor-pointer"
                    >
                        <button
                            type="button"
                            className={classNames({
                                "btn-cmd": true,
                                "-rotate-90": !expandedProperties,
                            })}
                        >
                            <icon.CaretDown />
                        </button>
                        <h4>Properties</h4>
                    </div>
                    <div
                        className={classNames({
                            "overflow-hidden": true,
                            "h-0": !expandedProperties,
                            "pb-2": expandedProperties,
                        })}
                    >
                        {sample.Properties.length > 0 ? (
                            <SampleDetailProperties
                                properties={sample.Properties}
                            />
                        ) : (
                            <div className="px-2">(no properties)</div>
                        )}
                    </div>
                </div>
                <div className="border-b">
                    <div
                        onMouseDown={toggle_expand_data}
                        className="flex gap-2 cursor-pointer"
                    >
                        <button
                            type="button"
                            className={classNames({
                                "btn-cmd": true,
                                "-rotate-90": !expandedData,
                            })}
                        >
                            <icon.CaretDown />
                        </button>
                        <h4>Data ({sample_resources.Data.length})</h4>
                        <div
                            className={classNames({
                                hidden: dataSelected.length === 0,
                            })}
                        >
                            <button
                                type="button"
                                className="btn-cmd"
                                onMouseDown={download_selected_data}
                            >
                                <icon.Download />
                            </button>
                        </div>
                    </div>
                    <div
                        className={classNames({
                            "overflow-hidden": true,
                            "h-0": !expandedData,
                            "pb-2": expandedData,
                        })}
                    >
                        {sample_resources.Data.length > 0 ? (
                            <SampleDetailData
                                data={sample_resources.Data}
                                data_schemas={sample_resources.DataSchemas}
                                users={sample_resources.Users}
                                onDataSelectionChange={set_selected_data}
                            />
                        ) : (
                            <div className="px-2">(no sample data)</div>
                        )}
                    </div>
                </div>
                <div className="border-b">
                    <div
                        onMouseDown={toggle_expand_notes}
                        className="flex gap-2 cursor-pointer"
                    >
                        <button
                            type="button"
                            className={classNames({
                                "btn-cmd": true,
                                "-rotate-90": !expandedNotes,
                            })}
                        >
                            <icon.CaretDown />
                        </button>
                        <h4>Notes ({sample_resources.ProjectNotes.length})</h4>
                    </div>
                    <div
                        className={classNames({
                            "overflow-hidden": true,
                            "h-0": !expandedNotes,
                            "pb-2": expandedNotes,
                        })}
                    >
                        {sample_resources.ProjectNotes.length > 0 ? (
                            <SampleDetailProjectNotes
                                notes={sample_resources.ProjectNotes}
                                users={sample_resources.Users}
                            />
                        ) : (
                            <div className="px-2">(no notes)</div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}

interface SampleDetailPropertiesProps {
    properties: app.Property[];
}
function SampleDetailProperties({ properties }: SampleDetailPropertiesProps) {
    const properties_sorted = properties.toSorted((a, b) => {
        if (a.Key < b.Key) {
            return -1;
        } else if (a.Key > b.Key) {
            return 0;
        } else {
            console.error(`duplicate property key ${a.Key}`);
            return 0;
        }
    });
    return (
        <ol className="grid grid-cols-[auto_auto]">
            {properties_sorted.map((property, index) => {
                return (
                    <li
                        key={property.Key}
                        className="grid col-span-full grid-cols-subgrid"
                        style={{
                            gridRow: index,
                        }}
                    >
                        <div className="col-1 pl-2">{property.Key}</div>
                        <div className="col-2 pr-2">
                            <SamplePropertyValue
                                type={property.Type}
                                value={property.Value}
                            />
                        </div>
                    </li>
                );
            })}
        </ol>
    );
}

interface SampleDetailDataProps {
    data: app.SampleData[];
    data_schemas: app.DataSchema[];
    users: app.User[];
    onDataSelectionChange: (data: UUID, selected: boolean) => void;
}
function SampleDetailData({
    data,
    data_schemas,
    users,
    onDataSelectionChange,
}: SampleDetailDataProps) {
    return (
        <ol>
            {data.map((datum, index) => {
                const data_schema_idx = data_schemas.findIndex(
                    (schema) => schema.Id === datum.Schema
                );
                if (data_schema_idx < 0) {
                    console.error(`could not find data schema ${datum.Schema}`);
                }

                const creator_idx = users.findIndex(
                    (user) => user.Id === datum.Creator
                );
                if (creator_idx < 0) {
                    console.error(`could not find user ${datum.Creator}`);
                }

                return (
                    <SampleDetailDataListItem
                        key={datum.Id}
                        index={index}
                        data={datum}
                        data_schema={data_schemas[data_schema_idx]}
                        creator={users[creator_idx]}
                        onSelectionChange={onDataSelectionChange}
                    />
                );
            })}
        </ol>
    );
}

interface SampleDetailDataListItemProps {
    index: number;
    data: app.SampleData;
    data_schema: app.DataSchema;
    creator: app.User;
    onSelectionChange: (data: UUID, selected: boolean) => void;
}
function SampleDetailDataListItem({
    data,
    data_schema,
    creator,
    onSelectionChange,
}: SampleDetailDataListItemProps) {
    const app_state = useContext(appStateCtx.Context);
    const timestamp = new Date(data.Timestamp);

    function on_selected(e: ChangeEvent<HTMLInputElement>) {
        const target = e.target as HTMLInputElement;
        onSelectionChange(data.Id, target.checked);
    }

    async function download(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        await app.DataService.SaveSampleDataSingle(data.Id)
            .then((path) => console.info(`data ${data.Id} saved to ${path}`))
            .catch((err) => {
                if (err.message === "CANCELLED_BY_USER") {
                    return;
                }

                console.error(err);
            });
    }

    return (
        <li className="px-2 flex gap-2">
            <div className="flex gap-2">
                <div>
                    <label>
                        <span className="hidden">Select data</span>
                        <input
                            type="checkbox"
                            id={`data[${data.Id}][select]`}
                            name={`data[${data.Id}][select]`}
                            onChange={on_selected}
                        />
                    </label>
                </div>
                <div>
                    <button
                        type="button"
                        onMouseDown={download}
                        className="btn-cmd"
                    >
                        <icon.Download />
                    </button>
                </div>
            </div>
            <div>{data_schema.Label}</div>
            <div>{timestamp.toLocaleString()}</div>
            <div>({creator.Name})</div>
        </li>
    );
}

interface SampleDetailProjectNotesProps {
    notes: app.ProjectSampleNote[];
    users: app.User[];
}
function SampleDetailProjectNotes({
    notes,
    users,
}: SampleDetailProjectNotesProps) {
    return (
        <ol>
            {notes.map((note, index) => {
                const creator_idx = users.findIndex(
                    (user) => user.Id === note.Creator
                );
                if (creator_idx < 0) {
                    console.error(`could not find user ${note.Creator}`);
                }

                return (
                    <SampleDetailProjectNoteListItem
                        key={note.Id}
                        note={note}
                        creator={users[creator_idx]}
                    />
                );
            })}
        </ol>
    );
}

interface SampleDetailProjectNoteListItemProps {
    note: app.ProjectSampleNote;
    creator: app.User;
}
function SampleDetailProjectNoteListItem({
    note,
    creator,
}: SampleDetailProjectNoteListItemProps) {
    const appState = useContext(appStateCtx.Context);

    return (
        <li>
            <div>
                <div>{note.Timestamp}</div>
                <div>
                    {appState.user.Id === creator.Id ? "you" : creator.Name}
                </div>
            </div>
            <div>{note.Content}</div>
        </li>
    );
}

interface ResourceBrowserSampleDataProps {
    data: UUID[];
    className?: string;
}
function ResourceBrowserSampleData({
    data,
    className,
}: ResourceBrowserSampleDataProps) {
    console.debug(data);
    return <div>data results</div>;
}

interface ProjectSamplePropertyProps {
    property: app.Property;
}
function ProjectSampleProperty({ property }: ProjectSamplePropertyProps) {
    const title = `${property.Key} (${property.Type})`;
    return (
        <div title={title}>
            <span className="font-bold">{property.Key}</span>:{" "}
            <span>
                <ProjectSamplePropertyValue
                    type={property.Type}
                    value={property.Value}
                />
            </span>
        </div>
    );
}

interface ProjectSamplePropertyValueProps {
    type: app.PropertyType;
    value: any;
}
function ProjectSamplePropertyValue({
    type,
    value,
}: ProjectSamplePropertyValueProps) {
    switch (type) {
        case app.PropertyType.PROPERTY_TYPE_STRING:
        case app.PropertyType.PROPERTY_TYPE_INT:
        case app.PropertyType.PROPERTY_TYPE_UINT:
        case app.PropertyType.PROPERTY_TYPE_FLOAT:
            return <>value</>;
        case app.PropertyType.PROPERTY_TYPE_BOOL:
            switch (value) {
                case true:
                    return <>true</>;
                case false:
                    return <>false</>;
                default:
                    throw new Error("incompatible sample property value");
            }
            break;
        case app.PropertyType.PROPERTY_TYPE_QUANTITY:
            return (
                <>
                    <span>{value.MagnitudeString}</span>{" "}
                    <span>{value.Unit}</span>
                </>
            );
        case app.PropertyType.PROPERTY_TYPE_TIMESTAMP:
            return <>value</>;
    }
}
