import {
    MouseButton,
    QUERY_KEY_DATA,
    QUERY_KEY_DATA_VALUES,
    QUERY_KEY_USER_PROJECTS,
    timestampToString,
    uuidToString,
} from "@/common";
import { Loading, StandardError, SuspenseError } from "@/components";
import { Note as NoteView } from "@/components/Note";
import { value_to_string } from "@/components/Property";
import { VisibilityBadge } from "@/components/Visibility";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import projectService from "@/service/project.service";
import {
    DataSchemaCardinalityMultiple,
    DataSchemaCardinalitySingle,
    DataSourceCardinalityMultiple,
    DataSourceCardinalitySingle,
    DataStorageExternal,
    DataStorageInternal,
    ValueTypeBoolean,
    ValueTypeFloat,
    ValueTypeInt,
    ValueTypeString,
    ValueTypeTimestamp,
    ValueTypeUint,
    type DataCreator,
    type DataCreatorTransformInfo,
    type DataCreatorUserInfo,
    type DataOriginRx,
    type DataProjectResources,
    type DataRx,
    type DataSchemaCardinality,
    type DataSource,
    type DataStorage,
    type DataType,
    type Note,
    type Property,
    type SchemaFieldValues,
    type User,
    type Visibility,
} from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";
import React, {
    Suspense,
    useState,
    type Dispatch,
    type MouseEvent,
    type SetStateAction,
    type SubmitEvent,
    type ComponentProps,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, Navigate, useNavigate, useParams } from "react-router";
import type { UUIDTypes } from "uuid";

export default function () {
    const { data_id } = useParams();
    if (!data_id) {
        return <Navigate to="/" replace />;
    }

    return (
        <ErrorBoundary FallbackComponent={LoadError}>
            <Suspense fallback={<Loading />}>
                <Data data_id={data_id} />
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

interface DataProps {
    data_id: UUIDTypes;
}
function Data({ data_id }: DataProps) {
    const { data: resources } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA, data_id],
        queryFn: async () => await dataService.dataGet(data_id),
    });

    const data = resources.Data as DataRx;
    const dataType = resources.DataType as DataType;
    const properties = resources.Properties as Property[];
    const notes = resources.Notes as Note[];
    const creator = resources.Creator as DataCreator;
    const projectResources =
        resources.ProjectResources as DataProjectResources[];
    const users = resources.Users as User[];

    return (
        <div>
            <Header
                data_id={data_id}
                dtype_label={dataType.Label}
                timestamp={timestampToString(data.Timestamp)}
                visibility={data.Visibility}
            />
            <InfoCards
                creator={creator}
                timestamp={data.Timestamp}
                className="px-4 py-2"
            />
            <Properties properties={properties} className="pt-4" />
            <Notes notes={notes} users={users} className="pt-4" />
            <ProjectResources
                dataId={data_id}
                projectResources={projectResources}
                users={users}
                className="pt-4"
            />
            <DataValues
                data={data_id}
                storage={dataType.Storage}
                className="pt-4"
            />
        </div>
    );
}

interface DataHeaderProps {
    data_id: UUIDTypes;
    dtype_label: string;
    timestamp: string;
    visibility: Visibility;
}
function Header({
    data_id,
    dtype_label,
    timestamp,
    visibility,
}: DataHeaderProps) {
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
                    <span className="text-sm">
                        <VisibilityBadge visibility={visibility} />
                    </span>
                </h1>
                <div className="invisible group-hover:visible">
                    <a
                        href={`/data/${data_id}/edit`}
                        title="Edit data"
                        className="align-middle"
                    >
                        <button type="button" className="btn-cmd">
                            <Icon.Pen />
                        </button>
                    </a>
                </div>
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

interface InfoCardsProps extends ComponentProps<"div"> {
    creator: DataCreator;
    timestamp: Date;
}
function InfoCards({
    creator,
    timestamp,
    className,
    ...props
}: InfoCardsProps) {
    return (
        <div
            className={`flex gap-4 divide-x divide-gray-400 ${className ?? ""}`}
            {...props}
        >
            <CreatorCard creator={creator} className="pr-4" />
            <TimestampCard timestamp={timestamp} className="pr-4" />
            {Object.hasOwn(creator, "Origin") ? (
                <OriginCard origin={creator.Origin} className="pr-4" />
            ) : null}
        </div>
    );
}

interface CreatorCardProps extends ComponentProps<"div"> {
    creator: DataCreator;
}
function CreatorCard({ creator, ...props }: CreatorCardProps) {
    if (Object.hasOwn(creator, "User")) {
        return <CreatorUser creator={creator} {...props} />;
    } else if (
        Object.hasOwn(creator, "Id") &&
        Object.hasOwn(creator, "Label") &&
        Object.hasOwn(creator, "Description")
    ) {
        return <CreatorTransform creator={creator} {...props} />;
    } else {
        console.debug(creator);
        throw new Error("invalid data creator");
    }
}

interface CreatorUserProps extends ComponentProps<"div"> {
    creator: DataCreatorUserInfo;
}
function CreatorUser({ creator, ...props }: CreatorUserProps) {
    return (
        <div {...props}>
            <div className="h-full inline-flex gap-2 py-2">
                <div>
                    <Icon.User />
                </div>
                <div>
                    <div className="text-xs text-secondary">Created by</div>
                    <div className="font-bold select-all">
                        {creator.User.Name || creator.User.Email}
                    </div>
                </div>
            </div>
        </div>
    );
}

interface OriginCardProps extends ComponentProps<"div"> {
    origin: DataOriginRx;
}
function OriginCard({ origin, ...props }: OriginCardProps) {
    return (
        <div {...props}>
            <div className="h-full inline-flex gap-2 py-2">
                <div>
                    <Icon.LocationPin />
                </div>
                <div>
                    <div className="text-xs text-secondary">Origin</div>
                    <div className="font-bold select-all">{origin.Label}</div>
                </div>
            </div>
        </div>
    );
}

interface CreatorTransformProps extends ComponentProps<"div"> {
    creator: DataCreatorTransformInfo;
}
function CreatorTransform({ creator, ...props }: CreatorTransformProps) {
    return (
        <div {...props}>
            <div className="h-full inline-flex gap-2 py-2">
                <div>
                    <Icon.Gears />
                </div>
                <div>
                    <div className="text-xs text-secondary">Created by</div>
                    <div className="font-bold select-all">{creator.Label}</div>
                </div>
            </div>
        </div>
    );
}

interface TimestampCardProps extends ComponentProps<"div"> {
    timestamp: Date;
}
function TimestampCard({ timestamp, ...props }: TimestampCardProps) {
    const date = timestamp.toLocaleDateString(navigator.languages, {
        month: "short",
        day: "2-digit",
        year: "numeric",
    });
    const time = timestamp.toLocaleTimeString(navigator.languages, {
        hour12: false,
    });
    return (
        <div {...props}>
            <div className={"h-full inline-flex gap-2 py-2"}>
                <div>
                    <Icon.Calendar />
                </div>
                <div>
                    <div className="text-xs text-secondary">Created on</div>
                    <div className="font-bold select-all">
                        <span>{date}</span>, <span>{time}</span>
                    </div>
                </div>
            </div>
        </div>
    );
}

interface PropertiesProps extends ComponentProps<"div"> {
    properties: Property[];
}
function Properties({ properties, ...props }: PropertiesProps) {
    return (
        <div {...props}>
            <div className="px-4 flex gap-2 items-center">
                <Icon.Tag />
                <h2 className="text-lg font-bold">Properties</h2>
            </div>
            {properties.length === 0 ? (
                <div className="px-4 pt-2 text-secondary">(no properties)</div>
            ) : (
                <div className="px-4">
                    <table className="border-separate border-spacing-x-0 border-spacing-y-1">
                        <tbody>
                            {properties.map((property) => (
                                <tr key={property.Key}>
                                    <th className="px-4 text-left py-1 rounded-l-lg bg-gray-100 dark:bg-gray-700">
                                        {property.Key}
                                    </th>
                                    <td className="px-4 text-secondary py-1 bg-gray-100 dark:bg-gray-700">
                                        <div className="px-4 rounded-full text-center bg-primary-300 dark:bg-primary-800 dark:text-white">
                                            {property.Type}
                                        </div>
                                    </td>
                                    <td className="px-4 py-1 select-all rounded-r-lg bg-gray-100 dark:bg-gray-700">
                                        {value_to_string(property)}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
}

interface NotesProps extends ComponentProps<"div"> {
    notes: Note[];
    users: User[];
}
function Notes({ notes, users, ...props }: NotesProps) {
    return (
        <div {...props}>
            <div className="px-4 pb-2 flex gap-2 items-center">
                <Icon.Notebook />
                <h2 className="text-lg font-bold">Notes</h2>
            </div>
            {notes.length === 0 ? (
                <div className="px-4 pt-2 text-sm text-secondary">
                    (no notes)
                </div>
            ) : (
                <div>
                    <ol>
                        {notes.map((note) => {
                            const creator = users.find(
                                (user) => user.Id === note.Creator,
                            );
                            return (
                                <li
                                    key={uuidToString(note.Id)}
                                    className="px-4"
                                >
                                    <NoteView
                                        note={note}
                                        creator={creator}
                                        className="rounded-lg border overflow-hidden"
                                    />
                                </li>
                            );
                        })}
                    </ol>
                </div>
            )}
        </div>
    );
}

// TODO: Show data tree.
// interface FamilyTreeProps {
//     tree: any
// }
// function FamilyTree({tree}: FamilyTreeProps) {
//     return <div>
//         <div>
//             <h2>Family tree</h2>
//         </div>
//         <ul>

//         </ul>
//     </div>
// }

interface ProjectResourcesProps extends ComponentProps<"div"> {
    dataId: UUIDTypes;
    projectResources: DataProjectResources[];
    users: User[];
}
function ProjectResources({
    dataId,
    projectResources,
    users,
    ...props
}: ProjectResourcesProps) {
    const [showAddProject, setShowAddProject] = useState(false);

    function showAddProjectFn(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        setShowAddProject(true);
    }

    const currentProjects = projectResources.map((prj) => prj.Project.Id);

    return (
        <div {...props}>
            <div className="px-4 pb-2 flex gap-2 items-baseline justify-between">
                <div className="flex gap-2 items-baseline">
                    <Icon.Folder />
                    <h2 className="text-lg font-bold">Projects</h2>
                </div>
                <button
                    type="button"
                    className="btn-cmd"
                    onMouseDown={showAddProjectFn}
                >
                    <Icon.Plus />
                </button>
            </div>
            <div
                className={classNames({
                    "px-4 py-2": true,
                    hidden: !showAddProject,
                })}
            >
                <AddProject
                    data_id={dataId}
                    currentProjects={currentProjects}
                    setShowAddProject={setShowAddProject}
                />
            </div>
            {projectResources.length > 0 ? (
                <ProjectList projects={projectResources} users={users} />
            ) : (
                <ProjectsEmpty />
            )}
        </div>
    );
}

interface AddProjectProps {
    data_id: UUIDTypes;
    currentProjects: UUIDTypes[];
    setShowAddProject: Dispatch<SetStateAction<boolean>>;
}
function AddProject({
    data_id,
    currentProjects,
    setShowAddProject,
}: AddProjectProps) {
    const { data: projects } = useSuspenseQuery({
        queryKey: [QUERY_KEY_USER_PROJECTS],
        queryFn: projectService.getUserProjects,
    });

    async function addProject(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const data = new FormData(e.target);
        const project = data.get("add-project")!.toString();
        await projectService
            .dataMembershipCreate(project, data_id)
            .then((_) => {
                setShowAddProject(false);
            });
    }

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        setShowAddProject(false);
    }

    return (
        <form onSubmit={addProject}>
            <div className="pb-2 flex gap-2">
                <div>
                    <select
                        id="add-project"
                        name="add-project"
                        className="input-basic"
                        defaultValue=""
                    >
                        <option value="" disabled hidden>
                            Choose a project
                        </option>
                        {projects
                            .filter(
                                (resource) =>
                                    !currentProjects.includes(
                                        resource.Project.Id,
                                    ),
                            )
                            .map((resource) => (
                                <option
                                    key={resource.Project.Id.toString()}
                                    value={resource.Project.Id.toString()}
                                >
                                    {resource.Project.Label}
                                </option>
                            ))}
                    </select>
                </div>
                <div className="flex gap-2">
                    <div>
                        <button type="submit" className="btn-submit">
                            <Icon.Plus />
                        </button>
                    </div>
                    <div>
                        <button
                            type="button"
                            className="btn-submit"
                            onMouseDown={close}
                        >
                            <Icon.Close />
                        </button>
                    </div>
                </div>
            </div>
        </form>
    );
}

function ProjectsEmpty() {
    return (
        <div className="px-4 text-sm text-secondary">
            <p>This data does not belong to any projects, yet.</p>
            <p>
                Add it to one by clicking the <Icon.Plus className="inline" />{" "}
                above.
            </p>
        </div>
    );
}

interface ProjectListProps {
    projects: DataProjectResources[];
    users: User[];
}
function ProjectList({ projects, users }: ProjectListProps) {
    return (
        <table className="px-4">
            <tbody>
                {projects.map((project) => (
                    <ProjectItem
                        key={project.Project.Id.toString()}
                        resources={project}
                        users={users}
                    />
                ))}
            </tbody>
        </table>
    );
}

interface ProjectItemProps {
    resources: DataProjectResources;
    users: User[];
}
function ProjectItem({ resources }: ProjectItemProps) {
    return (
        <tr className="group pb-1">
            <td className="font-bold pl-4 pr-2">{resources.Project.Label}</td>
            <td className="pr-2">
                {resources.Label ?? (
                    <span className="text-secondary-700 dark:text-secondary-300">
                        (no label)
                    </span>
                )}
            </td>
            <td className="pr-2">{resources.Project.Description}</td>
            <td className="pr-2">
                <div className="invisible group-hover:visible">
                    <Link
                        to={`/project/${resources.Project.Id}`}
                        className="align-middle"
                    >
                        <button type="button" className="btn-cmd">
                            <Icon.Eye />
                        </button>
                    </Link>
                </div>
            </td>
        </tr>
    );
}

function dataValuesTitleResources(storage: DataStorage): {
    title: string;
    icon: React.ReactNode;
} {
    switch (storage) {
        case DataStorageInternal:
            return {
                title: "Values",
                icon: <Icon.Chart />,
            };
        case DataStorageExternal:
            return {
                title: "Sources",
                icon: <Icon.Files />,
            };
        default:
            throw new Error(`invalid data storage type: ${storage}`);
    }
}

interface DataValuesProps extends ComponentProps<"div"> {
    data: UUIDTypes;
    storage: DataStorage;
}
function DataValues({ data, storage, ...props }: DataValuesProps) {
    return (
        <ErrorBoundary
            fallbackRender={(props) => (
                <ValuesError storage={storage} {...props} />
            )}
        >
            <Suspense fallback={<ValuesLoading storage={storage} />}>
                <DataValuesInner data={data} {...props} />
            </Suspense>
        </ErrorBoundary>
    );
}

interface ValuesErrorProps extends FallbackProps {
    storage: DataStorage;
}
function ValuesError({ storage, error, resetErrorBoundary }: ValuesErrorProps) {
    const { icon, title } = dataValuesTitleResources(storage);
    return (
        <div>
            <div className="px-4 flex gap-2 group justify-baseline">
                {icon}
                <h2 className="text-lg font-bold">{title}</h2>
            </div>
            <StandardError
                title="Could not load data values"
                error={error as Error}
                resetErrorBoundary={resetErrorBoundary}
            />
        </div>
    );
}

interface ValuesLoadingProps {
    storage: DataStorage;
}
function ValuesLoading({ storage }: ValuesLoadingProps) {
    const { icon, title } = dataValuesTitleResources(storage);
    return (
        <div>
            <div className="px-4 flex gap-2 group justify-baseline">
                {icon}
                <h2 className="text-lg font-bold">{title}</h2>
            </div>
            <div className="px-4">Loading values</div>
        </div>
    );
}

interface DataValuesInnerProps extends ComponentProps<"div"> {
    data: UUIDTypes;
}
function DataValuesInner({ data, ...props }: DataValuesInnerProps) {
    const { data: values } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_VALUES, data],
        queryFn: async () => await dataService.dataValues(data),
    });

    let component;
    let title;
    let icon;
    switch (values.Storage) {
        case DataStorageInternal:
            title = "Values";
            icon = <Icon.Chart />;
            component = (
                <DataValuesInternal
                    cardinality={values.Values[0].Cardinality}
                    values={values.Values}
                />
            );
            break;
        case DataStorageExternal:
            title = "Sources";
            icon = <Icon.Files />;
            component = (
                <DataValuesExternal data_id={data} sources={values.Values} />
            );
            break;
    }

    const params = new URLSearchParams();
    params.append("id", uuidToString(data));
    const download = `/resource/data?${params}`;
    return (
        <div {...props}>
            <div className="px-4 flex gap-2 group items-baseline">
                {icon}
                <h2 className="text-lg font-bold">{title}</h2>
                <div className="invisible group-hover:visible">
                    <a href={download}>
                        <button type="button" className="btn-cmd">
                            <Icon.Download />
                        </button>
                    </a>
                </div>
            </div>
            <div className="px-4 pt-2">{component}</div>
        </div>
    );
}

interface DataValuesInternalProps {
    cardinality: DataSchemaCardinality;
    values: any;
}
function DataValuesInternal({ cardinality, values }: DataValuesInternalProps) {
    switch (cardinality) {
        case DataSchemaCardinalitySingle:
            return <DataValuesInternalSingle values={values} />;
        case DataSchemaCardinalityMultiple:
            return <DataValuesInternalMultiple values={values} />;
    }
}

interface DataValuesInternalSingleProps {
    values: SchemaFieldValues[];
}
function DataValuesInternalSingle({ values }: DataValuesInternalSingleProps) {
    return (
        <table className="table-std">
            <tbody>
                {values.map((field) => {
                    return (
                        <tr key={field.Label}>
                            <th>{field.Label}</th>
                            <td>{field.Values}</td>
                        </tr>
                    );
                })}
            </tbody>
        </table>
    );
}

interface DataValuesInternalMultipleProps {
    values: SchemaFieldValues[];
}
function DataValuesInternalMultiple({
    values,
}: DataValuesInternalMultipleProps) {
    const rx_cnt = values[0]!.Values.length;
    const col_cnt = values.length;
    const records = new Array<Array<any>>(rx_cnt);
    for (let idx = 0; idx < rx_cnt; idx++) {
        records[idx] = new Array(col_cnt);
    }
    for (let vdx = 0; vdx < col_cnt; vdx++) {
        for (let idx = 0; idx < rx_cnt; idx++) {
            records[idx]![vdx] = values[vdx]!.Values[idx];
        }
    }

    const is_numeric = new Array<boolean>(col_cnt);
    for (let idx = 0; idx < col_cnt; idx++) {
        switch (values[idx]!.DType) {
            case ValueTypeFloat:
            case ValueTypeInt:
            case ValueTypeUint:
                is_numeric[idx] = true;
                break;
            case ValueTypeBoolean:
            case ValueTypeString:
            case ValueTypeTimestamp:
                is_numeric[idx] = false;
                break;
        }
    }

    return (
        <table className="w-full text-left">
            <thead>
                <tr>
                    <th
                        className="px-4 align-middle bg-gray-200 dark:bg-gray-700 rounded-tl-lg"
                        rowSpan={2}
                    >
                        #
                    </th>
                    {values.map((field) => (
                        <th
                            key={field.Label}
                            className="px-4 bg-gray-200 dark:bg-gray-700 last:rounded-tr-lg"
                        >
                            {field.Label}
                        </th>
                    ))}
                </tr>
                <tr>
                    {values.map((field) => (
                        <th
                            key={field.Label}
                            className="px-4 bg-gray-200 dark:bg-gray-700 text-secondary text-sm text-left font-normal"
                        >
                            {field.DType}
                        </th>
                    ))}
                </tr>
            </thead>
            <tbody>
                {records.map((rx, idx) => (
                    <tr
                        key={idx}
                        className="even:bg-gray-200 dark:even:bg-gray-700 hover:bg-primary-400 dark:hover:bg-primary-800"
                    >
                        <th className="px-4 py-1 font-normal">{idx + 1}</th>
                        {rx.map((vx, vdx) => {
                            return (
                                <td key={vdx} className="px-4 py-1 select-all">
                                    {vx}
                                </td>
                            );
                        })}
                    </tr>
                ))}
            </tbody>
        </table>
    );
}

interface DataValuesExternalProps {
    data_id: UUIDTypes;
    sources: DataSource[];
}
function DataValuesExternal({ data_id, sources }: DataValuesExternalProps) {
    return (
        <div>
            {sources.map((source) => (
                <DataValuesExternalSource
                    key={source.Label}
                    data_id={data_id}
                    source={source}
                />
            ))}
        </div>
    );
}

interface DataValuesExternalSourceProps {
    data_id: UUIDTypes;
    source: DataSource;
}
function DataValuesExternalSource({
    data_id,
    source,
}: DataValuesExternalSourceProps) {
    switch (source.Cardinality) {
        case DataSourceCardinalitySingle:
            return (
                <DataValuesExternalSourceSingle
                    data_id={data_id}
                    source={source}
                />
            );
        case DataSourceCardinalityMultiple:
            return (
                <DataValuesExternalSourceMultiple
                    data_id={data_id}
                    source={source}
                />
            );
    }
}

interface DataValuesExternalSourceSingleProps {
    data_id: UUIDTypes;
    source: DataSource;
}
function DataValuesExternalSourceSingle({
    data_id,
    source,
}: DataValuesExternalSourceSingleProps) {
    const params = new URLSearchParams();
    params.append("data", uuidToString(data_id));
    params.append("source", source.Label);
    const download = `/resource/data/source?${params}`;

    return (
        <div className="px-4 flex gap-2 group">
            <div title="Source">{source.Label}</div>
            <div className="invisible group-hover:visible">
                <a href={download} title="Download">
                    <button type="button" className="btn-cmd">
                        <Icon.Download />
                    </button>
                </a>
            </div>
        </div>
    );
}

interface DataValuesExternalSourceMultipleProps {
    data_id: UUIDTypes;
    source: DataSource;
}
function DataValuesExternalSourceMultiple({
    data_id,
    source,
}: DataValuesExternalSourceMultipleProps) {
    const params = new URLSearchParams();
    params.append("data", uuidToString(data_id));
    params.append("source", source.Label);
    const download = `/resource/data/source?${params}`;

    return (
        <div>
            <div className="px-4 flex gap-2 group">
                <div title="Source">{source.Label}</div>
                <div className="invisible group-hover:visible">
                    <a href={download} title="Download">
                        <button type="button" className="btn-cmd">
                            <Icon.Download />
                        </button>
                    </a>
                </div>
            </div>
            <ol className="list-decimal list-inside">
                {source.Source.sort((src) => src.Index).map((src) => {
                    const params = new URLSearchParams();
                    params.append("data", uuidToString(data_id));
                    params.append("source", src.Label);
                    params.append("index", src.Index);
                    const download = `/resource/data/source?${params}`;

                    return (
                        <li key={src.Index.toString()} className="px-4">
                            <div className="group/item inline-flex gap-2">
                                <div>{src.Label}</div>
                                <div className="invisible group-hover/item:visible">
                                    <a href={download} title="Download">
                                        <button
                                            type="button"
                                            className="btn-cmd"
                                        >
                                            <Icon.Download />
                                        </button>
                                    </a>
                                </div>
                            </div>
                        </li>
                    );
                })}
            </ol>
        </div>
    );
}
