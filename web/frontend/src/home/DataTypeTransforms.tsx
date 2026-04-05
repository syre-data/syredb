import {
    MouseButton,
    QUERY_KEY_DATA_TYPE_TRANSFORMS,
    QUERY_KEY_DATA_TYPES,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import type { DataType, DataTypeTransform, DataTypeTransformRx } from "@/types";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Suspense, type MouseEvent } from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { Link, useNavigate } from "react-router";

export default function () {
    return (
        <ErrorBoundary FallbackComponent={DataTypeTransformsError}>
            <Suspense fallback={<Loading />}>
                <DataTypeTransforms />
            </Suspense>
        </ErrorBoundary>
    );
}

function DataTypeTransformsError({ error, resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            <div>Could not load data type transforms</div>
        </SuspenseError>
    );
}

function DataTypeTransforms() {
    const { data: transforms } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPE_TRANSFORMS],
        queryFn: dataService.dataTypeTransformsGetAll,
    });

    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    return (
        <div>
            <div className="flex justify-between px-4 pt-2">
                <div className="flex gap-2">
                    <h2 className="text-lg">Data type transforms</h2>
                    <div>
                        <Link to="/data-type-transform/create">
                            <button type="button" className="btn-cmd">
                                <Icon.Plus />
                            </button>
                        </Link>
                    </div>
                </div>
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
            {transforms.length === 0 ? (
                <DataTypeTransformsEmpty />
            ) : (
                <DataTypeTransformsContent transforms={transforms} />
            )}
        </div>
    );
}

function DataTypeTransformsEmpty() {
    return (
        <div className="px-4">
            Create a data type transform by clicking the{" "}
            <Icon.Plus className="inline" /> above
        </div>
    );
}

interface DataTypeTransformsContentProps {
    transforms: DataTypeTransform[];
}
function DataTypeTransformsContent({
    transforms,
}: DataTypeTransformsContentProps) {
    const { data: data_types } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPES],
        queryFn: dataService.dataTypesGetAll,
    });

    return (
        <ul className="px-4 grid grid-cols-[repeat(2,min-content)]">
            {transforms.map((transform) => (
                <DataTypeTransformItem
                    key={transform.Id.toString()}
                    transform={transform}
                    dataTypes={data_types}
                />
            ))}
        </ul>
    );
}

interface DataTypeTransformItemProps {
    transform: DataTypeTransform;
    dataTypes: DataType[];
}
function DataTypeTransformItem({
    transform,
    dataTypes,
}: DataTypeTransformItemProps) {
    const source_idx = dataTypes.findIndex(
        (type) => type.Id === transform.Source,
    );
    if (source_idx < 0) {
        console.error("invalid transform source");
    }
    const destination_idx = dataTypes.findIndex(
        (type) => type.Id === transform.Destination,
    );
    if (source_idx < 0) {
        console.error("invalid transform destination");
    }

    const source = dataTypes[source_idx];
    const destination = dataTypes[destination_idx];
    return (
        <li className="col-span-full grid grid-cols-subgrid">
            <div className="col-1 whitespace-nowrap">{transform.Label}</div>
            <div className="col-2 flex gap-2 items-center">
                <span className="whitespace-nowrap">{source.Label}</span>
                <Icon.ArrowRight className="inline" />
                <span className="whitespace-nowrap">{destination.Label}</span>
            </div>
        </li>
    );
}
