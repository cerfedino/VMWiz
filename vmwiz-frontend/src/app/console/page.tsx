"use client";

import { useAuth } from "@/context/auth";
import { Separator } from "@/components/ui/separator";
import { VMDelete } from "@/components/admin/vm-delete";
import { SurveyAdmin } from "@/components/admin/survey-admin";
import { VMRequestAdmin } from "@/components/admin/vm-request-admin";
import { ClipboardList, BarChart3, Trash2, User, Server } from "lucide-react";
import { fetchFreeIPv4Count } from "@/lib/api";
import { useEffect, useState } from "react";

export default function ConsolePage() {
    const { user, loading } = useAuth();
    const [freeIPs, setFreeIPs] = useState<number | null>(null);

    useEffect(() => {
        if (user) {
            fetchFreeIPv4Count()
                .then((count) => setFreeIPs(count))
                .catch(console.error);
        }
    }, [user]);

    return (
        <div className="mx-auto w-full max-w-4xl space-y-10 p-6 pb-16">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-bold">Admin Console</h1>
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <User className="h-4 w-4" />
                    {loading ? (
                        <span className="animate-pulse">…</span>
                    ) : user ? (
                        <span>{user.email}</span>
                    ) : (
                        <span>Not logged in</span>
                    )}
                </div>
            </div>

            {user && typeof freeIPs === "number" && (
                <div className="flex w-full items-center justify-between rounded-lg border border-border bg-card p-4 shadow-sm">
                    <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-blue-500/10 text-blue-500">
                            <Server className="h-5 w-5" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-muted-foreground">
                                IPv4 Allocation
                            </p>
                            <p className="text-2xl font-bold">
                                {freeIPs} <span className="text-base font-normal text-muted-foreground">available IPs</span>
                            </p>
                        </div>
                    </div>
                </div>
            )}

            <section className="space-y-4">
                <h2 className="flex items-center gap-2 text-lg font-semibold">
                    <Trash2 className="h-5 w-5" />
                    Delete VM
                </h2>
                <VMDelete />
            </section>

            <Separator className="opacity-30" />
            <section className="space-y-4">
                <h2 className="flex items-center gap-2 text-lg font-semibold">
                    <ClipboardList className="h-5 w-5" />
                    VM Requests
                </h2>
                <VMRequestAdmin />
            </section>

            <Separator className="opacity-30" />

            <section className="space-y-4">
                <h2 className="flex items-center gap-2 text-lg font-semibold">
                    <BarChart3 className="h-5 w-5" />
                    Surveys
                </h2>
                <SurveyAdmin />
            </section>
        </div>
    );
}
