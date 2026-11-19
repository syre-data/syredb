import { Context } from "@/AppStateContext";
import {
    hasDbPermission,
    MouseButton,
    QUERY_KEY_DATA_SCHEMA,
    QUERY_KEY_DATA_TYPE,
    QUERY_KEY_DATA_TYPE_RESOURCES,
    timestampToString,
    uuidToString,
    uuidV7ToTimestamp,
} from "@/common";
import { Loading, StandardError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import {
    DataCreatorTypeTransform,
    DataCreatorTypeUser,
    DataSourceCardinalityMultiple,
    DataSourceCardinalitySingle,
    DataStorageExternal,
    DataStorageInternal,
    DbPermissionDataTypeModify,
    type DataType,
    type DataTypeExternal,
    type DataTypeExternalSourceRx,
    type DataTypeInternal,
    type DataTypeResources,
} from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
    Suspense,
    useContext,
    type ComponentProps,
    type MouseEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, useNavigate, useParams } from "react-router";

export default function () {
    const navigate = useNavigate();
    const { data_type_id } = useParams();
    if (data_type_id === undefined) {
        console.debug("data type id not present");
        navigate(-1);
        return;
    }

    return (
        <ErrorBoundary FallbackComponent={DataTypeError}>
            <Suspense fallback={<Loading />}>
                <DataType data_type_id={data_type_id} />
            </Suspense>
        </ErrorBoundary>
    );
}

function DataTypeError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <StandardError
            resetErrorBoundary={resetErrorBoundary}
            error={error as Error}
            title={"Could not load data type resources"}
        />
    );
}

interface DataTypeProps {
    data_type_id: string;
}
function DataType({ data_type_id }: DataTypeProps) {
    const RECENT_DATA_LIMIT = 25;
    const { data: dataType } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPE, data_type_id],
        queryFn: async () => dataService.dataTypeGet(data_type_id),
    });
    const { data: dataTypeResources } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPE_RESOURCES, data_type_id],
        queryFn: async () =>
            dataService.dataTypeResources(data_type_id, RECENT_DATA_LIMIT),
    });

    return (
        <div>
            <Header dataType={dataType} />
            <div className="px-4 pt-1">{dataType.Description}</div>
            <OverviewCards
                dataType={dataType}
                dataCount={dataTypeResources.DataCount}
                className="px-4 pt-4"
            />
            {dataType.Storage === DataStorageExternal ? (
                <StorageExternal
                    dataType={dataType as DataTypeExternal}
                    className="px-4 pt-2"
                />
            ) : null}
            <DataList dataType={dataType} resources={dataTypeResources} />
        </div>
    );
}

interface HeaderProps {
    dataType: DataType;
}
function Header({ dataType }: HeaderProps) {
    const navigate = useNavigate();
    const ctx = useContext(Context);
    const user = ctx.user;

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    const canModifyType = hasDbPermission(
        DbPermissionDataTypeModify,
        user.DbPermissions,
    );

    return (
        <div>
            <div className="px-4 pt-2 flex justify-between">
                <div className="flex gap-2 items-center">
                    <h1 className="title">{dataType.Label}</h1>
                    <div>
                        {dataType.Active ? (
                            <Icon.CircleCheck
                                title="Active"
                                className=" text-syre-green-700 dark:text-syre-green-500"
                            />
                        ) : (
                            <Icon.CircleX
                                title="Inactive"
                                className="text-secondary"
                            />
                        )}
                    </div>
                    <div>
                        {canModifyType ? (
                            <Link to={`/data-type/${dataType.Id}/edit`}>
                                <button type="button" className="btn-cmd">
                                    <Icon.Pen />
                                </button>
                            </Link>
                        ) : null}
                    </div>
                </div>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        title="Close"
                        onMouseDown={close}
                    >
                        <Icon.Close />
                    </button>
                </div>
            </div>
        </div>
    );
}

interface OverviewCards extends ComponentProps<"div"> {
    dataType: DataType;
    dataCount: number;
}
function OverviewCards({
    dataType,
    dataCount,
    className,
    ...props
}: OverviewCards) {
    let storageTitle;
    let storageIcon;
    switch (dataType.Storage) {
        case DataStorageInternal:
            storageTitle = "Table";
            storageIcon = <Icon.Columns />;
            break;
        case DataStorageExternal:
            storageTitle = "Files";
            storageIcon = <Icon.Files />;
            break;
        default:
            console.error(`invalid data type storage: ${dataType}`);
            throw new Error("invalid data type storage");
    }
    return (
        <div className={`flex gap-4 ${className}`} {...props}>
            <div className="card px-4">
                <div>Records</div>
                <p className="pt-2 w-full text-right text-lg font-bold select-all">
                    {dataCount}
                </p>
            </div>
            <div className="card px-4">
                <div>Storage</div>
                <div className="pt-2 inline-flex gap-1 items-center">
                    {storageIcon}
                    <span className="font-bold select-all">{storageTitle}</span>
                </div>
            </div>
            {dataType.Storage === DataStorageInternal ? (
                <div className="card px-4">
                    <div>Schema</div>
                    <div className="pt-2 text-right">
                        <OverviewCardStorageInternal
                            dataType={dataType as DataTypeInternal}
                            className="font-bold hover:underline"
                        />
                    </div>
                </div>
            ) : null}
        </div>
    );
}

interface OverviewCardStorageInternalProps extends ComponentProps<"a"> {
    dataType: DataTypeInternal;
}
function OverviewCardStorageInternal({
    dataType,
    ...props
}: OverviewCardStorageInternalProps) {
    const { data: schemaResources } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_SCHEMA, dataType.Schema],
        queryFn: async () =>
            await dataService.dataSchemaResourcesGet(dataType.Schema),
    });

    return (
        <Link
            to={`/data-schema/${dataType.Schema}`}
            title="To data schema"
            {...props}
        >
            {schemaResources.DataSchema.Label}
        </Link>
    );
}

interface StorageExternalProps extends ComponentProps<"div"> {
    dataType: DataTypeExternal;
}
function StorageExternal({ dataType, ...props }: StorageExternalProps) {
    return (
        <div {...props}>
            <div className="pb-2">
                <h2 className="text-lg font-bold">Data sources</h2>
            </div>
            <ol className="flex gap-4">
                {dataType.Sources.map((source) => (
                    <DataTypeSource
                        key={source.Id.toString()}
                        source={source}
                        className="card px-4 py-2"
                    />
                ))}
            </ol>
        </div>
    );
}

interface DataTypeSourceProps extends ComponentProps<"li"> {
    source: DataTypeExternalSourceRx;
}
function DataTypeSource({ source, ...props }: DataTypeSourceProps) {
    return (
        <li {...props}>
            <div className="flex items-center">
                <div>
                    <span title="Label" className="font-bold">
                        {source.Label}
                    </span>
                    {source.Required ? (
                        <span title="Required" className="pl-0.5">
                            *
                        </span>
                    ) : null}
                </div>
                <div className="pl-2">
                    {source.Cardinality === DataSourceCardinalitySingle ? (
                        <Icon.File title="Single file source" />
                    ) : null}
                    {source.Cardinality === DataSourceCardinalityMultiple ? (
                        <Icon.Files title="Multiple file source" />
                    ) : null}
                </div>
            </div>
            <div className="text-sm text-secondary">
                {source.Description ?? "(no description)"}
            </div>
            <div className="pt-2 text-sm">
                <div>Media types</div>
                <div className="pt-1">
                    {source.MediaTypes.length > 0 ? (
                        source.MediaTypes.map((media) => {
                            return (
                                <span className="px-2 py-0.5 rounded-full text-white font-semibold bg-primary-600 select-all">
                                    {media}
                                </span>
                            );
                        })
                    ) : (
                        <span className="text-secondary">(no filters)</span>
                    )}
                </div>
            </div>
        </li>
    );
}

interface DataListProps extends ComponentProps<"div"> {
    dataType: DataType;
    resources: DataTypeResources;
}
function DataList({ dataType, resources, ...props }: DataListProps) {
    return (
        <div {...props}>
            <div className="px-4 pt-4">
                <h2 className="text-lg font-bold pb-2 px-4">Recent data</h2>
                <table className="table-compact w-full [&_th]:last:w-full [&_td]:last:w-full [&_td]:text-nowrap [&_th]:first:pl-4 [&_td]:first:pl-4">
                    <thead>
                        <tr>
                            <th className="bg-gray-200 dark:bg-gray-700 rounded-tl-lg">
                                #
                            </th>
                            <th className="bg-gray-200 dark:bg-gray-700">
                                Created
                            </th>
                            <th className="bg-gray-200 dark:bg-gray-700">By</th>
                            <th className="bg-gray-200 dark:bg-gray-700 rounded-tr-lg"></th>
                        </tr>
                    </thead>
                    <tbody>
                        {resources.RecentData.map((data, idx) => {
                            const created = uuidV7ToTimestamp(data.Data.Id);
                            let creator;
                            let toUrl;
                            switch (data.Data.CreatorType) {
                                case DataCreatorTypeTransform:
                                    const transform = resources.Transforms.find(
                                        (transform) =>
                                            transform.Id === data.Creator,
                                    );
                                    if (!transform) {
                                        throw new Error(
                                            "invalid data creator transform",
                                        );
                                    }
                                    creator = transform.Label;
                                    toUrl = `/data-type-transform/${uuidToString(transform.Id)}`;
                                    break;
                                case DataCreatorTypeUser:
                                    const user = resources.Users.find(
                                        (transform) =>
                                            transform.Id === data.Creator,
                                    );
                                    if (!user) {
                                        throw new Error(
                                            "invalid data creator user",
                                        );
                                    }
                                    if (user.Name) {
                                        creator = user.Name;
                                    } else {
                                        creator = user.Email;
                                    }
                                    toUrl = `/user/${uuidToString(user.Id)}`;
                                    break;
                                default:
                                    throw new Error(
                                        "invaldi data creator type",
                                    );
                            }
                            return (
                                <tr
                                    key={data.Data.Id.toString()}
                                    className="group even:bg-gray-200 dark:even:bg-gray-700 \
                                    hover:bg-primary-400 dark:hover:bg-primary-700"
                                >
                                    <td>{idx + 1}</td>
                                    <td>{timestampToString(created)}</td>
                                    <td>{creator}</td>
                                    <td>
                                        <div className="invisible group-hover:visible">
                                            <Link to={toUrl}>
                                                <Icon.Eye />
                                            </Link>
                                        </div>
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>
        </div>
    );
}
