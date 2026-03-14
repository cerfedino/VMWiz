"use client";

import { useEffect } from "react";
import { toastError } from "@/lib/toast-error";
import { Toaster } from "./ui/sonner";

/** Listens to uncaught errors and unhandled promise rejections and shows them as toasts. */
export function ErrorToasterWrapper() {
    useEffect(() => {
        function onUnhandledRejection(e: PromiseRejectionEvent) {
            if (e.reason instanceof Error) toastError(e.reason);
        }

        function onError(e: ErrorEvent) {
            if (e.error instanceof Error) toastError(e.error);
        }

        window.addEventListener("unhandledrejection", onUnhandledRejection);
        window.addEventListener("error", onError);
        return () => {
            window.removeEventListener(
                "unhandledrejection",
                onUnhandledRejection,
            );
            window.removeEventListener("error", onError);
        };
    }, []);

    return (
        <Toaster
            position="top-right"
            richColors
            closeButton
            visibleToasts={Infinity}
        />
    );
}
