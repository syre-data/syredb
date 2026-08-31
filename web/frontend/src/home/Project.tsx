import {
    MouseButton,
    QUERY_KEY_DATA_PREVIEW,
    QUERY_KEY_PROJECT_RESOURCES,
    timestampToString,
    uuidToString,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
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
    type DataSource,
    type DataStorage,
    type DataType,
    type DataValuesPreviewInternal,
    type DataValuesPreviewInternalMultiple,
    type ProjectData,
    type SchemaFieldValues,
    type SourceFileInfo,
} from "@/types";
import { useDismiss, useFloating, useInteractions } from "@floating-ui/react";
import {
    QueryClient,
    useQuery,
    useQueryClient,
    useSuspenseQuery,
} from "@tanstack/react-query";
import classNames from "classnames";
import type { types } from "node:util";
import React, { Suspense, useState, type MouseEvent, type Ref } from "react";
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
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            <div>Could not load project</div>
        </SuspenseError>
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
            />
        </div>
    );
}

interface ProjectDataProps {
    projectId: uuid.UUIDTypes;
    data: ProjectData[];
    relations: { [key: string]: uuid.UUIDTypes[] };
    types: DataType[];
}
function ProjectData({ projectId, data, relations, types }: ProjectDataProps) {
    const params = new URLSearchParams();
    params.append("project", uuidToString(projectId));
    const download = `/resource/project/data?${params}`;

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
                    <div className="invisible group-hover:visible">
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
            </div>
            {data.length === 0 ? (
                <ProjectDataEmpty />
            ) : (
                <ProjectDataList
                    projectId={projectId}
                    data={data}
                    relations={relations}
                    types={types}
                />
            )}
        </div>
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

interface ProjectDataListProps {
    projectId: uuid.UUIDTypes;
    data: ProjectData[];
    relations: { [key: string]: uuid.UUIDTypes[] };
    types: DataType[];
}
function ProjectDataList({
    projectId,
    data,
    relations,
    types,
}: ProjectDataListProps) {
    let parents = new Map();
    for (const parent in relations) {
        const children = relations[parent]!;
        for (const child of children) {
            parents.set(child, parent);
        }
    }

    return (
        <table className="table-std">
            <tbody>
                {data.map((datum, idx) => {
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

                    return (
                        <ProjectDataItem
                            key={datum.Id.toString()}
                            index={idx}
                            projectId={projectId}
                            data={datum}
                            type={type}
                            parent={parent}
                            parentLabel={parent_label}
                        />
                    );
                })}
            </tbody>
        </table>
    );
}

interface ProjectDataItemProps {
    index: number;
    projectId: uuid.UUIDTypes;
    data: ProjectData;
    type: DataType;
    parent: ProjectData | undefined;
    parentLabel: string | undefined;
}
function ProjectDataItem({
    index,
    projectId,
    data,
    type,
    parent,
    parentLabel,
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
        queryClient.prefetchQuery({
            queryKey: [QUERY_KEY_DATA_PREVIEW, data.Id],
            queryFn: async () => await dataService.dataPreview(data.Id),
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

    const params = new URLSearchParams();
    params.append("id", data.Id.toString());
    params.append("project", uuidToString(projectId));
    const download = `/resource/data?${params}`;
    return (
        <tr id={`data-${data.Id}`} data-id={data.Id} className="group">
            <td className="text-nowrap  font-semibold w-0">{type.Label}</td>
            <td className="text-nowrap w-0 group/label">
                {data.Label ? (
                    <>
                        <span className="pr-2 font-semibold">{data.Label}</span>
                        <span className="text-gray-500 invisible group-hover/label:visible">
                            ({timestampToString(new Date(data.Timestamp))})
                        </span>
                    </>
                ) : (
                    <span className="font-semibold">
                        {timestampToString(new Date(data.Timestamp))}
                    </span>
                )}
            </td>
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
                                className="btn-cmd"
                                title="View data"
                            >
                                <Icon.Eye />
                            </button>
                        </Link>
                    </div>
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            title="Preview values"
                            onFocus={fetch_preview}
                            onMouseEnter={fetch_preview}
                            onMouseDown={open_preview}
                        >
                            <Icon.MagnifyingGlass />
                        </button>
                    </div>
                    <div>
                        <a href={download} title="Download">
                            <button type="button" className="btn-cmd">
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
