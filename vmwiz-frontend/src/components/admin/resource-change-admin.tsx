"use client";

import { type ReactElement, useCallback, useEffect, useState } from "react";
import { Check, Eye, Pencil, RefreshCw, Server, X } from "lucide-react";
import { formatDate } from "@/lib/utils";
import { StatusBadge } from "@/components/admin/vm-request-detail-dialog";
import { FilterBar } from "@/components/admin/filter-bar";
import { FetchDialog } from "@/components/fetch-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
    Tooltip,
    TooltipContent,
    TooltipTrigger,
} from "@/components/ui/tooltip";
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
import { ResourceChangeRequest } from "@/lib/types/api";
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
const filterOptions = [
    { value: "all", label: "All" },
    { value: "pending", label: "Pending" },
    { value: "accepted", label: "Accepted" },
    { value: "rejected", label: "Rejected" },
    { value: "failed", label: "Failed" },
    { value: "cancelled", label: "Cancelled" },
] as const;
type Filter = (typeof filterOptions)[number]["value"];

function matches(request: ResourceChangeRequest, filter: Filter) {
    return (
        filter === "all" ||
        request.Status === filter ||
        (filter === "accepted" && request.Status === "auto_approved")
    );
}

export function ResourceChangeAdmin() {
    const [requests, setRequests] = useState<ResourceChangeRequest[]>([]);
    const [loading, setLoading] = useState(true);
    const [filter, setFilter] = useState<Filter>("pending");
    const [selectedID, setSelectedID] = useState<number | null>(null);

    const load = useCallback(async () => {
        setLoading(true);
        try {
            setRequests(await fetchResourceChanges());
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

    const counts = Object.fromEntries(
        filterOptions.map(({ value }) => [
            value,
            requests.filter((request) => matches(request, value)).length,
        ]),
    ) as Record<Filter, number>;
    const selected = requests.find((request) => request.ID === selectedID);

    return (
        <div className="space-y-4">
            {loading ? (
                <Skeleton className="h-64 w-full" />
            ) : (
                <>
                    <div className="flex items-center justify-between gap-3">
                        <FilterBar
                            options={filterOptions}
                            filter={filter}
                            onFilterChange={setFilter}
                            counts={counts}
                        />
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
                    {visible.length === 0 ? (
                        <div className="flex flex-col items-center justify-center gap-2 py-12 text-muted-foreground">
                            <Eye className="size-8 opacity-40" />
                            <p className="text-sm">
                                No {filter !== "all" ? `${filter} ` : ""}
                                resource requests found.
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
                                        <TableHead key={label}>
                                            {label}
                                        </TableHead>
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
                                        onClick={() =>
                                            setSelectedID(request.ID)
                                        }
                                        onKeyDown={(event) => {
                                            if (
                                                event.key === "Enter" ||
                                                event.key === " "
                                            ) {
                                                event.preventDefault();
                                                setSelectedID(request.ID);
                                            }
                                        }}
                                    >
                                        <TableCell>
                                            <StatusBadge
                                                status={request.Status}
                                            />
                                        </TableCell>
                                        <TableCell className="font-mono text-xs">
                                            #{request.ID}
                                        </TableCell>
                                        <TableCell>
                                            {request.VMHostname}
                                        </TableCell>
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
                </>
            )}
            {selected && (
                <ResourceChangeDetail
                    key={selected.ID}
                    request={selected}
                    onClose={() => setSelectedID(null)}
                    onChange={() => void load()}
                />
            )}
        </div>
    );
}

// Disabled buttons get no pointer events, so the tooltip hangs on a wrapper.
function DisabledHint({
    hint,
    children,
}: {
    hint: string | null;
    children: ReactElement;
}) {
    if (!hint) return children;
    return (
        <Tooltip>
            <TooltipTrigger render={<span className="inline-flex" />}>
                {children}
            </TooltipTrigger>
            <TooltipContent>{hint}</TooltipContent>
        </Tooltip>
    );
}

function ResourceChangeDetail({
    request,
    onClose,
    onChange,
}: {
    request: ResourceChangeRequest;
    onClose: () => void;
    onChange: () => void;
}) {
    const [fields, setFields] = useState<ResourceChangeEditFields>({
        newCores: request.NewCores,
        newRamGB: request.NewRamGB,
        newDiskGB: request.NewDiskGB,
        newSecondaryDiskGB: request.NewSecondaryDiskGB,
    });
    const [reason, setReason] = useState("");
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
            successDescription: "Changes saved.",
        },
        accept: {
            title: "Accept resource request",
            request: prepareAcceptResourceChange(request.ID, reason.trim()),
            successDescription: "Resource change applied.",
        },
        reject: {
            title: "Reject resource request",
            request: prepareRejectResourceChange(request.ID, reason.trim()),
            successDescription: "Resource request rejected.",
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
                            <StatusBadge status={request.Status} />
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
                                            value={
                                                pending
                                                    ? fields[`new${key}`]
                                                    : request[`New${key}`]
                                            }
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
                        {pending && (
                            <label className="block space-y-1 text-xs text-muted-foreground">
                                <span className="block font-medium text-foreground">
                                    Reason for decision
                                </span>
                                <span className="block">
                                    Emailed to the VM owner. Required to reject,
                                    optional when accepting.
                                </span>
                                <Textarea
                                    maxLength={1000}
                                    value={reason}
                                    onChange={(event) =>
                                        setReason(event.target.value)
                                    }
                                />
                            </label>
                        )}
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
                            <DisabledHint
                                hint={
                                    reason.trim()
                                        ? null
                                        : "You have to provide a reason"
                                }
                            >
                                <Button
                                    variant="destructive"
                                    disabled={!reason.trim()}
                                    onClick={() => setAction("reject")}
                                >
                                    <X className="size-3.5" />
                                    Reject
                                </Button>
                            </DisabledHint>
                            <DisabledHint
                                hint={dirty ? "Save your changes first" : null}
                            >
                                <Button
                                    disabled={dirty}
                                    onClick={() => setAction("accept")}
                                >
                                    <Check className="size-3.5" />
                                    Accept
                                </Button>
                            </DisabledHint>
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
                    description={`Request #${request.ID} for ${request.VMHostname}. ${action === "edit" ? "Save the modified target resources?" : action === "accept" ? "Apply the requested resources? CPU and RAM changes take effect on reboot." : "Reject this request? The reason is emailed to the VM owner."}`}
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
                    onSuccess={onChange}
                    onError={onChange}
                />
            )}
        </>
    );
}
