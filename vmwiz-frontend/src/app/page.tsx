import { Suspense } from "react";
import { VMRequestFormProvider } from "@/context/vm-request-form";
import { VMRequestForm } from "@/components/vm-request-form";

export default function Home() {
    return (
        <Suspense>
            <VMRequestFormProvider>
                <VMRequestForm />
            </VMRequestFormProvider>
        </Suspense>
    );
}
