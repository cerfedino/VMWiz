import { toast } from "sonner";
import { FetchError } from "@/lib/api";
import { formatDate } from "@/lib/utils";
import { getReasonPhrase } from "http-status-codes";
import { RequestDebugPanel } from "@/components/request-debug-panel";

function ToastTitle({ children }: { children: React.ReactNode }) {
    const timestamp = formatDate(new Date().toISOString());
    return (
        <div className="flex w-full items-baseline justify-between gap-3">
            <span>{children}</span>
            <span className="shrink-0 text-xs font-normal opacity-60">
                {timestamp}
            </span>
        </div>
    );
}

function FetchErrorToastDescription({ err }: { err: FetchError }) {
    return (
        <div>
            {err.message && <p>{err.message}</p>}
            <RequestDebugPanel
                requestInfo={err.request}
                responseInfo={{
                    status: err.response.status,
                    body: err.message,
                }}
            />
        </div>
    );
}

function toastFetchError(err: FetchError) {
    const status = err.response?.status;
    let title = "Request failed";
    if (status && status > 0) {
        title = `${status} ${getReasonPhrase(status)}`;
    }
    toast.error(<ToastTitle>{title}</ToastTitle>, {
        description: <FetchErrorToastDescription err={err} />,
        duration: Infinity,
        style: { height: "auto" },
    });
}

function toastGenericError(err: Error) {
    const title = `Unexpected Error${err.name !== "Error" ? ": " + err.name : ""}`;
    toast.error(<ToastTitle>{title}</ToastTitle>, {
        description: err.message || undefined,
        duration: Infinity,
    });
}

/** Show an error toast */
export function toastError(err: Error) {
    if (err instanceof FetchError) {
        toastFetchError(err);
    } else {
        toastGenericError(err);
    }
}
