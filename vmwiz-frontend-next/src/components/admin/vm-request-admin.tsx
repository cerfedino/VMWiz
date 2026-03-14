"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { formatDate } from "@/lib/utils";
import { fetchVMRequests } from "@/lib/api";
import type {
    VMRequest,
    VMRequestStatus,
    VMRequestListResponse,
} from "@/lib/types/api";
import {
    Table,
    TableHeader,
    TableBody,
    TableRow,
    TableHead,
    TableCell,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Eye, RefreshCw } from "lucide-react";
import {
    StatusBadge,
    RequestDetailDialog,
} from "@/components/admin/vm-request-detail-dialog";

type StatusFilter = "all" | VMRequestStatus;

/** Sort by pending first, then by creation date (newest first) */
function sortRequests(requests: VMRequest[]): VMRequest[] {
    return [...requests]
        .sort((a, b) => {
            // One of the two is pending
            if (
                (a.RequestStatus === "pending") !=
                (b.RequestStatus === "pending")
            ) {
                return (
                    (b.RequestStatus === "pending" ? 0 : 1) -
                    (a.RequestStatus === "pending" ? 0 : 1)
                );
            }

            // Otherwise sort by creation date
            return (
                new Date(a.RequestCreatedAt).getTime() -
                new Date(b.RequestCreatedAt).getTime()
            );
        })
        .reverse();
}

/** Set of buttons to filter by request status */
function FilterBar({
    filter,
    onFilterChange,
    counts,
}: {
    filter: StatusFilter;
    onFilterChange: (f: StatusFilter) => void;
    counts: Record<StatusFilter, number>;
}) {
    const options: { value: StatusFilter; label: string }[] = [
        { value: "all", label: "All" },
        { value: "pending", label: "Pending" },
        { value: "accepted", label: "Accepted" },
        { value: "rejected", label: "Rejected" },
    ];

    return (
        <div className="flex flex-wrap gap-1.5">
            {options.map(({ value, label }) => (
                <Button
                    key={value}
                    variant={filter === value ? "default" : "outline"}
                    size="sm"
                    onClick={() => onFilterChange(value)}
                >
                    {label}
                    <Badge
                        variant="secondary"
                        className="ml-1.5 px-1.5 text-[0.65rem]"
                    >
                        {counts[value]}
                    </Badge>
                </Button>
            ))}
        </div>
    );
}

function LoadingSkeleton() {
    return <Skeleton className="h-64 w-full" />;
}

export function VMRequestAdmin() {
    const [requests, setRequests] = useState<VMRequestListResponse>([]);
    const [loading, setLoading] = useState(true);
    const [filter, setFilter] = useState<StatusFilter>("all");
    const [filterInitialized, setFilterInitialized] = useState(false);
    const [selectedRequest, setSelectedRequest] = useState<VMRequest | null>(
        null,
    );
    const [editFormOpen, setEditFormOpen] = useState(false);

    /** Loads VM Requests */
    const loadRequests = useCallback(async () => {
        setLoading(true);
        try {
            const data = (await fetchVMRequests()) ?? [];
            setRequests(data);

            // If no filter is selected, default to "pending" if there are pending requests, otherwise "all"
            if (!filterInitialized) {
                const hasPending = data.some(
                    (r) => r.RequestStatus === "pending",
                );
                setFilter(hasPending ? "pending" : "all");
                setFilterInitialized(true);
            }
        } finally {
            setLoading(false);
        }
    }, [filterInitialized]);

    useEffect(() => {
        loadRequests();
    }, [loadRequests]);

    /** Counts how many requests there are for each status */
    const counts = useMemo(() => {
        const c: Record<StatusFilter, number> = {
            all: requests.length,
            pending: 0,
            accepted: 0,
            rejected: 0,
        };
        for (const r of requests) {
            c[r.RequestStatus]++;
        }
        return c;
    }, [requests]);

    /** Apply filter and sorting to requests */
    const displayedRequests = useMemo(() => {
        const filtered =
            filter === "all"
                ? requests
                : requests.filter((r) => r.RequestStatus === filter);
        return sortRequests(filtered);
    }, [requests, filter]);

    /** Handle successfull edit of a request */
    const handleEditSuccess = useCallback(() => {
        loadRequests();
    }, [loadRequests]);

    const openEditForm = useCallback((req: VMRequest) => {
        setSelectedRequest(req);
        setEditFormOpen(true);
    }, []);

    return (
        <div className="space-y-4">
            {loading ? (
                <LoadingSkeleton />
            ) : (
                <>
                    <div className="flex items-center justify-between gap-3">
                        <FilterBar
                            filter={filter}
                            onFilterChange={setFilter}
                            counts={counts}
                        />
                        <Button
                            variant="outline"
                            size="icon-sm"
                            onClick={loadRequests}
                            title="Refresh"
                        >
                            <RefreshCw className="size-3.5" />
                        </Button>
                    </div>

                    {displayedRequests.length === 0 ? (
                        <div className="flex flex-col items-center justify-center gap-2 py-12 text-muted-foreground">
                            <Eye className="size-8 opacity-40" />
                            <p className="text-sm">
                                No {filter !== "all" ? `${filter} ` : ""}VM
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
                                    <TableHead>Email</TableHead>
                                    <TableHead>Cores</TableHead>
                                    <TableHead>RAM</TableHead>
                                    <TableHead>Disk</TableHead>
                                    <TableHead>Created</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {displayedRequests.map((req) => {
                                    return (
                                        <TableRow
                                            key={req.ID}
                                            className={`font-medium cursor-pointer ${req.RequestStatus === "pending" ? "" : "opacity-60"}`}
                                            onClick={() => openEditForm(req)}
                                        >
                                            <TableCell>
                                                <StatusBadge
                                                    status={req.RequestStatus}
                                                />
                                            </TableCell>
                                            <TableCell className="font-mono text-xs">
                                                #{req.ID}
                                            </TableCell>
                                            <TableCell>
                                                {req.Hostname}
                                            </TableCell>
                                            <TableCell>{req.Email}</TableCell>
                                            <TableCell>{req.Cores}</TableCell>
                                            <TableCell>
                                                {req.RamGB} GB
                                            </TableCell>
                                            <TableCell>
                                                {req.DiskGB} GB
                                            </TableCell>
                                            <TableCell className="text-muted-foreground">
                                                {formatDate(
                                                    req.RequestCreatedAt,
                                                )}
                                            </TableCell>
                                        </TableRow>
                                    );
                                })}
                            </TableBody>
                        </Table>
                    )}
                </>
            )}

            <RequestDetailDialog
                key={selectedRequest?.ID}
                request={selectedRequest}
                open={editFormOpen}
                onOpenChange={setEditFormOpen}
                onMutationSuccess={handleEditSuccess}
            />
        </div>
    );
}
