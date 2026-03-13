"use client";

import { useState } from "react";
import { useSearchParams } from "next/navigation";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import { FetchDialog } from "@/components/fetch-dialog";
import { submitSurveyResponse } from "@/lib/api";
import { AlertTriangle, Check, X } from "lucide-react";

function InvalidLinkCard() {
    return (
        <div className="flex min-h-[70vh] items-center justify-center px-4">
            <Card className="w-full max-w-md text-center animate-in fade-in-0 zoom-in-95 duration-300">
                <CardHeader>
                    <div className="mx-auto mb-2 flex h-14 w-14 items-center justify-center rounded-full bg-destructive/10">
                        <AlertTriangle className="h-7 w-7 text-destructive" />
                    </div>
                    <CardTitle className="text-xl">
                        Invalid Survey Link
                    </CardTitle>
                    <CardDescription className="text-balance">
                        This survey link appears to be missing required
                        parameters. Please use the link from the email you
                        received.
                    </CardDescription>
                </CardHeader>
            </Card>
        </div>
    );
}

export function SurveyForm() {
    const searchParams = useSearchParams();

    const pollId = searchParams.get("id") ?? "";
    const hostname = searchParams.get("hostname") ?? "";

    const [keepDialog, setKeepDialog] = useState(false);
    const [removeDialog, setRemoveDialog] = useState(false);
    const [submitted, setSubmitted] = useState(false);

    if (!pollId || !hostname) {
        return <InvalidLinkCard />;
    }

    return (
        <>
            <FetchDialog
                open={keepDialog}
                onOpenChange={setKeepDialog}
                fetchFn={(onConfirm) =>
                    submitSurveyResponse(pollId, true, onConfirm).then(
                        (data) => ({ data }),
                    )
                }
                immediate
                title="VM Usage Survey"
                successDescription={
                    <>
                        Your response has been recorded. Your VM{" "}
                        <strong>{hostname}</strong> will remain active.
                    </>
                }
                onSuccess={() => setSubmitted(true)}
            />

            <FetchDialog
                open={removeDialog}
                onOpenChange={setRemoveDialog}
                fetchFn={(onConfirm) =>
                    submitSurveyResponse(pollId, false, onConfirm).then(
                        (data) => ({ data }),
                    )
                }
                title="Confirm Removal"
                description={`Are you sure you want to give up access to ${hostname}? It will be stopped and removed.`}
                cancelLabel="No, keep it"
                proceedLabel="Yes, I don't need it"
                proceedVariant="destructive"
                successDescription={
                    <>
                        Your response has been recorded.{" "}
                        <strong>{hostname}</strong> will be stopped and removed.
                    </>
                }
                onSuccess={() => setSubmitted(true)}
            />

            <div className="flex min-h-[70vh] items-center justify-center px-4">
                <Card className="w-full max-w-md py-8 text-center">
                    <CardHeader className="gap-2 px-8">
                        <CardTitle className="text-xl">
                            VM Usage Survey
                        </CardTitle>
                        <CardDescription>
                            Do you still need this virtual machine?
                        </CardDescription>
                    </CardHeader>

                    <CardContent className="space-y-6 px-8 pt-2">
                        <div className="inline-flex items-center rounded-lg border bg-muted/50 px-5 py-2.5 font-mono text-sm font-semibold">
                            {hostname}
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <button
                                type="button"
                                disabled={submitted}
                                onClick={() => setKeepDialog(true)}
                                className="group flex items-center justify-center gap-2.5 rounded-lg bg-teal-50 px-4 py-4 transition-all hover:bg-teal-100 hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-500 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50"
                            >
                                <Check className="h-5 w-5 text-teal-600" />
                                <span className="text-sm font-semibold text-teal-900">
                                    Yes, keep it
                                </span>
                            </button>

                            <button
                                type="button"
                                disabled={submitted}
                                onClick={() => setRemoveDialog(true)}
                                className="group flex items-center justify-center gap-2.5 rounded-lg bg-red-50 px-4 py-4 transition-all hover:bg-red-100 hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50"
                            >
                                <X className="h-5 w-5 text-red-600" />
                                <span className="text-sm font-semibold text-red-900">
                                    {"No, I don't need it"}
                                </span>
                            </button>
                        </div>

                        <p className="text-xs text-muted-foreground/60">
                            If you do not respond, your VM may be shut down
                            automatically.
                        </p>
                    </CardContent>
                </Card>
            </div>
        </>
    );
}
