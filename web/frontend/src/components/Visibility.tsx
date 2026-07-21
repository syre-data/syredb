import Icon from "@/icon";
import { VisibilityPrivate, VisibilityPublic, type Visibility } from "@/types";
import classNames from "classnames";
import { useState } from "react";

interface VisibilityIconProps {
    visibility: Visibility;
}
export function VisibilityIcon({ visibility }: VisibilityIconProps) {
    switch (visibility) {
        case VisibilityPrivate:
            return <Icon.EyeSlash title="Private" />;
        case VisibilityPublic:
            return <Icon.Eye title="Public" />;
    }
}

interface VisibilityFormToggleProps {
    defaultValue: Visibility;
    className?: string;
}
export function VisibilityFormToggle({
    defaultValue,
    className,
}: VisibilityFormToggleProps) {
    const iconClass =
        "text-syre-grey-600 dark:text-syre-grey-300 \
        peer-checked:bg-primary-700 dark:peer-checked:bg-primary-500 \
        peer-checked:text-white dark:peer-checked:text-black \
        rounded-full block \
        h-full aspect-square p-1 cursor-pointer";

    let parentClass = "flex";
    if (className) {
        parentClass += " " + className;
    }

    return (
        <fieldset className={parentClass}>
            <label
                className="rounded-l-full border-l border-t border-b pr-0.5 cursor-pointer"
                title="Public"
            >
                <input
                    type="radio"
                    name="visibility"
                    value={VisibilityPublic}
                    defaultChecked={defaultValue === VisibilityPublic}
                    className="hidden peer"
                />
                <span className="sr-only">Public</span>
                <span className={iconClass}>
                    <Icon.Eye />
                </span>
            </label>
            <label
                className="rounded-r-full border-r border-t border-b pl-0.5 cursor-pointer"
                title="Private"
            >
                <input
                    type="radio"
                    name="visibility"
                    value={VisibilityPrivate}
                    defaultChecked={defaultValue === VisibilityPrivate}
                    className="hidden peer"
                />
                <span className="sr-only">Private</span>
                <span className={iconClass}>
                    <Icon.EyeSlash />
                </span>
            </label>
        </fieldset>
    );
}
