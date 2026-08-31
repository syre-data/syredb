import authService from "@/service/auth.service";
import { StatusCodes } from "http-status-codes";
import { useEffect } from "react";

export default function () {
    useEffect(() => {
        authService.logout().then((res) => {
            console.debug(res);
            if (res.status === StatusCodes.OK) {
                console.debug("LOGOUT");
                window.location.replace("/");
            } else {
                console.debug("ERROR");
            }
        });
    }, []);

    return <div className="pt-4 text-center">Logging out</div>;
}
