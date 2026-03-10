"use client";

import { useAuth } from "@/context/auth";
import { Loader2 } from "lucide-react";

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
        <div className="mx-auto w-full max-w-4xl space-y-8 p-6 pb-16">
            <h1 className="text-2xl font-bold">Admin Console</h1>
            <p className="text-muted-foreground">Ciao mamma</p>
        </div>
    );
}
