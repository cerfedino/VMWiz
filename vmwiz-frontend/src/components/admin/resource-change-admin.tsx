"use client";

import { useCallback, useEffect, useState } from "react";
import { Check, Eye, Pencil, RefreshCw, Server, X } from "lucide-react";
import { formatDate } from "@/lib/utils";
import { StatusBadge } from "@/components/admin/vm-request-detail-dialog";
import { FetchDialog } from "@/components/fetch-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { ResourceChangeRequest, ResourceChangeStatus } from "@/lib/types/api";
import {
    fetchResourceChanges,
    prepareAcceptResourceChange,
    prepareEditResourceChange,
    prepareRejectResourceChange,
    ResourceChangeEditFields,
} from "@/lib/api";

const resources = [
    { label: "Cores", key: "Cores", unit: "" },
    { label: "RAM", key: "RamGB", unit: " GB" },
    { label: "SSD", key: "DiskGB", unit: " GB" },
    { label: "HDD", key: "SecondaryDiskGB", unit: " GB" },
] as const;
const filters = [
    "all",
    "pending",
    "accepted",
    "rejected",
    "failed",
    "cancelled",
] as const;
type Filter = (typeof filters)[number];

function matches(request: ResourceChangeRequest, filter: Filter) {
    return (
        filter === "all" ||
        request.Status === filter ||
        (filter === "accepted" && request.Status === "auto_approved")
    );
}

function ResourceStatusBadge({ status }: { status: ResourceChangeStatus }) {
    if (status === "pending" || status === "accepted" || status === "rejected")
        return <StatusBadge status={status} />;
    return (
        <Badge variant={status === "failed" ? "destructive" : "secondary"}>
            {status === "auto_approved"
                ? "Auto-approved"
                : status === "failed"
                  ? "Failed"
                  : "Cancelled"}
        </Badge>
    );
}

export function ResourceChangeAdmin() {
    const [requests, setRequests] = useState<ResourceChangeRequest[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [filter, setFilter] = useState<Filter>("pending");
    const [selected, setSelected] = useState<ResourceChangeRequest | null>(
        null,
    );

    const load = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            setRequests(await fetchResourceChanges());
        } catch (error) {
            setError(error instanceof Error ? error.message : String(error));
        } finally {
            setLoading(false);
        }
    }, []);
    useEffect(() => {
        void load();
    }, [load]);

    const visible = requests
        .filter((request) => matches(request, filter))
        .sort(
            (a, b) =>
                Number(b.Status === "pending") -
                    Number(a.Status === "pending") ||
                Date.parse(b.CreatedAt) - Date.parse(a.CreatedAt) ||
                b.ID - a.ID,
        );

    if (loading) return <Skeleton className="h-64 w-full" />;

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between gap-3">
                <div className="flex min-w-0 gap-1.5 overflow-x-auto">
                    {filters.map((value) => (
                        <Button
                            key={value}
                            variant={filter === value ? "default" : "outline"}
                            size="xs"
                            onClick={() => setFilter(value)}
                        >
                            {value[0].toUpperCase() + value.slice(1)}
                            <Badge
                                variant="secondary"
                                className="ml-1.5 px-1.5 text-[0.65rem]"
                            >
                                {
                                    requests.filter((request) =>
                                        matches(request, value),
                                    ).length
                                }
                            </Badge>
                        </Button>
                    ))}
                </div>
                <Button
                    variant="outline"
                    size="icon-sm"
                    className="shrink-0"
                    onClick={load}
                    title="Refresh"
                    aria-label="Refresh resource requests"
                >
                    <RefreshCw className="size-3.5" />
                </Button>
            </div>
            {error ? (
                <p role="alert" className="text-sm text-destructive">
                    {error}
                </p>
            ) : visible.length === 0 ? (
                <div className="flex flex-col items-center justify-center gap-2 py-12 text-muted-foreground">
                    <Eye className="size-8 opacity-40" />
                    <p className="text-sm">
                        No {filter !== "all" ? `${filter} ` : ""}resource
                        requests found.
                    </p>
                </div>
            ) : (
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Status</TableHead>
                            <TableHead>ID</TableHead>
                            <TableHead>Hostname</TableHead>
                            {resources.map(({ label }) => (
                                <TableHead key={label}>{label}</TableHead>
                            ))}
                            <TableHead>Created</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {visible.map((request) => (
                            <TableRow
                                key={request.ID}
                                className={`font-medium cursor-pointer ${request.Status === "pending" ? "" : "opacity-60"}`}
                                tabIndex={0}
                                onClick={() => setSelected(request)}
                                onKeyDown={(event) => {
                                    if (
                                        event.key === "Enter" ||
                                        event.key === " "
                                    ) {
                                        event.preventDefault();
                                        setSelected(request);
                                    }
                                }}
                            >
                                <TableCell>
                                    <ResourceStatusBadge
                                        status={request.Status}
                                    />
                                </TableCell>
                                <TableCell className="font-mono text-xs">
                                    #{request.ID}
                                </TableCell>
                                <TableCell>{request.VMHostname}</TableCell>
                                {resources.map(({ key, unit }) => (
                                    <TableCell key={key}>
                                        {request[`Current${key}`] ===
                                        request[`New${key}`]
                                            ? request[`New${key}`]
                                            : `${request[`Current${key}`]} → ${request[`New${key}`]}`}
                                        {unit}
                                    </TableCell>
                                ))}
                                <TableCell className="text-muted-foreground">
                                    {formatDate(request.CreatedAt)}
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            )}
            {selected && (
                <ResourceChangeDetail
                    key={selected.ID}
                    request={selected}
                    onClose={() => setSelected(null)}
                    onSuccess={() => {
                        setSelected(null);
                        void load();
                    }}
                />
            )}
        </div>
    );
}

function ResourceChangeDetail({
    request,
    onClose,
    onSuccess,
}: {
    request: ResourceChangeRequest;
    onClose: () => void;
    onSuccess: () => void;
}) {
    const [fields, setFields] = useState<ResourceChangeEditFields>({
        newCores: request.NewCores,
        newRamGB: request.NewRamGB,
        newDiskGB: request.NewDiskGB,
        newSecondaryDiskGB: request.NewSecondaryDiskGB,
    });
    const [action, setAction] = useState<"edit" | "accept" | "reject" | null>(
        null,
    );
    const pending = request.Status === "pending";
    const dirty = resources.some(
        ({ key }) => fields[`new${key}`] !== request[`New${key}`],
    );
    const valid = resources.every(
        ({ key }) =>
            Number.isInteger(fields[`new${key}`]) &&
            fields[`new${key}`] >=
                (key === "SecondaryDiskGB" || key === "DiskGB"
                    ? request[`Current${key}`]
                    : 1),
    );
    const actions = {
        edit: {
            title: "Save changes",
            request: prepareEditResourceChange(request.ID, fields),
        },
        accept: {
            title: "Accept resource request",
            request: prepareAcceptResourceChange(request.ID),
        },
        reject: {
            title: "Reject resource request",
            request: prepareRejectResourceChange(request.ID),
        },
    };

    return (
        <>
            <Dialog
                open={!action}
                onOpenChange={(open) => {
                    if (!open) onClose();
                }}
            >
                <DialogContent className="sm:max-w-2xl">
                    <DialogHeader>
                        <div className="flex items-center gap-3">
                            <ResourceStatusBadge status={request.Status} />
                            <DialogTitle>
                                Resource request #{request.ID}
                            </DialogTitle>
                        </div>
                        <DialogDescription>
                            {request.VMHostname} · Created{" "}
                            {formatDate(request.CreatedAt)}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="max-h-[65vh] space-y-6 overflow-y-auto pr-1">
                        <div className="space-y-3 text-sm">
                            {request.RequestReason && (
                                <div>
                                    <p className="mb-1 font-medium">Reason</p>
                                    <p className="whitespace-pre-wrap">
                                        {request.RequestReason}
                                    </p>
                                </div>
                            )}
                            {request.NeedsReviewReasons.length > 0 && (
                                <div>
                                    <p className="mb-1 font-medium">
                                        Review required
                                    </p>
                                    <ul className="list-disc space-y-1 pl-4">
                                        {request.NeedsReviewReasons.map(
                                            (reason, i) => (
                                                <li key={i}>{reason}</li>
                                            ),
                                        )}
                                    </ul>
                                </div>
                            )}
                        </div>
                        <Separator />
                        <div className="space-y-3">
                            <div className="flex items-center gap-2 text-sm font-medium">
                                <Server className="size-4 text-muted-foreground" />
                                VM Specification
                            </div>
                            <div className="grid grid-cols-2 gap-x-6 gap-y-4">
                                {resources.map(({ label, key, unit }) => (
                                    <label
                                        key={key}
                                        className="space-y-1 text-xs text-muted-foreground"
                                    >
                                        <span>
                                            {label}
                                            {unit && " (GB)"} · currently{" "}
                                            {request[`Current${key}`]}
                                        </span>
                                        <Input
                                            type="number"
                                            disabled={!pending}
                                            required
                                            step={1}
                                            min={
                                                key === "SecondaryDiskGB" ||
                                                key === "DiskGB"
                                                    ? request[`Current${key}`]
                                                    : 1
                                            }
                                            value={fields[`new${key}`]}
                                            onChange={(event) =>
                                                setFields({
                                                    ...fields,
                                                    [`new${key}`]: Number(
                                                        event.target.value,
                                                    ),
                                                })
                                            }
                                        />
                                    </label>
                                ))}
                            </div>
                            <p className="text-xs text-muted-foreground">
                                CPU and RAM changes take effect on reboot.
                                Existing disks cannot shrink.
                            </p>
                        </div>
                        {request.RequesterFingerprint && (
                            <div className="space-y-1 text-xs">
                                <p className="text-muted-foreground">
                                    Requester fingerprint
                                </p>
                                <p className="break-all font-mono">
                                    {request.RequesterFingerprint}
                                </p>
                            </div>
                        )}
                        {request.ProcessedAt && (
                            <div className="space-y-1 text-sm">
                                <p className="text-xs text-muted-foreground">
                                    Processed {formatDate(request.ProcessedAt)}
                                </p>
                                <p className="whitespace-pre-wrap">
                                    {request.ProcessedNote}
                                </p>
                            </div>
                        )}
                    </div>
                    {pending && (
                        <DialogFooter>
                            <Button
                                variant="outline"
                                disabled={!dirty || !valid}
                                onClick={() => setAction("edit")}
                            >
                                <Pencil className="size-3.5" />
                                Save changes
                            </Button>
                            <Button
                                variant="destructive"
                                onClick={() => setAction("reject")}
                            >
                                <X className="size-3.5" />
                                Reject
                            </Button>
                            <Button
                                disabled={dirty}
                                onClick={() => setAction("accept")}
                            >
                                <Check className="size-3.5" />
                                Accept
                            </Button>
                        </DialogFooter>
                    )}
                </DialogContent>
            </Dialog>
            {action && (
                <FetchDialog
                    open
                    onOpenChange={(open) => {
                        if (!open) setAction(null);
                    }}
                    {...actions[action]}
                    description={`Request #${request.ID} for ${request.VMHostname}. ${action === "edit" ? "Save the modified target resources?" : action === "accept" ? "Apply the requested resources? CPU and RAM changes take effect on reboot." : "Reject this request?"}`}
                    proceedLabel={
                        action === "edit"
                            ? "Save"
                            : action === "accept"
                              ? "Accept"
                              : "Reject"
                    }
                    proceedVariant={
                        action === "reject" ? "destructive" : "default"
                    }
                    onSuccess={onSuccess}
                />
            )}
        </>
    );
}
