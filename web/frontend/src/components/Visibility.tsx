import Icon from "@/icon";
import { VisibilityPrivate, VisibilityPublic, type Visibility } from "@/types";

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
