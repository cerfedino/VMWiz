"use client";

import { useAuth } from "@/context/auth";
import { Separator } from "@/components/ui/separator";
import { VMDelete } from "@/components/admin/vm-delete";
import { DnsDelete } from "@/components/admin/dns-delete";
import { SurveyAdmin } from "@/components/admin/survey-admin";
import { VMRequestAdmin } from "@/components/admin/vm-request-admin";
import { ClipboardList, BarChart3, Trash2, User, Server } from "lucide-react";
import { fetchFreeIPv4Count } from "@/lib/api";
import { useEffect, useState } from "react";

export default function ConsolePage() {
    const { user, loading } = useAuth();
    const [freeIPs, setFreeIPs] = useState<number | null>(null);

    useEffect(() => {
        fetchFreeIPv4Count().then((count) => setFreeIPs(count));
    }, [user]);

    return (
        <div className="mx-auto w-full max-w-4xl space-y-10 p-6 pb-16">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-bold">Admin Console</h1>
                <div className="flex items-center gap-3 text-sm text-muted-foreground">
                    {typeof freeIPs === "number" && (
                        <div className="inline-flex items-center gap-1.5 rounded-md border border-border bg-muted/50 px-2.5 py-1">
                            <Server className="h-3.5 w-3.5" />
                            <span className="font-semibold tabular-nums text-foreground">
                                {freeIPs}
                            </span>
                            <span>free IPv4</span>
                        </div>
                    )}
                    <div className="flex items-center gap-2">
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
            </div>

            <section className="space-y-4">
                <h2 className="flex items-center gap-2 text-lg font-semibold">
                    <Trash2 className="h-5 w-5" />
                    Delete VM
                </h2>
                <VMDelete />
            </section>

            <section className="space-y-4">
                <h2 className="flex items-center gap-2 text-lg font-semibold">
                    <Trash2 className="h-5 w-5" />
                    Delete DNS
                </h2>
                <DnsDelete />
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
