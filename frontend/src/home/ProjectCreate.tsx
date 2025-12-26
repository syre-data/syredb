import classNames from "classnames";
import { FormEvent, MouseEvent, useState } from "react";
import * as common from "../common";
import * as app from "../../bindings/syredb/app";
import { useNavigate } from "react-router";

export default function () {
    const [visibility, setVisibility] = useState("private");
    const [pending, setPending] = useState(false);
    const [error, setError] = useState("");
    const navigate = useNavigate();

    async function create_project(e: FormEvent<HTMLFormElement>) {
        e.preventDefault();
        setError("");

        const form = e.target as HTMLFormElement;
        const data = new FormData(form);
        const label = data.get("label")!.toString().trim();
        const description = data.get("description")!.toString().trim();
        const visibility_str = data.get("visibility")!.toString();
        const visibility =
            common.project_visibility_string_to_variant(visibility_str);
        if (!visibility) {
            console.error(`invalid project visibility: ${visibility_str}`);
            const input = document.getElementById(
                "visibility"
            )! as HTMLSelectElement;
            input.setCustomValidity("invalid project visibility");
            form.reportValidity();
            return;
        }

        if (label.length === 0) {
            const input = document.getElementById("label")! as HTMLInputElement;
            input.setCustomValidity("label can not be empty");
            form.reportValidity();
            return;
        }

        const project = new app.ProjectCreate({
            Label: label,
            Description: description.length === 0 ? undefined : description,
            Visibility: visibility,
        });
        await app.AppService.CreateProject(project)
            .then((id) => navigate(`/project/${id}`))
            .catch((err) => setError(err));
        setPending(false);
    }

    function set_project_visiblity(e: MouseEvent<HTMLButtonElement>) {
        if (e.button != common.MouseButton.Primary) {
            return;
        }

        const target = e.target as HTMLButtonElement;
        setVisibility(target.name);
    }

    return (
        <div className="flex flex-col gap-2 items-center pt-4">
            <h2>Create project</h2>
            <div className="text-red-600">{error}</div>
            <form
                id="create-project-form"
                onSubmit={create_project}
                className="flex flex-col gap-2 items-center"
            >
                <div className="flex flex-col gap-2 items-center">
                    <div>
                        <label>
                            <span className="hidden">Name</span>
                            <input
                                type="text"
                                id="label"
                                name="label"
                                placeholder="Name"
                                className="input-basic"
                                required
                            />
                        </label>
                    </div>
                    <div className="flex gap-2 items-start w-full">
                        <button
                            type="button"
                            name="private"
                            className={classNames({
                                "border-2 px-2 py-0.5 rounded-full cursor-pointer":
                                    true,
                                "border-blue-400 bg-blue-100 dark:border-blue-800 dark:bg-blue-950":
                                    visibility === "private",
                            })}
                            onMouseDown={set_project_visiblity}
                        >
                            Private
                        </button>

                        <button
                            type="button"
                            name="public"
                            className={classNames({
                                "border-2 px-2 py-0.5 rounded-full cursor-pointer":
                                    true,
                                "border-blue-400 bg-blue-100 dark:border-blue-800 dark:bg-blue-950":
                                    visibility === "public",
                            })}
                            onMouseDown={set_project_visiblity}
                        >
                            Public
                        </button>

                        <fieldset className="hidden">
                            <legend>Visibility</legend>
                            <label>
                                <input
                                    type="radio"
                                    id="project_visibility_private"
                                    name="visibility"
                                    value="private"
                                    checked={visibility === "private"}
                                    readOnly
                                />
                                Private
                            </label>
                            <label>
                                <input
                                    type="radio"
                                    id="project_visibily_public"
                                    name="visibility"
                                    value="public"
                                    checked={visibility === "public"}
                                    readOnly
                                />
                                Public
                            </label>
                        </fieldset>
                    </div>
                    <div className="w-full">
                        <label>
                            <span className="hidden">Description</span>
                            <textarea
                                name="description"
                                placeholder="Description"
                                className="min-w-full border px-1 py-0.5"
                            ></textarea>
                        </label>
                    </div>
                </div>
                <div className="flex gap-2">
                    <button
                        type="submit"
                        id="submit"
                        className="btn-submit"
                        disabled={pending}
                    >
                        Create
                    </button>
                    <button
                        type="button"
                        id="cancel"
                        className="btn-submit"
                        disabled={pending}
                    >
                        Cancel
                    </button>
                </div>
            </form>
        </div>
    );
}
