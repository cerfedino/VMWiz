"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";

export interface LogLine {
    ts: string;
    level: string;
    scope: string;
    msg: string;
}

/**
 * Streams the live logs of a backend task scope over SSE, rendering each line
 * as it arrives and invoking onDone when the task finishes.
 */
export function LogStream({
    logScopeId,
    onDone,
}: {
    logScopeId: string;
    onDone?: (failed: boolean) => void;
}) {
    const [lines, setLines] = useState<LogLine[]>([]);
    const [done, setDone] = useState(false);
    const bottomRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const es = new EventSource(
            `/api/logs/${logScopeId}/stream?subscopes=true`,
        );
        es.onmessage = (e) => {
            try {
                setLines((prev) => [...prev, JSON.parse(e.data) as LogLine]);
            } catch {
                // ignore malformed line
            }
        };
        es.addEventListener("done", (e) => {
            es.close();
            setDone(true);
            let failed = false;
            try {
                failed = (
                    JSON.parse((e as MessageEvent).data) as { failed: boolean }
                ).failed;
            } catch {
                // ignore malformed terminal event
            }
            onDone?.(failed);
        });
        return () => es.close();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [logScopeId]);

    useEffect(() => {
        bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    }, [lines, done]);

    return (
        <div className="max-h-[60vh] overflow-y-auto rounded-md bg-zinc-950 p-3 font-mono text-xs leading-relaxed text-zinc-100">
            {lines.length === 0 && (
                <div className="text-zinc-400">Waiting for logs…</div>
            )}
            {lines.map((l, i) => (
                <div
                    key={i}
                    className={cn(
                        "break-all whitespace-pre-wrap",
                        l.level === "ERROR" && "text-red-400",
                    )}
                >
                    {l.msg}
                </div>
            ))}
            {!done && (
                <div className="mt-1 flex gap-1" aria-label="Streaming logs">
                    <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-zinc-400 [animation-delay:0ms]" />
                    <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-zinc-400 [animation-delay:200ms]" />
                    <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-zinc-400 [animation-delay:400ms]" />
                </div>
            )}
            <div ref={bottomRef} />
        </div>
    );
}
