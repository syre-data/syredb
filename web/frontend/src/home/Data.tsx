import {
    MouseButton,
    QUERY_KEY_DATA,
    QUERY_KEY_DATA_VALUES,
    QUERY_KEY_USER_PROJECTS,
    timestampToString,
    uuidToString,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import projectService from "@/service/project.service";
import {
    DataCreatorTypeTransform,
    DataCreatorTypeUser,
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
    type DataCreatorType,
    type DataCreatorUserInfo,
    type DataProjectResources,
    type DataRx,
    type DataSchemaCardinality,
    type DataSource,
    type DataType,
    type Note,
    type Property,
    type SchemaFieldValues,
    type User,
} from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";
import { StatusCodes } from "http-status-codes";
import {
    Suspense,
    useState,
    type Dispatch,
    type MouseEvent,
    type SetStateAction,
    type SubmitEvent,
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
        <SuspenseError resetErrorBoundary={resetErrorBoundary}>
            Could not load data
        </SuspenseError>
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
    const [showAddProject, setShowAddProject] = useState(false);
    const navigate = useNavigate();

    function showAddProjectFn(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        setShowAddProject(true);
    }

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    const data = resources.Data as DataRx;
    const data_type = resources.DataType as DataType;
    const properties = resources.Properties as Property[];
    const notes = resources.Notes as Note[];
    const creator = resources.Creator as DataCreator;
    const project_resources =
        resources.ProjectResources as DataProjectResources[];
    const users = resources.Users as User[];
    const currentProjects = project_resources.map(
        (resource) => resource.Project.Id,
    );

    let creatorContent;
    if (Object.hasOwn(creator, "User") && Object.hasOwn(creator, "Origin")) {
        creatorContent = <CreatorUser creator={creator} />;
    } else if (
        Object.hasOwn(creator, "Id") &&
        Object.hasOwn(creator, "Label") &&
        Object.hasOwn(creator, "Description")
    ) {
        creatorContent = <CreatorTransform creator={creator} />;
    } else {
        console.debug(creator);
        throw new Error("invalid data creator");
    }

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <div className="group flex gap-2">
                    <h1 className="title flex gap-2">
                        <div className="flex gap-2 items-center">
                            <Icon.DataType /> {data_type.Label}
                        </div>
                        |
                        <div>{timestampToString(new Date(data.Timestamp))}</div>
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
            <div className="px-4 pt-2">
                <div>{data.Visibility}</div>
                <div>{creatorContent}</div>
            </div>
            <Properties properties={properties} />
            <Notes notes={notes} />
            <div className="pt-2">
                <div className="px-4 flex gap-2">
                    <h2 className="text-lg">Projects</h2>
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
                        data_id={data_id}
                        currentProjects={currentProjects}
                        setShowAddProject={setShowAddProject}
                    />
                </div>
                {project_resources.length > 0 ? (
                    <ProjectList projects={project_resources} users={users} />
                ) : (
                    <ProjectsEmpty />
                )}
            </div>
            <ErrorBoundary FallbackComponent={ValuesError}>
                <Suspense fallback={<ValuesLoading />}>
                    <DataValues data={data_id} />
                </Suspense>
            </ErrorBoundary>
        </div>
    );
}

interface CreatorUserProps {
    creator: DataCreatorUserInfo;
}
function CreatorUser({ creator }: CreatorUserProps) {
    return (
        <div>
            Created by <span>{creator.User.Name || creator.User.Email}</span>{" "}
            from <span>{creator.Origin.Label}</span>
        </div>
    );
}

interface CreatorTransformProps {
    creator: DataCreatorTransformInfo;
}
function CreatorTransform({ creator }: CreatorTransformProps) {
    return (
        <div>
            Created by script <span>{creator.Label}</span>
        </div>
    );
}

interface PropertiesProps {
    properties: Property[];
}
function Properties({ properties }: PropertiesProps) {
    if (properties.length === 0) {
        return (
            <div className="px-4 text-secondary-700 dark:text-secondary-300">
                (no properties)
            </div>
        );
    } else {
        return (
            <div className="pt-2">
                <div className="px-4">
                    <h2 className="text-lg">Properties</h2>
                </div>
                <div className="px-4">
                    <table>
                        <tbody>
                            {properties.map((property) => (
                                <tr key={property.Key}>
                                    <th className="pr-2">{property.Key}</th>
                                    <td className="px-2">{property.Value}</td>
                                    <td className="pl-2 text-syre-grey-700 dark:text-syre-grey-300">
                                        ({property.Type})
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </div>
        );
    }
}

interface NotesProps {
    notes: Note[];
}
function Notes({ notes }: NotesProps) {
    if (notes.length === 0) {
        return (
            <div className="px-4 text-secondary-700 dark:text-secondary-300">
                (no notes)
            </div>
        );
    } else {
        return (
            <div className="pt-2">
                <div className="px-4 pb-2">
                    <h2>Notes</h2>
                </div>
                <div>
                    <ol>
                        {notes.map((note) => (
                            <li key={note.Id.toString()}>
                                <div className="px-4 pb-2">
                                    <h4>
                                        {timestampToString(
                                            new Date(note.Timestamp),
                                        )}
                                    </h4>
                                    <div>{note.Content}</div>
                                </div>
                            </li>
                        ))}
                    </ol>
                </div>
            </div>
        );
    }
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
            .then((res) => {
                if (res.status === StatusCodes.OK) {
                    setShowAddProject(false);
                }
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
                                (project) =>
                                    !currentProjects.includes(project.Id),
                            )
                            .map((project) => (
                                <option
                                    key={project.Id.toString()}
                                    value={project.Id.toString()}
                                >
                                    {project.Label}
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
        <div className="px-4">
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

function ValuesError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError resetErrorBoundary={resetErrorBoundary}>
            Could not load data values
        </SuspenseError>
    );
}

function ValuesLoading() {
    return <div>Loading values</div>;
}

interface DataValuesProps {
    data: UUIDTypes;
}
function DataValues({ data }: DataValuesProps) {
    const { data: values } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_VALUES, data],
        queryFn: async () => await dataService.dataValues(data),
    });

    let component;
    let title;
    switch (values.Storage) {
        case DataStorageInternal:
            title = "Values";
            component = (
                <DataValuesInternal
                    cardinality={values.Values[0].Cardinality}
                    values={values.Values}
                />
            );
            break;
        case DataStorageExternal:
            title = "Sources";
            component = (
                <DataValuesExternal data_id={data} sources={values.Values} />
            );
            break;
    }

    const params = new URLSearchParams();
    params.append("id", uuidToString(data));
    const download = `/resource/data?${params}`;
    return (
        <div className="pt-2">
            <div className="px-4 flex gap-2 group">
                <h2 className="text-lg">{title}</h2>
                <div className="invisible group-hover:visible">
                    <a href={download}>
                        <button type="button" className="btn-cmd">
                            <Icon.Download />
                        </button>
                    </a>
                </div>
            </div>
            {component}
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
        <table>
            <thead>
                <tr>
                    <th></th>
                    {values.map((field) => (
                        <th key={field.Label} className="px-2">
                            {field.Label}
                        </th>
                    ))}
                </tr>
                <tr>
                    <th></th>
                    {values.map((field) => (
                        <th key={field.Label} className="px-2">
                            {field.DType}
                        </th>
                    ))}
                </tr>
            </thead>
            <tbody>
                {records.map((rx, idx) => (
                    <tr
                        key={idx}
                        className="hover:bg-gray-50 dark:hover:bg-gray-800"
                    >
                        <th className="pl-4 pr-2">{idx + 1}.</th>
                        {rx.map((vx, vdx) => {
                            return (
                                <td
                                    key={vdx}
                                    className={classNames({
                                        "px-2": true,
                                        "text-right": is_numeric[vdx],
                                    })}
                                >
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
                {source.Source.sort((src) => src.Index).map((src, idx) => {
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
