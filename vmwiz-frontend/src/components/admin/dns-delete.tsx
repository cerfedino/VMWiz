"use client";

import { useState } from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { FetchDialog } from "@/components/fetch-dialog";
import { prepareDeleteDNS } from "@/lib/api";
import { Trash2 } from "lucide-react";

export function DnsDelete() {
    const [hostname, setHostname] = useState("");
    const [dialogOpen, setDialogOpen] = useState(false);

    return (
        <div className="space-y-4">
            <div className="flex items-end gap-3">
                <div className="flex-1 space-y-2">
                    <Label htmlFor="delete-dns-hostname">Hostname</Label>
                    <Input
                        id="delete-dns-hostname"
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
                    Delete DNS
                </Button>
            </div>

            <FetchDialog
                open={dialogOpen}
                onOpenChange={setDialogOpen}
                request={prepareDeleteDNS(hostname)}
                title="Delete DNS entries"
                description={`You are about to delete the DNS entries for "${hostname}". This cannot be undone.`}
                onSuccess={() => setHostname("")}
            />
        </div>
    );
}
