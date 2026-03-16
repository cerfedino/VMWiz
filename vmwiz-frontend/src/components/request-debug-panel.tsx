"use client";

import { useState } from "react";
import { cn } from "@/lib/utils";
import { ChevronRight } from "lucide-react";
import { getReasonPhrase } from "http-status-codes";
import type { BackendRequest } from "@/lib/api";

export interface ResponseInfo {
    status: number;
    body?: string;
}

/**
 * The collapsible debug panel that shows request and response details.
 */
export function RequestDebugPanel({
    requestInfo,
    responseInfo,
}: {
    requestInfo?: BackendRequest;
    responseInfo?: ResponseInfo;
}) {
    const [expanded, setExpanded] = useState(false);

    if (!requestInfo) return null;

    return (
        <div className="mt-2 min-w-0 overflow-hidden">
            <button
                type="button"
                onClick={() => setExpanded((v) => !v)}
                className="flex items-center gap-0.5 text-[0.7rem] opacity-50 hover:opacity-80 transition-opacity"
            >
                <ChevronRight
                    className={cn(
                        "h-3 w-3 transition-transform duration-150",
                        expanded && "rotate-90",
                    )}
                />
                <span>Details</span>
            </button>

            <div
                className={cn(
                    "mt-2 grid transition-[grid-template-rows] duration-150 ease-in-out",
                    expanded ? "grid-rows-[1fr]" : "grid-rows-[0fr]",
                )}
            >
                <div className="overflow-hidden">
                    <div className="max-h-60 overflow-auto rounded-md bg-muted/50 p-3 text-left font-mono text-xs leading-relaxed">
                        <div>
                            <span className="font-semibold text-foreground">
                                {requestInfo.method}
                            </span>{" "}
                            <span className="text-muted-foreground">
                                {requestInfo.path}
                            </span>
                        </div>

                        {requestInfo.headers &&
                            Object.keys(requestInfo.headers).length > 0 && (
                                <div className="mt-1.5">
                                    <span className="font-semibold text-foreground">
                                        Headers
                                    </span>
                                    <pre className="mt-0.5 whitespace-pre-wrap break-all text-muted-foreground">
                                        {JSON.stringify(
                                            requestInfo.headers,
                                            null,
                                            2,
                                        )}
                                    </pre>
                                </div>
                            )}

                        {requestInfo.body !== undefined && (
                            <div className="mt-1.5">
                                <span className="font-semibold text-foreground">
                                    Body
                                </span>
                                <pre className="mt-0.5 whitespace-pre-wrap break-all text-muted-foreground">
                                    {JSON.stringify(
                                        JSON.parse(requestInfo.body),
                                        null,
                                        2,
                                    )}
                                </pre>
                            </div>
                        )}

                        {responseInfo && (
                            <div className="mt-2 border-t border-border pt-2">
                                <div>
                                    <span className="font-semibold text-foreground">
                                        Response
                                    </span>{" "}
                                    <span
                                        className={cn(
                                            responseInfo.status >= 200 &&
                                                responseInfo.status < 300
                                                ? "text-teal-600"
                                                : "text-destructive",
                                        )}
                                    >
                                        {responseInfo.status}{" "}
                                        {getReasonPhrase(responseInfo.status)}
                                    </span>
                                </div>
                                {responseInfo.body && (
                                    <pre className="mt-0.5 whitespace-pre-wrap break-all text-muted-foreground">
                                        {responseInfo.body}
                                    </pre>
                                )}
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
