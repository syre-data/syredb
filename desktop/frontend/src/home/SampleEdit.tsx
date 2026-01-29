import {
    Suspense,
    MouseEvent,
    FormEvent,
    ChangeEvent,
    useContext,
    Dispatch,
    useState,
} from "react";
import { ErrorBoundary, FallbackProps } from "react-error-boundary";
import { Link, useNavigate, useParams } from "react-router";
import * as common from "../common";
import { useSuspenseQuery } from "@tanstack/react-query";
import * as app from "../../bindings/syredb/app";
import icon from "../icon";
import { UUID } from "../../bindings/github.com/google/uuid";
import {
    InputPropertyValue,
    value_to_string as property_value_to_string,
    SelectPropertyType,
    type_string_to_variant,
} from "../components/Property";
import { immerable } from "immer";
import { useImmerReducer } from "use-immer";
import * as appStateCtx from "../AppStateContext";

export default function () {
    const navigate = useNavigate();
    const { project_id, sample_id } = useParams();
    if (project_id && sample_id) {
        return (
            <ErrorBoundary FallbackComponent={SampleError}>
                <Suspense fallback={<Loading />}>
                    <Sample project_id={project_id} sample_id={sample_id} />
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

function SampleError({ error, resetErrorBoundary }: FallbackProps) {
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

class SampleState {
    [immerable] = true;
    dirty: boolean;
    label: string;
    tags: string[];
    properties: app.Property[];
    project_notes: app.ProjectSampleNote[];
    data: app.SampleData[];
    users: app.User[];
    user_permissions: app.SampleUserPermissions[];

    constructor(resources: app.ProjectSampleResources) {
        this.dirty = false;
        this.label = resources.ProjectMembership.Label;
        this.tags = resources.ProjectTags;
        this.properties = resources.Properties;
        this.project_notes = resources.ProjectNotes;
        this.data = resources.Data;
        this.users = resources.Users;
        this.user_permissions = resources.UserPermissions;
    }
}

type SampleStateAction =
    | { type: "set_dirty" }
    | { type: "remove_property"; payload: { key: string } };

function sampleStateReducer(draft: SampleState, action: SampleStateAction) {
    switch (action.type) {
        case "set_dirty":
            draft.dirty = true;
            break;
        case "remove_property":
            draft.properties = draft.properties.filter(
                (property) => property.Key !== action.payload.key,
            );
            draft.dirty = true;
            break;
    }
}

interface SampleProps {
    project_id: UUID;
    sample_id: UUID;
}
function Sample({ project_id, sample_id }: SampleProps) {
    const { data: sample_resources } = useSuspenseQuery({
        queryKey: ["project_sample_resources", project_id, sample_id],
        queryFn: async () =>
            app.ProjectService.GetProjectSampleResources(project_id, sample_id),
    });
    const [sampleState, sampleStateDispatch] = useImmerReducer(
        sampleStateReducer,
        new SampleState(sample_resources),
    );
    const appState = useContext(appStateCtx.Context);
    const navigate = useNavigate();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        if (sampleState.dirty) {
            const proceed = window.confirm(
                "You have unsaved changes. Are you sure you want to discard them?",
            );
            if (!proceed) {
                return;
            }
        }

        navigate(-1);
    }

    function validate_sample_label(e: ChangeEvent<HTMLInputElement>) {
        const value = e.target.value;
    }

    function update_sample(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
    }

    const user_permissions =
        sample_resources.UserPermissions.find(
            (permissions) => permissions.User === appState.user.Id,
        )?.Permissions ?? [];

    const user_can_modify_properties =
        user_permissions.includes(
            app.SampleUserPermission.SAMPLE_USER_PERMISSION_MODIFY_PROPERTIES,
        ) ||
        user_permissions.includes(
            app.SampleUserPermission.SAMPLE_USER_PERMISSION_OWNER,
        );

    const user_can_add_data =
        user_permissions.includes(
            app.SampleUserPermission.SAMPLE_USER_PERMISSION_ADD_DATA,
        ) ||
        user_permissions.includes(
            app.SampleUserPermission.SAMPLE_USER_PERMISSION_OWNER,
        );

    const FORM_ROW_CLASSNAME = "pb-2 col-span-full grid grid-cols-subgrid";
    return (
        <div>
            <div className="flex px-4 py-2">
                <h1 className="grow text-lg font-bold">
                    {sample_resources.ProjectMembership.Label}
                </h1>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={close}
                    >
                        <icon.Close />
                    </button>
                </div>
            </div>
            <form onSubmit={update_sample}>
                <div className="pb-2">
                    <section className="px-4 pb-2 grid grid-cols-[repeat(2,min-content)] gap-2">
                        <div className={FORM_ROW_CLASSNAME}>
                            <label className="contents">
                                <span className="col-1">Label</span>
                                <input
                                    type="text"
                                    defaultValue={
                                        sample_resources.ProjectMembership.Label
                                    }
                                    placeholder="Label"
                                    className="col-2 input-basic"
                                    onChange={validate_sample_label}
                                />
                            </label>
                        </div>
                        <div className={FORM_ROW_CLASSNAME}>
                            <label className="contents">
                                <span className="col-1">Tags</span>
                                <input
                                    type="text"
                                    defaultValue={sample_resources.ProjectTags.join(
                                        ", ",
                                    )}
                                    placeholder="Tags"
                                    className="col-2 input-basic"
                                />
                            </label>
                        </div>
                    </section>
                    {user_can_modify_properties ? (
                        <SamplePropertiesEditable
                            properties={sampleState.properties}
                            sampleStateDispatch={sampleStateDispatch}
                        />
                    ) : (
                        <SampleProperties properties={sampleState.properties} />
                    )}
                    {user_can_add_data ? (
                        <SampleDataEditable data={sampleState.data} />
                    ) : (
                        <SampleData data={sample_resources.Data} />
                    )}
                </div>
                <div className="flex gap-2 justify-center">
                    <div>
                        <button type="submit" className="btn-submit">
                            Save
                        </button>
                    </div>
                </div>
            </form>
        </div>
    );
}

interface SamplePropertiesProps {
    properties: app.Property[];
}
function SampleProperties({ properties }: SamplePropertiesProps) {
    const FORM_ROW_CLASSNAME = "pb-2 col-span-full grid grid-cols-subgrid";

    return (
        <section>
            <div>
                <h2 className="text-lg font-bold">Properties</h2>
            </div>
            <div className="grid gap-2 grid-cols-[]">
                {properties.map((property) => (
                    <PropertyStatic
                        property={property}
                        className={FORM_ROW_CLASSNAME}
                    />
                ))}
            </div>
        </section>
    );
}

interface PropertyStaticProps {
    property: app.Property;
    className?: string;
}
function PropertyStatic({ property, className }: PropertyStaticProps) {
    return (
        <div className={className ?? ""}>
            <div className="col-1">{property.Key}</div>
            <div className="col-2">{property_value_to_string(property)}</div>
        </div>
    );
}

interface SamplePropertiesEditableProps {
    properties: app.Property[];
    sampleStateDispatch: Dispatch<SampleStateAction>;
}
function SamplePropertiesEditable({
    properties,
    sampleStateDispatch,
}: SamplePropertiesEditableProps) {
    const [newProperties, setNewProperties] = useState<number[]>([]);

    function add_property(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        const new_id =
            newProperties.length === 0
                ? 0
                : newProperties[newProperties.length - 1] + 1;
        setNewProperties([...newProperties, new_id]);
    }

    function remove_new_property(id: number) {
        setNewProperties(newProperties.filter((pid) => pid !== id));
    }

    return (
        <section>
            <div className="flex gap-2 items-center px-4 pb-2">
                <h2 className="text-g font-bold">Properties</h2>
                <div>
                    <button
                        type="button"
                        className="btn-cmd align-middle"
                        onMouseDown={add_property}
                    >
                        <icon.Plus />
                    </button>
                </div>
            </div>
            <div className="pb-2 grid gap-2 grid-cols-[repeat(3,min-content)]">
                {properties.map((property) => (
                    <PropertyEditable
                        property={property}
                        className="px-4 col-span-full grid grid-cols-subgrid"
                        onRemove={(key) =>
                            sampleStateDispatch({
                                type: "remove_property",
                                payload: { key: key },
                            })
                        }
                    />
                ))}
            </div>
            <div className="grid gap-2 grid-cols-[repeat(4,min-content)]">
                {newProperties.map((id) => {
                    return (
                        <NewProperty
                            key={id}
                            id={id}
                            className="px-4 col-span-full grid grid-cols-subgrid"
                            onRemove={remove_new_property}
                        />
                    );
                })}
            </div>
        </section>
    );
}

interface PropertyEditableProps {
    property: app.Property;
    onRemove: (key: string) => void;
    className?: string;
}
function PropertyEditable({
    property,
    onRemove,
    className,
}: PropertyEditableProps) {
    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        onRemove(property.Key);
    }

    const classname = `group/sample-row ${className}`;
    return (
        <div className={classname}>
            <div className="col-1 whitespace-nowrap">
                <span className="inline-block pr-1">{property.Key}</span>
                <span>({property.Type})</span>
            </div>
            <InputPropertyValue
                type={property.Type}
                defaultValue={property.Value}
                className="col-2 input-basic"
            />
            <div className="col-3 flex gap-1">
                <button
                    className="btn-cmd invisible group-hover/sample-row:visible"
                    title="Remove property"
                    onMouseDown={remove}
                >
                    <icon.Trash />
                </button>
            </div>
        </div>
    );
}

interface NewPropertyProps {
    id: number;
    className?: string;
    onRemove: (id: number) => void;
}
function NewProperty({ id, className, onRemove }: NewPropertyProps) {
    const [type, setType] = useState(app.PropertyType.PROPERTY_TYPE_STRING);

    function set_type(e: ChangeEvent<HTMLSelectElement>) {
        const new_type = type_string_to_variant(e.target.value);
        if (new_type) {
            setType(new_type);
        }
    }

    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        onRemove(id);
    }

    const classname = `group/sample-row ${className}`;
    return (
        <div className={classname}>
            <div className="col-1">
                <label>
                    <span className="hidden">Key</span>
                    <input
                        type="text"
                        id={`property[${id}][key]`}
                        name={`property[${id}][key]`}
                        placeholder="Key"
                        className="input-basic"
                    />
                </label>
            </div>
            <div className="col-2">
                <label>
                    <span className="hidden">Type</span>
                    <SelectPropertyType
                        className="input-basic"
                        value={type}
                        onChange={set_type}
                    />
                </label>
            </div>
            <div className="col-3">
                <label className="flex gap-2">
                    <span className="hidden">Value</span>
                    <InputPropertyValue type={type} className="input-basic" />
                </label>
            </div>
            <div className="col-4 invisible group-hover/sample-row:visible">
                <button type="button" className="btn-cmd" onMouseDown={remove}>
                    <icon.Trash />
                </button>
            </div>
        </div>
    );
}

interface SampleDataProps {
    data: app.SampleData[];
}
function SampleData({ data }: SampleDataProps) {
    return (
        <section>
            <div>
                <h2 className="text-lg font-bold">Data</h2>
            </div>
        </section>
    );
}

interface SampleDataEditableProps {
    data: app.SampleData[];
}
function SampleDataEditable({ data }: SampleDataEditableProps) {
    return (
        <section>
            <div className="flex gap-1 px-4">
                <h2 className="text-lg font-bold">Data</h2>
                <div>
                    <button type="button" className="btn-cmd">
                        <icon.Plus />
                    </button>
                </div>
            </div>
        </section>
    );
}
