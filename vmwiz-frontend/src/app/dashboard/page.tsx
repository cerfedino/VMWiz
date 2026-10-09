"use client";

import { useAuth } from "@/context/auth";
import { Separator } from "@/components/ui/separator";
import { VMDelete } from "@/components/admin/vm-delete";
import { DnsDelete } from "@/components/admin/dns-delete";
import { LogScopesMenu } from "@/components/admin/log-scopes-menu";
import { SurveyAdmin } from "@/components/admin/survey-admin";
import { VMRequestAdmin } from "@/components/admin/vm-request-admin";
import { RequestClosureAdmin } from "@/components/admin/request-closure-admin";
import { AnimatedHeight } from "@/components/animated-height";
import { ClipboardList, BarChart3, Trash2, User, Server } from "lucide-react";
import { fetchFreeIPv4Count } from "@/lib/api";
import { useEffect, useState } from "react";

export default function DashboardPage() {
    const { user, loading } = useAuth();
    const [freeIPs, setFreeIPs] = useState<number | null>(null);

    useEffect(() => {
        fetchFreeIPv4Count().then((count) => setFreeIPs(count));
    }, [user]);

    return (
        <div className="mx-auto w-full max-w-4xl space-y-10 p-6 pb-16">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                <h1 className="text-2xl font-bold">Dashboard</h1>
                <div className="relative flex items-center gap-3 text-sm text-muted-foreground">
                    <LogScopesMenu />
                    {typeof freeIPs === "number" && (
                        <div className="inline-flex items-center gap-1.5 rounded-md border border-border bg-muted/50 px-2.5 py-1">
                            <Server className="h-3.5 w-3.5" />
                            <span className="font-semibold tabular-nums text-foreground">
                                {freeIPs}
                            </span>
                            <span>free IPv4</span>
                        </div>
                    )}
                    <div className="flex min-w-0 items-center gap-2">
                        <User className="h-4 w-4 shrink-0" />
                        {loading ? (
                            <span className="animate-pulse">…</span>
                        ) : user ? (
                            <span className="truncate">{user.email}</span>
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
                <AnimatedHeight>
                    <RequestClosureAdmin />
                </AnimatedHeight>
                <AnimatedHeight>
                    <VMRequestAdmin />
                </AnimatedHeight>
            </section>

            <Separator className="opacity-30" />

            <section className="space-y-4">
                <h2 className="flex items-center gap-2 text-lg font-semibold">
                    <BarChart3 className="h-5 w-5" />
                    Surveys
                </h2>
                <AnimatedHeight>
                    <SurveyAdmin />
                </AnimatedHeight>
            </section>
        </div>
    );
}
