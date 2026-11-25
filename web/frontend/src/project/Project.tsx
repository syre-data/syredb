import {
    MouseButton,
    QUERY_KEY_DATA_PREVIEW,
    QUERY_KEY_PROJECT_RESOURCES,
    timestampToString,
    uuidToString,
} from "@/common";
import { Loading, StandardError } from "@/components";
import { typeIsNumeric, valueToString } from "@/components/Property";
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
    type DataCreator,
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
import { useDismiss, useFloating, useInteractions } from "@floating-ui/react";
import { useForm } from "@tanstack/react-form";
import { noop, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import {
    createColumnHelper,
    flexRender,
    columnVisibilityFeature,
    tableFeatures,
    useTable,
    type ColumnDef,
    type Row,
    type Table,
    type CellContext,
} from "@tanstack/react-table";
import classNames from "classnames";
import React, {
    Suspense,
    useMemo,
    useState,
    type ComponentProps,
    type MouseEvent,
    type Ref,
    type SubmitEvent,
} from "react";
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
            <div className="px-4 text-secondary">
                {resources.Project.Description}
            </div>
            <HeaderCards
                dataCount={resources.Data.length}
                userCount={resources.Users.length}
                className="pt-4 px-4"
            />
            <ProjectData
                projectId={projectId}
                data={resources.Data}
                relations={resources.DataRelations}
                types={resources.DataTypes}
                origins={resources.DataOrigins}
                users={resources.Users}
                className="pt-8"
            />
        </div>
    );
}

interface HeaderCardsProps extends ComponentProps<"div"> {
    dataCount: number;
    userCount: number;
}
function HeaderCards({
    dataCount,
    userCount,
    className,
    ...props
}: HeaderCardsProps) {
    return (
        <div className={`flex gap-4 ${className}`} {...props}>
            <div className="card">
                <div className="px-4 pt-2">Data count</div>
                <div className="flex items-center justify-end px-4 pb-2 text-right text-lg font-bold">
                    <div className="pr-2">
                        <Icon.Files />
                    </div>
                    <div>{dataCount}</div>
                </div>
            </div>
            <div className="card">
                <div className="px-4 pt-2">Collaborators</div>
                <div className="flex items-center justify-end px-4 pb-2 text-right text-lg font-bold">
                    <div className="pr-2">
                        <Icon.Users />
                    </div>
                    <div>{userCount}</div>
                </div>
            </div>
        </div>
    );
}

interface ProjectDataProps extends ComponentProps<"div"> {
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
    className,
    ...props
}: ProjectDataProps) {
    const params = new URLSearchParams();
    params.append("project", uuidToString(projectId));
    const download = `/resource/project/data?${params}`;

    const [dataFilter, setDataFilter] = useState<null | uuid.UUIDTypes[]>(null);

    return (
        <div className={`group/table-wrapper ${className}`} {...props}>
            <div className="px-4 flex gap-2 items-center">
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
                        "invisible group-hover/table-wrapper:visible in-focus-within:visible": true,
                        visible: dataFilter,
                    })}
                />
            </div>
            {data.length === 0 ? (
                <ProjectDataEmpty className="pt-4 px-4" />
            ) : (
                <div className="pt-4">
                    <ProjectDataTableView
                        projectId={projectId}
                        data={data}
                        relations={relations}
                        types={types}
                        origins={origins}
                        users={users}
                        filter={dataFilter}
                    />
                </div>
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

function ProjectDataEmpty(props: ComponentProps<"div">) {
    return (
        <div {...props}>
            <p>No data yet.</p>
            <p>
                Click the <Icon.Plus className="inline" /> above to create some.
            </p>
        </div>
    );
}

interface ProjectDataTableViewProps extends ComponentProps<"div"> {
    projectId: uuid.UUIDTypes;
    data: ProjectData[];
    relations: { [key: string]: uuid.UUIDTypes[] };
    types: DataType[];
    origins: DataOriginRx[];
    users: User[];
    filter: uuid.UUIDTypes<Uint8Array<ArrayBufferLike>>[] | null;
}
function ProjectDataTableView({
    projectId,
    data,
    relations,
    types,
    origins,
    users,
    filter,
    ...props
}: ProjectDataTableViewProps) {
    if (!filter) {
        return (
            <ProjectDataTable
                projectId={projectId}
                data={data}
                relations={relations}
                types={types}
                origins={origins}
                users={users}
                {...props}
            />
        );
    } else if (filter.length == 0) {
        return <ProjectDataFilterEmpty />;
    } else {
        const filteredData = data.filter((data) => filter.includes(data.Id));
        return (
            <div {...props}>
                <div className="text-sm px-4 pb-2">
                    Filtered {filteredData.length} out of {data.length} data
                </div>
                <ProjectDataTable
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

interface ProjectDataTableProps extends ComponentProps<"div"> {
    projectId: uuid.UUIDTypes;
    data: ProjectData[];
    relations: { [key: string]: uuid.UUIDTypes[] };
    types: DataType[];
    origins: DataOriginRx[];
    users: User[];
}

const dataTableFeatures = tableFeatures({
    columnVisibilityFeature,
});

function ProjectDataTable({
    projectId,
    data,
    relations,
    types,
    origins,
    users,
    ...props
}: ProjectDataTableProps) {
    const propertiesAll = data.flatMap((datum) =>
        datum.Properties
            ? datum.Properties.map((property) => {
                  return {
                      _id: `${property.Key}.${property.Type}`,
                      key: property.Key,
                      kind: property.Type,
                  };
              })
            : [],
    );
    const propertyInfo = [
        ...new Map(
            propertiesAll.map((property) => [property._id, property]),
        ).values(),
    ];
    propertyInfo.sort((a, b) => {
        if (a._id < b._id) {
            return -1;
        }
        if (b._id < a._id) {
            return 1;
        }
        throw new Error("property set was not unique");
    });

    const colHelper = createColumnHelper<
        typeof dataTableFeatures,
        ProjectData
    >();
    const colDataType = colHelper.accessor("Type", {
        header: "Type",
        cell: ({ getValue, table }) => {
            const typeId = getValue();
            const dataTypes = table.options.meta!.dataTypes as DataType[];
            const dtype = dataTypes.find((dtype) => dtype.Id === typeId);
            if (!dtype) {
                throw new Error("invalid data type");
            }

            return dtype.Label;
        },
    });
    const colLabel = colHelper.accessor("Label", {
        header: "Label",
    });
    const colCreator = colHelper.accessor("Creator", {
        header: "Creator",
        cell: ({ getValue, table }) => {
            const creator = getValue() as DataCreator;
            const createdByUser = !!creator.Origin;
            if (createdByUser) {
                const user: User = table.options.meta!.users.find(
                    (user: User) => creator.Id === user.Id,
                );
                if (!user) {
                    throw new Error("invalid user creator");
                }

                if (user.Name) {
                    return (
                        <div className="flex gap-2 items-center">
                            <span>{user.Name}</span>
                            <span>
                                <a
                                    href={`mailto:${user.Email}`}
                                    target="_blank"
                                    title={user.Email}
                                    className="invisible group-hover:visible"
                                >
                                    <Icon.Mail />
                                </a>
                            </span>
                        </div>
                    );
                } else {
                    return user.Email;
                }
            } else {
                throw new Error("TODO: created by transform");
            }
        },
    });
    const colOrigin = colHelper.accessor("Creator.Origin", {
        header: "Origin",
        cell: ({ getValue, table }) => {
            const originId = getValue();
            if (!originId) {
                return "";
            }

            const origins: DataOriginRx[] = table.options.meta!.origins;
            const origin = origins.find((origin) => origin.Id === originId);
            return origin?.Label;
        },
    });
    const colTimestamp = colHelper.accessor("Timestamp", {
        header: "Timestamp",
        cell: (cell) => timestampToString(cell.getValue()),
    });
    const colProperties = colHelper.group({
        id: "properties",
        header: "Properties",
        columns: colHelper.columns(
            propertyInfo.map((property) => {
                return colHelper.accessor(
                    (data) => {
                        return data.Properties?.find(
                            (prop) =>
                                prop.Key === property.key &&
                                prop.Type === property.kind,
                        );
                    },
                    {
                        id: `property.${property.key}.${property.kind}`,
                        cell: (cell) => {
                            const property = cell.getValue();
                            if (!property) {
                                return (
                                    <span className="inline-block w-full text-secondary text-center">
                                        -
                                    </span>
                                );
                            }
                            const value = valueToString(property);
                            if (typeIsNumeric(property.Type)) {
                                return <span>{value}</span>;
                            } else {
                                return <span>{value}</span>;
                            }
                        },
                        header: () => {
                            return (
                                <div>
                                    <span>{property.key}</span>
                                    <span className="text-secondary text-sm font-normal pl-1">
                                        ({property.kind})
                                    </span>
                                </div>
                            );
                        },
                        meta: {
                            property,
                        },
                    },
                );
            }),
        ),
    });
    const colActions = colHelper.display({
        id: "actions",
        cell: (cell) => <DataRowActions cell={cell} />,
        enableHiding: false,
    });
    const columns = useMemo<
        Array<ColumnDef<typeof dataTableFeatures, ProjectData>>
    >(
        () =>
            colHelper.columns([
                colDataType,
                colLabel,
                colCreator,
                colOrigin,
                colTimestamp,
                colProperties,
                colActions,
            ]),
        [],
    );

    const parents = new Map();
    for (const parent in relations) {
        const children = relations[parent]!;
        for (const child of children) {
            parents.set(child, parent);
        }
    }

    const table = useTable({
        key: "project-data-table",
        features: dataTableFeatures,
        columns,
        data,
        getRowId: (data) => uuidToString(data.Id),
        meta: {
            projectId: projectId,
            dataTypes: types,
            users,
            origins,
            parents,
        },
    });

    return (
        <div {...props}>
            <ProjectDataTableColumnSelector
                table={table}
                className="px-4 pb-2"
            />
            <div className="w-full overflow-x-scroll">
                <ProjectDataTableInner table={table} />
            </div>
        </div>
    );
}

interface DataRowActionsProps {
    cell: CellContext<typeof dataTableFeatures, ProjectData>;
}
function DataRowActions({ cell }: DataRowActionsProps) {
    const projectId = cell.table.options.meta!.projectId;
    const dataId = cell.row.original.Id;
    const dataType: DataType | undefined =
        cell.table.options.meta!.dataTypes.find(
            (dtype) => dtype.Id === cell.row.original.Type,
        );
    if (!dataType) {
        throw new Error("invalid data type");
    }

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

    function highlight_parent(_: MouseEvent<HTMLElement>, active: boolean) {
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
            queryKey: [QUERY_KEY_DATA_PREVIEW, dataId],
            queryFn: async () =>
                await dataService.dataPreview(dataId).catch(noop),
        });
    }

    function open_preview(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        setPreviewOpen(true);
    }

    const download_params = new URLSearchParams();
    download_params.append("id", dataId.toString());
    download_params.append("project", uuidToString(projectId));
    const download_path = `/resource/data?${download_params}`;

    return (
        <div
            className={classNames({
                "flex gap-1": true,
                "invisible group-hover:visible": !previewOpen,
            })}
        >
            <div>
                {/* TODO: Check permissions */}
                <Link to={`/data/${dataId}?project=${projectId}`}>
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
            {previewOpen ? (
                <div className="relative">
                    <DataPreview
                        ref={refs.setFloating}
                        style={floatingStyles}
                        floatingProps={getFloatingProps}
                        data_id={dataId}
                        storage={dataType.Storage}
                        setPreviewOpen={setPreviewOpen}
                    />
                </div>
            ) : null}
        </div>
    );
}

interface ProjectDataTablePropertySelectorProps extends ComponentProps<"div"> {
    table: Table<typeof dataTableFeatures, ProjectData>;
}
function ProjectDataTableColumnSelector({
    table,
    ...props
}: ProjectDataTablePropertySelectorProps) {
    return (
        <div {...props}>
            <form>
                <div className="flex gap-2">
                    {table
                        .getAllColumns()
                        .filter((column) => column.id !== "properties")
                        .map((column) => {
                            if (!column.getCanHide()) {
                                return null;
                            }
                            return (
                                <div key={column.id}>
                                    <label className="flex gap-1">
                                        <input
                                            type="checkbox"
                                            checked={column.getIsVisible()}
                                            onChange={column.getToggleVisibilityHandler()}
                                        />
                                        <div>
                                            <span>{column.id}</span>
                                            <span className="pl-1 text-secondary">
                                                {/* ({property.kind}) */}
                                            </span>
                                        </div>
                                    </label>
                                </div>
                            );
                        })}
                </div>
                <div className="flex gap-2">
                    <span className="font-bold pr-2">Properties</span>
                    <div className="flex gap-2">
                        {table
                            .getColumn("properties")!
                            .columns.map((column) => {
                                const property =
                                    column.columnDef.meta!.property;
                                return (
                                    <div key={column.id}>
                                        <label className="flex gap-1">
                                            <input
                                                type="checkbox"
                                                checked={column.getIsVisible()}
                                                disabled={!column.getCanHide()}
                                                onChange={column.getToggleVisibilityHandler()}
                                            />
                                            <div>
                                                <span>{property.key}</span>
                                                <span className="pl-1 text-secondary">
                                                    ({property.kind})
                                                </span>
                                            </div>
                                        </label>
                                    </div>
                                );
                            })}
                    </div>
                </div>
            </form>
        </div>
    );
}

interface ProjectDataTableInnerProps {
    table: Table<typeof dataTableFeatures, ProjectData>;
}
function ProjectDataTableInner({ table }: ProjectDataTableInnerProps) {
    return (
        <table
            className="table-compact w-full \
            [&_th]:last:w-full [&_td]:last:w-full [&_td]:text-nowrap \
            [&_th]:first:pl-4 [&_td]:first:pl-4"
        >
            <thead className="[&_th]:bg-gray-200 dark:[&_th]:bg-gray-700">
                {table.getHeaderGroups().map((headerGroup) => {
                    return (
                        <tr key={headerGroup.id}>
                            {headerGroup.headers.map((header) => {
                                return (
                                    <th
                                        key={header.id}
                                        colSpan={header.colSpan}
                                    >
                                        {header.isPlaceholder ? null : (
                                            <table.FlexRender header={header} />
                                        )}
                                    </th>
                                );
                            })}
                        </tr>
                    );
                })}
            </thead>
            <tbody>
                {table.getRowModel().rows.map((row) => {
                    const parent = table.options.meta!.parents[row.original.Id];
                    return (
                        <ProjectDataRow
                            key={row.id}
                            row={row}
                            parent={parent}
                            className="group"
                        />
                    );
                })}
            </tbody>
        </table>
    );
}

interface ProjectDataRowProps extends ComponentProps<"tr"> {
    row: Row<typeof dataTableFeatures, ProjectData>;
    parent?: ProjectData;
}
function ProjectDataRow({
    row,
    parent,
    className,
    ...props
}: ProjectDataRowProps) {
    const meta = row.table.options.meta!;
    const data = row.original;
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

    function highlight_parent(_: MouseEvent<HTMLElement>, active: boolean) {
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

    const download_params = new URLSearchParams();
    download_params.append("id", data.Id.toString());
    download_params.append("project", uuidToString(meta.projectId));
    const download_path = `/resource/data?${download_params}`;

    return (
        <tr
            id={`data-${data.Id}`}
            data-id={data.Id}
            className={`group hover:bg-primary-600 ${className}`}
            {...props}
        >
            {row.getVisibleCells().map((cell) => {
                return (
                    <td key={cell.id} className="select-all">
                        {flexRender(
                            cell.column.columnDef.cell,
                            cell.getContext(),
                        )}
                    </td>
                );
            })}
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
        <StandardError
            resetErrorBoundary={resetErrorBoundary}
            error={error as Error}
            title="Could not load data preview"
        />
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
