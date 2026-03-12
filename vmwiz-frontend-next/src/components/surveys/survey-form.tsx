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
import { Button } from "@/components/ui/button";
import { ConfirmationDialog } from "@/components/confirmation-dialog";
import { submitSurveyResponse } from "@/lib/api";
import {
    CheckCircle2,
    XCircle,
    AlertTriangle,
    Check,
    X,
    Loader2,
} from "lucide-react";

type SurveyStatus = "idle" | "submitting" | "success" | "error";

/** The dialog that shows the status of the survey submission (successful/error) */
function StatusCard({
    icon,
    iconClassName,
    title,
    description,
    children,
}: {
    /** The icon to display at the top of the status card */
    icon: React.ReactNode;
    iconClassName: string;
    title: string;
    description: React.ReactNode;
    children?: React.ReactNode;
}) {
    return (
        <div className="flex min-h-[70vh] items-center justify-center px-4">
            <Card
                className={`w-full max-w-md text-center animate-in fade-in-0 zoom-in-95 duration-300`}
            >
                <CardHeader>
                    <div
                        className={`mx-auto mb-2 flex h-14 w-14 items-center justify-center rounded-full ${iconClassName}`}
                    >
                        {icon}
                    </div>
                    <CardTitle className="text-xl">{title}</CardTitle>
                    <CardDescription className="text-balance">
                        {description}
                    </CardDescription>
                </CardHeader>
                {children && <CardContent>{children}</CardContent>}
            </Card>
        </div>
    );
}

export function SurveyForm() {
    const searchParams = useSearchParams();

    const pollId = searchParams.get("id") ?? "";
    const hostname = searchParams.get("hostname") ?? "";

    const [status, setStatus] = useState<SurveyStatus>("idle");
    const [showConfirm, setShowConfirm] = useState(false);
    const [keptVM, setKeptVM] = useState<boolean | null>(null);

    async function submitChoice(keep: boolean) {
        setStatus("submitting");
        setKeptVM(keep);
        setShowConfirm(false);

        try {
            const response = await submitSurveyResponse(pollId, keep);

            if (response.ok) {
                setStatus("success");
            } else {
                setStatus("error");
            }
        } catch (err) {
            console.error("Error submitting survey response:", err);
            setStatus("error");
        }
    }

    if (!pollId || !hostname) {
        return (
            <StatusCard
                icon={<AlertTriangle className="h-7 w-7 text-destructive" />}
                iconClassName="bg-destructive/10"
                title="Invalid Survey Link"
                description="This survey link appears to be missing required parameters. Please use the link from the email you received."
            />
        );
    }

    if (status === "success") {
        return (
            <StatusCard
                icon={<CheckCircle2 className="h-7 w-7 text-teal-600" />}
                iconClassName="bg-teal-100"
                title="Thank you!"
                description={
                    keptVM ? (
                        <>
                            Your response has been recorded. Your VM{" "}
                            <strong className="text-foreground">
                                {hostname}
                            </strong>{" "}
                            will remain active.
                        </>
                    ) : (
                        <>
                            Your response has been recorded.{" "}
                            <strong className="text-foreground">
                                {hostname}
                            </strong>{" "}
                            will be stopped and removed.
                        </>
                    )
                }
            >
                <p className="text-xs text-muted-foreground">
                    You can close this window.
                </p>
            </StatusCard>
        );
    }

    if (status === "error") {
        return (
            <StatusCard
                icon={<XCircle className="h-7 w-7 text-destructive" />}
                iconClassName="bg-destructive/10"
                title="Something went wrong"
                description="An error occurred while submitting your response. Please try again later. If the problem persists, contact us via email."
            >
                <Button variant="outline" onClick={() => setStatus("idle")}>
                    Try again
                </Button>
            </StatusCard>
        );
    }

    return (
        <>
            <ConfirmationDialog
                open={showConfirm}
                onOpenChange={setShowConfirm}
                title="Confirm Answer"
                description="Are you sure you want to lose access to the VM?"
                icon={<AlertTriangle className="h-6 w-6 text-amber-600" />}
                iconClassName="bg-amber-100"
                footer={
                    <>
                        <Button
                            variant="outline"
                            onClick={() => setShowConfirm(false)}
                        >
                            No, keep it
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={() => submitChoice(false)}
                        >
                            {"Yes, I don't need it"}
                        </Button>
                    </>
                }
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
                                disabled={status === "submitting"}
                                onClick={() => submitChoice(true)}
                                className="group flex items-center justify-center gap-2.5 rounded-lg bg-teal-50 px-4 py-4 transition-all hover:bg-teal-100 hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-500 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50"
                            >
                                <Check className="h-5 w-5 text-teal-600" />
                                <span className="text-sm font-semibold text-teal-900">
                                    Yes, keep it
                                </span>
                            </button>

                            <button
                                type="button"
                                disabled={status === "submitting"}
                                onClick={() => setShowConfirm(true)}
                                className="group flex items-center justify-center gap-2.5 rounded-lg bg-red-50 px-4 py-4 transition-all hover:bg-red-100 hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50"
                            >
                                <X className="h-5 w-5 text-red-600" />
                                <span className="text-sm font-semibold text-red-900">
                                    {"No, I don't need it"}
                                </span>
                            </button>
                        </div>

                        {status === "submitting" && (
                            <div className="animate-in fade-in-0 duration-200 flex items-center justify-center gap-2 text-sm text-muted-foreground">
                                <Loader2 className="h-4 w-4 animate-spin" />
                                Submitting…
                            </div>
                        )}

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
