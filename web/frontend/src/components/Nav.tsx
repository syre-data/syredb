import Icon from "@/icon";
import classNames from "classnames";
import type React from "react";
import type { ComponentProps } from "react";
import { Link } from "react-router";
import { SyreBadge, ThemeLight } from "./SyreBadge";

export const PageDashboard = "dashboard";
export const PageUserProfile = "user_profile";
export const PageUsers = "users";
export const PageDataSchemas = "data_schemas";
export const PageDataTypes = "data_types";
export const PageDataOrigins = "data_origins";
export const PageDataTransforms = "data_type_transforms";
export const PageLogout = "logout";
export type PageId =
    | typeof PageDashboard
    | typeof PageUserProfile
    | typeof PageUsers
    | typeof PageDataSchemas
    | typeof PageDataTypes
    | typeof PageDataOrigins
    | typeof PageDataTransforms
    | typeof PageLogout;

interface MainNavProps extends ComponentProps<"div"> {
    current?: PageId;
}

export function MainNav({ current, className, ...props }: MainNavProps) {
    return (
        <div
            className={`px-2 flex flex-col gap-2 bg-gray-800 text-white ${className}`}
            {...props}
        >
            <ol className="grow flex flex-col gap-4 divide-y divide-gray-600">
                <li className="pb-4">
                    <MainNavItem
                        title="Dashboard"
                        icon={<Icon.Home />}
                        url="/"
                        highlighted={current === PageDashboard}
                    />
                    <ol>
                        <MainNavItem
                            title="Profile"
                            icon={<Icon.User />}
                            url="/profile"
                            highlighted={current === PageUserProfile}
                        />
                        <MainNavItem
                            url="/users"
                            title="Users"
                            icon={<Icon.Users />}
                            highlighted={current === PageUsers}
                        />
                        <MainNavItem
                            url="/data-schemas"
                            title="Data schemas"
                            icon={<Icon.Columns />}
                            highlighted={current === PageDataSchemas}
                        />
                        <MainNavItem
                            url="/data-types"
                            title="Data types"
                            icon={<Icon.DataType />}
                            highlighted={current === PageDataTypes}
                        />
                        <MainNavItem
                            url="/data-origins"
                            title="Data origins"
                            icon={<Icon.LocationPin />}
                            highlighted={current === PageDataOrigins}
                        />
                        <MainNavItem
                            url="/data-type-transforms"
                            title="Data transforms"
                            icon={<Icon.Function />}
                            highlighted={current === PageDataTransforms}
                            popoverTitle="Data type transforms"
                        />
                    </ol>
                </li>
                <MainNavItem
                    url="/logout"
                    title="Log out"
                    icon={<Icon.Logout />}
                    highlighted={current === PageLogout}
                />
            </ol>
            <SyreBadge theme={ThemeLight} asLink={true} />
        </div>
    );
}

interface MainNavItemProps {
    title: string;
    icon: React.ReactNode;
    url: string;
    highlighted: boolean;
    popoverTitle?: string;
}
function MainNavItem({
    title,
    icon,
    url,
    highlighted,
    popoverTitle,
}: MainNavItemProps) {
    popoverTitle = popoverTitle ?? title;
    return (
        <li
            className={classNames({
                "px-4 py-2 rounded-lg": true,
                "bg-primary-800 font-bold": highlighted,
                "hover:bg-gray-700": !highlighted,
            })}
        >
            <Link
                to={url}
                title={popoverTitle}
                className="flex gap-2 items-center"
            >
                {icon} {title}
            </Link>
        </li>
    );
}
