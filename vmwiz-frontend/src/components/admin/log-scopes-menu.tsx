"use client";

import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { LogStream } from "@/components/log-stream";
import { fetchLogScopes } from "@/lib/api";
import type { LogScope } from "@/lib/types/api";
import {
    ScrollText,
    Loader2,
    CheckCircle2,
    AlertTriangle,
    Server,
} from "lucide-react";
import { cn } from "@/lib/utils";

// The catch-all scope ("0") isn't returned by the list endpoint; it's pinned.
const CATCH_ALL: LogScope = {
    id: "0",
    label: "System / general logs",
    startedAt: "",
    ended: false,
    failed: false,
    available: true,
};

function ScopeIcon({ scope }: { scope: LogScope }) {
    if (!scope.ended)
        return (
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        );
    if (scope.failed) return <AlertTriangle className="h-4 w-4 text-red-500" />;
    return <CheckCircle2 className="h-4 w-4 text-teal-600" />;
}

const PAGE_SIZE = 30;

// Upserts scopes by id and keeps them newest-first (UUIDv7 ids sort by time).
function merge(prev: LogScope[], incoming: LogScope[]): LogScope[] {
    const byId = new Map(prev.map((s) => [s.id, s]));
    for (const s of incoming) byId.set(s.id, s);
    return Array.from(byId.values()).sort((a, b) => (a.id < b.id ? 1 : -1));
}

export function LogScopesMenu() {
    const [open, setOpen] = useState(false);
    const [scopes, setScopes] = useState<LogScope[]>([]);
    const [hasMore, setHasMore] = useState(true);
    const [loading, setLoading] = useState(false);
    const [selected, setSelected] = useState<LogScope | null>(null);
    const ref = useRef<HTMLDivElement>(null);
    const listRef = useRef<HTMLDivElement>(null);

    // On open: load the first page and poll it so running scopes stay fresh.
    // The merge keeps any older pages already loaded by scrolling.
    useEffect(() => {
        if (!open) return;
        setScopes([]);
        setHasMore(true);
        let active = true;
        const refresh = () =>
            fetchLogScopes(undefined, PAGE_SIZE)
                .then((s) => active && setScopes((prev) => merge(prev, s)))
                .catch(() => {});
        refresh();
        const t = setInterval(refresh, 3000);
        return () => {
            active = false;
            clearInterval(t);
        };
    }, [open]);

    // Loads the page of scopes older than the oldest one currently shown.
    const loadMore = () => {
        if (loading || !hasMore || scopes.length === 0) return;
        setLoading(true);
        const oldest = scopes[scopes.length - 1].id;
        fetchLogScopes(oldest, PAGE_SIZE)
            .then((older) => {
                setScopes((prev) => merge(prev, older));
                setHasMore(older.length === PAGE_SIZE);
            })
            .catch(() => {})
            .finally(() => setLoading(false));
    };

    // Keep loading new pages until the there is the need of a scrollbar. after that we keep fetching pages only when we reach the end of the scrollbar
    useEffect(() => {
        const el = listRef.current;
        if (!el || loading || !hasMore || scopes.length === 0) return;
        if (el.scrollHeight <= el.clientHeight + 1) {
            loadMore();
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [open, scopes, hasMore, loading]);

    // Close the dropdown when clicking outside it.
    useEffect(() => {
        if (!open) return;
        const onDown = (e: MouseEvent) => {
            if (ref.current && !ref.current.contains(e.target as Node)) {
                setOpen(false);
            }
        };
        document.addEventListener("mousedown", onDown);
        return () => document.removeEventListener("mousedown", onDown);
    }, [open]);

    return (
        <div className="relative" ref={ref}>
            <Button
                variant="outline"
                size="sm"
                onClick={() => setOpen((o) => !o)}
            >
                <ScrollText className="h-4 w-4" />
                Logs
            </Button>

            {open && (
                <div className="absolute right-0 z-50 mt-2 w-180 overflow-hidden rounded-md border border-border bg-background shadow-lg">
                    <button
                        className="flex w-full items-center gap-2 border-b border-border px-3 py-2 text-left text-sm font-medium hover:bg-muted"
                        onClick={() => {
                            setSelected(CATCH_ALL);
                            setOpen(false);
                        }}
                    >
                        <Server className="h-4 w-4 text-muted-foreground" />
                        <span className="flex-1 truncate">
                            {CATCH_ALL.label}
                        </span>
                    </button>
                    <div
                        ref={listRef}
                        className="max-h-96 overflow-y-auto"
                        onScroll={(e) => {
                            const el = e.currentTarget;
                            if (
                                el.scrollTop + el.clientHeight >=
                                el.scrollHeight - 32
                            ) {
                                loadMore();
                            }
                        }}
                    >
                        {scopes.length === 0 && (
                            <div className="p-4 text-center text-sm text-muted-foreground">
                                No log scopes yet.
                            </div>
                        )}
                        {scopes.map((s) => (
                            <button
                                key={s.id}
                                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-muted"
                                onClick={() => {
                                    setSelected(s);
                                    setOpen(false);
                                }}
                            >
                                <ScopeIcon scope={s} />
                                <span
                                    className={cn(
                                        "flex-1 truncate",
                                        !s.available && "text-muted-foreground",
                                    )}
                                >
                                    {s.label}
                                </span>
                                {!s.available && (
                                    <span className="shrink-0 text-xs text-muted-foreground italic">
                                        unavailable
                                    </span>
                                )}
                                <span className="shrink-0 text-xs text-muted-foreground">
                                    {new Date(s.startedAt).toLocaleTimeString()}
                                </span>
                            </button>
                        ))}
                        {loading && (
                            <div className="flex justify-center py-2">
                                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                            </div>
                        )}
                    </div>
                </div>
            )}

            <Dialog
                open={selected !== null}
                onOpenChange={(v) => !v && setSelected(null)}
            >
                <DialogContent className="sm:max-w-3xl">
                    <DialogHeader>
                        <DialogTitle>{selected?.label}</DialogTitle>
                    </DialogHeader>
                    {selected &&
                        (selected.available ? (
                            <LogStream
                                key={selected.id}
                                logScopeId={selected.id}
                            />
                        ) : (
                            <p className="py-4 text-center text-sm text-muted-foreground">
                                These logs are no longer available
                            </p>
                        ))}
                </DialogContent>
            </Dialog>
        </div>
    );
}
