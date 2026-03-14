"use client";

import { useState } from "react";
import { formatDate } from "@/lib/utils";
import {
    acceptVMRequest,
    prepareAcceptVMRequest,
    rejectVMRequest,
    prepareRejectVMRequest,
    editVMRequest,
    prepareEditVMRequest,
} from "@/lib/api";
import type {
    VMRequest,
    VMRequestStatus,
    VMRequestEditFields,
} from "@/lib/types/api";
import { FetchDialog } from "@/components/fetch-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import {
    Check,
    X,
    Clock,
    Pencil,
    Server,
    Key,
    MessageSquare,
    User,
} from "lucide-react";

export function StatusBadge({ status }: { status: VMRequestStatus }) {
    switch (status) {
        case "pending":
            return (
                <Badge className="bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400">
                    <Clock className="mr-1 size-3" />
                    Pending
                </Badge>
            );
        case "accepted":
            return (
                <Badge className="bg-teal-100 text-teal-800 dark:bg-teal-900/30 dark:text-teal-400">
                    <Check className="mr-1 size-3" />
                    Accepted
                </Badge>
            );
        case "rejected":
            return (
                <Badge variant="destructive">
                    <X className="mr-1 size-3" />
                    Rejected
                </Badge>
            );
    }
}

type EditState = Required<VMRequestEditFields>;

/** Returns true if any editable field has been changed from the original request */
function isEditDirty(req: VMRequest, fields: EditState): boolean {
    return (
        fields.Hostname !== req.Hostname ||
        fields.Cores !== req.Cores ||
        fields.RamGB !== req.RamGB ||
        fields.DiskGB !== req.DiskGB
    );
}

/** Builds the PATCH-like payload: only includes fields that differ from the original */
function buildEditPayload(
    req: VMRequest,
    fields: EditState,
): VMRequestEditFields {
    const payload: VMRequestEditFields = {};
    if (fields.Hostname !== req.Hostname) payload.Hostname = fields.Hostname;
    if (fields.Cores !== req.Cores) payload.Cores = fields.Cores;
    if (fields.RamGB !== req.RamGB) payload.RamGB = fields.RamGB;
    if (fields.DiskGB !== req.DiskGB) payload.DiskGB = fields.DiskGB;
    return payload;
}

/** Simple label + value pair used throughout the detail dialog */
function DetailField({
    label,
    children,
}: {
    label: string;
    children: React.ReactNode;
}) {
    return (
        <div className="space-y-1">
            <Label className="text-xs text-muted-foreground">{label}</Label>
            <div className="text-sm">{children}</div>
        </div>
    );
}

/** Edit dialog for VM requests. If the request is not pending, it simply shows the details without allowing edits or actions. */
export function RequestDetailDialog({
    request,
    open,
    onOpenChange,
    onMutationSuccess: onEditSuccess,
}: {
    request: VMRequest | null;
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onMutationSuccess: () => void;
}) {
    const [editFields, setEditFields] = useState<EditState>({
        Hostname: request?.Hostname ?? "",
        Cores: request?.Cores ?? 0,
        RamGB: request?.RamGB ?? 0,
        DiskGB: request?.DiskGB ?? 0,
    });
    const [acceptDialogOpen, setAcceptDialogOpen] = useState(false);
    const [rejectDialogOpen, setRejectDialogOpen] = useState(false);
    const [editDialogOpen, setEditDialogOpen] = useState(false);

    if (!request) return null;

    const isPending = request.RequestStatus === "pending";
    const dirty = isPending && isEditDirty(request, editFields);
    const editPayload = buildEditPayload(request, editFields);

    return (
        <>
            <Dialog open={open} onOpenChange={onOpenChange}>
                <DialogContent className="sm:max-w-2xl">
                    <DialogHeader>
                        <div className="flex items-center gap-3">
                            <StatusBadge status={request.RequestStatus} />
                            <DialogTitle>Request #{request.ID}</DialogTitle>
                        </div>
                        <DialogDescription>
                            Created {formatDate(request.RequestCreatedAt)}
                        </DialogDescription>
                    </DialogHeader>

                    <div className="max-h-[65vh] overflow-y-auto">
                        <div className="space-y-6 pr-1">
                            <div className="space-y-3">
                                <div className="flex items-center gap-2 text-sm font-medium">
                                    <User className="size-4 text-muted-foreground" />
                                    General Information
                                </div>
                                <div className="grid grid-cols-2 gap-x-6 gap-y-4">
                                    <DetailField label="Email">
                                        {request.Email}
                                    </DetailField>
                                    <DetailField label="Personal Email">
                                        {request.PersonalEmail || "—"}
                                    </DetailField>
                                    {request.IsOrganization && (
                                        <DetailField label="Organization">
                                            {request.OrgName}
                                        </DetailField>
                                    )}
                                </div>
                            </div>

                            {request.Comments && (
                                <>
                                    <div className="space-y-3">
                                        <div className="flex items-center gap-2 text-sm font-medium">
                                            <MessageSquare className="size-4 text-muted-foreground" />
                                            Comments
                                        </div>
                                        <p className="text-sm whitespace-pre-wrap">
                                            {request.Comments}
                                        </p>
                                    </div>
                                </>
                            )}
                            <Separator />

                            <div className="space-y-3">
                                <div className="flex items-center gap-2 text-sm font-medium">
                                    <Server className="size-4 text-muted-foreground" />
                                    VM Specification
                                </div>
                                <div className="grid grid-cols-2 gap-x-6 gap-y-4">
                                    <DetailField label="Image">
                                        {request.Image}
                                    </DetailField>

                                    <DetailField label="Hostname">
                                        <Input
                                            disabled={!isPending}
                                            value={editFields.Hostname}
                                            onChange={(e) => {
                                                const target =
                                                    e.target as HTMLInputElement | null;
                                                setEditFields((f) => ({
                                                    ...f,
                                                    Hostname:
                                                        target?.value ?? "",
                                                }));
                                            }}
                                        />
                                    </DetailField>

                                    <DetailField label="Cores">
                                        <Input
                                            type="number"
                                            disabled={!isPending}
                                            value={editFields.Cores}
                                            onChange={(e) => {
                                                const target =
                                                    e.target as HTMLInputElement | null;
                                                setEditFields((f) => ({
                                                    ...f,
                                                    Cores:
                                                        parseInt(
                                                            target?.value ??
                                                                "0",
                                                        ) || 0,
                                                }));
                                            }}
                                        />
                                    </DetailField>

                                    <DetailField label="RAM (GB)">
                                        <Input
                                            type="number"
                                            disabled={!isPending}
                                            value={editFields.RamGB}
                                            onChange={(e) => {
                                                const target =
                                                    e.target as HTMLInputElement | null;
                                                setEditFields((f) => ({
                                                    ...f,
                                                    RamGB:
                                                        parseInt(
                                                            target?.value ??
                                                                "0",
                                                        ) || 0,
                                                }));
                                            }}
                                        />
                                    </DetailField>

                                    <DetailField label="Disk (GB)">
                                        <Input
                                            type="number"
                                            disabled={!isPending}
                                            value={editFields.DiskGB}
                                            onChange={(e) => {
                                                const target =
                                                    e.target as HTMLInputElement | null;
                                                setEditFields((f) => ({
                                                    ...f,
                                                    DiskGB:
                                                        parseInt(
                                                            target?.value ??
                                                                "0",
                                                        ) || 0,
                                                }));
                                            }}
                                        />
                                    </DetailField>
                                </div>
                            </div>

                            <div className="space-y-3">
                                <div className="flex items-center gap-2 text-sm font-medium">
                                    <Key className="size-4 text-muted-foreground" />
                                    SSH Public Keys
                                </div>
                                {request.SshPubkeys &&
                                request.SshPubkeys.length > 0 ? (
                                    <div className="space-y-2">
                                        {request.SshPubkeys.map((key, i) => (
                                            <div
                                                key={i}
                                                className="flex gap-2.5 rounded-md bg-muted/50 px-3 py-2"
                                            >
                                                <span className="shrink-0 pt-px text-xs font-medium text-muted-foreground">
                                                    {i + 1}.
                                                </span>
                                                <span
                                                    className="font-mono text-xs break-all"
                                                    title={key}
                                                >
                                                    {key}
                                                </span>
                                            </div>
                                        ))}
                                    </div>
                                ) : (
                                    <p className="text-sm text-muted-foreground">
                                        No SSH keys provided.
                                    </p>
                                )}
                            </div>
                        </div>
                    </div>

                    {isPending && (
                        <DialogFooter>
                            {dirty && (
                                <Button
                                    variant="outline"
                                    onClick={() => setEditDialogOpen(true)}
                                >
                                    <Pencil className="size-3.5" />
                                    Save Changes
                                </Button>
                            )}
                            <Button
                                variant="destructive"
                                onClick={() => setRejectDialogOpen(true)}
                            >
                                <X className="size-3.5" />
                                Reject
                            </Button>
                            <Button onClick={() => setAcceptDialogOpen(true)}>
                                <Check className="size-3.5" />
                                Accept
                            </Button>
                        </DialogFooter>
                    )}
                </DialogContent>
            </Dialog>

            <FetchDialog
                open={acceptDialogOpen}
                onOpenChange={setAcceptDialogOpen}
                fetchFn={(onConfirm) =>
                    acceptVMRequest(request.ID, onConfirm).then((data) => ({
                        data,
                    }))
                }
                requestInfo={prepareAcceptVMRequest(request.ID)}
                title="Accept VM Request"
                description={`You are about to accept request #${request.ID} for "${request.Hostname}". This will provision the VM.`}
                proceedLabel="Accept"
                successDescription="VM request has been accepted successfully."
                onSuccess={() => {
                    onOpenChange(false);
                    onEditSuccess();
                }}
            />

            <FetchDialog
                open={rejectDialogOpen}
                onOpenChange={setRejectDialogOpen}
                fetchFn={(onConfirm) =>
                    rejectVMRequest(request.ID, onConfirm).then((data) => ({
                        data,
                    }))
                }
                requestInfo={prepareRejectVMRequest(request.ID)}
                title="Reject VM Request"
                description={`You are about to reject request #${request.ID} for "${request.Hostname}". This action cannot be undone.`}
                proceedLabel="Reject"
                proceedVariant="destructive"
                successDescription="VM request has been rejected."
                onSuccess={() => {
                    onOpenChange(false);
                    onEditSuccess();
                }}
            />

            <FetchDialog
                open={editDialogOpen}
                onOpenChange={setEditDialogOpen}
                fetchFn={(onConfirm) =>
                    editVMRequest(request.ID, editPayload, onConfirm).then(
                        (data) => ({
                            data,
                        }),
                    )
                }
                requestInfo={prepareEditVMRequest(request.ID, editPayload)}
                title="Save Changes"
                description={`Save the modified fields for request #${request.ID}?`}
                proceedLabel="Save"
                successDescription="Changes saved successfully."
                onSuccess={() => {
                    onEditSuccess();
                }}
            />
        </>
    );
}
