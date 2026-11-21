import {
    MouseButton,
    QUERY_KEY_DATA_PREVIEW,
    QUERY_KEY_PROJECT_RESOURCES,
    timestampToString,
    uuidToString,
} from "@/common";
import { Loading, StandardError, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import projectService from "@/service/project.service";
import {
    DataCreatorTypeUser,
    DataSchemaCardinalityMultiple,
    DataSchemaCardinalitySingle,
    DataSourceCardinalityMultiple,
    DataSourceCardinalitySingle,
    DataStorageExternal,
    DataStorageInternal,
    type DataOriginRx,
    type DataSource,
    type DataStorage,
    type DataType,
    type DataValuesPreviewInternal,
    type DataValuesPreviewInternalMultiple,
    type ProjectData,
    type SchemaFieldValues,
    type SourceFileInfo,
    type User,
} from "@/types";
import Users from "@/user/Users";
import { useDismiss, useFloating, useInteractions } from "@floating-ui/react";
import { useForm } from "@tanstack/react-form";
import { noop, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";
import React, {
    Suspense,
    useState,
    type ComponentProps,
    type MouseEvent,
    type Ref,
    type SubmitEvent,
} from "react";
import { createRoot } from "react-dom/client";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { data } from "react-router";
import { Link, redirect, useParams } from "react-router";
import * as uuid from "uuid";

export default function () {
    const { project_id } = useParams();
    if (!project_id) {
        console.error("project id not present");
        redirect("/");
        return;
    }

    return (
        <ErrorBoundary FallbackComponent={ProjectError}>
            <Suspense fallback={<Loading />}>
                <Project projectId={project_id} />
            </Suspense>
        </ErrorBoundary>
    );
}

function ProjectError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <StandardError
            resetErrorBoundary={resetErrorBoundary}
            error={error as Error}
            title={"Could not load project"}
        />
    );
}

interface ProjectProps {
    projectId: uuid.UUIDTypes;
}
function Project({ projectId }: ProjectProps) {
    const { data: resources } = useSuspenseQuery({
        queryKey: [QUERY_KEY_PROJECT_RESOURCES, projectId],
        queryFn: async () =>
            await projectService.getProjectResources(projectId),
    });

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <h1 className="title">{resources.Project.Label}</h1>
                <div>
                    <div>
                        <Link to="/">
                            <button type="button" className="btn-cmd">
                                <Icon.Home />
                            </button>
                        </Link>
                    </div>
                </div>
            </div>
            <ProjectData
                projectId={projectId}
                data={resources.Data}
                relations={resources.DataRelations}
                types={resources.DataTypes}
                origins={resources.DataOrigins}
                users={resources.Users}
            />
        </div>
    );
}

interface ProjectDataProps {
    projectId: uuid.UUIDTypes;
    data: ProjectData[];
    relations: { [key: string]: uuid.UUIDTypes[] };
    types: DataType[];
    origins: DataOriginRx[];
    users: User[];
}
function ProjectData({
    projectId,
    data,
    relations,
    types,
    origins,
    users,
}: ProjectDataProps) {
    const params = new URLSearchParams();
    params.append("project", uuidToString(projectId));
    const download = `/resource/project/data?${params}`;

    const [dataFilter, setDataFilter] = useState<null | uuid.UUIDTypes[]>(null);

    return (
        <div>
            <div className="px-4 flex gap-2 items-center group">
                <h2 className="text-xl">Data</h2>
                <div className="flex gap-2">
                    <div>
                        <Link to={`/project/${projectId}/data/create`}>
                            <button type="button" className="btn-cmd">
                                <Icon.Plus />
                            </button>
                        </Link>
                    </div>
                    <div
                        className={classNames({
                            "invisible group-hover:visible in-focus-within:visible": true,
                            visible: dataFilter,
                        })}
                    >
                        {data.length === 0 ? (
                            <button
                                type="button"
                                className="btn-cmd"
                                title="Download all data (disabled)"
                                disabled
                            >
                                <Icon.Download />
                            </button>
                        ) : (
                            <a href={download} title="Download all data">
                                <button type="button" className="btn-cmd">
                                    <Icon.Download />
                                </button>
                            </a>
                        )}
                    </div>
                </div>
                <DataQuery
                    projectId={projectId}
                    setDataFilter={setDataFilter}
                    className={classNames({
                        "invisible group-hover:visible in-focus-within:visible": true,
                        visible: dataFilter,
                    })}
                />
            </div>
            {data.length === 0 ? (
                <ProjectDataEmpty />
            ) : (
                <ProjectDataListView
                    projectId={projectId}
                    data={data}
                    relations={relations}
                    types={types}
                    origins={origins}
                    users={users}
                    filter={dataFilter}
                />
            )}
        </div>
    );
}

interface DataQuery {
    type?: string | string[];
    property: Map<string, any>;
    label?: string;
    user?: string;
    owner?: string;
}

interface DataQueryProps {
    projectId: uuid.UUIDTypes;
    setDataFilter: React.Dispatch<
        React.SetStateAction<
            uuid.UUIDTypes<Uint8Array<ArrayBufferLike>>[] | null
        >
    >;
    className?: string;
}
function DataQuery({ projectId, setDataFilter, className }: DataQueryProps) {
    const form = useForm({
        defaultValues: { query: "" },
        onSubmit: async ({ value }) => {
            const results = await dataService.search(
                projectId,
                value.query.trim(),
            );
            setDataFilter(results);
        },
    });

    function onsubmit(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        e.stopPropagation();
        form.handleSubmit();
    }

    return (
        <form className={className} onSubmit={onsubmit}>
            <div>
                <label className="flex gap-1 items-center">
                    <Icon.MagnifyingGlass />
                    <form.Field name="query">
                        {(field) => {
                            return (
                                <input
                                    type="text"
                                    className="input-basic"
                                    value={field.state.value}
                                    onChange={(e) => {
                                        const val = e.target.value;
                                        if (!val) {
                                            setDataFilter(null);
                                        }
                                        field.handleChange(val);
                                    }}
                                />
                            );
                        }}
                    </form.Field>
                </label>
            </div>
        </form>
    );
}

function ProjectDataEmpty() {
    return (
        <div className="px-4">
            <p>No data yet.</p>
            <p>
                Click the <Icon.Plus className="inline" /> above to create some.
            </p>
        </div>
    );
}

interface ProjectDataListViewProps {
    projectId: uuid.UUIDTypes;
    data: ProjectData[];
    relations: { [key: string]: uuid.UUIDTypes[] };
    types: DataType[];
    origins: DataOriginRx[];
    users: User[];
    filter: uuid.UUIDTypes<Uint8Array<ArrayBufferLike>>[] | null;
}
function ProjectDataListView({
    projectId,
    data,
    relations,
    types,
    origins,
    users,
    filter,
}: ProjectDataListViewProps) {
    if (!filter) {
        return (
            <ProjectDataList
                projectId={projectId}
                data={data}
                relations={relations}
                types={types}
                origins={origins}
                users={users}
            />
        );
    } else if (filter.length == 0) {
        return <ProjectDataFilterEmpty />;
    } else {
        const filteredData = data.filter((data) => filter.includes(data.Id));

        return (
            <div>
                <div className="text-sm px-4 pb-2">
                    Filtered {filteredData.length} out of {data.length} data
                </div>
                <ProjectDataList
                    projectId={projectId}
                    data={filteredData}
                    relations={relations}
                    types={types}
                    origins={origins}
                    users={users}
                />
            </div>
        );
    }
}

function ProjectDataFilterEmpty() {
    return <div className="pt-2 px-4 text-lg">No data match your query</div>;
}

interface ProjectDataListProps {
    projectId: uuid.UUIDTypes;
    data: ProjectData[];
    relations: { [key: string]: uuid.UUIDTypes[] };
    types: DataType[];
    origins: DataOriginRx[];
    users: User[];
}
function ProjectDataList({
    projectId,
    data,
    relations,
    types,
    origins,
    users,
}: ProjectDataListProps) {
    let parents = new Map();
    for (const parent in relations) {
        const children = relations[parent]!;
        for (const child of children) {
            parents.set(child, parent);
        }
    }

    // TODO: Add bulk download by selecting individual data
    return (
        <table
            className="table-compact w-full \
            [&_th]:last:w-full [&_td]:last:w-full [&_td]:text-nowrap \
            [&_th]:first:pl-4 [&_td]:first:pl-4"
        >
            <thead>
                <tr className="[&_th]:bg-gray-200 dark:[&_th]:bg-gray-700">
                    <th>Type</th>
                    <th>Label</th>
                    <th>Creator</th>
                    <th>Origin</th>
                    <th>Timestamp</th>
                    <th></th>
                    <th></th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
                {data.map((datum) => {
                    const parent_id = parents.get(datum.Id);
                    let parent: ProjectData | undefined = undefined;
                    let parent_label = undefined;
                    if (parent_id) {
                        const parent_idx = data.findIndex(
                            (datum) => datum.Id === parent_id,
                        )!;
                        parent = data[parent_idx]!;
                        if (parent.Label) {
                            const matching_label = data.findIndex(
                                (datum) =>
                                    datum.Label === parent.Label &&
                                    datum.Id !== parent.Id,
                            );
                            if (matching_label < 0) {
                                parent_label = parent.Label;
                            } else {
                                parent_label = `${parent.Label} (${parent_idx})`;
                            }
                        } else {
                            parent_label = parent_idx.toString();
                        }
                    }

                    const type = types.find((type) => type.Id === datum.Type);
                    if (!type) {
                        console.error(`could not get data type ${datum.Type}`);
                    }

                    let origin, userCreator;
                    if (datum.Creator.Origin) {
                        origin = origins.find(
                            (origin) => origin.Id === datum.Creator.Origin,
                        );
                        userCreator = users.find(
                            (user) => user.Id === datum.Creator.Id,
                        );
                    }

                    return (
                        <ProjectDataItem
                            key={datum.Id.toString()}
                            projectId={projectId}
                            data={datum}
                            type={type}
                            parent={parent}
                            parentLabel={parent_label}
                            origin={origin}
                            userCreator={userCreator}
                            className="even:bg-gray-200 dark:even:bg-gray-700 \
                                hover:bg-primary-400 dark:hover:bg-primary-700"
                        />
                    );
                })}
            </tbody>
        </table>
    );
}

interface ProjectDataItemProps extends ComponentProps<"tr"> {
    projectId: uuid.UUIDTypes;
    data: ProjectData;
    type: DataType;
    parent?: ProjectData;
    parentLabel?: string;
    origin?: DataOriginRx;
    userCreator?: User;
}
function ProjectDataItem({
    projectId,
    data,
    type,
    parent,
    parentLabel,
    origin,
    userCreator,
    className,
    ...props
}: ProjectDataItemProps) {
    const queryClient = useQueryClient();
    const [previewOpen, setPreviewOpen] = useState(false);
    const { refs, floatingStyles, context } = useFloating({
        placement: "right-start",

        open: previewOpen,
        onOpenChange: setPreviewOpen,
    });
    const dismissPreview = useDismiss(context);
    const { getReferenceProps, getFloatingProps } = useInteractions([
        dismissPreview,
    ]);

    function highlight_parent(e: MouseEvent<HTMLElement>, active: boolean) {
        const HIGHLIGHT_CLASS = ["bg-gray-200", "dark:bg-gray-800"];

        const parent_elm = document.getElementById(`data-${parent?.Id}`)!;
        if (active) {
            for (const st of HIGHLIGHT_CLASS) {
                parent_elm.classList.add(st);
            }
        } else {
            for (const st of HIGHLIGHT_CLASS) {
                parent_elm.classList.remove(st);
            }
        }
    }

    function goto_parent(e: MouseEvent<HTMLElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        const parent_elm = document.getElementById(`data-${parent?.Id}`)!;
        parent_elm.scrollIntoView();
    }

    async function fetch_preview() {
        queryClient.query({
            queryKey: [QUERY_KEY_DATA_PREVIEW, data.Id],
            queryFn: async () =>
                await dataService.dataPreview(data.Id).catch(noop),
        });
    }

    function open_preview(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        setPreviewOpen(true);
    }

    const date = new Date(data.Timestamp);
    const y = date.getUTCFullYear();
    const mo = date.getMonth() + 1;
    const d = date.getDate();
    const h = date.getHours();
    const m = date.getMinutes();
    const s = date.getSeconds();
    const timestamp = `${y}-${mo}-${d}T${h}-${m}-${s}`;
    let filename = data.Label ?? `data-${timestamp}`;
    switch (type.Storage) {
        case DataStorageExternal:
            // TODO: If source is a single file, use that extension.
            filename += ".zip";
            break;
        case DataStorageInternal:
            filename += ".csv";
            break;
        default:
            console.error("invalid data storage");
    }

    const download_params = new URLSearchParams();
    download_params.append("id", data.Id.toString());
    download_params.append("project", uuidToString(projectId));
    const download_path = `/resource/data?${download_params}`;

    return (
        <tr
            id={`data-${data.Id}`}
            data-id={data.Id}
            className={`group ${className}`}
            {...props}
        >
            <td className="text-nowrap w-0 font-semibold select-all">
                {type.Label}
            </td>
            <td className="text-nowrap w-0">
                {data.Label ? (
                    <span className="font-semibold select-all">
                        {data.Label}
                    </span>
                ) : (
                    <span className="text-secondary dark:hover:text-gray-800">
                        (no label)
                    </span>
                )}
            </td>
            <td>
                {userCreator ? (
                    <div className="flex items-center">
                        <span className="select-all">{userCreator.Name}</span>
                        <a
                            href={`mailto:${userCreator.Email}`}
                            target="_blank"
                            title={userCreator.Email}
                            className="invisible group-hover:visible pl-2"
                        >
                            <Icon.Mail />
                        </a>
                    </div>
                ) : null}
            </td>
            <td className="select-all">{origin ? origin.Label : null}</td>
            <td>{timestampToString(new Date(data.Timestamp))}</td>
            <td className="w-0">
                {parentLabel ? (
                    <span
                        className="text-nowrap text-gray-500 cursor-pointer"
                        onMouseEnter={(e) => highlight_parent(e, true)}
                        onMouseLeave={(e) => highlight_parent(e, false)}
                        onMouseDown={goto_parent}
                        title="Parent data"
                    >
                        (<Icon.ArrowReturn className="inline" /> {parentLabel})
                    </span>
                ) : null}
            </td>
            <td className="w-0" ref={refs.setReference} {...getReferenceProps}>
                <div
                    className={classNames({
                        "flex gap-1": true,
                        "invisible group-hover:visible": !previewOpen,
                    })}
                >
                    <div>
                        {/* TODO: Check permissions */}
                        <Link to={`/data/${data.Id}?project=${projectId}`}>
                            <button
                                type="button"
                                className="p-0.5 rounded-sm transition-colors cursor-pointer \
                                bg-white hover:bg-gray-100 dark:bg-gray-700 dark:hover:bg-gray-800"
                                title="View data"
                            >
                                <Icon.Eye />
                            </button>
                        </Link>
                    </div>
                    <div>
                        <button
                            type="button"
                            className="p-0.5 rounded-sm transition-colors cursor-pointer \
                            bg-white hover:bg-gray-100 dark:bg-gray-700 dark:hover:bg-gray-800"
                            title="Preview values"
                            onFocus={fetch_preview}
                            onMouseEnter={fetch_preview}
                            onMouseDown={open_preview}
                        >
                            <Icon.MagnifyingGlass />
                        </button>
                    </div>
                    <div>
                        <a href={download_path} title="Download">
                            <button
                                type="button"
                                className="p-0.5 rounded-sm transition-colors cursor-pointer \
                                bg-white hover:bg-gray-100 dark:bg-gray-700 dark:hover:bg-gray-800"
                            >
                                <Icon.Download />
                            </button>
                        </a>
                    </div>
                </div>
            </td>
            <td>
                {previewOpen ? (
                    <DataPreview
                        ref={refs.setFloating}
                        style={floatingStyles}
                        floatingProps={getFloatingProps}
                        data_id={data.Id}
                        storage={type.Storage}
                        setPreviewOpen={setPreviewOpen}
                    />
                ) : null}
            </td>
        </tr>
    );
}

interface DataPreviewProps {
    ref: Ref<HTMLDivElement>;
    style: React.CSSProperties;
    floatingProps: (
        userProps?: React.HTMLProps<HTMLElement>,
    ) => Record<string, unknown>;
    data_id: uuid.UUIDTypes;
    storage: DataStorage;
    setPreviewOpen: React.Dispatch<React.SetStateAction<boolean>>;
}
function DataPreview({
    ref,
    style,
    floatingProps,
    data_id,
    storage,
    setPreviewOpen,
}: DataPreviewProps) {
    return (
        <ErrorBoundary FallbackComponent={DataPreviewError}>
            <Suspense fallback={<Loading />}>
                <DataPreviewInner
                    ref={ref}
                    style={style}
                    floatingProps={floatingProps}
                    data_id={data_id}
                    storage={storage}
                    setPreviewOpen={setPreviewOpen}
                />
            </Suspense>
        </ErrorBoundary>
    );
}

function DataPreviewError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            <div>Could not load data preview</div>
        </SuspenseError>
    );
}

interface DataPreviewInnerProps {
    ref: Ref<HTMLDivElement>;
    style: React.CSSProperties;
    floatingProps: (
        userProps?: React.HTMLProps<HTMLElement>,
    ) => Record<string, unknown>;
    data_id: uuid.UUIDTypes;
    storage: DataStorage;
    setPreviewOpen: React.Dispatch<React.SetStateAction<boolean>>;
}
function DataPreviewInner({
    ref,
    style,
    floatingProps,
    data_id,
}: DataPreviewInnerProps) {
    const { data: preview } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_PREVIEW, data_id],
        queryFn: async () => await dataService.dataPreview(data_id),
    });

    let previewComponent;
    switch (preview.Storage) {
        case DataStorageExternal:
            previewComponent = (
                <DataPreviewExternal data={data_id} sources={preview.Preview} />
            );
            break;
        case DataStorageInternal:
            const values = preview.Preview as DataValuesPreviewInternal;
            switch (values.Cardinality) {
                case DataSchemaCardinalitySingle:
                    previewComponent = (
                        <DataPreviewInternalSingle fields={values.Values} />
                    );
                    break;
                case DataSchemaCardinalityMultiple:
                    previewComponent = (
                        <DataPreviewInternalMultiple preview={values.Values} />
                    );
                    break;
            }
            break;
    }

    const className =
        "bg-white dark:bg-syre-gray-900 border-2 border-secondary-700 dark:border-secondary-400 rounded";
    return (
        <div ref={ref} style={style} className={className} {...floatingProps}>
            {previewComponent}
        </div>
    );
}

interface DataPreviewExternalProps {
    data: uuid.UUIDTypes;
    sources: DataSource[];
}
function DataPreviewExternal({ data, sources }: DataPreviewExternalProps) {
    function downloadUrl(source: string, index?: number): string {
        const params = new URLSearchParams();
        params.append("data", uuidToString(data));
        params.append("source", source);
        if (index !== undefined) {
            console.debug(index);
            params.append("index", index.toString());
        }

        return `/resource/data/source?${params}`;
    }

    return (
        <ul className="overflow-hidden scroll-auto">
            {sources.map((source) => {
                switch (source.Cardinality) {
                    case DataSourceCardinalitySingle:
                        return (
                            <li
                                key={source.Label}
                                className="px-2 group/source"
                            >
                                <div className="flex gap-2">
                                    <div>{source.Label}</div>
                                    <div className="invisible group-hover/source:visible">
                                        <a href={downloadUrl(source.Label)}>
                                            <button
                                                type="button"
                                                className="btn-cmd align-middle"
                                            >
                                                <Icon.Download />
                                            </button>
                                        </a>
                                    </div>
                                </div>
                            </li>
                        );
                    case DataSourceCardinalityMultiple:
                        const srcs = source.Source as SourceFileInfo[];
                        srcs.sort((src, other) => src.Index - other.Index);
                        return (
                            <li key={source.Label} className="px-2">
                                <div className="flex gap-2 group/source">
                                    <div>{source.Label}</div>
                                    <div className="invisible group-hover/source:visible">
                                        <a href={downloadUrl(source.Label)}>
                                            <button
                                                type="button"
                                                className="btn-cmd align-middle"
                                            >
                                                <Icon.Download />
                                            </button>
                                        </a>
                                    </div>
                                </div>
                                <ol className="list-decimal list-inside">
                                    {srcs.map((src) => {
                                        return (
                                            <li
                                                key={src.Index.toString()}
                                                className="px-2 group/source-item"
                                            >
                                                <div className="inline-flex gap-2">
                                                    <div>{src.Label}</div>
                                                    <div className="invisible group-hover/source-item:visible">
                                                        <a
                                                            href={downloadUrl(
                                                                source.Label,
                                                                src.Index,
                                                            )}
                                                        >
                                                            <button
                                                                type="button"
                                                                className="btn-cmd align-middle"
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
                            </li>
                        );
                }
            })}
        </ul>
    );
}

interface DataPreviewInternalSingleProps {
    fields: SchemaFieldValues[];
}
function DataPreviewInternalSingle({ fields }: DataPreviewInternalSingleProps) {
    return (
        <div className="overflow-hidden scroll-auto">
            <table>
                <tbody>
                    {fields.map((field) => {
                        return (
                            <tr key={field.Label}>
                                <th>{field.Label}</th>
                                <td>{field.DType}</td>
                                <td>{field.Values}</td>
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
}

interface DataPreviewInternalMultipleProps {
    preview: DataValuesPreviewInternalMultiple;
}
function DataPreviewInternalMultiple({
    preview,
}: DataPreviewInternalMultipleProps) {
    const fields = preview.Values;
    fields.sort((field, other) => field.Label.localeCompare(other.Label));
    const values = new Array(preview.RecordCount);
    for (let idx = 0; idx < preview.RecordCount; idx++) {
        values[idx] = new Array(fields.length);
        for (const [fdx, field] of fields.entries()) {
            values[idx][fdx] = field.Values[fdx];
        }
    }

    return (
        <div>
            <div className="overflow-hidden scroll-auto">
                <table>
                    <thead>
                        <tr>
                            <th></th>
                            {fields.map((field) => (
                                <th key={field.Label}>{field.Label}</th>
                            ))}
                        </tr>
                        <tr>
                            <th></th>
                            {fields.map((field) => (
                                <th key={field.Label}>{field.DType}</th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {values.map((rx, idx) => {
                            return (
                                <tr key={idx.toString()}>
                                    <th>{idx + 1}</th>
                                    {rx.map((value, jdx) => {
                                        return <td key={jdx}>{value}</td>;
                                    })}
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>
            <div className="p-2">{preview.RecordCount} rows total</div>
        </div>
    );
}
