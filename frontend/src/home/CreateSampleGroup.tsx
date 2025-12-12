import {
    useState,
    FormEvent,
    MouseEventHandler,
    MouseEvent,
    SetStateAction,
    Dispatch,
} from "react";
import icons from "../icons";
import { MouseButton } from "../common";
import * as models from "../../wailsjs/go/models";
import * as app from "../../wailsjs/go/main/App";
import { useNavigate } from "react-router";

export default function () {
    let navigate = useNavigate();
    const [error, setError] = useState("");
    const [samples, setSamples] = useState<number[]>([]);

    function createSampleGroup(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
        const target = e.currentTarget;
        if (target === null) {
            console.error("could not get form");
            setError("form error");
            return;
        }
        const data = new FormData(target);

        let hasErrors = false;
        const groupLabelData = data.get("group-label")!;
        const groupDescriptionData = data.get("group-description")!;
        const groupParentData = data.get("group-parent");

        const groupLabel = groupLabelData.toString();
        if (groupLabel.length === 0) {
            const input = document.getElementById(
                "group-label"
            )! as HTMLInputElement;
            input.setCustomValidity("group label must be set");
            hasErrors = true;
        }

        let groupParents: string[] = [];
        if (groupParentData !== null) {
            // TODO
        }

        const groupSamples: models.main.SampleCreate[] = [];
        for (const idx of samples) {
            let label;

            const labelId = `sample[${idx}][label]`;
            const labelData = data.get(labelId);
            if (labelData === null) {
                const input = document.getElementById(
                    labelId
                ) as HTMLInputElement;
                input.setCustomValidity("label must be set");
                hasErrors = true;
            } else {
                const labelStr = labelData.toString();
                if (labelStr.length === 0) {
                    const input = document.getElementById(
                        labelId
                    ) as HTMLInputElement;
                    input.setCustomValidity("label must be set");
                    hasErrors = true;
                } else {
                    label = labelStr;
                }
            }

            console.assert(label !== undefined);
            groupSamples.push(new models.main.SampleCreate({ Label: label }));
        }

        if (hasErrors) {
            return;
        }

        const groupDescription = groupDescriptionData.toString();
        const group = new models.main.SampleGroupCreate({
            Label: groupLabel,
            Description: groupDescription,
            Parents: groupParents,
            Samples: groupSamples,
        });

        console.debug(group);
        app.CreateSampleGroup(group)
            .then(() => navigate("/"))
            .catch((err) => setError(err));
    }

    function addSample(e: MouseEvent) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        const id = samples.length ? samples[samples.length - 1] + 1 : 0;
        setSamples([...samples, id]);
    }

    function removeSample(id: number) {
        const filtered = samples.filter((sample_id) => sample_id != id);
        setSamples([...filtered]);
    }

    return (
        <div>
            <h2 className="text-lg font-bold">Create sample group</h2>
            <div>
                <form onSubmit={createSampleGroup}>
                    <div>
                        <div>
                            <label>
                                <span className="block">Group name</span>
                                <input
                                    type="text"
                                    id="group-label"
                                    name="group-label"
                                    minLength={1}
                                    required
                                    className="input-basic user-invalid:ring-red-600"
                                />
                            </label>
                        </div>
                        <div>
                            <label>
                                <span className="block">Description</span>
                                <textarea
                                    id="group-description"
                                    name="group-description"
                                    className="input-basic px-2 py-1"
                                ></textarea>
                            </label>
                        </div>
                        <div>
                            <label>
                                <span className="block">Parent</span>
                                <select
                                    id="group-parent"
                                    name="group-parent"
                                    multiple
                                    className="input-basic"
                                ></select>
                            </label>
                        </div>
                    </div>
                    <div>
                        <div className="flex gap-2">
                            <h3 className="text-lg grow">Samples</h3>
                        </div>
                        <ol className="list-inside list-decimal">
                            {samples.length ? (
                                samples.map((id) => (
                                    <li key={id}>
                                        <CreateSample
                                            id={id}
                                            onRemove={removeSample}
                                        />
                                    </li>
                                ))
                            ) : (
                                <NoSamples />
                            )}
                        </ol>
                        <div className="text-center">
                            <button
                                type="button"
                                onMouseDown={addSample}
                                className="btn-cmd"
                                title="Add a sample"
                            >
                                <icons.Plus />
                            </button>
                        </div>
                    </div>
                    <div className="text-center">
                        <button type="submit" className="cursor-pointer">
                            Create
                        </button>
                    </div>
                </form>
                <div>{error}</div>
            </div>
        </div>
    );
}

function NoSamples() {
    return <div>No samples added yet.</div>;
}

interface CreateSampleProps {
    id: number;
    onRemove: (id: number) => void;
}
function CreateSample({ id, onRemove }: CreateSampleProps) {
    function remove(e: MouseEvent) {
        if (e.button != MouseButton.Primary) {
            return;
        }

        onRemove(id);
    }

    return (
        <div className="inline-flex gap-2 pb-2">
            <div className="grow">
                <label>
                    <span className="hidden">label</span>
                    <input
                        id={`sample[${id}][label]`}
                        name={`sample[${id}][label]`}
                        type="text"
                        minLength={1}
                        required
                        placeholder="label"
                        className="input-basic user-invalid:ring-red-600"
                    />
                </label>
            </div>
            <div>
                <button type="button" onMouseDown={remove} className="btn-cmd">
                    <icons.Trash />
                </button>
            </div>
        </div>
    );
}
