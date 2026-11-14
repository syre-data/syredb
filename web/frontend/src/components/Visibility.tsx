import Icon from "@/icon";
import { VisibilityPrivate, VisibilityPublic, type Visibility } from "@/types";
import type { ComponentProps } from "react";

export function VisibilityBadgePrivate({
    className,
    ...props
}: ComponentProps<"div">) {
    return (
        <div
            className={`flex gap-1 items-center px-2 py-0.5
                ${className ?? ""}`}
            {...props}
        >
            <Icon.Lock />
            Private
        </div>
    );
}

export function VisibilityBadgePublic({
    className,
    ...props
}: ComponentProps<"div">) {
    return (
        <div
            className={`flex gap-1 items-center px-2 py-0.5
                ${className ?? ""}`}
            {...props}
        >
            <Icon.Earth />
            Public
        </div>
    );
}

interface VisibilityBadgeProps extends ComponentProps<"div"> {
    visibility: Visibility;
}
export function VisibilityBadge({
    visibility,
    className,
    ...props
}: VisibilityBadgeProps) {
    switch (visibility) {
        case VisibilityPrivate:
            return (
                <VisibilityBadgePrivate
                    className={`rounded-full border bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-200 ${className}`}
                    {...props}
                />
            );
        case VisibilityPublic:
            return (
                <VisibilityBadgePublic
                    className={`rounded-full border bg-syre-blue-200 text-syre-blue-700 dark:bg-syre-blue-700 dark:text-white ${className}`}
                    {...props}
                />
            );
    }
}

interface VisibilityFormToggleProps {
    defaultValue: Visibility;
    className?: string;
    onChange?: (visibility: Visibility) => void;
}
export function VisibilityFormToggle({
    defaultValue,
    className,
    onChange,
}: VisibilityFormToggleProps) {
    const iconClass =
        "text-secondary \
        bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-200 \
        peer-checked:bg-primary-700 dark:peer-checked:bg-primary-600 \
        peer-checked:text-white dark:peer-checked:text-black \
        transition-colors duration-200 ease-in-out\
        rounded block \
        h-full p-1 cursor-pointer";

    let parentClass = "flex";
    if (className) {
        parentClass += " " + className;
    }

    if (onChange === undefined) {
        onChange = (_) => {};
    }

    return (
        <fieldset className={parentClass}>
            <label
                className="rounded-l-full border-l border-t border-b cursor-pointer"
                title="Public"
            >
                <input
                    type="radio"
                    name="visibility"
                    value={VisibilityPublic}
                    defaultChecked={defaultValue === VisibilityPublic}
                    className="hidden peer"
                    onChange={(_) => onChange(VisibilityPublic)}
                />
                <span className="sr-only">Public</span>
                <VisibilityBadgePublic
                    className={`rounded-l-full ${iconClass}`}
                />
            </label>
            <label
                className="rounded-r-full border-r border-t border-b cursor-pointer"
                title="Private"
            >
                <input
                    type="radio"
                    name="visibility"
                    value={VisibilityPrivate}
                    defaultChecked={defaultValue === VisibilityPrivate}
                    className="hidden peer"
                    onChange={(_) => onChange(VisibilityPrivate)}
                />
                <span className="sr-only">Private</span>
                <VisibilityBadgePrivate
                    className={`rounded-r-full ${iconClass}`}
                />
            </label>
        </fieldset>
    );
}
