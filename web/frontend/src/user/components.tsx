import type { DbPermissionRecord } from "@/types";
import type { ComponentProps } from "react";

interface DbPermissionBadgesProps extends ComponentProps<"ul"> {
    permissions: DbPermissionRecord[];
}

export function DbPermissionBadges({
    permissions,
    ...props
}: DbPermissionBadgesProps) {
    return (
        <ul {...props}>
            {permissions.map((perm) => {
                return (
                    <li
                        key={perm.Id}
                        className="rounded-full px-4 py-1 bg-primary-700 text-white align-middle"
                        title={perm.Description}
                    >
                        {perm.Label}
                    </li>
                );
            })}
        </ul>
    );
}
