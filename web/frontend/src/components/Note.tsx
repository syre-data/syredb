import { timestampToString, uuidToString } from "@/common";
import type { Note, User } from "@/types";
import { VisibilityBadge } from "./Visibility";
import Icon from "@/icon";
import type { ComponentProps } from "react";

interface NoteProps extends ComponentProps<"div"> {
    note: Note;
    creator?: User;
}
export function Note({ note, creator, ...props }: NoteProps) {
    let creatorTitle = <span className="text-secondary">(unknown)</span>;
    if (creator) {
        if (creator.Name) {
            creatorTitle = <span>{creator.Name}</span>;
        } else if (creator.Email) {
            creatorTitle = <span>{creator.Email}</span>;
        } else {
            creatorTitle = (
                <span className="text-secondary">
                    {uuidToString(creator.Id)}
                </span>
            );
        }
    }

    return (
        <div {...props}>
            <div className="flex gap-2 items-center px-4 py-2 bg-gray-200 dark:bg-gray-700">
                <div>{timestampToString(note.Timestamp)}</div>
                <div className="flex gap-2 items-baseline">
                    <Icon.User />
                    {creatorTitle}
                </div>
                <div>
                    <VisibilityBadge
                        className="text-sm"
                        visibility={note.Visibility}
                    />
                </div>
            </div>
            <div className="px-4 py-2">{note.Content}</div>
        </div>
    );
}
