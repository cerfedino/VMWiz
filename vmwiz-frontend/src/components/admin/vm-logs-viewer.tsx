"use client";

import { useEffect, useState, useRef } from "react";
import { getVMLogsStreamUrl } from "@/lib/api";

export function VMLogsViewer({ operationID }: { operationID: string }) {
    const [logs, setLogs] = useState<string[]>([]);
    const endRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const url = getVMLogsStreamUrl(operationID);
        const source = new EventSource(url, { withCredentials: true });

        source.onmessage = (event) => {
            const raw = event.data;
            const message = raw.replace(/\\n/g, "\n");
            setLogs((prev) => [...prev, message]);
        };

        source.onerror = (err) => {
            console.error("SSE error", err);
            // In case the stream finishes or dies, event source might auto-reconnect.
            // If it's a permanent close from backend, it will stop.
            // To prevent endless reconnect loops if 404, we could close, but we leave default reconnect behavior on.
            if (source.readyState === EventSource.CLOSED) {
                 source.close();
            }
        };

        return () => {
            source.close();
        };
    }, [operationID]);

    useEffect(() => {
        if (endRef.current) {
            endRef.current.scrollIntoView({ behavior: "smooth" });
        }
    }, [logs]);

    return (
        <div className="mt-4 rounded-md bg-stone-950 p-4 font-mono text-xs text-stone-300 overflow-y-auto max-h-[300px] border border-border">
            {logs.length === 0 ? (
                <div className="text-muted-foreground italic">Waiting for logs...</div>
            ) : (
                logs.map((log, i) => (
                    <div key={i} className="whitespace-pre-wrap">{log}</div>
                ))
            )}
            <div ref={endRef} />
        </div>
    );
}
