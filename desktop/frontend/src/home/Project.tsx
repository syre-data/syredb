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
import * as property from "../components/Property";
import { SelectPropertyType, InputPropertyValue } from "../components/Property";

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
    const { project_id } = useParams();
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
    const err = error as common.BackendError;
    const navigate = useNavigate();

    if (err.message === common.USER_NOT_AUTHENTICATED_ERROR) {
        console.error(common.USER_NOT_AUTHENTICATED_ERROR);
        navigate("/");
        return null;
    } else {
        console.error(err);
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
            <div>{err.message}</div>
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
                    <div className="flex gap-2">
                        <div>
                            <Link
                                to={`/project/${project_info.project_id}/samples/create`}
                            >
                                <button
                                    type="button"
                                    className="flex gap-1 items-center border rounded-full \
                                whitespace-nowrap px-2 cursor-pointer"
                                    title="Create new samples"
                                >
                                    <icon.Plus className="inline" /> Create
                                    samples
                                </button>
                            </Link>
                        </div>
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
        PrimaryResourceType.Sample,
    );
    const [queryResults, setQueryResults] = useState<UUID[]>(
        project_resources.Samples.map((sample) => sample.Id),
    );

    useEffect(() => {
        const sample_ids = project_resources.Samples.map((sample) => sample.Id);
        setQueryResults(sample_ids);
    }, [project_resources.Samples]);

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
                    `invalid primary resource type: ${input.value}`,
                );
        }
    }

    function query_resources(e: FormEvent) {
        e.preventDefault();
    }

    async function download_all_project_data(
        hierarchy: app.SaveDataHierarchy[],
    ) {
        await app.DataService.SaveProjectDataAll(
            project_resources.Project.Id,
            hierarchy,
        )
            .then((path) => {
                console.info(`saved all project data to ${path}`);
            })
            .catch((err) => {
                console.error("could not save project data", err);
            });
    }

    return (
        <div className={`flex flex-col ${className ?? ""}`}>
            <div className="pb-2 flex">
                <form
                    onSubmit={query_resources}
                    className="grow flex gap-2 px-4"
                >
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
                <div className="flex gap-1 pr-1">
                    <ResourceBrowserDownloadDataBtn
                        onDownload={download_all_project_data}
                        disabled={project_resources.SampleData.length === 0}
                    />
                </div>
            </div>
            {(() => {
                switch (primaryResourceType) {
                    case PrimaryResourceType.Sample:
                        return (
                            <ResourceBrowserSamples
                                project_resources={project_resources}
                                filter={queryResults}
                                className="grow"
                            />
                        );
                    case PrimaryResourceType.Data:
                        return (
                            <ResourceBrowserSampleData
                                project_resources={project_resources}
                                filter={queryResults}
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

interface ResourceBrowserDownloadDataBtnProps {
    onDownload: (hierarchy: app.SaveDataHierarchy[]) => Promise<void>;
    disabled: boolean;
}
function ResourceBrowserDownloadDataBtn({
    onDownload,
    disabled,
}: ResourceBrowserDownloadDataBtnProps) {
    const [align, setAlign] = useState<"left" | "right">("left");
    const menuRef = useRef<HTMLDivElement>(null);

    useLayoutEffect(() => {
        const menu = menuRef.current;
        if (!menu) return;

        const rect = menu.getBoundingClientRect();
        const viewportWidth = window.innerWidth;

        if (rect.right > viewportWidth) {
            setAlign("right");
        } else if (rect.left < 0) {
            setAlign("left");
        }
    }, []);

    function on_download(
        e: MouseEvent<HTMLButtonElement>,
        hierarchy: app.SaveDataHierarchy[],
    ) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        onDownload(hierarchy);
    }

    const MENU_LIST_ITEM_CLASS =
        "hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer";
    const MENU_LIST_ITEM_BTN_CLASS =
        "w-full h-full px-4 py-2 text-left cursor-pointer whitespace-nowrap";
    return (
        <div className="flex gap-1 cursor-pointer">
            <button
                type="button"
                className="cursor-pointer"
                title="Download all project data"
                disabled={disabled}
                onMouseDown={(e) => on_download(e, [])}
            >
                <icon.Download />
            </button>
            <div className="relative group/menu">
                <button type="button" className="align-middle cursor-pointer">
                    <icon.CaretDown />
                </button>
                <div
                    ref={menuRef}
                    className={classNames({
                        "invisible absolute top-full border bg-white dark:bg-black \
                        transition-[visibility] delay-200": true,
                        "group-hover/menu:visible": !disabled,
                        "left-0": align === "left",
                        "right-0": align === "right",
                    })}
                >
                    <ol>
                        <li className={MENU_LIST_ITEM_CLASS}>
                            <button
                                type="button"
                                className={MENU_LIST_ITEM_BTN_CLASS}
                                title="Download all project data in the same folder"
                                onMouseDown={(e) => on_download(e, [])}
                            >
                                Flat
                            </button>
                        </li>
                        <li className={MENU_LIST_ITEM_CLASS}>
                            <button
                                type="button"
                                className={MENU_LIST_ITEM_BTN_CLASS}
                                title="Download all project data organized by sample"
                                onMouseDown={(e) =>
                                    on_download(e, [
                                        app.SaveDataHierarchy
                                            .SAVE_DATA_HIERARCHY_SAMPLE,
                                    ])
                                }
                            >
                                Sample
                            </button>
                        </li>
                        <li className={MENU_LIST_ITEM_CLASS}>
                            <button
                                type="button"
                                className={MENU_LIST_ITEM_BTN_CLASS}
                                title="Download all project data organized by data schema"
                                onMouseDown={(e) =>
                                    on_download(e, [
                                        app.SaveDataHierarchy
                                            .SAVE_DATA_HIERARCHY_DATA_SCHEMA,
                                    ])
                                }
                            >
                                Data schema
                            </button>
                        </li>
                        <li className={MENU_LIST_ITEM_CLASS}>
                            <button
                                type="button"
                                className={MENU_LIST_ITEM_BTN_CLASS}
                                title="Download all project data organized by sample then data schema"
                                onMouseDown={(e) =>
                                    on_download(e, [
                                        app.SaveDataHierarchy
                                            .SAVE_DATA_HIERARCHY_SAMPLE,
                                        app.SaveDataHierarchy
                                            .SAVE_DATA_HIERARCHY_DATA_SCHEMA,
                                    ])
                                }
                            >
                                Sample &gt; Data schema
                            </button>
                        </li>
                        <li className={MENU_LIST_ITEM_CLASS}>
                            <button
                                type="button"
                                className={MENU_LIST_ITEM_BTN_CLASS}
                                title="Download all project data organized by data schema then sample"
                                onMouseDown={(e) =>
                                    on_download(e, [
                                        app.SaveDataHierarchy
                                            .SAVE_DATA_HIERARCHY_DATA_SCHEMA,
                                        app.SaveDataHierarchy
                                            .SAVE_DATA_HIERARCHY_SAMPLE,
                                    ])
                                }
                            >
                                Data schema &gt; Sample
                            </button>
                        </li>
                    </ol>
                </div>
            </div>
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
            (sample) => samples.findIndex((id) => sample.Id === id) > -1,
        );

        this.sample_property_keys = this.samples
            .flatMap((sample) =>
                sample.Properties.map((property) => property.Key),
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
        const cols = ["min-content"];
        for (const col of this.sample_property_keys) {
            cols.push("min-content");
        }

        return cols.join(" ");
    }
}

type SampleBrowserStateAction =
    | {
          type: "set_project_resources";
          payload: { project_resources: app.ProjectResources; filter: UUID[] };
      }
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
    action: SampleBrowserStateAction,
) {
    switch (action.type) {
        case "set_project_resources":
            draft.samples = action.payload.project_resources.Samples.filter(
                (sample) =>
                    action.payload.filter.findIndex((id) => sample.Id === id) >
                    -1,
            );

            draft.sample_property_keys = draft.samples
                .flatMap((sample) =>
                    sample.Properties.map((property) => property.Key),
                )
                .sort()
                .reduce((keys, key) => {
                    if (keys[keys.length - 1] !== key) {
                        keys.push(key);
                    }
                    return keys;
                }, [] as string[]);
            break;
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
    new SampleBrowserState(new app.ProjectResources(), []),
);

const SampleBrowserStateDispatchCtx = createContext<
    ActionDispatch<[SampleBrowserStateAction]>
>(() => {});

interface ResourceBrowserSamplesProps {
    project_resources: app.ProjectResources;
    filter: UUID[];
    className?: string;
}
function ResourceBrowserSamples({
    project_resources,
    filter,
    className,
}: ResourceBrowserSamplesProps) {
    const [state, stateDispatch] = useImmerReducer(
        sample_browser_state_reducer,
        new SampleBrowserState(project_resources, filter),
    );

    useEffect(() => {
        stateDispatch({
            type: "set_project_resources",
            payload: { project_resources: project_resources, filter: filter },
        });
    }, [project_resources.Samples, filter]);

    return (
        <SampleBrowserStateCtx value={state}>
            <SampleBrowserStateDispatchCtx value={stateDispatch}>
                {state.samples.length === 0 ? (
                    <ResourceBrowserSamplesEmpty />
                ) : (
                    <ResourceBrowserSamplesInner className={className} />
                )}
            </SampleBrowserStateDispatchCtx>
        </SampleBrowserStateCtx>
    );
}

function ResourceBrowserSamplesEmpty() {
    return (
        <div className="px-4">This project doesn't have any samples yet.</div>
    );
}

interface ResourceBrowserSamplesInnerProps {
    className?: string;
}
function ResourceBrowserSamplesInner({
    className,
}: ResourceBrowserSamplesInnerProps) {
    const state = useContext(SampleBrowserStateCtx);
    const stateDispatch = useContext(SampleBrowserStateDispatchCtx);
    return (
        <div className={`flex ${className ?? ""}`}>
            <div
                className="grow grid gap-x-4"
                style={{
                    gridTemplateColumns:
                        state.grid_template_columns() + " auto",
                    gridTemplateRows: "min-content min-content",
                }}
            >
                <ResourceBrowserSamplesHeader className="row-1" />
                <ol className="row-2 col-span-full grid grid-cols-subgrid">
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
                            (sample) => sample.Id === state.active_sample,
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
            className={`col-span-full grid grid-cols-subgrid ${
                className ?? ""
            }`}
        >
            <div
                className="row-1"
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
                    className="row-2"
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
            className={classNames({
                "grid grid-cols-subgrid col-span-full cursor-pointer \
                hover:bg-gray-50 dark:hover:bg-gray-900 px-4 \
                [&.file-drop-target-active]:bg-gray-50 dark:[&.file-drop-target-active]:bg-gray-900":
                    true,
                "bg-gray-50 dark:bg-gray-900": isActive,
            })}
            style={{ gridRow: index + 1 }}
            data-sample-id={sample.Id}
            data-file-drop-target
            onMouseDown={toggle_active_state}
        >
            <div style={{ gridColumn: SAMPLE_BROWSER_LABEL_COL }}>
                {sample.Label}
            </div>
            {state.sample_property_keys.map((property, idx) => {
                const sample_prop = sample.Properties.find(
                    (sample_prop) => sample_prop.Key === property,
                );
                if (sample_prop === undefined) {
                    return <div key={property}></div>;
                } else {
                    return (
                        <div
                            key={property}
                            className="whitespace-nowrap"
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
            val_typed = value as property.QuantityProperty;
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
    const err = error as common.BackendError;
    return (
        <div>
            <div>Could not load sample details.</div>
            <div>{err.message}</div>
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
                sample.Id,
            ),
    });

    const [expandedProperties, setExpandedProperties] = useState(true);
    const [expandedData, setExpandedData] = useState(false);
    const [expandedNotes, setExpandedNotes] = useState(false);
    const [dataSelected, setDataSelected] = useState<UUID[]>([]);
    const [newProperties, setNewProperties] = useState<number[]>([]);

    useEffect(() => {
        setExpandedProperties(true);
        setExpandedData(false);
        setExpandedNotes(false);
        setDataSelected([]);
    }, [sample.Id]);

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
                (selected) => selected !== data,
            );
            setDataSelected(selected);
        }
    }

    async function download_selected_data(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }
        e.stopPropagation();

        if (dataSelected.length === 0) {
            console.error("attempted to download 0 data resources");
        } else if (dataSelected.length === 1) {
            await app.DataService.SaveSampleDataSingle(dataSelected[0])
                .then((path) => {
                    console.info(`data ${dataSelected[0]} saved to ${path}`);
                    setDataSelected([]);
                })
                .catch((err) => {
                    console.error(err);
                });
        } else {
            await app.DataService.SaveSampleDataMultiple(
                dataSelected,
                project_data.project_id,
                [],
            )
                .then((path) => {
                    console.info(`data saved to ${path}`, dataSelected);
                    setDataSelected([]);
                })
                .catch((err) => {
                    console.error(err);
                });
        }
    }

    function add_sample_property(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }
        e.stopPropagation();

        if (newProperties.length === 0) {
            setNewProperties([0]);
        } else {
            const next_id = newProperties[newProperties.length - 1] + 1;
            setNewProperties([...newProperties, next_id]);
        }
    }

    return (
        <div className={`bg-white dark:bg-black ${className ?? ""}`}>
            <div className="flex gap-2 px-2">
                <div className="grow flex gap-2">
                    <h3 className="text-xl font-bold">{sample.Label}</h3>
                    <div>
                        <Link
                            to={`/project/{${project_data.project_id}}/sample/${sample.Id}/edit`}
                        >
                            <button type="button" className="btn-cmd">
                                <icon.Pen />
                            </button>
                        </Link>
                    </div>
                </div>
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
                                title="Download selected sample data"
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
                                selectedData={dataSelected}
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
        <ol className="grid grid-cols-[min-content_min-content] gap-x-2">
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

interface SampleDetailNewPropertyProps {
    id: number;
}
function SampleDetailNewProperty({ id }: SampleDetailNewPropertyProps) {
    const [type, setType] = useState<app.PropertyType>(
        app.PropertyType.PROPERTY_TYPE_STRING,
    );

    function add_property(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
    }

    function set_type(e: ChangeEvent<HTMLSelectElement>) {
        const type_value = property.type_string_to_variant(e.target.value);
        if (type_value === undefined) {
            console.error(`invlaid property type string ${e.target.value}`);
            return;
        }

        setType(type_value);
    }

    return (
        <form className="flex gap-2 px-2" onSubmit={add_property}>
            <div>
                <label>
                    <span className="hidden">Property key</span>
                    <input
                        type="text"
                        className="input-basic"
                        placeholder="Label"
                    />
                </label>
            </div>
            <div>
                <label>
                    <span className="hidden">Property label</span>
                    <SelectPropertyType
                        className="input-basic"
                        onChange={set_type}
                    />
                </label>
            </div>
            <div>
                <label>
                    <span className="hidden">Property value</span>
                    <InputPropertyValue type={type} className="input-basic" />
                </label>
            </div>
        </form>
    );
}

interface SampleDetailDataProps {
    data: app.SampleData[];
    data_schemas: app.DataSchema[];
    users: app.User[];
    selectedData: UUID[];
    onDataSelectionChange: (data: UUID, selected: boolean) => void;
}
function SampleDetailData({
    data,
    data_schemas,
    users,
    selectedData,
    onDataSelectionChange,
}: SampleDetailDataProps) {
    return (
        <ol>
            {data.map((datum, index) => {
                const data_schema_idx = data_schemas.findIndex(
                    (schema) => schema.Id === datum.Schema,
                );
                if (data_schema_idx < 0) {
                    console.error(`could not find data schema ${datum.Schema}`);
                }

                const creator_idx = users.findIndex(
                    (user) => user.Id === datum.Creator,
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
                        selected={selectedData.includes(datum.Id)}
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
    selected: boolean;
    onSelectionChange: (data: UUID, selected: boolean) => void;
}
function SampleDetailDataListItem({
    data,
    data_schema,
    creator,
    selected,
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
                            checked={selected}
                        />
                    </label>
                </div>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        title="Download sample data"
                        onMouseDown={download}
                    >
                        <icon.Download />
                    </button>
                </div>
            </div>
            <div>{data_schema.Label}</div>
            <div className="whitespace-nowrap">
                {timestamp.toLocaleString()}
            </div>
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
                    (user) => user.Id === note.Creator,
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
                <div>{creator.Name}</div>
            </div>
            <div>{note.Content}</div>
        </li>
    );
}

interface ResourceBrowserSampleDataProps {
    project_resources: app.ProjectResources;
    filter: UUID[];
    className?: string;
}
function ResourceBrowserSampleData({
    project_resources,
    filter,
    className,
}: ResourceBrowserSampleDataProps) {
    const [dataSelected, setDataSelected] = useState<UUID[]>([]);
    const data_schema_sample_data = new Map<UUID, app.SampleData[]>();
    for (const sample_data of project_resources.SampleData) {
        const schema_datas = data_schema_sample_data.get(sample_data.Schema);
        if (schema_datas === undefined) {
            data_schema_sample_data.set(sample_data.Schema, [sample_data]);
        } else {
            schema_datas.push(sample_data);
        }
    }

    function toggle_data_selection(sample_data: UUID, selected: boolean) {
        if (selected && !dataSelected.includes(sample_data)) {
            setDataSelected([...dataSelected, sample_data]);
        } else if (!selected && dataSelected.includes(sample_data)) {
            const filtered = dataSelected.filter((id) => id !== sample_data);
            setDataSelected(filtered);
        }
    }

    const sample_data_arr = Array.from(data_schema_sample_data.entries());
    return sample_data_arr.length === 0 ? (
        <ResourceBrowserSampleDataEmpty />
    ) : (
        <div className={className ?? ""}>
            {sample_data_arr.map(([data_schema_id, schema_sample_data]) => {
                const data_schema = project_resources.DataSchemas.find(
                    (schema) => schema.Id === data_schema_id,
                )!;
                return (
                    <ResourceBrowserSampleDataSchemaGroup
                        project_resources={project_resources}
                        key={data_schema_id}
                        data_schema={data_schema}
                        sample_data={schema_sample_data}
                        selected_data={dataSelected}
                        onSelectionToggle={toggle_data_selection}
                    />
                );
            })}
        </div>
    );
}

function ResourceBrowserSampleDataEmpty() {
    return (
        <div className="px-4">
            This project doesn't have any sample data yet.
        </div>
    );
}

interface ResourceBrowserSampleDataSchemaGroupProps {
    project_resources: app.ProjectResources;
    data_schema: app.DataSchema;
    sample_data: app.SampleData[];
    selected_data: UUID[];
    onSelectionToggle: (sample_data: UUID, selected: boolean) => void;
}
function ResourceBrowserSampleDataSchemaGroup({
    project_resources,
    data_schema,
    sample_data,
    selected_data,
    onSelectionToggle,
}: ResourceBrowserSampleDataSchemaGroupProps) {
    const project_data = useContext(CommonProjectDataCtx);

    async function download_all_schema_data(
        hierarchy: app.SaveDataHierarchy[],
    ) {
        const sample_data_ids = sample_data.map((data) => data.Id);
        await app.DataService.SaveDataSchemaSampleDataAll(
            data_schema.Id,
            project_data.project_id,
            hierarchy,
        )
            .then((path) => {
                console.info(`data saved to ${path}`, sample_data_ids);
            })
            .catch((err) => {
                console.error("could not download data", err);
            });
    }

    return (
        <div>
            <div className="px-4 flex gap-2">
                <h4 className="text-lg font-bold">{data_schema.Label}</h4>
                <div>
                    <ResourceBrowserSampleDataSchemaGroupDownloadDataBtn
                        onDownload={download_all_schema_data}
                    />
                </div>
            </div>
            <ol
                className="grid gap-x-4"
                style={{
                    gridTemplateColumns: "repeat(3, min-content) auto",
                }}
            >
                {sample_data.map((data, idx) => {
                    const sample = project_resources.Samples.find(
                        (sample) => sample.Id === data.Sample,
                    )!;

                    const selected = selected_data.includes(data.Id);
                    return (
                        <ResourceBrowserSampleDataSchemaGroupListItem
                            key={data.Id}
                            index={idx}
                            sample={sample}
                            sample_data={data}
                            selected={selected}
                            onSelectionToggle={onSelectionToggle}
                        />
                    );
                })}
            </ol>
        </div>
    );
}

interface ResourceBrowserSampleDataSchemaGroupDownloadDataBtnProps {
    onDownload: (hierarchy: app.SaveDataHierarchy[]) => Promise<void>;
}
function ResourceBrowserSampleDataSchemaGroupDownloadDataBtn({
    onDownload,
}: ResourceBrowserSampleDataSchemaGroupDownloadDataBtnProps) {
    const [align, setAlign] = useState<"left" | "right">("left");
    const menuRef = useRef<HTMLDivElement>(null);

    useLayoutEffect(() => {
        const menu = menuRef.current;
        if (!menu) return;

        const rect = menu.getBoundingClientRect();
        const viewportWidth = window.innerWidth;

        if (rect.right > viewportWidth) {
            setAlign("right");
        } else if (rect.left < 0) {
            setAlign("left");
        }
    }, []);

    function on_download(
        e: MouseEvent<HTMLButtonElement>,
        hierarchy: app.SaveDataHierarchy[],
    ) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        onDownload(hierarchy);
    }

    const MENU_LIST_ITEM_CLASS =
        "hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer";
    const MENU_LIST_ITEM_BTN_CLASS =
        "w-full h-full px-4 py-2 text-left cursor-pointer whitespace-nowrap";
    return (
        <div className="flex gap-1 cursor-pointer">
            <button
                type="button"
                className="cursor-pointer"
                title="Download all schema data"
                onMouseDown={(e) => on_download(e, [])}
            >
                <icon.Download />
            </button>
            <div className="relative group/menu">
                <button type="button" className="align-middle cursor-pointer">
                    <icon.CaretDown />
                </button>
                <div
                    ref={menuRef}
                    className={classNames({
                        "invisible group-hover/menu:visible absolute top-full border \
                        bg-white dark:bg-black transition-[visibility] delay-200":
                            true,
                        "left-0": align === "left",
                        "right-0": align === "right",
                    })}
                >
                    <ol>
                        <li className={MENU_LIST_ITEM_CLASS}>
                            <button
                                type="button"
                                className={MENU_LIST_ITEM_BTN_CLASS}
                                title="Download all project data in the same folder"
                                onMouseDown={(e) => on_download(e, [])}
                            >
                                Flat
                            </button>
                        </li>
                        <li className={MENU_LIST_ITEM_CLASS}>
                            <button
                                type="button"
                                className={MENU_LIST_ITEM_BTN_CLASS}
                                title="Download all project data organized by sample"
                                onMouseDown={(e) =>
                                    on_download(e, [
                                        app.SaveDataHierarchy
                                            .SAVE_DATA_HIERARCHY_SAMPLE,
                                    ])
                                }
                            >
                                Sample
                            </button>
                        </li>
                    </ol>
                </div>
            </div>
        </div>
    );
}

interface ResourceBrowserSampleDataSchemaGroupListItemProps {
    index: number;
    sample: app.ProjectSample;
    sample_data: app.SampleData;
    selected: boolean;
    onSelectionToggle: (sample_data: UUID, selected: boolean) => void;
}
function ResourceBrowserSampleDataSchemaGroupListItem({
    index,
    sample,
    sample_data,
    selected,
    onSelectionToggle,
}: ResourceBrowserSampleDataSchemaGroupListItemProps) {
    const timestamp = new Date(sample_data.Timestamp);

    function on_selection_change(e: ChangeEvent<HTMLInputElement>) {
        const input = e.target as HTMLInputElement;
        onSelectionToggle(sample_data.Id, input.checked);
    }

    return (
        <li
            className={classNames({
                "px-2 grid grid-cols-subgrid col-span-full cursor-pointer \
                group/sample-data-row \
                hover:bg-gray-50 dark:hover:bg-gray-900": true,
                "bg-gray-50 dark:bg-gray-900": selected,
            })}
            style={{
                gridRow: index + 1,
            }}
        >
            <div className="row-1 col-1">{sample.Label}</div>
            <div className="row-1 col-2 whitespace-nowrap">
                {timestamp.toLocaleString()}
            </div>
            <div
                className={classNames({
                    "flex gap-1 group-hover/sample-data-row:visible -col-1": true,
                    invisible: !selected,
                })}
            >
                <input
                    type="checkbox"
                    checked={selected}
                    onChange={on_selection_change}
                />
                <button type="button" className="btn-cmd" title="Download data">
                    <icon.Download />
                </button>
            </div>
        </li>
    );
}
