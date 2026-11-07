import { http, HttpResponse } from "msw";

const MOCK_ERROR_CODE = "MOCK_ERROR";

export const handlers = [
    // http.get("/api/data/orphaned", () => {
    //     return HttpResponse.json(
    //         {
    //             requestId: "0",
    //             code: MOCK_ERROR_CODE,
    //             message: "Mock MSW",
    //             payload: { origin: "msw mock" },
    //         },
    //         { status: 500 },
    //     );
    // }),
];
