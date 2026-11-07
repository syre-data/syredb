import { type AppError as AppErrorInt } from "@/types/errors";
export class AppError extends Error {
    requestId: string;
    payload: any;

    constructor(data: AppErrorInt) {
        super(data.message, { cause: data.code });

        // Maintains proper stack trace for where our error was thrown (non-standard)
        if (Error.captureStackTrace) {
            Error.captureStackTrace(this, AppError);
        }

        this.name = "AppError";
        this.requestId = data.requestId;
        this.payload = data.payload;
    }

    override toString() {
        const info = { code: this.cause, payload: this.payload };
        return `[request: ${this.requestId}] ${this.message} ${JSON.stringify(info)}`;
    }

    toObject() {
        return {
            requestId: this.requestId,
            cause: this.cause,
            message: this.message,
            payload: this.payload,
        };
    }
}
