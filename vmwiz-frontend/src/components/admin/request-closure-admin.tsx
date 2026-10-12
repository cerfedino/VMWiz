"use client";

import { useCallback, useEffect, useState, type ReactNode } from "react";
import { cn, formatDate } from "@/lib/utils";
import {
    fetchRequestClosure,
    prepareCloseRequests,
    prepareReopenRequests,
    prepareFetchWaitlist,
} from "@/lib/api";
import type {
    RequestClosure,
    WaitlistEntry,
    WaitlistListResponse,
} from "@/lib/types/api";
import { toast } from "sonner";
import { FetchDialog } from "@/components/fetch-dialog";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { Copy, Lock, Users } from "lucide-react";

/**
 * Toggle for accepting new VM requests, plus the list of people waiting to be notified when requests reopen.
 */
export function RequestClosureAdmin() {
    const [closure, setClosure] = useState<RequestClosure | null>(null);
    const [reason, setReason] = useState("");
    const [reasonDialogOpen, setReasonDialogOpen] = useState(false);
    const [closeDialogOpen, setCloseDialogOpen] = useState(false);
    const [reopenDialogOpen, setReopenDialogOpen] = useState(false);
    const [waitlistDialogOpen, setWaitlistDialogOpen] = useState(false);

    const loadClosure = useCallback(async () => {
        setClosure(await fetchRequestClosure());
    }, []);

    useEffect(() => {
        loadClosure();
    }, [loadClosure]);

    if (!closure) return <Skeleton className="h-14 w-full" />;

    return (
        <div className="flex items-center justify-between gap-3 rounded-lg border p-3">
            <div className="flex min-w-0 items-center gap-3">
                <Switch
                    id="accept-requests"
                    className="shrink-0"
                    checked={!closure.closed}
                    onCheckedChange={(checked) => {
                        if (checked) {
                            setReopenDialogOpen(true);
                        } else {
                            setReason("");
                            setReasonDialogOpen(true);
                        }
                    }}
                />
                <div className="space-y-0.5">
                    <Label htmlFor="accept-requests">
                        {closure.closed
                            ? "Not accepting new VM requests"
                            : "Accepting new VM requests"}
                    </Label>
                    {closure.closed && (
                        <p className="text-xs text-muted-foreground">
                            Closed {formatDate(closure.closedAt ?? "")}:{" "}
                            {closure.reason}
                        </p>
                    )}
                </div>
            </div>
            <Button
                variant="outline"
                size="sm"
                className="shrink-0"
                onClick={() => setWaitlistDialogOpen(true)}
            >
                <Users className="h-4 w-4" />
                Waitlist
            </Button>

            {/* Ask for the reason before closing */}
            <Dialog open={reasonDialogOpen} onOpenChange={setReasonDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Close VM requests</DialogTitle>
                        <DialogDescription>
                            The reason is shown to users on the request form.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-2">
                        <Label htmlFor="closure-reason">Reason</Label>
                        <Textarea
                            id="closure-reason"
                            value={reason}
                            onChange={(e) => setReason(e.target.value)}
                            placeholder="e.g. We are out of IPv4 addresses until the next allocation."
                            autoFocus
                        />
                    </div>
                    <DialogFooter>
                        <Button
                            variant="outline"
                            onClick={() => setReasonDialogOpen(false)}
                        >
                            Cancel
                        </Button>
                        <Button
                            variant="destructive"
                            disabled={!reason.trim()}
                            onClick={() => {
                                setReasonDialogOpen(false);
                                setCloseDialogOpen(true);
                            }}
                        >
                            <Lock className="h-4 w-4" />
                            Close requests
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <FetchDialog
                open={closeDialogOpen}
                onOpenChange={setCloseDialogOpen}
                request={prepareCloseRequests(reason.trim())}
                immediate
                title="Closing VM requests"
                successDescription="VM requests are now closed."
                onSuccess={loadClosure}
            />

            <FetchDialog
                open={reopenDialogOpen}
                onOpenChange={setReopenDialogOpen}
                request={prepareReopenRequests()}
                title="Reopen VM requests"
                description="Users will be able to submit VM requests again."
                proceedLabel="Reopen"
                successDescription="VM requests are open again."
                onSuccess={loadClosure}
            />

            {/* Dialog showing the waitlist */}
            <FetchDialog
                open={waitlistDialogOpen}
                onOpenChange={setWaitlistDialogOpen}
                request={prepareFetchWaitlist()}
                immediate
                showIcon={false}
                title="Waitlist"
                successDescription={null}
                successContent={(data) => (
                    <ClosureTimeline closures={data as WaitlistListResponse} />
                )}
            />
        </div>
    );
}

/** Timeline UI for the waitlist */
function ClosureTimeline({ closures }: { closures: WaitlistListResponse }) {
    if (closures.length === 0) {
        return (
            <p className="py-4 text-center text-sm text-muted-foreground">
                VM requests have never been closed.
            </p>
        );
    }
    return (
        <ol className="ml-1.5 space-y-6 border-l pl-6 text-sm">
            {closures.map((closure) => (
                <li key={closure.id}>
                    <ul className="space-y-2">
                        <TimelineItem dot="event">
                            <div className="flex items-center gap-2">
                                <span className="font-medium">Closed</span>
                                <span className="text-xs text-muted-foreground">
                                    {formatDate(closure.closedAt)}
                                </span>
                                {closure.waitlist.length > 0 && (
                                    <Button
                                        variant="ghost"
                                        size="icon-xs"
                                        className="ml-auto"
                                        title="Copy emails"
                                        onClick={() =>
                                            copyEmails(closure.waitlist)
                                        }
                                    >
                                        <Copy />
                                        <span className="sr-only">
                                            Copy emails
                                        </span>
                                    </Button>
                                )}
                            </div>
                            <p className="text-muted-foreground">
                                {closure.reason}
                            </p>
                        </TimelineItem>
                        {closure.waitlist.length === 0 && (
                            <li className="text-xs text-muted-foreground">
                                Nobody signed up
                            </li>
                        )}
                        {closure.waitlist.map((entry) => (
                            <TimelineItem key={entry.email} dot="person">
                                <div className="flex items-baseline justify-between gap-4">
                                    <span className="font-mono text-xs">
                                        {entry.email}
                                    </span>
                                    <span className="shrink-0 text-xs text-muted-foreground">
                                        {formatDate(entry.createdAt)}
                                    </span>
                                </div>
                            </TimelineItem>
                        ))}
                        {closure.reopenedAt ? (
                            <TimelineItem dot="event">
                                <div className="flex items-baseline gap-2">
                                    <span className="font-medium">
                                        Reopened
                                    </span>
                                    <span className="text-xs text-muted-foreground">
                                        {formatDate(closure.reopenedAt)}
                                    </span>
                                </div>
                            </TimelineItem>
                        ) : (
                            <TimelineItem dot="open">
                                <span className="text-muted-foreground">
                                    Still closed
                                </span>
                            </TimelineItem>
                        )}
                    </ul>
                </li>
            ))}
        </ol>
    );
}

async function copyEmails(entries: WaitlistEntry[]) {
    await navigator.clipboard.writeText(entries.map((e) => e.email).join(", "));
    toast.success(
        `Copied ${entries.length} email${entries.length === 1 ? "" : "s"}`,
    );
}

const TIMELINE_DOTS = {
    event: "top-1.5 size-2.5 bg-foreground",
    open: "top-1.5 size-2.5 border-2 border-foreground bg-background",
    person: "top-1.5 size-1.5 bg-muted-foreground",
};

function TimelineItem({
    dot,
    children,
}: {
    dot: keyof typeof TIMELINE_DOTS;
    children: ReactNode;
}) {
    return (
        <li className="relative">
            <span
                className={cn(
                    "absolute -left-6 -translate-x-1/2 rounded-full ring-2 ring-background",
                    TIMELINE_DOTS[dot],
                )}
            />
            {children}
        </li>
    );
}
