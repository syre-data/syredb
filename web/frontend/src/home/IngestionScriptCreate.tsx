import {
    MouseButton,
    QUERY_KEY_DATA_TYPES,
    QUERY_KEY_INGESTION_SCRIPTS,
} from "@/common";
import { Loading, SuspenseError } from "@/components";
import Icon from "@/icon";
import dataService from "@/service/data.service";
import {
    DataSourceCardinalityMultiple,
    DataSourceCardinalitySingle,
    type ExternalSourceCreate,
    type IngestionScriptCreate,
} from "@/types";
import type { IngestionScriptCreateData } from "@/types/handler";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import classNames from "classnames";
import {
    Suspense,
    useState,
    type ChangeEvent,
    type MouseEvent,
    type SubmitEvent,
} from "react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import { useNavigate } from "react-router";

export default function () {
    return (
        <ErrorBoundary FallbackComponent={IngestionScriptCreateError}>
            <Suspense fallback={<Loading />}>
                <IngestionScriptCreate />
            </Suspense>
        </ErrorBoundary>
    );
}

function IngestionScriptCreateError({ resetErrorBoundary }: FallbackProps) {
    return (
        <SuspenseError
            resetErrorBoundary={resetErrorBoundary}
            className="text-center pt-4"
        >
            Could not load ingestion scripts
        </SuspenseError>
    );
}

function IngestionScriptCreate() {
    const { data: scripts } = useSuspenseQuery({
        queryKey: [QUERY_KEY_INGESTION_SCRIPTS],
        queryFn: dataService.ingestionScriptsGetAll,
    });
    const { data: data_types } = useSuspenseQuery({
        queryKey: [QUERY_KEY_DATA_TYPES],
        queryFn: dataService.dataTypesGetAll,
    });
    const navigate = useNavigate();
    const queryClient = useQueryClient();

    function close(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        navigate(-1);
    }

    function validate_label(e: ChangeEvent<HTMLInputElement>) {
        const input = e.target;
        input.setCustomValidity("");

        const label = input.value;
        if (label.length === 0) {
            input.setCustomValidity("Label is required");
            return;
        }
        if (scripts.findIndex((script) => script.Label === label) > -1) {
            input.setCustomValidity("Label already exists");
            return;
        }
    }

    async function create_ingestion_script(e: SubmitEvent<HTMLFormElement>) {
        e.preventDefault();

        const data = new FormData(e.target);
        const source_keys_arr = data
            .keys()
            .filter((key) => key.startsWith("source"))
            .map((key) => {
                const end = key.indexOf("]");
                return key.substring("source[".length, end);
            });
        const source_keys = new Set(source_keys_arr);
        const sources = [];
        for (const key of source_keys) {
            const label = data.get(`source[${key}][label]`)!.toString();
            const required = !!data.get(`source[${key}][required]`);
            const cardinality = data
                .get(`source[${key}][cardinality]`)!
                .toString();
            const description = data
                .get(`source[${key}][description]`)!
                .toString();
            const ext_filter = data
                .get(`source[${key}][ext_filter]`)!
                .toString();
            const source = {
                Label: label,
                Required: required,
                Cardinality: cardinality,
                Description: description,
                ExtFilter: ext_filter,
            };
            sources.push(source);
        }

        const ingestion_data = {
            Type: data.get("data_type")!.toString(),
            Label: data.get("label")!.toString(),
            Description: data.get("description")!.toString(),
            Cmd: "",
            Args: [],
            Sources: sources,
        } satisfies IngestionScriptCreateData;

        const script = data.get("script")! as File;
        await dataService
            .ingestionScriptCreate(ingestion_data, script)
            .then((resp) => {
                if (resp.ok) {
                    queryClient.invalidateQueries({
                        queryKey: [QUERY_KEY_INGESTION_SCRIPTS],
                    });
                    navigate(-1);
                }
            });
    }

    return (
        <div>
            <div className="flex justify-between px-4 pt-2">
                <div className="flex gap-2">
                    <h2 className="text-xl">Create ingestion script</h2>
                </div>
                <div>
                    <button
                        type="button"
                        className="btn-cmd"
                        onMouseDown={close}
                        title="Close"
                    >
                        <Icon.Close />
                    </button>
                </div>
            </div>
            <form
                className="px-4 pt-2 flex flex-col gap-2"
                onSubmit={create_ingestion_script}
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
                                className="input-basic"
                                onChange={validate_label}
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="sr-only">Data type</span>
                            <select
                                id="data_type"
                                name="data_type"
                                className="input-basic"
                                required
                            >
                                <option disabled selected hidden>
                                    Data type
                                </option>
                                {data_types.map((type) => (
                                    <option
                                        key={type.Id.toString()}
                                        value={type.Id.toString()}
                                    >
                                        {type.Label}
                                    </option>
                                ))}
                            </select>
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="pr-2">Script</span>
                            <input
                                type="file"
                                id="script"
                                name="script"
                                accept=".py"
                                className="input-basic"
                                required
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            <span className="sr-only">Description</span>
                            <textarea
                                id="description"
                                name="description"
                                className="input-basic"
                                placeholder="Description"
                            ></textarea>
                        </label>
                    </div>
                    <Sources />
                </div>
                <div>
                    <button type="submit" className="btn-submit">
                        Create
                    </button>
                </div>
            </form>
        </div>
    );
}

interface Source {
    id: number;
}
function Sources() {
    const [sources, setSources] = useState<Source[]>([{ id: 0 }]);

    function add_source(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }

        const id = Math.max(...sources.map((source) => source.id)) + 1;
        const source = {
            id,
        };
        setSources([...sources, source]);
    }

    function remove_source(id: number) {
        setSources(sources.filter((source) => source.id !== id));
    }

    return (
        <div>
            <fieldset>
                <legend className="flex gap-2">
                    Sources
                    <div>
                        <button
                            type="button"
                            className="btn-cmd"
                            onMouseDown={add_source}
                        >
                            <Icon.Plus />
                        </button>
                    </div>
                </legend>
                <ol className="px-4 pt-2 list-decimal">
                    {sources.map((source) => (
                        <Source
                            key={source.id}
                            source={source}
                            canRemove={sources.length > 1}
                            onRemove={remove_source}
                        />
                    ))}
                </ol>
            </fieldset>
        </div>
    );
}

interface SourceProps {
    source: Source;
    canRemove: boolean;
    onRemove: (id: number) => void;
}
function Source({ source, canRemove, onRemove }: SourceProps) {
    function remove(e: MouseEvent<HTMLButtonElement>) {
        if (e.button !== MouseButton.Primary) {
            return;
        }
        if (!canRemove) {
            return;
        }

        onRemove(source.id);
    }

    const source_name = `source[${source.id}]`;
    return (
        <li className="pb-2">
            <div className="flex flex-col gap-1">
                <div className="flex gap-2">
                    <label>
                        <span className="sr-only">Label</span>
                        <input
                            type="text"
                            id={`${source_name}[label]`}
                            name={`${source_name}[label]`}
                            placeholder="Label"
                            className="input-basic"
                            required
                        />
                    </label>
                    <div className={classNames({ hidden: !canRemove })}>
                        <button
                            type="button"
                            className="btn-cmd"
                            onMouseDown={remove}
                        >
                            <Icon.Minus />
                        </button>
                    </div>
                </div>
                <div>
                    <label>
                        <input
                            type="checkbox"
                            id={`${source_name}[required]`}
                            name={`${source_name}[required]`}
                            className="input-basic"
                        />
                        <span className="pl-2">Required</span>
                    </label>
                </div>
                <div>
                    <label>
                        <select
                            id={`${source_name}[cardinality]`}
                            name={`${source_name}[cardinality]`}
                            className="input-basic"
                            required
                        >
                            <option hidden selected disabled>
                                Cardinality
                            </option>
                            <option value={DataSourceCardinalitySingle}>
                                Single
                            </option>
                            <option value={DataSourceCardinalityMultiple}>
                                Multiple
                            </option>
                        </select>
                    </label>
                </div>
                <div>
                    <label>
                        <span className="sr-only">description</span>
                        <textarea
                            id={`${source_name}[description]`}
                            name={`${source_name}[description]`}
                            className="input-basic"
                            placeholder="Description"
                        ></textarea>
                    </label>
                </div>
                <div>
                    <label title="A file type specifier (e.g. `image/*`) or a comma separated list of file extensions (e.g. `.png, .jpg, .bmp`)">
                        <span className="sr-only">ext_filter</span>
                        <input
                            type="text"
                            id={`${source_name}[ext_filter]`}
                            name={`${source_name}[ext_filter]`}
                            className="input-basic"
                            placeholder="Extension filter"
                        />
                    </label>
                </div>
            </div>
        </li>
    );
}
