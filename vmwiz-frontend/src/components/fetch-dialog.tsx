"use client";

import React, { useCallback, useEffect, useRef, useState } from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

import {
    FetchError,
    type OnConfirmCallback,
    type BackendRequest,
} from "@/lib/api";
import {
    RequestDebugPanel,
    type ResponseInfo,
} from "@/components/request-debug-panel";
import { AlertTriangle, CheckCircle2, Loader2 } from "lucide-react";

class CancelledError extends Error {
    constructor() {
        super("Cancelled");
        this.name = "CancelledError";
    }
}

type Phase = "idle" | "loading" | "confirming" | "success" | "error";

function PhaseIcon({ phase }: { phase: Phase }) {
    const base =
        "mx-auto mb-1 flex h-12 w-12 items-center justify-center rounded-full";

    switch (phase) {
        case "idle":
        case "confirming":
            return (
                <div className={cn(base, "bg-amber-100")}>
                    <AlertTriangle className="h-6 w-6 text-amber-600" />
                </div>
            );
        case "loading":
            return (
                <div className={cn(base, "bg-muted")}>
                    <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
            );
        case "success":
            return (
                <div className={cn(base, "bg-teal-100")}>
                    <CheckCircle2 className="h-6 w-6 text-teal-600" />
                </div>
            );
        case "error":
            return (
                <div className={cn(base, "bg-red-100")}>
                    <AlertTriangle className="h-6 w-6 text-red-600" />
                </div>
            );
    }
}

interface FetchDialogProps {
    /** Whether the dialog is open or not. */
    open: boolean;
    onOpenChange: (open: boolean) => void;

    /**
     * The fetch function that performs the actual request.
     * The dialog injects its `onConfirmRequired` callback so it can show the confirmation prompt if needed.
     */
    fetchFn: (
        onConfirmRequired: OnConfirmCallback,
    ) => Promise<{ data: unknown }>;

    /** Optional request metadata shown in the debug info panel. Built by prepare* helpers from api.ts. */
    requestInfo?: BackendRequest;

    title: string;
    description?: string;
    /** Body to show if the request si successfull */
    successDescription?: React.ReactNode;
    /** Custom content rendered in the dialog on success */
    successContent?: (data: unknown) => React.ReactNode;
    /**  Whether to fire the request immediately as soon as the dialog is shown. If false, the user has to manually click a button to proceed */
    immediate?: boolean;
    /**  Cancel button label (only shown if `immediate` is false) */
    cancelLabel?: string;
    /**  Cancel button appearance (only shown if `immediate` is false) */
    cancelVariant?:
        | "outline"
        | "destructive"
        | "default"
        | "secondary"
        | "ghost"
        | "link";
    /** Proceed button label */
    proceedLabel?: string;
    /** Proceed button appearance */
    proceedVariant?:
        | "outline"
        | "destructive"
        | "default"
        | "secondary"
        | "ghost"
        | "link";

    /** Whether to show the phase icon. Defaults to true. */
    showIcon?: boolean;
    /** Callback that gets called with the response data if everything goes well */
    onSuccess?: (data: unknown) => void;
    /** If onError returns true, the dialog closes (error handled externally).
     Otherwise the dialog shows the error phase. */
    onError?: (error: Error) => boolean | void;
}

export function FetchDialog({
    open,
    onOpenChange,
    fetchFn,
    requestInfo,
    title,
    description,
    successDescription = "Completed successfully",
    successContent,
    cancelLabel = "Cancel",
    cancelVariant = "outline",
    proceedLabel = "Proceed",
    proceedVariant = "default",
    immediate = false,
    showIcon = true,
    onSuccess,
    onError,
}: FetchDialogProps) {
    const [phase, setPhase] = useState<Phase>("idle");
    const [errorMessage, setErrorMessage] = useState("");
    const [confirmInput, setConfirmInput] = useState("");
    const [expectedToken, setExpectedToken] = useState("");
    const [successData, setSuccessData] = useState<unknown>(undefined);
    const [responseInfo, setResponseInfo] = useState<ResponseInfo | undefined>(
        undefined,
    );

    const pendingConfirmation = useRef<{
        resolve: (token: string) => void;
        reject: (reason: Error) => void;
    } | null>(null);

    // Reset state when dialog is hidden
    useEffect(() => {
        if (!open) {
            const t = setTimeout(() => {
                setPhase("idle");
                setErrorMessage("");
                setConfirmInput("");
                setExpectedToken("");
                setSuccessData(undefined);
                setResponseInfo(undefined);
                pendingConfirmation.current = null;
            }, 150);
            return () => clearTimeout(t);
        }
    }, [open]);

    /** Callback when the request requires user confirmation */
    const onConfirmRequired: OnConfirmCallback = useCallback(
        (token: string) => {
            setExpectedToken(token);
            setConfirmInput("");
            setPhase("confirming");
            return new Promise<string>((resolve, reject) => {
                pendingConfirmation.current = { resolve, reject };
            });
        },
        [],
    );

    /** Function to fire the request */
    const fireRequest = useCallback(async () => {
        setPhase("loading");
        setErrorMessage("");
        setResponseInfo(undefined);

        try {
            const { data } = await fetchFn(onConfirmRequired);
            setResponseInfo({
                status: 200,
                body:
                    data !== undefined
                        ? JSON.stringify(data, null, 2)
                        : undefined,
            });
            setSuccessData(data);
            setPhase("success");
            onSuccess?.(data);
        } catch (err) {
            if (err instanceof CancelledError) {
                return;
            }

            // Extract response info from FetchError if available
            if (err instanceof FetchError) {
                setResponseInfo({
                    status: err.response.status,
                    body: err.message,
                });
            }

            const error =
                err instanceof Error ? err : new Error("Unknown error");
            const handled = onError?.(error);
            // Error handled externally, just close the dialog
            if (handled) {
                onOpenChange(false);
                return;
            }
            // Show error state in the dialog
            setErrorMessage(error.message);
            setPhase("error");
        }
    }, [fetchFn, onConfirmRequired, onSuccess, onError, onOpenChange]);

    const hasFired = useRef(false);
    // Handle auto-fire the request when the dialog is opened and `immediate` is true
    useEffect(() => {
        if (open && immediate && !hasFired.current) {
            hasFired.current = true;
            setTimeout(fireRequest, 0);
        }
        if (!open) {
            hasFired.current = false;
        }
    }, [open, immediate, fireRequest]);

    /** Completes the confirmation step that is waiting on the user's input */
    function handleConfirm() {
        const pending = pendingConfirmation.current;
        if (!pending) return;
        pendingConfirmation.current = null;
        setPhase("loading");
        pending.resolve(confirmInput);
    }

    /** Handle canceling the request (e.g pressing on cancel button) */
    function handleCancel() {
        const pending = pendingConfirmation.current;
        if (pending) {
            pendingConfirmation.current = null;
            pending.reject(new CancelledError());
        }
        onOpenChange(false);
    }

    const closeable = phase !== "loading";

    const phaseDescriptions: Record<Phase, React.ReactNode> = {
        idle: description,
        loading: "Please wait...",
        confirming: (description ?? "") + " This action requires confirmation",
        success: successDescription,
        error: errorMessage ?? "Something went wrong.",
    };

    const phaseDescription = phaseDescriptions[phase];

    const descriptionClassName = cn(
        "text-center text-balance",
        phase === "success" && "text-teal-600",
        phase === "error" && "text-destructive",
    );

    return (
        <Dialog
            open={open}
            onOpenChange={(v) => {
                // Dont do anything if dialog is not closeable
                if (!v && !closeable) return;

                if (!v && phase === "confirming") {
                    handleCancel();
                    return;
                }
                onOpenChange(v);
            }}
        >
            <DialogContent
                showCloseButton={false}
                className="max-h-[85vh] flex flex-col"
            >
                <DialogHeader>
                    {showIcon && <PhaseIcon phase={phase} />}
                    <DialogTitle className="text-center">{title}</DialogTitle>
                    {phaseDescription && (
                        <DialogDescription className={descriptionClassName}>
                            {phaseDescription}
                        </DialogDescription>
                    )}
                </DialogHeader>

                {phase === "success" && successContent && (
                    <div className="min-h-0 overflow-y-auto">
                        {successContent(successData)}
                    </div>
                )}

                {phase === "confirming" && (
                    <div className="space-y-2 py-2">
                        <Label htmlFor="fetch-dialog-confirm">
                            Type{" "}
                            <span className="font-mono font-semibold">
                                {expectedToken}
                            </span>{" "}
                            to confirm
                        </Label>
                        <Input
                            id="fetch-dialog-confirm"
                            value={confirmInput}
                            onChange={(e) => {
                                const target =
                                    e.target as HTMLInputElement | null;
                                setConfirmInput(target?.value ?? "");
                            }}
                            placeholder={expectedToken}
                            autoFocus
                            autoComplete="off"
                            onKeyDown={(e) => {
                                if (
                                    e.key === "Enter" &&
                                    confirmInput === expectedToken
                                ) {
                                    handleConfirm();
                                }
                            }}
                        />
                    </div>
                )}

                {/*Collapsible panel showing request info*/}
                <RequestDebugPanel
                    requestInfo={requestInfo}
                    responseInfo={responseInfo}
                />

                {/*Footer buttons*/}
                <DialogFooter className={cn(phase === "loading" && "hidden")}>
                    {phase === "idle" && (
                        <>
                            <Button
                                variant={cancelVariant}
                                onClick={() => onOpenChange(false)}
                            >
                                {cancelLabel}
                            </Button>
                            <Button
                                variant={proceedVariant}
                                onClick={fireRequest}
                            >
                                {proceedLabel}
                            </Button>
                        </>
                    )}

                    {phase === "confirming" && (
                        <>
                            <Button variant="outline" onClick={handleCancel}>
                                Cancel
                            </Button>
                            <Button
                                variant="destructive"
                                disabled={confirmInput !== expectedToken}
                                onClick={handleConfirm}
                            >
                                Confirm
                            </Button>
                        </>
                    )}

                    {(phase === "success" || phase === "error") && (
                        <Button
                            variant="outline"
                            onClick={() => onOpenChange(false)}
                        >
                            Close
                        </Button>
                    )}
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
