import { useSuspenseQuery } from "@tanstack/react-query";
import { useParams, Link, useNavigate } from "react-router";
import * as app from "../../bindings/syredb/app";
import icon from "../icon";
import { ErrorBoundary, FallbackProps } from "react-error-boundary";
import {
    createContext,
    FormEvent,
    MouseEvent,
    Suspense,
    useContext,
    useEffect,
    useLayoutEffect,
    useRef,
    useState,
} from "react";
import * as common from "../common";
import classNames from "classnames";

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

    if (error === common.USER_NOT_AUTHENTICATED_ERROR) {
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
            <div>{error}</div>
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
    const navigate = useNavigate();
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
            <div>
                <ProjectHeader project={project_resources.Project} />
                <ProjectSampleList
                    samples={project_resources.Samples}
                    className="pt-2"
                />
            </div>
        </CommonProjectDataCtx>
    );
}

interface ProjectHeaderProps {
    project: app.Project;
}
function ProjectHeader({ project }: ProjectHeaderProps) {
    return (
        <div className="flex gap-2 pt-1 pb-2 px-2 border-b">
            <h2
                className={`group/project-header-label flex gap-2 grow font-bold`}
            >
                {project.Label}
            </h2>
            <div className="flex gap-1 items-center">
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

interface ProjectSampleListProps {
    samples: app.ProjectSample[];
    className: string;
}
function ProjectSampleList({ samples, className }: ProjectSampleListProps) {
    const project_data = useContext(CommonProjectDataCtx);

    return (
        <div className={className}>
            <div className="flex gap-2 px-4">
                <h3 className="font-bold">Samples</h3>
                {common.is_admin_or_owner(project_data.user_permission) ? (
                    <div>
                        <Link
                            to={`/project/${project_data.project_id}/samples/create`}
                        >
                            <button type="button" className="btn-cmd">
                                <icon.Plus />
                            </button>
                        </Link>
                    </div>
                ) : null}
            </div>
            {samples.length == 0 ? (
                <ProjectSampleListEmpty />
            ) : (
                <ul className="grid grid-cols-[50px_100px_200px]">
                    {samples.map((sample, index) => (
                        <li key={sample.Id.toString()} className="contents">
                            <ProjectSampleListItem
                                index={index}
                                sample={sample}
                            />
                        </li>
                    ))}
                </ul>
            )}
        </div>
    );
}

function ProjectSampleListEmpty() {
    const project_data = useContext(CommonProjectDataCtx);
    return (
        <div className="px-4">
            <div>No samples</div>
            {common.is_admin_or_owner(project_data.user_permission) ? (
                <div>
                    <small>
                        Click the <icon.Plus className="inline" /> to add some
                    </small>
                </div>
            ) : null}
        </div>
    );
}

interface ProjectSampleListItemProps {
    index: number;
    sample: app.ProjectSample;
}
function ProjectSampleListItem({ index, sample }: ProjectSampleListItemProps) {
    const ROW_SPAN = 2;
    const [expanded, setExpanded] = useState(false);

    function toggle_expand(e: MouseEvent<HTMLDivElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        setExpanded(!expanded);
    }

    const properties = sample.Properties.toSorted((a, b) => {
        const ka = a.Key.toLowerCase().charCodeAt(0);
        const kb = b.Key.toLowerCase().charCodeAt(0);
        return ka - kb;
    });

    const row_idx = index * ROW_SPAN;
    const main_row_idx = row_idx + 1;
    const properties_row_idx = main_row_idx + 1;
    return (
        <div className="contents group/sample-row">
            <div
                className={classNames({
                    "col-start-1 pl-4 cursor-pointer": true,
                    "invisible group-hover/sample-row:visible": !expanded,
                })}
                style={{
                    gridRow: main_row_idx,
                }}
                onMouseDown={toggle_expand}
            >
                <button
                    type="button"
                    className={classNames({
                        "transition-[rotate]": true,
                        "-rotate-90": !expanded,
                    })}
                >
                    <icon.CaretDown />
                </button>
            </div>
            <div
                className="col-start-2 cursor-pointer"
                style={{ gridRow: main_row_idx }}
                onMouseDown={toggle_expand}
            >
                {sample.Label}
            </div>
            <div className="col-start-3" style={{ gridRow: main_row_idx }}>
                {sample.Tags.length === 0
                    ? "(no tags)"
                    : sample.Tags.join(", ")}
            </div>
            <div
                className={classNames({
                    "col-start-2 -col-end-1 flex gap-2 overflow-y-hidden transition-[height]":
                        true,
                    "h-0": !expanded,
                })}
                style={{ gridRow: properties_row_idx }}
            >
                {properties.length === 0 ? (
                    <ProjectSamplePropertiesEmpty />
                ) : (
                    properties.map((property) => (
                        <ProjectSampleProperty
                            key={property.Key}
                            property={property}
                        />
                    ))
                )}
            </div>
        </div>
    );
}

function ProjectSamplePropertiesEmpty() {
    return <div>(no properties)</div>;
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
