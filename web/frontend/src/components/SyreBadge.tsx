import type { ComponentProps } from "react";

export const ThemeLight = "light";
export const ThemeDark = "dark";
export type Theme = typeof ThemeLight | typeof ThemeDark;

interface SyreBadgeProps extends ComponentProps<"div"> {
    theme?: Theme;
    asLink?: boolean;
}
export function SyreBadge({
    theme,
    asLink,
    className,
    ...props
}: SyreBadgeProps) {
    let srcLogo = "/logo-black-icon.svg";
    if (theme === ThemeLight) {
        srcLogo = "/logo-white-icon.svg";
    }
    const badge = (
        <div
            className={`flex items-center gap-2 ${className ?? ""}`}
            {...props}
        >
            <div>
                <img src={srcLogo} className="h-8" />
            </div>
            <div>
                <div className="text-sm">powered by</div>
                <div className="font-primary">SyreDB</div>
            </div>
        </div>
    );

    return asLink ? (
        <a href="https://syre.ai" target="_blank">
            {badge}
        </a>
    ) : (
        badge
    );
}
