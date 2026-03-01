import { ErrorBoundary } from "react-error-boundary";
import { useSuspenseQuery, useQueryClient } from "@tanstack/react-query";
import {
    Suspense,
    type MouseEvent,
    type SubmitEvent,
    type ChangeEvent,
    useState,
    useRef,
} from "react";
import { useNavigate, useParams, Link } from "react-router";
import Loading from "../components/Loading";
import { SuspenseError } from "@/components";
import type { FallbackProps } from "react-error-boundary";
import * as types from "@/types";
import * as uuid from "uuid";
import data_service from "@/service/data.service";
import icon from "@/icon";
import * as common from "@/common";

export default function () {
    const navigate = useNavigate();
    const { data_schema_id } = useParams();
    if (data_schema_id) {
        return (
            <ErrorBoundary FallbackComponent={DataSchemaError}>
                <Suspense fallback={<Loading />}>
                    <DataSchema data_schema_id={data_schema_id} />
                </Suspense>
            </ErrorBoundary>
        );
    } else {
        navigate("/");
        return null;
    }
}

function DataSchemaError({ error, resetErrorBoundary }: FallbackProps) {
    const err = error as types.AppError;

    return (
        <SuspenseError resetErrorBoundary={resetErrorBoundary}>
            <div>Could not load project</div>
            <div>{err.Message}</div>
        </SuspenseError>
    );
}

interface DataSchemaProps {
    data_schema_id: uuid.UUIDTypes;
}
function DataSchema({ data_schema_id }: DataSchemaProps) {
    const queryClient = useQueryClient();

    const { data: data_schema_resources } = useSuspenseQuery({
        queryKey: ["data_schema_resources", data_schema_id],
        queryFn: async () =>
            data_service.getDataSchemaResources(data_schema_id),
    });
    const { data: data_schemas } = useSuspenseQuery({
        queryKey: ["data_schemas"],
        queryFn: async () => data_service.getDataSchemasAll(),
    });
    const data_schema = data_schema_resources.DataSchema;

    const [createTransformEditor, setCreateTransformEditor] = useState(false);

    function addTransform(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        setCreateTransformEditor(true);
    }

    async function createTransform(form_data: FormData) {
        const label = form_data.get("label") as string;
        const description = form_data.get("description") as string;
        const schema_id = form_data.get("schema") as string;
        const script_file = form_data.get("script") as File;

        if (!label || !schema_id || !script_file) {
            console.error("missing required fields for creating transform");
            return;
        }

        await data_service
            .transformCreate({
                Input: data_schema.Id,
                Output: schema_id,
                Script: script_file,
                Label: label,
                Description: description,
            } satisfies types.TransformCreate)
            .then((res) => {
                if (res.status !== 200) {
                }
                queryClient.invalidateQueries({
                    queryKey: ["data_schema_resources", data_schema_id],
                });
                setCreateTransformEditor(false);
            })
            .catch((err) => {
                console.error("could not create transform", err);
            });
    }

    return (
        <div>
            <div className="pt-2 px-4">
                <div className="flex gap-2 items-stretch">
                    <h1 className="text-xl">Data schema {data_schema.Label}</h1>
                    <div className="flex gap-2">
                        <Link to="/">
                            <button type="button" className="btn-cmd">
                                <icon.Home />
                            </button>
                        </Link>
                    </div>
                </div>
                <div>{data_schema.Description}</div>
            </div>
            <div className="px-4">
                <h2 className="text-lg">Schema</h2>
                <div className="flex gap-2">
                    {data_schema.Schema.map((col, idx) => (
                        <>
                            {idx !== 0 ? <div>|</div> : null}
                            <div key={col.label} className="flex gap-1">
                                <div>{col.label}</div>
                                <div>({data_type_to_string(col.dtype)})</div>
                            </div>
                        </>
                    ))}
                </div>
            </div>
            <div>
                <div className="px-4 flex gap-2">
                    <h2 className="text-lg">Transforms</h2>
                    <div>
                        <button
                            type="button"
                            className="btn-cmd align-middle"
                            onMouseDown={addTransform}
                            disabled={createTransformEditor}
                        >
                            <icon.Plus />
                        </button>
                    </div>
                </div>
                {createTransformEditor && (
                    <div className="px-4">
                        <TransformCreate
                            dataSchemas={data_schemas}
                            onSubmit={createTransform}
                            onCancel={() => setCreateTransformEditor(false)}
                        />
                    </div>
                )}
                <div className="px-4">
                    <ul>
                        {data_schema_resources.Transforms.map((transform) => (
                            <li
                                key={transform.Id.toString()}
                                className="flex gap-2"
                            >
                                <div>{transform.Label}</div>
                                <div>
                                    (
                                    {
                                        data_schemas.find(
                                            (s) => s.Id === transform.Output,
                                        )!.Label
                                    }
                                    )
                                </div>
                            </li>
                        ))}
                    </ul>
                </div>
            </div>
        </div>
    );
}

function data_type_to_string(data_type: types.DataType): string {
    switch (data_type) {
        case types.DataTypeString:
            return "string";
        case types.DataTypeInt:
            return "int";
        case types.DataTypeUint:
            return "uint";
        case types.DataTypeFloat:
            return "float";
        case types.DataTypeBoolean:
            return "boolean";
        case types.DataTypeTimestamp:
            return "timestamp";
    }
}

interface TransformCreateProps {
    dataSchemas: types.DataSchema[];
    onSubmit: (e: FormData) => void;
    onCancel: () => void;
}
function TransformCreate({
    dataSchemas: data_schemas,
    onSubmit,
    onCancel,
}: TransformCreateProps) {
    function cancel(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== common.MouseButton.Primary) {
            return;
        }

        onCancel();
    }

    function submit(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        const form_data = new FormData(e.currentTarget);
        onSubmit(form_data);
    }

    return (
        <form className="flex flex-col gap-2" onSubmit={submit}>
            <div className="flex flex-col gap-2">
                <div>
                    <label>
                        <span className="hidden">Label</span>
                        <input
                            type="text"
                            name="label"
                            placeholder="Label"
                            className="input-basic"
                            required
                        />
                    </label>
                </div>
                <div>
                    <label>
                        <span className="hidden">Output schema</span>
                        <select
                            name="schema"
                            className="input-basic"
                            required
                            defaultValue=""
                        >
                            <option value="" disabled>
                                Select output schema
                            </option>
                            {data_schemas.map((schema) => (
                                <option
                                    key={schema.Id.toString()}
                                    value={schema.Id.toString()}
                                >
                                    {schema.Label}
                                </option>
                            ))}
                        </select>
                    </label>
                </div>
                <div>
                    <label className="flex gap-2">
                        <span>Script</span>
                        <input
                            type="file"
                            name="script"
                            accept=".py"
                            className="input-basic"
                            required
                        />
                    </label>
                </div>
                <div>
                    <label>
                        <span className="hidden">Description</span>
                        <textarea
                            name="description"
                            placeholder="Description"
                            className="input-basic"
                        ></textarea>
                    </label>
                </div>
            </div>
            <div className="flex gap-2">
                <div>
                    <button type="submit" className="btn-submit">
                        Create
                    </button>
                </div>
                <div>
                    <button
                        type="button"
                        className="btn-submit"
                        onMouseDown={cancel}
                    >
                        Cancel
                    </button>
                </div>
            </div>
        </form>
    );
}

interface ColumnSchema {
    id: number;
}

interface ColumnSchemaProps {
    schema: ColumnSchema;
    onRemove: (id: number) => void;
    onChangeLabel: (id: number, event: ChangeEvent<HTMLInputElement>) => void;
}
function ColumnSchema({ schema, onRemove, onChangeLabel }: ColumnSchemaProps) {
    const labelNode = useRef<HTMLInputElement>(null);

    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        onRemove(schema.id);
    }

    function on_change_label(e: ChangeEvent<HTMLInputElement>) {
        const input = labelNode.current;
        if (
            input &&
            !input.validity.customError &&
            input.value.trim().length > 0
        ) {
            input.setCustomValidity("");
        }

        onChangeLabel(schema.id, e);
    }

    return (
        <div className="flex gap-2">
            <div>
                <label>
                    <span className="hidden">Label</span>
                    <input
                        ref={labelNode}
                        type="text"
                        id={`column[${schema.id}][label]`}
                        name={`column[${schema.id}][label]`}
                        placeholder="Label"
                        title="Column label"
                        className="input-basic invalid:ring-red-600"
                        onChange={on_change_label}
                    />
                </label>
            </div>
            <div>
                <label>
                    <span className="hidden">Data type</span>
                    <select
                        id={`column[${schema.id}][dtype]`}
                        name={`column[${schema.id}][dtype]`}
                        title="Column data type"
                        defaultValue="string"
                        className="input-basic invalid:ring-red-600"
                    >
                        <option value="string">String</option>
                        <option value="int">Int</option>
                        <option value="uint">Uint</option>
                        <option value="float">Float</option>
                        <option value="boolean">Boolean</option>
                        <option value="timestamp">Timestamp</option>
                    </select>
                </label>
            </div>
            <div>
                <button type="button" onMouseDown={remove} className="btn-cmd">
                    <icon.Trash />
                </button>
            </div>
        </div>
    );
}
