import authService from "@/service/auth.service";
import { StatusCodes } from "http-status-codes";
import { useEffect } from "react";

export default function () {
    useEffect(() => {
        authService.logout().then((res) => {
            if (res.status === StatusCodes.OK) {
                window.location.replace("/");
            } else {
                console.error(res);
            }
        });
    }, []);

    return <div className="pt-4 text-center">Logging out</div>;
}
