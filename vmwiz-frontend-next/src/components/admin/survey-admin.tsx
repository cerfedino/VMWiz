"use client";

import { useState, useEffect, useCallback } from "react";
import {
    fetchSurveyIds,
    fetchSurveyInfo,
    fetchSurveyResponses,
    prepareFetchSurveyResponses,
    createSurvey,
    prepareCreateSurvey,
    resendUnsent,
    prepareResendUnsent,
    resendUnanswered,
    prepareResendUnanswered,
} from "@/lib/api";
import type { SurveyInfo, SurveyResponseCategory } from "@/lib/types/api";
import { FetchDialog } from "@/components/fetch-dialog";
import {
    Accordion,
    AccordionItem,
    AccordionTrigger,
    AccordionContent,
} from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

import { Skeleton } from "@/components/ui/skeleton";
import {
    ThumbsUp,
    ThumbsDown,
    HelpCircle,
    MailWarning,
    Plus,
    RotateCcw,
    BellRing,
} from "lucide-react";

/** Dialog titles when drilling into a response category. */
const RESPONSE_TITLES: Record<SurveyResponseCategory, string> = {
    positive: "Positive Responses",
    negative: "Negative Responses",
    none: "Unanswered",
    notsent: "Unsent Emails",
};

/** Per-category display config. */
const STAT_CONFIG: Record<
    SurveyResponseCategory,
    {
        label: string;
        icon: typeof ThumbsUp;
        colorClass: string;
        countClass: string;
    }
> = {
    positive: {
        label: "Positive",
        icon: ThumbsUp,
        colorClass: "text-teal-600",
        countClass: "text-teal-700 hover:text-teal-900",
    },
    negative: {
        label: "Negative",
        icon: ThumbsDown,
        colorClass: "text-red-600",
        countClass: "text-red-700 hover:text-red-900",
    },
    none: {
        label: "Unanswered",
        icon: HelpCircle,
        colorClass: "text-blue-600",
        countClass: "text-blue-700 hover:text-blue-900",
    },
    notsent: {
        label: "Not Sent",
        icon: MailWarning,
        colorClass: "text-amber-600",
        countClass: "text-amber-700 hover:text-amber-900",
    },
};

const RESPONSE_CATEGORIES = Object.keys(
    STAT_CONFIG,
) as SurveyResponseCategory[];

const getCount = (survey: SurveyInfo, category: SurveyResponseCategory) => {
    switch (category) {
        case "positive":
            return survey.positive;
        case "negative":
            return survey.negative;
        case "none":
            return survey.not_responded;
        case "notsent":
            return survey.not_sent;
    }
};

/** Formats an ISO date to a readable format (e.g. "May 12, 2025"). */
function formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    return date.toLocaleDateString("en-US", {
        year: "numeric",
        month: "short",
        day: "numeric",
    });
}

/**
 * Admin panel for managing VM usage surveys.
 */
export function SurveyAdmin() {
    const [surveys, setSurveys] = useState<SurveyInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [expandedItems, setExpandedItems] = useState<string[]>([]);
    const [createSurveyOpen, setCreateSurveyOpen] = useState(false);
    const [responseDialog, setResponseDialog] = useState<{
        open: boolean;
        surveyId: number;
        category: SurveyResponseCategory;
    }>({ open: false, surveyId: 0, category: "positive" });

    const [resendDialog, setResendDialog] = useState<{
        open: boolean;
        type: "unsent" | "unanswered";
        surveyId: number;
    }>({ open: false, type: "unsent", surveyId: 0 });

    /** Fetches all survey IDs and their info. */
    const loadSurveys = useCallback(async () => {
        setLoading(true);
        try {
            const { surveyIds } = await fetchSurveyIds();
            const infos = await Promise.all(
                surveyIds.map((id) => fetchSurveyInfo(id)),
            );
            infos.sort(
                (a, b) =>
                    new Date(b.date).getTime() - new Date(a.date).getTime(),
            );
            setSurveys(infos);
        } catch (err) {
            console.error("Failed to load surveys", err);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        loadSurveys();
    }, [loadSurveys]);

    // Show the dialog containing the VM for a given survey and category
    const showResponses = useCallback(
        (surveyId: number, category: SurveyResponseCategory) => {
            setResponseDialog({ open: true, surveyId, category });
        },
        [],
    );

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <Button onClick={() => setCreateSurveyOpen(true)}>
                    <Plus className="h-4 w-4" />
                    Create New Survey
                </Button>
            </div>

            {loading ? (
                <div className="space-y-3">
                    <Skeleton className="h-12 w-full" />
                    <Skeleton className="h-12 w-full" />
                    <Skeleton className="h-12 w-full" />
                </div>
            ) : surveys.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                    No surveys found.
                </p>
            ) : (
                <Accordion
                    value={expandedItems}
                    onValueChange={setExpandedItems}
                >
                    {surveys.map((survey) => {
                        const total =
                            survey.positive +
                            survey.negative +
                            survey.not_responded +
                            survey.not_sent;
                        return (
                            <AccordionItem
                                key={survey.surveyId}
                                value={String(survey.surveyId)}
                            >
                                <AccordionTrigger>
                                    <div className="flex items-center gap-3">
                                        <span className="font-semibold">
                                            Survey #{survey.surveyId}
                                        </span>
                                        <span className="text-muted-foreground">
                                            {formatDate(survey.date)}
                                        </span>
                                        <Badge variant="secondary">
                                            {total} total
                                        </Badge>
                                    </div>
                                </AccordionTrigger>
                                <AccordionContent>
                                    <div className="space-y-4">
                                        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                                            {RESPONSE_CATEGORIES.map(
                                                (category) => {
                                                    const config =
                                                        STAT_CONFIG[category];
                                                    const Icon = config.icon;
                                                    const count = getCount(
                                                        survey,
                                                        category,
                                                    );
                                                    return (
                                                        <div
                                                            key={category}
                                                            className="flex flex-col items-center gap-1.5 rounded-lg border p-3"
                                                        >
                                                            <Icon
                                                                className={`h-4 w-4 ${config.colorClass}`}
                                                            />
                                                            <span className="text-xs text-muted-foreground">
                                                                {config.label}
                                                            </span>
                                                            <button
                                                                type="button"
                                                                onClick={() =>
                                                                    showResponses(
                                                                        survey.surveyId,
                                                                        category,
                                                                    )
                                                                }
                                                                className={`text-lg font-bold underline underline-offset-2 ${config.countClass}`}
                                                            >
                                                                {count}
                                                            </button>
                                                        </div>
                                                    );
                                                },
                                            )}
                                        </div>
                                        <div className="flex gap-2">
                                            <Button
                                                variant="outline"
                                                size="sm"
                                                onClick={() =>
                                                    setResendDialog({
                                                        open: true,
                                                        type: "unsent",
                                                        surveyId:
                                                            survey.surveyId,
                                                    })
                                                }
                                            >
                                                <RotateCcw className="h-4 w-4" />
                                                Retry unsent
                                            </Button>
                                            <Button
                                                variant="outline"
                                                size="sm"
                                                onClick={() =>
                                                    setResendDialog({
                                                        open: true,
                                                        type: "unanswered",
                                                        surveyId:
                                                            survey.surveyId,
                                                    })
                                                }
                                            >
                                                <BellRing className="h-4 w-4" />
                                                Send reminder
                                            </Button>
                                        </div>
                                    </div>
                                </AccordionContent>
                            </AccordionItem>
                        );
                    })}
                </Accordion>
            )}

            {/*Dialog for creating a new survey*/}
            <FetchDialog
                open={createSurveyOpen}
                onOpenChange={setCreateSurveyOpen}
                fetchFn={(onConfirm) =>
                    createSurvey(onConfirm).then((data) => ({ data }))
                }
                requestInfo={prepareCreateSurvey()}
                title="Create New Survey"
                description="This will create a new usage survey and send emails to all VM owners. Continue?"
                proceedLabel="Create"
                successDescription="Survey created successfully."
                onSuccess={loadSurveys}
            />

            {/* Dialog for sending reminder emails*/}
            <FetchDialog
                open={resendDialog.open}
                onOpenChange={(open) =>
                    setResendDialog((prev) => ({ ...prev, open }))
                }
                fetchFn={(onConfirm) =>
                    resendDialog.type === "unsent"
                        ? resendUnsent(resendDialog.surveyId, onConfirm).then(
                              (data) => ({ data }),
                          )
                        : resendUnanswered(
                              resendDialog.surveyId,
                              onConfirm,
                          ).then((data) => ({ data }))
                }
                requestInfo={
                    resendDialog.type === "unsent"
                        ? prepareResendUnsent(resendDialog.surveyId)
                        : prepareResendUnanswered(resendDialog.surveyId)
                }
                title={
                    resendDialog.type === "unsent"
                        ? "Retry Unsent Emails"
                        : "Send Reminder"
                }
                description={
                    resendDialog.type === "unsent"
                        ? `Retry sending emails that failed for survey #${resendDialog.surveyId}?`
                        : `Send a reminder to all users who haven't responded to survey #${resendDialog.surveyId}?`
                }
                proceedLabel={resendDialog.type === "unsent" ? "Retry" : "Send"}
                successDescription={
                    resendDialog.type === "unsent"
                        ? "Unsent emails have been retried."
                        : "Reminders have been sent."
                }
                onSuccess={loadSurveys}
            />

            {/* Dialog showing responses for one category (e.g positive responses */}
            <FetchDialog
                open={responseDialog.open}
                onOpenChange={(open) =>
                    setResponseDialog((prev) => ({ ...prev, open }))
                }
                fetchFn={() =>
                    fetchSurveyResponses(
                        responseDialog.surveyId,
                        responseDialog.category,
                    ).then((data) => ({ data }))
                }
                requestInfo={prepareFetchSurveyResponses(
                    responseDialog.surveyId,
                    responseDialog.category,
                )}
                immediate
                showIcon={false}
                title={RESPONSE_TITLES[responseDialog.category]}
                successContent={(data) => {
                    const items = data as string[];
                    if (items.length === 0) {
                        return (
                            <p className="py-4 text-center text-sm text-muted-foreground">
                                No entries.
                            </p>
                        );
                    }
                    return (
                        <ul className="space-y-1">
                            {items.map((item, i) => (
                                <li
                                    key={i}
                                    className="rounded px-2 py-1 font-mono text-sm even:bg-muted/50"
                                >
                                    {item}
                                </li>
                            ))}
                        </ul>
                    );
                }}
                successDescription={null}
            />
        </div>
    );
}
