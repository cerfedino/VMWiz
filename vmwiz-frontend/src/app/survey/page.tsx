import { Suspense } from "react";
import { SurveyForm } from "@/components/surveys/survey-form";
import { Loader2 } from "lucide-react";
import type { Metadata } from "next";

export const metadata: Metadata = {
    title: "VM Usage Survey - VMWiz",
    description: "VSOS VM usage survey",
};

function SurveyLoading() {
    return (
        <div className="flex min-h-[70vh] items-center justify-center">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
    );
}

export default function SurveyPage() {
    return (
        <Suspense fallback={<SurveyLoading />}>
            <SurveyForm />
        </Suspense>
    );
}
