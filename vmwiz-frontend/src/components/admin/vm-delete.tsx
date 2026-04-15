"use client";

import { useState } from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@/components/ui/button";
import { FetchDialog } from "@/components/fetch-dialog";
import { deleteVM, prepareDeleteVM } from "@/lib/api";
import { Trash2 } from "lucide-react";
import { VMLogsViewer } from "@/components/admin/vm-logs-viewer";

export function VMDelete() {
    const [hostname, setHostname] = useState("");
    const [deleteDNS, setDeleteDNS] = useState(true);
    const [dialogOpen, setDialogOpen] = useState(false);
    const [lastDeletedVM, setLastDeletedVM] = useState<string | null>(null);

    return (
        <div className="space-y-4">
            <div className="flex items-end gap-3">
                <div className="flex-1 space-y-2">
                    <Label htmlFor="delete-hostname">Hostname</Label>
                    <Input
                        id="delete-hostname"
                        value={hostname}
                        onChange={(e) => {
                            const target = e.target as HTMLInputElement | null;
                            setHostname(target?.value ?? "");
                        }}
                        placeholder="my-vm.vsos.ethz.ch"
                    />
                </div>
                <Button
                    variant="destructive"
                    disabled={!hostname.trim() || dialogOpen}
                    onClick={() => setDialogOpen(true)}
                >
                    <Trash2 className="h-4 w-4" />
                    Delete VM
                </Button>
            </div>

            <label className="flex items-center gap-2 text-sm">
                <Checkbox
                    checked={deleteDNS}
                    onCheckedChange={(checked) =>
                        setDeleteDNS(checked === true)
                    }
                />
                Also delete DNS entry
            </label>

            <FetchDialog
                open={dialogOpen}
                onOpenChange={setDialogOpen}
                fetchFn={(onConfirm) =>
                    deleteVM(hostname, deleteDNS, onConfirm).then((data) => ({
                        data,
                    }))
                }
                requestInfo={prepareDeleteVM(hostname, deleteDNS)}
                title="Delete VM"
                description={`You are about to delete "${hostname}"${deleteDNS ? " and its DNS entries" : ""}. This cannot be undone.`}
                onSuccess={() => {
                    setLastDeletedVM(hostname);
                    setHostname("");
                }}
            />

            {lastDeletedVM && (
                <div className="mt-6 space-y-3 pb-8 animate-in fade-in slide-in-from-top-4 duration-500">
                    <div className="flex items-center gap-2 text-sm font-medium">
                        <Trash2 className="size-4 text-muted-foreground" />
                        VM Deletion Logs for &quot;{lastDeletedVM}&quot;
                    </div>
                    <VMLogsViewer operationID={`vmdelete-${lastDeletedVM}`} />
                </div>
            )}
        </div>
    );
}
