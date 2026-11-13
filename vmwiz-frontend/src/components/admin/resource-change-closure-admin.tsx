"use client";

import { useCallback, useEffect, useState } from "react";
import { formatDate } from "@/lib/utils";
import {
    fetchResourceChangeClosure,
    prepareCloseResourceChanges,
    prepareReopenResourceChanges,
} from "@/lib/api";
import type { RequestClosure } from "@/lib/types/api";
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
import { Lock } from "lucide-react";

/**
 * Toggle for accepting new resource change requests from the SSH portal.
 */
export function ResourceChangeClosureAdmin() {
    const [closure, setClosure] = useState<RequestClosure | null>(null);
    const [reason, setReason] = useState("");
    const [reasonDialogOpen, setReasonDialogOpen] = useState(false);
    const [closeDialogOpen, setCloseDialogOpen] = useState(false);
    const [reopenDialogOpen, setReopenDialogOpen] = useState(false);

    const loadClosure = useCallback(async () => {
        setClosure(await fetchResourceChangeClosure());
    }, []);

    useEffect(() => {
        loadClosure();
    }, [loadClosure]);

    if (!closure) return <Skeleton className="h-14 w-full" />;

    return (
        <div className="flex items-center gap-3 rounded-lg border p-3">
            <Switch
                id="accept-resource-requests"
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
                <Label htmlFor="accept-resource-requests">
                    {closure.closed
                        ? "Not accepting new resource requests"
                        : "Accepting new resource requests"}
                </Label>
                {closure.closed && (
                    <p className="text-xs text-muted-foreground">
                        Closed {formatDate(closure.closedAt ?? "")}:{" "}
                        {closure.reason}
                    </p>
                )}
            </div>

            {/* Ask for the reason before closing */}
            <Dialog open={reasonDialogOpen} onOpenChange={setReasonDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Close resource requests</DialogTitle>
                        <DialogDescription>
                            The reason is shown to VM owners in the SSH portal
                            when they try to submit a request. Pending requests
                            can still be accepted or rejected.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-2">
                        <Label htmlFor="resource-closure-reason">Reason</Label>
                        <Textarea
                            id="resource-closure-reason"
                            value={reason}
                            onChange={(e) => setReason(e.target.value)}
                            placeholder="e.g. The cluster is at capacity until the new nodes arrive."
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
                request={prepareCloseResourceChanges(reason.trim())}
                immediate
                title="Closing resource requests"
                successDescription="Resource requests are now closed."
                onSuccess={loadClosure}
            />

            <FetchDialog
                open={reopenDialogOpen}
                onOpenChange={setReopenDialogOpen}
                request={prepareReopenResourceChanges()}
                title="Reopen resource requests"
                description="VM owners will be able to submit resource requests again."
                proceedLabel="Reopen"
                successDescription="Resource requests are open again."
                onSuccess={loadClosure}
            />
        </div>
    );
}
