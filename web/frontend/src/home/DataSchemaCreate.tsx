import type { ChangeEvent, MouseEvent, SubmitEvent } from "react";
import { useRef, useState } from "react";
import { useNavigate } from "react-router";
import * as common from "@/common";
import data_service from "@/service/data.service";
import icon from "@/icon";
import * as types from "@/types";
import { useQueryClient } from "@tanstack/react-query";

interface ColumnSchema {
    id: number;
}

export default function () {
    const DEFAULT_STORAGE = types.DataStorageInternal;
    const INVALID_LABEL_ERROR =
        "Label can only contain letters, numbers, and underscores (_)";

    const queryClient = useQueryClient();
    const navigate = useNavigate();
    const [error, setError] = useState("");
    const [cols, setCols] = useState([{ id: 0 }] as ColumnSchema[]);

    function add_column(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        const id = Math.max(...cols.map((col) => col.id)) + 1;
        setCols([...cols, { id }]);
    }

    function remove_column(id: number) {
        setCols(cols.filter((col) => col.id !== id));
    }

    function cancel(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    function add_column_if_needed(
        id: number,
        e: ChangeEvent<HTMLInputElement>,
    ) {
        const idx = cols.findIndex((col) => col.id === id);
        if (idx < 0) {
            console.error(`invalid column id: ${id}`);
            return;
        }
        if (idx + 1 === cols.length && e.target.value.length > 0) {
            const id = Math.max(...cols.map((col) => col.id)) + 1;
            setCols([...cols, { id }]);
        }
    }

    function is_column_label_valid(label: string): boolean {
        const PATTERN = new RegExp("^[\\w_]+$", "gm");
        return PATTERN.test(label);
    }

    function on_label_change(id: number, e: ChangeEvent<HTMLInputElement>) {
        const LABEL_DUPLICATE_ERROR_KEY = "data-error-label-duplicate";

        const form = document.getElementById("schema-form")! as HTMLFormElement;
        const input = document.getElementById(
            `column[${id}][label]`,
        ) as HTMLInputElement;
        input.setCustomValidity("");
        const label = input.value.trim();
        for (const col of cols) {
            if (col.id === id) {
                continue;
            }

            const other = document.getElementById(
                `column[${col.id}][label]`,
            ) as HTMLInputElement;
            const other_label = other.value.trim();
            if (other_label === label) {
                input.setAttribute(
                    LABEL_DUPLICATE_ERROR_KEY,
                    col.id.toString(),
                );
                other.setAttribute(LABEL_DUPLICATE_ERROR_KEY, id.toString());

                input.setCustomValidity("label must be unique");
                form.reportValidity();
                return;
            } else {
                const other_dup_id = other.getAttribute(
                    LABEL_DUPLICATE_ERROR_KEY,
                );
                if (other_dup_id !== null) {
                    const other_dup = parseInt(other_dup_id);
                    if (other_dup === id) {
                        other.removeAttribute(LABEL_DUPLICATE_ERROR_KEY);
                        other.setCustomValidity("");
                    }
                }
            }
        }

        if (!is_column_label_valid(label)) {
            input.setCustomValidity(INVALID_LABEL_ERROR);
            form.reportValidity();
            return;
        }

        add_column_if_needed(id, e);
    }

    async function create_data_schema(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();
        setError("");

        const form = document.getElementById("schema-form")! as HTMLFormElement;
        const data = new FormData(e.target as HTMLFormElement);
        for (const key of data.keys()) {
            const input = document.getElementById(key) as any;
            if (input && input.setCustomValidity) {
                input.setCustomValidity("");
            }
        }

        const label = data.get("label")!.toString().trim();
        const description = data.get("description")!.toString().trim();
        const storage_str = data.get("storage")!.toString();
        const storage = common.data_storage_string_to_variant(storage_str);
        if (storage === null) {
            const input = document.getElementById(
                "storage",
            )! as HTMLSelectElement;
            input.setCustomValidity("invalid value");
            form.reportValidity();
            return;
        }

        if (label.length === 0) {
            const input = document.getElementById("label")! as HTMLInputElement;
            input.setCustomValidity("label must be set");
        }

        const columns = [];
        for (const { id } of cols) {
            const label_key = `column[${id}][label]`;
            const dtype_key = `column[${id}][dtype]`;
            const label = data.get(label_key)!.toString().trim();
            const dtype_str = data.get(dtype_key)!.toString();
            const dtype = common.data_type_string_to_variant(dtype_str);
            if (!dtype) {
                console.error(`invalid data type: ${dtype}`);
                const input = document.getElementById(
                    dtype_key,
                )! as HTMLSelectElement;
                input.setCustomValidity("invalid data type");
                form.reportValidity();
                return;
            }

            if (label.length === 0) {
                continue;
            }

            const label_input = document.getElementById(
                label_key,
            )! as HTMLInputElement;
            label_input.setCustomValidity("");

            if (!is_column_label_valid(label)) {
                label_input.setCustomValidity(INVALID_LABEL_ERROR);
            } else {
                columns.push({
                    label,
                    dtype: dtype,
                } satisfies types.ColumnSchema);
            }
        }
        if (columns.length === 0) {
            if (cols.length === 0) {
                setError("Schema must have at least one column");
                return;
            } else {
                const input = document.getElementById(
                    `column[${cols[0]!.id}][label]`,
                )! as HTMLInputElement;
                input.setCustomValidity("Schema must have at least one column");
                form.reportValidity();
                return;
            }
        }

        form.reportValidity();
        const data_schema = {
            Schema: columns,
            Storage: storage!,
            Label: label,
            Description: description,
        } satisfies types.DataSchemaCreate;

        await data_service
            .dataSchemaCreate(data_schema)
            .then((res) => {
                if (res.status !== 200) {
                    console.error(
                        `failed to create data schema: ${res.status}`,
                    );
                    return;
                }

                queryClient.invalidateQueries({
                    queryKey: [common.QUERY_KEY_DATA_SCHEMA],
                });
                navigate(-1);
            })
            .catch((err) => {
                console.error(err);
                setError(err);
            });
    }

    return (
        <div>
            <div>
                <h2 className="px-4 text-lg font-bold pb-4">
                    Create a data schema
                </h2>
            </div>
            <div>
                <form
                    id="schema-form"
                    onSubmit={create_data_schema}
                    className="flex flex-col gap-2 px-4"
                >
                    <div className="flex flex-col gap-2">
                        <div>
                            <label>
                                <span className="sr-only">Label</span>
                                <input
                                    type="text"
                                    id="label"
                                    name="label"
                                    placeholder="Label"
                                    className="input-basic invalid:ring-red-600"
                                    required
                                />
                            </label>
                        </div>
                        <div>
                            <label>
                                <span className="pr-2">Storage</span>
                                <select
                                    id="storage"
                                    name="storage"
                                    defaultValue={DEFAULT_STORAGE}
                                    className="input-basic invalid:ring-red-600"
                                >
                                    <option
                                        value={types.DataStorageInternal}
                                        title="Store data as a table in the database"
                                    >
                                        Internal
                                    </option>
                                    <option
                                        value={types.DataStorageExternal}
                                        title="Store data externally"
                                    >
                                        External
                                    </option>
                                </select>
                            </label>
                        </div>
                        <div>
                            <label>
                                <span className="sr-only">Description</span>
                                <textarea
                                    id="description"
                                    name="description"
                                    cols={80}
                                    placeholder="Description"
                                    className="input-basic"
                                ></textarea>
                            </label>
                        </div>
                        <div>
                            <div className="flex gap-2">
                                <h3 className="text-lg pb-2">Columns</h3>
                                <div>
                                    <button
                                        type="button"
                                        onMouseDown={add_column}
                                        className="btn-cmd"
                                    >
                                        <icon.Plus />
                                    </button>
                                </div>
                            </div>
                            <ol>
                                {cols.map((col, index) => (
                                    <li
                                        key={col.id}
                                        className="flex gap-2 pb-2"
                                    >
                                        <div>{index + 1}.</div>
                                        <ColumnSchema
                                            schema={col}
                                            onRemove={remove_column}
                                            onChangeLabel={on_label_change}
                                        />
                                    </li>
                                ))}
                            </ol>
                        </div>
                    </div>
                    <div className="flex gap-2 justify-center pt-4">
                        <div>
                            <button type="submit" className="btn-submit">
                                Save
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
            </div>
            <div className="px-4">{error}</div>
        </div>
    );
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
                    <span className="sr-only">Label</span>
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
                    <span className="sr-only">Data type</span>
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
