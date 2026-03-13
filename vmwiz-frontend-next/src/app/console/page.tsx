"use client";

import { useAuth } from "@/context/auth";
import { Loader2, ClipboardList, BarChart3, Trash2 } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import { VMDelete } from "@/components/admin/vm-delete";
import { SurveyAdmin } from "@/components/admin/survey-admin";

export default function ConsolePage() {
    const { status } = useAuth();

    if (status !== "authenticated") {
        return (
            <div className="flex min-h-[60vh] items-center justify-center">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
        );
    }

    return (
        <div className="mx-auto w-full max-w-4xl space-y-10 p-6 pb-16">
            <h1 className="text-2xl font-bold">Admin Console</h1>

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
                <p className="text-muted-foreground">
                    VM requests will go here.
                </p>
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
